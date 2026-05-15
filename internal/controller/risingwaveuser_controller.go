/*
Copyright 2025 RisingWave Labs.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
	"github.com/risingwavelabs/risingwave-resource-operator/internal/constants"
	"github.com/risingwavelabs/risingwave-resource-operator/internal/metrics"
	"github.com/risingwavelabs/risingwave-resource-operator/internal/rwclient"
	"github.com/risingwavelabs/risingwave-resource-operator/internal/utils"
)

const (
	risingWaveUserFinalizer = "risingwaveuser.risingwave.risingwavelabs.com/finalizer"
)

// RisingWaveUserReconciler reconciles a RisingWaveUser object.
type RisingWaveUserReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	ConnectionPool *rwclient.Pool
}

//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveusers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile reconciles a RisingWaveUser object.
func (r *RisingWaveUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("RisingWaveUserReconciler")

	// 1. Fetch RisingWaveUser
	rwUser := &v1alpha1.RisingWaveUser{}
	if err := r.Get(ctx, req.NamespacedName, rwUser); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("RisingWaveUser not found, skipping", "namespacedName", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get RisingWaveUser")
		return ctrl.Result{}, err
	}
	logger.Info("Fetched RisingWaveUser", "name", rwUser.Name, "namespace", rwUser.Namespace)

	// 2. Check pause annotation
	if rwUser.GetAnnotations()[constants.PauseReconcileAnnotation] == "true" {
		logger.Info("Reconciliation paused via annotation")
		return ctrl.Result{}, nil
	}

	// 3. Build connection key and info
	connRef := rwUser.Spec.ConnectionRef
	port := connRef.Port
	if port == 0 {
		port = 4567
	}

	adminUser, adminPassword, err := r.resolveAdminCredentials(ctx, rwUser)
	if err != nil {
		logger.Error(err, "Failed to resolve admin credentials")
		rwUser.Status.Phase = constants.PhaseNotReady
		rwUser.Status.Reason = constants.ReasonFailedToGetSecret
		if serr := r.Status().Update(ctx, rwUser); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	connKey := rwclient.ConnectionKeyFrom(rwUser.Namespace, connRef.Host, port)
	connInfo := rwclient.DefaultConnectionInfo(connRef.Host, port, adminUser, adminPassword)

	db, err := r.ConnectionPool.Get(ctx, connKey, connInfo)
	if err != nil {
		logger.Error(err, "Failed to connect to RisingWave", "host", connRef.Host, "port", port)
		rwUser.Status.Phase = constants.PhaseNotReady
		rwUser.Status.Reason = constants.ReasonConnectionFailed
		if serr := r.Status().Update(ctx, rwUser); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	logger.Info("Connected to RisingWave", "host", connRef.Host, "port", port)

	// 4. Handle finalizer
	finalizerResult, finalizerErr := utils.HandleFinalizer(utils.FinalizerParams{
		Object:    rwUser,
		Context:   ctx,
		Client:    r.Client,
		Finalizer: risingWaveUserFinalizer,
		FinalizationFunc: func() error {
			return r.finalizeRisingWaveUser(ctx, db, rwUser, connKey)
		},
		OnFailure: func(err error) error {
			rwUser.Status.Phase = constants.PhaseNotReady
			rwUser.Status.Reason = constants.ReasonFailedToFinalize
			if serr := r.Status().Update(ctx, rwUser); serr != nil {
				logger.Error(serr, "Failed to update finalization status")
				return serr
			}
			return nil
		},
	})

	if finalizerErr != nil || !rwUser.GetDeletionTimestamp().IsZero() {
		return finalizerResult, finalizerErr
	}

	// Effective user name
	userName := rwUser.Spec.Name
	if userName == "" {
		userName = rwUser.Name
	}

	// 5. Determine password (only for password auth)
	rotatePassword := rwUser.GetAnnotations()[constants.RotatePasswordAnnotation] == "true"
	secretName := fmt.Sprintf("risingwave-user-%s", rwUser.Name)

	var password string
	var managedSecret bool
	if r.isPasswordAuth(rwUser) {
		password, managedSecret, err = r.resolveUserPassword(ctx, rwUser, secretName, rotatePassword)
		if err != nil {
			logger.Error(err, "Failed to resolve user password")
			rwUser.Status.Phase = constants.PhaseNotReady
			rwUser.Status.Reason = constants.ReasonFailedToGetSecret
			if serr := r.Status().Update(ctx, rwUser); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
	} else {
		// OAuth/LDAP auth - no password needed
		password = ""
		managedSecret = false
	}

	// 6. Ensure user exists in RisingWave
	userExists, err := r.checkUserExists(ctx, db, userName)
	if err != nil {
		logger.Error(err, "Failed to check if user exists", "user", userName)
		return ctrl.Result{}, err
	}

	if !userExists {
		createSQL, err := r.buildCreateUserSQL(rwUser, userName)
		if err != nil {
			logger.Error(err, "Failed to build CREATE USER statement", "user", userName)
			rwUser.Status.Phase = constants.PhaseNotReady
			rwUser.Status.Reason = constants.ReasonFailedToCreateUser
			if serr := r.Status().Update(ctx, rwUser); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		logger.V(1).Info("Creating user", "sql", redactSQL(createSQL))
		if _, err := db.ExecContext(ctx, createSQL); err != nil {
			logger.Error(err, "Failed to create user", "user", userName)
			rwUser.Status.Phase = constants.PhaseNotReady
			rwUser.Status.Reason = constants.ReasonFailedToCreateUser
			if serr := r.Status().Update(ctx, rwUser); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		logger.Info("Created user", "user", userName)
		rwUser.Status.UserCreated = true
		metrics.RisingWaveUserCreatedTotal.Increment()
	} else {
		rwUser.Status.UserCreated = true
		// Password rotation only applies to password auth type
		if rotatePassword && r.isPasswordAuth(rwUser) {
			alterSQL := rwclient.BuildAlterUserPasswordSQL(userName, password)
			logger.V(1).Info("Updating user password", "sql", redactSQL(alterSQL))
			if _, err := db.ExecContext(ctx, alterSQL); err != nil {
				logger.Error(err, "Failed to update user password", "user", userName)
				rwUser.Status.Phase = constants.PhaseNotReady
				rwUser.Status.Reason = constants.ReasonFailedToUpdatePassword
				if serr := r.Status().Update(ctx, rwUser); serr != nil {
					return ctrl.Result{RequeueAfter: time.Second}, nil
				}
				return ctrl.Result{}, err
			}
			logger.Info("Updated user password", "user", userName)
		}
	}

	// 7. Reconcile user-level permissions
	if len(rwUser.Spec.Permissions) > 0 {
		alterSQL := rwclient.BuildAlterUserPermissionsSQL(userName, rwUser.Spec.Permissions)
		if alterSQL != "" {
			if _, err := db.ExecContext(ctx, alterSQL); err != nil {
				logger.Error(err, "Failed to update user permissions", "user", userName)
				rwUser.Status.Phase = constants.PhaseNotReady
				rwUser.Status.Reason = constants.ReasonFailedToGrant
				if serr := r.Status().Update(ctx, rwUser); serr != nil {
					return ctrl.Result{RequeueAfter: time.Second}, nil
				}
				return ctrl.Result{}, err
			}
			logger.Info("Updated user permissions", "user", userName)
		}
	}

	// 8. Sync privileges
	if rwUser.Spec.Grants != nil {
		if err := r.syncPrivileges(ctx, db, rwUser, userName); err != nil {
			logger.Error(err, "Failed to sync privileges", "user", userName)
			rwUser.Status.Phase = constants.PhaseNotReady
			rwUser.Status.Reason = constants.ReasonFailedToGrant
			if serr := r.Status().Update(ctx, rwUser); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		rwUser.Status.PrivilegesSynced = true
		logger.Info("Synced privileges", "user", userName)
	}

	// 9. Create/Update K8s Secret with credentials (only if we manage the password)
	if managedSecret {
		if err := r.ensureCredentialSecret(ctx, rwUser, secretName, userName, password); err != nil {
			logger.Error(err, "Failed to ensure credential secret")
			rwUser.Status.Phase = constants.PhaseNotReady
			rwUser.Status.Reason = constants.ReasonFailedToCreateSecret
			if serr := r.Status().Update(ctx, rwUser); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		rwUser.Status.SecretCreated = true
		rwUser.Status.SecretName = secretName
		logger.Info("Ensured credential secret", "secret", secretName)
	}

	// 10. Remove rotate-password annotation if present
	if rotatePassword {
		patch := client.MergeFrom(rwUser.DeepCopy())
		annotations := rwUser.GetAnnotations()
		if annotations == nil {
			annotations = make(map[string]string)
		}
		delete(annotations, constants.RotatePasswordAnnotation)
		rwUser.SetAnnotations(annotations)
		if patchErr := r.Patch(ctx, rwUser, patch); patchErr != nil {
			logger.Error(patchErr, "Failed to remove rotate-password annotation")
		}
	}

	// 11. Update status to Ready
	rwUser.Status.Phase = constants.PhaseReady
	rwUser.Status.Reason = constants.ReasonCompleted
	rwUser.Status.ObservedGeneration = rwUser.Generation
	if serr := r.Status().Update(ctx, rwUser); serr != nil {
		logger.Error(serr, "Failed to update RisingWaveUser status")
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: constants.ReconciliationPeriod}, nil
}

// resolveAdminCredentials resolves the admin username and password for connecting to RisingWave.
func (r *RisingWaveUserReconciler) resolveAdminCredentials(ctx context.Context, rwUser *v1alpha1.RisingWaveUser) (string, string, error) {
	connRef := rwUser.Spec.ConnectionRef
	adminUser := "root"
	adminPassword := ""

	if connRef.Credentials != nil {
		if connRef.Credentials.Username != "" {
			adminUser = connRef.Credentials.Username
		}

		if connRef.Credentials.PasswordSecretRef != nil {
			secretKey := types.NamespacedName{
				Namespace: connRef.Credentials.PasswordSecretRef.Namespace,
				Name:      connRef.Credentials.PasswordSecretRef.Name,
			}
			if secretKey.Namespace == "" {
				secretKey.Namespace = rwUser.Namespace
			}
			adminSecret := &v1.Secret{}
			if err := r.Get(ctx, secretKey, adminSecret); err != nil {
				return "", "", fmt.Errorf("failed to get admin credentials secret: %w", err)
			}
			pwdKey := connRef.Credentials.PasswordSecretRef.Key
			if pwdKey == "" {
				pwdKey = "password"
			}
			adminPassword = string(adminSecret.Data[pwdKey])
		} else {
			adminPassword = connRef.Credentials.Password
		}
	}

	return adminUser, adminPassword, nil
}

// resolveUserPassword determines the password to use for the user.
// Returns (password, managedSecret, error) where managedSecret indicates we should create/update a K8s Secret.
func (r *RisingWaveUserReconciler) resolveUserPassword(ctx context.Context, rwUser *v1alpha1.RisingWaveUser, secretName string, rotatePassword bool) (string, bool, error) {
	if rwUser.Spec.Password != nil && rwUser.Spec.Password.SecretRef != nil {
		// Use externally managed secret
		pwdSecretKey := types.NamespacedName{
			Namespace: rwUser.Spec.Password.SecretRef.Namespace,
			Name:      rwUser.Spec.Password.SecretRef.Name,
		}
		if pwdSecretKey.Namespace == "" {
			pwdSecretKey.Namespace = rwUser.Namespace
		}
		pwdSecret := &v1.Secret{}
		if err := r.Get(ctx, pwdSecretKey, pwdSecret); err != nil {
			return "", false, fmt.Errorf("failed to get password secret: %w", err)
		}
		pwdKey := rwUser.Spec.Password.SecretRef.Key
		if pwdKey == "" {
			pwdKey = "password"
		}
		return string(pwdSecret.Data[pwdKey]), false, nil
	}

	// Auto-generate or reuse existing managed secret
	existingSecret := &v1.Secret{}
	secretKey := types.NamespacedName{Namespace: rwUser.Namespace, Name: secretName}
	if err := r.Get(ctx, secretKey, existingSecret); err == nil && !rotatePassword {
		// Reuse existing generated password
		return string(existingSecret.Data["password"]), true, nil
	}

	// Generate new password
	length := 16
	if rwUser.Spec.Password != nil && rwUser.Spec.Password.GenerateRandomLength != nil {
		length = int(*rwUser.Spec.Password.GenerateRandomLength)
	}
	return utils.GenerateRandomPassword(length), true, nil
}

// checkUserExists checks if a user exists in RisingWave.
func (r *RisingWaveUserReconciler) checkUserExists(ctx context.Context, db *sql.DB, userName string) (bool, error) {
	row := db.QueryRowContext(ctx, "SELECT name FROM rw_catalog.rw_users WHERE name = $1", userName)
	var foundName string
	if err := row.Scan(&foundName); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// finalizeRisingWaveUser drops the user from RisingWave.
func (r *RisingWaveUserReconciler) finalizeRisingWaveUser(ctx context.Context, db *sql.DB, rwUser *v1alpha1.RisingWaveUser, connKey rwclient.ConnectionKey) error {
	logger := log.FromContext(ctx).WithName("RisingWaveUserReconciler").WithValues("user", rwUser.Name)

	if rwUser.Status.UserCreated {
		userName := rwUser.Spec.Name
		if userName == "" {
			userName = rwUser.Name
		}
		dropSQL := rwclient.BuildDropUserSQL(userName)

		if _, err := db.ExecContext(ctx, dropSQL); err != nil {
			logger.Error(err, "Failed to drop user", "sql", dropSQL)
			return err
		}
		logger.Info("Dropped user", "user", userName)
		metrics.RisingWaveUserDeletedTotal.Increment()

		// Remove connection from pool
		r.ConnectionPool.Remove(connKey)
	}

	return nil
}

// syncPrivileges uses snapshot-based reconciliation to grant/revoke privileges.
//
//nolint:gocyclo // Complex privilege sync logic across 8 object types; refactoring would reduce readability
func (r *RisingWaveUserReconciler) syncPrivileges(ctx context.Context, db *sql.DB, rwUser *v1alpha1.RisingWaveUser, userName string) error {
	logger := log.FromContext(ctx).WithName("syncPrivileges").WithValues("user", userName)

	// If no grants specified, we're done
	if rwUser.Spec.Grants == nil {
		logger.Info("No grants specified, skipping privilege sync")
		return nil
	}

	// Fetch actual privilege snapshot from RisingWave
	actualSnapshot, err := rwclient.FetchPrivilegeSnapshotForUser(ctx, db, userName)
	if err != nil {
		return fmt.Errorf("failed to fetch privilege snapshot: %w", err)
	}

	logger.V(1).Info("Fetched actual privilege snapshot", "databases", len(actualSnapshot.Databases))

	// Build a map of actual databases for quick lookup
	actualDBMap := make(map[string]rwclient.DatabasePrivilegeSnapshot)
	for _, dbSnap := range actualSnapshot.Databases {
		actualDBMap[dbSnap.Name] = dbSnap
	}

	// Group statements by database for proper execution context
	type dbStatements struct {
		database   string
		statements []string
	}
	var statementsByDB = make(map[string]*dbStatements)
	var globalStatements []string // Database-level statements (can be executed from any DB)

	// Process each database in the spec
	for _, dbSpec := range rwUser.Spec.Grants.Databases {
		actualDB, hasActual := actualDBMap[dbSpec.Name]

		// Calculate database-level diff
		actualDBPrivs := rwclient.DatabasePrivilegeSnapshot{
			Name:            dbSpec.Name,
			Privileges:      []string{},
			WithGrantOption: false,
		}
		if hasActual {
			actualDBPrivs.Privileges = actualDB.Privileges
			actualDBPrivs.WithGrantOption = actualDB.WithGrantOption
		}
		dbDiff := rwclient.CalculateDatabaseDiff(userName, actualDBPrivs, &dbSpec)
		// Database-level statements can be executed from any database context
		globalStatements = append(globalStatements, dbDiff.ToRevoke...)
		globalStatements = append(globalStatements, dbDiff.ToGrant...)

		// Initialize statements group for this database
		if _, exists := statementsByDB[dbSpec.Name]; !exists {
			statementsByDB[dbSpec.Name] = &dbStatements{database: dbSpec.Name}
		}

		// Build map of actual schemas for this database
		actualSchemaMap := make(map[string]rwclient.SchemaPrivilegeSnapshot)
		if hasActual {
			for _, schemaSnap := range actualDB.Schemas {
				actualSchemaMap[schemaSnap.Name] = schemaSnap
			}
		}

		// Process each schema in the spec
		for _, schemaSpec := range dbSpec.Schemas {
			actualSchema, hasActualSchema := actualSchemaMap[schemaSpec.Name]

			// Calculate schema-level diff
			actualSchemaPrivs := rwclient.SchemaPrivilegeSnapshot{
				Name:            schemaSpec.Name,
				Privileges:      []string{},
				WithGrantOption: false,
			}
			if hasActualSchema {
				actualSchemaPrivs.Privileges = actualSchema.Privileges
				actualSchemaPrivs.WithGrantOption = actualSchema.WithGrantOption
			}
			schemaDiff := rwclient.CalculateSchemaDiff(userName, actualSchemaPrivs, &schemaSpec)
			statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, schemaDiff.ToRevoke...)
			statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, schemaDiff.ToGrant...)

			// Revoke orphaned objects (objects that exist in actual but not in spec)
			if hasActualSchema {
				// Revoke orphaned tables
				actualTablesMap := make(map[string]rwclient.ObjectPrivilege)
				for _, t := range actualSchema.Tables {
					actualTablesMap[t.Name] = t
				}
				hasWildcardTable := false
				for _, tableSpec := range schemaSpec.Tables {
					if tableSpec.Name == "*" {
						hasWildcardTable = true
					}
					delete(actualTablesMap, tableSpec.Name)
				}
				for _, orphanTable := range actualTablesMap {
					if orphanTable.Name == "*" {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON ALL TABLES IN SCHEMA %s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteUser(userName)))
					} else if !hasWildcardTable {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON TABLE %s.%s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteIdentifier(orphanTable.Name),
								rwclient.QuoteUser(userName)))
					}
				}

				// Revoke orphaned views
				actualViewsMap := make(map[string]rwclient.ObjectPrivilege)
				for _, v := range actualSchema.Views {
					actualViewsMap[v.Name] = v
				}
				hasWildcardView := false
				for _, viewSpec := range schemaSpec.Views {
					if viewSpec.Name == "*" {
						hasWildcardView = true
					}
					delete(actualViewsMap, viewSpec.Name)
				}
				for _, orphanView := range actualViewsMap {
					if orphanView.Name == "*" {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON ALL VIEWS IN SCHEMA %s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteUser(userName)))
					} else if !hasWildcardView {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON VIEW %s.%s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteIdentifier(orphanView.Name),
								rwclient.QuoteUser(userName)))
					}
				}

				// Revoke orphaned materialized views
				actualMVsMap := make(map[string]rwclient.ObjectPrivilege)
				for _, mv := range actualSchema.MaterializedViews {
					actualMVsMap[mv.Name] = mv
				}
				hasWildcardMV := false
				for _, mvSpec := range schemaSpec.MaterializedViews {
					if mvSpec.Name == "*" {
						hasWildcardMV = true
					}
					delete(actualMVsMap, mvSpec.Name)
				}
				for _, orphanMV := range actualMVsMap {
					if orphanMV.Name == "*" {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON ALL MATERIALIZED VIEWS IN SCHEMA %s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteUser(userName)))
					} else if !hasWildcardMV {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON MATERIALIZED VIEW %s.%s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteIdentifier(orphanMV.Name),
								rwclient.QuoteUser(userName)))
					}
				}

				// Revoke orphaned sources
				actualSourcesMap := make(map[string]rwclient.ObjectPrivilege)
				for _, s := range actualSchema.Sources {
					actualSourcesMap[s.Name] = s
				}
				hasWildcardSource := false
				for _, sourceSpec := range schemaSpec.Sources {
					if sourceSpec.Name == "*" {
						hasWildcardSource = true
					}
					delete(actualSourcesMap, sourceSpec.Name)
				}
				for _, orphanSource := range actualSourcesMap {
					if orphanSource.Name == "*" {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON ALL SOURCES IN SCHEMA %s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteUser(userName)))
					} else if !hasWildcardSource {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON SOURCE %s.%s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteIdentifier(orphanSource.Name),
								rwclient.QuoteUser(userName)))
					}
				}

				// Revoke orphaned sinks
				actualSinksMap := make(map[string]rwclient.ObjectPrivilege)
				for _, s := range actualSchema.Sinks {
					actualSinksMap[s.Name] = s
				}
				hasWildcardSink := false
				for _, sinkSpec := range schemaSpec.Sinks {
					if sinkSpec.Name == "*" {
						hasWildcardSink = true
					}
					delete(actualSinksMap, sinkSpec.Name)
				}
				for _, orphanSink := range actualSinksMap {
					if orphanSink.Name == "*" {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON ALL SINKS IN SCHEMA %s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteUser(userName)))
					} else if !hasWildcardSink {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON SINK %s.%s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteIdentifier(orphanSink.Name),
								rwclient.QuoteUser(userName)))
					}
				}

				// Revoke orphaned secrets
				actualSecretsMap := make(map[string]rwclient.ObjectPrivilege)
				for _, s := range actualSchema.Secrets {
					actualSecretsMap[s.Name] = s
				}
				hasWildcardSecret := false
				for _, secretSpec := range schemaSpec.Secrets {
					if secretSpec.Name == "*" {
						hasWildcardSecret = true
					}
					delete(actualSecretsMap, secretSpec.Name)
				}
				for _, orphanSecret := range actualSecretsMap {
					if orphanSecret.Name == "*" {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON ALL SECRETS IN SCHEMA %s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteUser(userName)))
					} else if !hasWildcardSecret {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON SECRET %s.%s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteIdentifier(orphanSecret.Name),
								rwclient.QuoteUser(userName)))
					}
				}

				// Revoke orphaned functions
				actualFuncsMap := make(map[string]rwclient.ObjectPrivilege)
				for _, f := range actualSchema.Functions {
					actualFuncsMap[f.Name] = f
				}
				hasWildcardFunc := false
				for _, funcSpec := range schemaSpec.Functions {
					if funcSpec.Name == "*" {
						hasWildcardFunc = true
					}
					delete(actualFuncsMap, funcSpec.Name)
				}
				for _, orphanFunc := range actualFuncsMap {
					if orphanFunc.Name == "*" {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON ALL FUNCTIONS IN SCHEMA %s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteUser(userName)))
					} else if !hasWildcardFunc {
						statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements,
							fmt.Sprintf("REVOKE ALL ON FUNCTION %s.%s FROM %s",
								rwclient.QuoteIdentifier(schemaSpec.Name),
								rwclient.QuoteIdentifier(orphanFunc.Name),
								rwclient.QuoteUser(userName)))
					}
				}
			}

			// Process tables
			for _, tableSpec := range schemaSpec.Tables {
				actualObj := findObjectPrivilege(actualSchema.Tables, tableSpec.Name)
				desiredPrivs := privilegeSliceToString(tableSpec.Privileges)
				objType := "TABLE"
				if tableSpec.Name == "*" {
					objType = "ALL TABLES IN SCHEMA"
				}
				tableDiff := rwclient.CalculateObjectDiff(userName, dbSpec.Name, schemaSpec.Name, objType, actualObj, tableSpec.Name, desiredPrivs)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, tableDiff.ToRevoke...)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, tableDiff.ToGrant...)
			}

			// Process views
			for _, viewSpec := range schemaSpec.Views {
				actualObj := findObjectPrivilege(actualSchema.Views, viewSpec.Name)
				desiredPrivs := privilegeSliceToString(viewSpec.Privileges)
				objType := "VIEW"
				if viewSpec.Name == "*" {
					objType = "ALL VIEWS IN SCHEMA"
				}
				viewDiff := rwclient.CalculateObjectDiff(userName, dbSpec.Name, schemaSpec.Name, objType, actualObj, viewSpec.Name, desiredPrivs)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, viewDiff.ToRevoke...)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, viewDiff.ToGrant...)
			}

			// Process materialized views
			for _, mvSpec := range schemaSpec.MaterializedViews {
				actualObj := findObjectPrivilege(actualSchema.MaterializedViews, mvSpec.Name)
				desiredPrivs := privilegeSliceToString(mvSpec.Privileges)
				objType := "MATERIALIZED VIEW"
				if mvSpec.Name == "*" {
					objType = "ALL MATERIALIZED VIEWS IN SCHEMA"
				}
				mvDiff := rwclient.CalculateObjectDiff(userName, dbSpec.Name, schemaSpec.Name, objType, actualObj, mvSpec.Name, desiredPrivs)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, mvDiff.ToRevoke...)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, mvDiff.ToGrant...)
			}

			// Process sources
			for _, sourceSpec := range schemaSpec.Sources {
				actualObj := findObjectPrivilege(actualSchema.Sources, sourceSpec.Name)
				desiredPrivs := privilegeSliceToString(sourceSpec.Privileges)
				objType := "SOURCE"
				if sourceSpec.Name == "*" {
					objType = "ALL SOURCES IN SCHEMA"
				}
				sourceDiff := rwclient.CalculateObjectDiff(userName, dbSpec.Name, schemaSpec.Name, objType, actualObj, sourceSpec.Name, desiredPrivs)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, sourceDiff.ToRevoke...)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, sourceDiff.ToGrant...)
			}

			// Process sinks
			for _, sinkSpec := range schemaSpec.Sinks {
				actualObj := findObjectPrivilege(actualSchema.Sinks, sinkSpec.Name)
				desiredPrivs := privilegeSliceToString(sinkSpec.Privileges)
				objType := "SINK"
				if sinkSpec.Name == "*" {
					objType = "ALL SINKS IN SCHEMA"
				}
				sinkDiff := rwclient.CalculateObjectDiff(userName, dbSpec.Name, schemaSpec.Name, objType, actualObj, sinkSpec.Name, desiredPrivs)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, sinkDiff.ToRevoke...)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, sinkDiff.ToGrant...)
			}

			// Process secrets
			for _, secretSpec := range schemaSpec.Secrets {
				actualObj := findObjectPrivilege(actualSchema.Secrets, secretSpec.Name)
				desiredPrivs := privilegeSliceToString(secretSpec.Privileges)
				objType := "SECRET"
				if secretSpec.Name == "*" {
					objType = "ALL SECRETS IN SCHEMA"
				}
				secretDiff := rwclient.CalculateObjectDiff(userName, dbSpec.Name, schemaSpec.Name, objType, actualObj, secretSpec.Name, desiredPrivs)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, secretDiff.ToRevoke...)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, secretDiff.ToGrant...)
			}

			// Process functions
			for _, funcSpec := range schemaSpec.Functions {
				actualObj := findObjectPrivilege(actualSchema.Functions, funcSpec.Name)
				desiredPrivs := privilegeSliceToString(funcSpec.Privileges)
				objType := "FUNCTION"
				if funcSpec.Name == "*" {
					objType = "ALL FUNCTIONS IN SCHEMA"
				}
				funcDiff := rwclient.CalculateObjectDiff(userName, dbSpec.Name, schemaSpec.Name, objType, actualObj, funcSpec.Name, desiredPrivs)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, funcDiff.ToRevoke...)
				statementsByDB[dbSpec.Name].statements = append(statementsByDB[dbSpec.Name].statements, funcDiff.ToGrant...)
			}
		}
	}

	// Handle orphaned databases (databases in actual but not in spec)
	for dbName, actualDB := range actualDBMap {
		found := false
		for _, dbSpec := range rwUser.Spec.Grants.Databases {
			if dbSpec.Name == dbName {
				found = true
				break
			}
		}
		if !found && len(actualDB.Privileges) > 0 {
			// Revoke all database-level privileges
			globalStatements = append(globalStatements,
				fmt.Sprintf("REVOKE ALL ON DATABASE %s FROM %s",
					rwclient.QuoteIdentifier(dbName),
					rwclient.QuoteUser(userName)))
		}
	}

	// Execute statements in proper database context
	// 1. Execute global statements (database-level) first - can be executed from any DB
	// 2. For each database, execute USE <database> then that database's object-level statements

	var allStatementsToSort []string
	allStatementsToSort = append(allStatementsToSort, globalStatements...)

	// Collect all database-specific statements for sorting
	for _, dbStmts := range statementsByDB {
		allStatementsToSort = append(allStatementsToSort, dbStmts.statements...)
	}

	// Sort to put REVOKEs before GRANTs
	sort.Slice(allStatementsToSort, func(i, j int) bool {
		isRevokeI := strings.HasPrefix(allStatementsToSort[i], "REVOKE")
		isRevokeJ := strings.HasPrefix(allStatementsToSort[j], "REVOKE")
		if isRevokeI && !isRevokeJ {
			return true
		}
		if !isRevokeI && isRevokeJ {
			return false
		}
		return allStatementsToSort[i] < allStatementsToSort[j]
	})

	if len(allStatementsToSort) > 0 {
		logger.Info("Executing privilege changes", "count", len(allStatementsToSort), "databases", len(statementsByDB))

		// Track current database to avoid unnecessary USE statements
		currentDB := ""

		for i, stmt := range allStatementsToSort {
			// Determine which database this statement belongs to
			targetDB := ""
			for dbName, dbStmts := range statementsByDB {
				for _, s := range dbStmts.statements {
					if s == stmt {
						targetDB = dbName
						break
					}
				}
				if targetDB != "" {
					break
				}
			}

			// If this is a database-specific statement and we're not in that database, switch
			if targetDB != "" && targetDB != currentDB {
				useStmt := fmt.Sprintf("USE %s", rwclient.QuoteIdentifier(targetDB))
				logger.V(1).Info("Switching database context", "statement", redactSQL(useStmt))
				if _, err := db.ExecContext(ctx, useStmt); err != nil {
					logger.Error(err, "Failed to switch database", "database", targetDB)
					// Continue anyway - might work if already in that DB
				} else {
					currentDB = targetDB
				}
			}

			logger.Info("Executing privilege statement",
				"statement", redactSQL(stmt),
				"index", i+1,
				"total", len(allStatementsToSort))

			_, err := db.ExecContext(ctx, stmt)
			if err != nil {
				// Log but don't fail for idempotency
				errMsg := strings.ToLower(err.Error())
				if strings.Contains(errMsg, "does not exist") ||
					strings.Contains(errMsg, "not found") ||
					strings.Contains(errMsg, "must be owner") {
					logger.V(1).Info("Statement failed (expected for idempotency)", "statement", redactSQL(stmt), "error", err)
				} else {
					return fmt.Errorf("failed to execute statement %d/%d: %s: %w", i+1, len(allStatementsToSort), redactSQL(stmt), err)
				}
			}
		}

		logger.Info("Successfully executed all privilege statements", "count", len(allStatementsToSort))
	} else {
		logger.Info("No privilege changes needed", "user", userName)
	}

	return nil
}

// findObjectPrivilege finds an object privilege by name in a slice.
func findObjectPrivilege(objects []rwclient.ObjectPrivilege, name string) rwclient.ObjectPrivilege {
	for _, obj := range objects {
		if obj.Name == name {
			return obj
		}
	}
	return rwclient.ObjectPrivilege{
		Name:            name,
		Privileges:      []string{},
		WithGrantOption: false,
	}
}

// privilegeSliceToString converts a slice of privilege types to a slice of strings.
func privilegeSliceToString[T ~string](privs []T) []string {
	result := make([]string, len(privs))
	for i, p := range privs {
		result[i] = string(p)
	}
	return result
}

// buildCreateUserSQL builds the appropriate CREATE USER statement based on auth type.
func (r *RisingWaveUserReconciler) buildCreateUserSQL(rwUser *v1alpha1.RisingWaveUser, userName string) (string, error) {
	authType := v1alpha1.AuthTypePassword
	if rwUser.Spec.Auth != nil && rwUser.Spec.Auth.Type != nil {
		authType = *rwUser.Spec.Auth.Type
	}

	switch authType {
	case v1alpha1.AuthTypeOAuth:
		if rwUser.Spec.Auth != nil && rwUser.Spec.Auth.OAuth != nil {
			return rwclient.BuildCreateUserWithOAuthSQL(userName, rwUser.Spec.Auth.OAuth), nil
		}
		return rwclient.BuildCreateUserWithOAuthSQL(userName, &v1alpha1.OAuthConfig{}), nil

	case v1alpha1.AuthTypeLDAP:
		if rwUser.Spec.Auth != nil && rwUser.Spec.Auth.LDAP != nil {
			return rwclient.BuildCreateUserWithLDAPSQL(userName, rwUser.Spec.Auth.LDAP), nil
		}
		return rwclient.BuildCreateUserWithLDAPSQL(userName, &v1alpha1.LDAPConfig{}), nil

	case v1alpha1.AuthTypePassword:
		fallthrough
	default:
		// Password auth - resolve password first
		secretName := fmt.Sprintf("risingwave-user-%s", rwUser.Name)
		password, _, err := r.resolveUserPassword(context.Background(), rwUser, secretName, false)
		if err != nil {
			return "", fmt.Errorf("failed to resolve password: %w", err)
		}
		return rwclient.BuildCreateUserSQL(rwUser, password), nil
	}
}

// isPasswordAuth returns true if the user uses password authentication.
func (r *RisingWaveUserReconciler) isPasswordAuth(rwUser *v1alpha1.RisingWaveUser) bool {
	if rwUser.Spec.Auth == nil || rwUser.Spec.Auth.Type == nil {
		return true // Default to password auth
	}
	return *rwUser.Spec.Auth.Type == v1alpha1.AuthTypePassword
}

// ensureCredentialSecret creates or updates the K8s Secret with user credentials.
func (r *RisingWaveUserReconciler) ensureCredentialSecret(ctx context.Context, rwUser *v1alpha1.RisingWaveUser, secretName, userName, password string) error {
	secret := &v1.Secret{}
	secretKey := types.NamespacedName{Namespace: rwUser.Namespace, Name: secretName}

	err := r.Get(ctx, secretKey, secret)
	if apierrors.IsNotFound(err) {
		newSecret := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: rwUser.Namespace,
				Labels: map[string]string{
					"app.kubernetes.io/managed-by":       "risingwave-resource-operator",
					"risingwave.risingwavelabs.com/user": rwUser.Name,
				},
			},
			StringData: map[string]string{
				"username": userName,
				"password": password,
			},
		}
		return r.Create(ctx, newSecret)
	}
	if err != nil {
		return err
	}

	// Update existing secret
	secret.StringData = map[string]string{
		"username": userName,
		"password": password,
	}
	return r.Update(ctx, secret)
}

// redactSQL redacts sensitive information (passwords, secrets) from SQL statements for logging.
func redactSQL(sql string) string {
	// Redact PASSWORD '...' clauses using regex
	passwordRegex := regexp.MustCompile(` WITH PASSWORD '[^']*'`)
	result := passwordRegex.ReplaceAllString(sql, " WITH PASSWORD '***'")

	// Redact SECRET names in ON clause (for consistency, though secret names aren't usually sensitive)
	result = strings.ReplaceAll(result, " ON SECRET ", " ON [REDACTED] ")

	return result
}

// SetupWithManager sets up the controller with the Manager.
func (r *RisingWaveUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RisingWaveUser{}).
		Complete(r)
}
