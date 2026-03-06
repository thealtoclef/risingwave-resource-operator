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

const risingWaveDatabaseFinalizer = "risingwavedatabase.risingwave.risingwavelabs.com/finalizer"

// RisingWaveDatabaseReconciler reconciles a RisingWaveDatabase object.
type RisingWaveDatabaseReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	ConnectionPool *rwclient.Pool
}

//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwavedatabases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwavedatabases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwavedatabases/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile reconciles a RisingWaveDatabase object.
func (r *RisingWaveDatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("RisingWaveDatabaseReconciler")

	// 1. Fetch RisingWaveDatabase
	rwDB := &v1alpha1.RisingWaveDatabase{}
	if err := r.Get(ctx, req.NamespacedName, rwDB); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("RisingWaveDatabase not found, skipping", "namespacedName", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get RisingWaveDatabase")
		return ctrl.Result{}, err
	}
	logger.Info("Fetched RisingWaveDatabase", "name", rwDB.Name, "namespace", rwDB.Namespace)

	// 2. Check pause annotation
	if rwDB.GetAnnotations()[constants.PauseReconcileAnnotation] == "true" {
		logger.Info("Reconciliation paused via annotation")
		return ctrl.Result{}, nil
	}

	// 3. Build connection key and info
	connRef := rwDB.Spec.ConnectionRef
	port := connRef.Port
	if port == 0 {
		port = 4567
	}

	adminUser, adminPassword, err := r.resolveAdminCredentials(ctx, rwDB)
	if err != nil {
		logger.Error(err, "Failed to resolve admin credentials")
		rwDB.Status.Phase = constants.PhaseNotReady
		rwDB.Status.Reason = constants.ReasonFailedToGetSecret
		if serr := r.Status().Update(ctx, rwDB); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	connKey := rwclient.ConnectionKeyFrom(rwDB.Namespace, connRef.Host, port)
	connInfo := rwclient.DefaultConnectionInfo(connRef.Host, port, adminUser, adminPassword)

	db, err := r.ConnectionPool.Get(ctx, connKey, connInfo)
	if err != nil {
		logger.Error(err, "Failed to connect to RisingWave", "host", connRef.Host, "port", port)
		rwDB.Status.Phase = constants.PhaseNotReady
		rwDB.Status.Reason = constants.ReasonConnectionFailed
		if serr := r.Status().Update(ctx, rwDB); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	logger.Info("Connected to RisingWave", "host", connRef.Host, "port", port)

	// 4. Handle finalizer
	finalizerResult, finalizerErr := utils.HandleFinalizer(utils.FinalizerParams{
		Object:    rwDB,
		Context:   ctx,
		Client:    r.Client,
		Finalizer: risingWaveDatabaseFinalizer,
		FinalizationFunc: func() error {
			return r.finalizeRisingWaveDatabase(ctx, db, rwDB, connKey)
		},
		OnFailure: func(err error) error {
			rwDB.Status.Phase = constants.PhaseNotReady
			rwDB.Status.Reason = constants.ReasonFailedToFinalize
			if serr := r.Status().Update(ctx, rwDB); serr != nil {
				logger.Error(serr, "Failed to update finalization status")
				return serr
			}
			return nil
		},
	})

	if finalizerErr != nil || !rwDB.GetDeletionTimestamp().IsZero() {
		return finalizerResult, finalizerErr
	}

	// Effective database name
	dbName := rwDB.GetDatabaseName()

	// 5. Ensure database exists
	dbExists, err := rwclient.CheckDatabaseExists(ctx, db, dbName)
	if err != nil {
		logger.Error(err, "Failed to check if database exists", "database", dbName)
		return ctrl.Result{}, err
	}

	if !dbExists {
		createSQL := rwclient.BuildCreateDatabaseSQL(dbName)
		logger.V(1).Info("Creating database", "sql", createSQL)
		if _, err := db.ExecContext(ctx, createSQL); err != nil {
			logger.Error(err, "Failed to create database", "database", dbName)
			rwDB.Status.Phase = constants.PhaseNotReady
			rwDB.Status.Reason = constants.ReasonFailedToCreateDatabase
			if serr := r.Status().Update(ctx, rwDB); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		logger.Info("Created database", "database", dbName)
		rwDB.Status.DatabaseCreated = true
		metrics.RisingWaveDatabaseCreatedTotal.Increment()
	} else {
		rwDB.Status.DatabaseCreated = true
	}

	// 6. Update database owner if specified
	if rwDB.Spec.Owner != "" {
		currentOwner, err := rwclient.GetDatabaseOwner(ctx, db, dbName)
		if err != nil {
			logger.Error(err, "Failed to get database owner", "database", dbName)
			rwDB.Status.Phase = constants.PhaseNotReady
			rwDB.Status.Reason = constants.ReasonFailedToUpdateOwner
			if serr := r.Status().Update(ctx, rwDB); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}

		if currentOwner != rwDB.Spec.Owner {
			alterSQL := rwclient.BuildAlterDatabaseOwnerSQL(dbName, rwDB.Spec.Owner)
			logger.V(1).Info("Updating database owner", "sql", alterSQL)
			if _, err := db.ExecContext(ctx, alterSQL); err != nil {
				logger.Error(err, "Failed to update database owner", "database", dbName, "owner", rwDB.Spec.Owner)
				rwDB.Status.Phase = constants.PhaseNotReady
				rwDB.Status.Reason = constants.ReasonFailedToUpdateOwner
				if serr := r.Status().Update(ctx, rwDB); serr != nil {
					return ctrl.Result{RequeueAfter: time.Second}, nil
				}
				return ctrl.Result{}, err
			}
			logger.Info("Updated database owner", "database", dbName, "owner", rwDB.Spec.Owner)
		}
	}

	// 7. Update status to Ready
	rwDB.Status.Phase = constants.PhaseReady
	rwDB.Status.Reason = constants.ReasonCompleted
	rwDB.Status.ObservedGeneration = rwDB.Generation
	if serr := r.Status().Update(ctx, rwDB); serr != nil {
		logger.Error(serr, "Failed to update RisingWaveDatabase status")
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: constants.ReconciliationPeriod}, nil
}

// resolveAdminCredentials resolves the admin username and password for connecting to RisingWave.
func (r *RisingWaveDatabaseReconciler) resolveAdminCredentials(ctx context.Context, rwDB *v1alpha1.RisingWaveDatabase) (string, string, error) {
	connRef := rwDB.Spec.ConnectionRef
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
				secretKey.Namespace = rwDB.Namespace
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

// finalizeRisingWaveDatabase handles the finalization of a RisingWaveDatabase.
// Safe-by-default: only drops the database if deletion-policy is explicitly set to "delete".
func (r *RisingWaveDatabaseReconciler) finalizeRisingWaveDatabase(
	ctx context.Context, db *sql.DB, rwDB *v1alpha1.RisingWaveDatabase, connKey rwclient.ConnectionKey,
) error {
	logger := log.FromContext(ctx).WithName("RisingWaveDatabaseReconciler").WithValues("database", rwDB.Name)

	// Safe-by-default: only drop if explicitly requested
	if rwDB.GetAnnotations()[constants.DeletionPolicyAnnotation] != constants.DeletionPolicyDelete {
		logger.Info("Deletion policy is not 'delete', retaining database",
			"database", rwDB.GetDatabaseName(),
			"policy", rwDB.GetAnnotations()[constants.DeletionPolicyAnnotation])
		r.ConnectionPool.Remove(connKey)
		return nil
	}

	// Explicit delete requested - drop database
	dbName := rwDB.GetDatabaseName()
	dropSQL := rwclient.BuildDropDatabaseSQL(dbName)
	if _, err := db.ExecContext(ctx, dropSQL); err != nil {
		return fmt.Errorf("failed to drop database: %w", err)
	}

	logger.Info("Dropped database", "database", dbName)
	metrics.RisingWaveDatabaseDeletedTotal.Increment()
	r.ConnectionPool.Remove(connKey)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RisingWaveDatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RisingWaveDatabase{}).
		Complete(r)
}
