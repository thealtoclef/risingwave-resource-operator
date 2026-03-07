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
	"time"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

const risingWaveSchemaFinalizer = "risingwaveschema.risingwave.risingwavelabs.com/finalizer"

// RisingWaveSchemaReconciler reconciles a RisingWaveSchema object.
type RisingWaveSchemaReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	ConnectionPool *rwclient.Pool
}

//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveschemas,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveschemas/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveschemas/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile reconciles a RisingWaveSchema object.
func (r *RisingWaveSchemaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("RisingWaveSchemaReconciler")

	// 1. Fetch RisingWaveSchema
	rwSchema := &v1alpha1.RisingWaveSchema{}
	if err := r.Get(ctx, req.NamespacedName, rwSchema); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("RisingWaveSchema not found, skipping", "namespacedName", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get RisingWaveSchema")
		return ctrl.Result{}, err
	}
	logger.Info("Fetched RisingWaveSchema", "name", rwSchema.Name, "namespace", rwSchema.Namespace)

	// 2. Check pause annotation
	if rwSchema.GetAnnotations()[constants.PauseReconcileAnnotation] == "true" {
		logger.Info("Reconciliation paused via annotation")
		return ctrl.Result{}, nil
	}

	// 3. Build connection key and info
	connRef := rwSchema.Spec.ConnectionRef
	port := connRef.Port
	if port == 0 {
		port = 4567
	}

	adminUser, adminPassword, err := r.resolveAdminCredentials(ctx, rwSchema)
	if err != nil {
		logger.Error(err, "Failed to resolve admin credentials")
		rwSchema.Status.Phase = constants.PhaseNotReady
		rwSchema.Status.Reason = constants.ReasonFailedToGetSecret
		if serr := r.Status().Update(ctx, rwSchema); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	connKey := rwclient.ConnectionKeyFrom(rwSchema.Namespace, connRef.Host, port)
	connInfo := rwclient.DefaultConnectionInfo(connRef.Host, port, adminUser, adminPassword)

	db, err := r.ConnectionPool.Get(ctx, connKey, connInfo)
	if err != nil {
		logger.Error(err, "Failed to connect to RisingWave", "host", connRef.Host, "port", port)
		rwSchema.Status.Phase = constants.PhaseNotReady
		rwSchema.Status.Reason = constants.ReasonConnectionFailed
		if serr := r.Status().Update(ctx, rwSchema); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	logger.Info("Connected to RisingWave", "host", connRef.Host, "port", port)

	// 4. Handle finalizer
	finalizerResult, finalizerErr := utils.HandleFinalizer(utils.FinalizerParams{
		Object:    rwSchema,
		Context:   ctx,
		Client:    r.Client,
		Finalizer: risingWaveSchemaFinalizer,
		FinalizationFunc: func() error {
			return r.finalizeRisingWaveSchema(ctx, db, rwSchema, connKey)
		},
		OnFailure: func(err error) error {
			rwSchema.Status.Phase = constants.PhaseNotReady
			rwSchema.Status.Reason = constants.ReasonFailedToFinalize
			if serr := r.Status().Update(ctx, rwSchema); serr != nil {
				logger.Error(serr, "Failed to update finalization status")
				return serr
			}
			return nil
		},
	})

	if finalizerErr != nil || !rwSchema.GetDeletionTimestamp().IsZero() {
		return finalizerResult, finalizerErr
	}

	// Effective schema and database names
	dbName := rwSchema.Spec.DatabaseRef.Name
	schemaName := rwSchema.GetSchemaName()

	// 5. Switch to target database
	useSQL := rwclient.BuildUseDatabaseSQL(dbName)
	if _, err := db.ExecContext(ctx, useSQL); err != nil {
		logger.Error(err, "Failed to switch to database", "database", dbName)
		rwSchema.Status.Phase = constants.PhaseNotReady
		rwSchema.Status.Reason = constants.ReasonFailedToCreateSchema
		if serr := r.Status().Update(ctx, rwSchema); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// 6. Ensure schema exists
	schemaExists, err := rwclient.CheckSchemaExists(ctx, db, schemaName)
	if err != nil {
		logger.Error(err, "Failed to check if schema exists", "schema", schemaName)
		return ctrl.Result{}, err
	}

	if !schemaExists {
		var createSQL string
		if rwSchema.Spec.Owner != "" {
			createSQL = rwclient.BuildCreateSchemaWithOwnerSQL(schemaName, rwSchema.Spec.Owner)
		} else {
			createSQL = rwclient.BuildCreateSchemaSQL(schemaName)
		}
		logger.V(1).Info("Creating schema", "sql", createSQL)
		if _, err := db.ExecContext(ctx, createSQL); err != nil {
			logger.Error(err, "Failed to create schema", "schema", schemaName)
			rwSchema.Status.Phase = constants.PhaseNotReady
			rwSchema.Status.Reason = constants.ReasonFailedToCreateSchema
			if serr := r.Status().Update(ctx, rwSchema); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		logger.Info("Created schema", "schema", schemaName, "database", dbName)
		rwSchema.Status.SchemaCreated = true
		metrics.RisingWaveSchemaCreatedTotal.Increment()
	} else {
		rwSchema.Status.SchemaCreated = true
	}

	// 7. Update schema owner if specified
	if rwSchema.Spec.Owner != "" {
		currentOwner, err := rwclient.GetSchemaOwner(ctx, db, schemaName)
		if err != nil {
			logger.Error(err, "Failed to get schema owner", "schema", schemaName)
			rwSchema.Status.Phase = constants.PhaseNotReady
			rwSchema.Status.Reason = constants.ReasonFailedToUpdateOwner
			if serr := r.Status().Update(ctx, rwSchema); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}

		if currentOwner != rwSchema.Spec.Owner {
			alterSQL := rwclient.BuildAlterSchemaOwnerSQL(schemaName, rwSchema.Spec.Owner)
			logger.V(1).Info("Updating schema owner", "sql", alterSQL)
			if _, err := db.ExecContext(ctx, alterSQL); err != nil {
				logger.Error(err, "Failed to update schema owner", "schema", schemaName, "owner", rwSchema.Spec.Owner)
				rwSchema.Status.Phase = constants.PhaseNotReady
				rwSchema.Status.Reason = constants.ReasonFailedToUpdateOwner
				if serr := r.Status().Update(ctx, rwSchema); serr != nil {
					return ctrl.Result{RequeueAfter: time.Second}, nil
				}
				return ctrl.Result{}, err
			}
			logger.Info("Updated schema owner", "schema", schemaName, "owner", rwSchema.Spec.Owner)
		}
	}

	// 8. Update status to Ready
	rwSchema.Status.Phase = constants.PhaseReady
	rwSchema.Status.Reason = constants.ReasonCompleted
	rwSchema.Status.ObservedGeneration = rwSchema.Generation
	if serr := r.Status().Update(ctx, rwSchema); serr != nil {
		logger.Error(serr, "Failed to update RisingWaveSchema status")
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: constants.ReconciliationPeriod}, nil
}

// resolveAdminCredentials resolves the admin username and password for connecting to RisingWave.
func (r *RisingWaveSchemaReconciler) resolveAdminCredentials(ctx context.Context, rwSchema *v1alpha1.RisingWaveSchema) (string, string, error) {
	connRef := rwSchema.Spec.ConnectionRef
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
				secretKey.Namespace = rwSchema.Namespace
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

// finalizeRisingWaveSchema handles the finalization of a RisingWaveSchema.
// Safe-by-default: only drops the schema if deletion-policy is explicitly set to "delete".
func (r *RisingWaveSchemaReconciler) finalizeRisingWaveSchema(
	ctx context.Context, db *sql.DB, rwSchema *v1alpha1.RisingWaveSchema, connKey rwclient.ConnectionKey,
) error {
	logger := log.FromContext(ctx).WithName("RisingWaveSchemaReconciler").WithValues("schema", rwSchema.Name)

	// Safe-by-default: only drop if explicitly requested
	if rwSchema.GetAnnotations()[constants.DeletionPolicyAnnotation] != constants.DeletionPolicyDelete {
		logger.Info("Deletion policy is not 'delete', retaining schema",
			"schema", rwSchema.GetSchemaName(),
			"policy", rwSchema.GetAnnotations()[constants.DeletionPolicyAnnotation])
		r.ConnectionPool.Remove(connKey)
		return nil
	}

	// Never drop the default "public" schema
	schemaName := rwSchema.GetSchemaName()
	if schemaName == constants.DefaultSchemaName {
		logger.Info("Cannot delete default schema, retaining",
			"schema", schemaName,
			"reason", "public schema is created by default and should not be deleted")
		r.ConnectionPool.Remove(connKey)
		return nil
	}

	// Explicit delete requested - switch to database and drop schema
	dbName := rwSchema.Spec.DatabaseRef.Name

	// Switch to database
	useSQL := rwclient.BuildUseDatabaseSQL(dbName)
	if _, err := db.ExecContext(ctx, useSQL); err != nil {
		return fmt.Errorf("failed to switch to database: %w", err)
	}

	// Drop schema with CASCADE
	dropSQL := rwclient.BuildDropSchemaSQL(schemaName)
	if _, err := db.ExecContext(ctx, dropSQL); err != nil {
		return fmt.Errorf("failed to drop schema: %w", err)
	}

	logger.Info("Dropped schema", "schema", schemaName, "database", dbName)
	metrics.RisingWaveSchemaDeletedTotal.Increment()
	r.ConnectionPool.Remove(connKey)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RisingWaveSchemaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RisingWaveSchema{}).
		Complete(r)
}
