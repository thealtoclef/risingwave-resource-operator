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
	"strings"
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

const risingWaveConnectionFinalizer = "risingwaveconnection.risingwave.risingwavelabs.com/finalizer"

// RisingWaveConnectionReconciler reconciles a RisingWaveConnection object.
type RisingWaveConnectionReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	ConnectionPool *rwclient.Pool
}

//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveconnections,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveconnections/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=risingwave.risingwavelabs.com,resources=risingwaveconnections/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
//+kubebuilder:rbac:groups=core,resources=events,verbs=create;update;patch

// Reconcile reconciles a RisingWaveConnection object.
func (r *RisingWaveConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("RisingWaveConnectionReconciler")

	// 1. Fetch RisingWaveConnection
	rwConn := &v1alpha1.RisingWaveConnection{}
	if err := r.Get(ctx, req.NamespacedName, rwConn); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("RisingWaveConnection not found, skipping", "namespacedName", req.NamespacedName)
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get RisingWaveConnection")
		return ctrl.Result{}, err
	}
	logger.Info("Fetched RisingWaveConnection", "name", rwConn.Name, "namespace", rwConn.Namespace)

	// 2. Check pause annotation
	if rwConn.GetAnnotations()[constants.PauseReconcileAnnotation] == "true" {
		logger.Info("Reconciliation paused via annotation")
		return ctrl.Result{}, nil
	}

	// 3. Build connection key and info
	connRef := rwConn.Spec.ConnectionRef
	port := connRef.Port
	if port == 0 {
		port = 4567
	}

	adminUser, adminPassword, err := r.resolveAdminCredentials(ctx, rwConn)
	if err != nil {
		logger.Error(err, "Failed to resolve admin credentials")
		rwConn.Status.Phase = constants.PhaseNotReady
		rwConn.Status.Reason = constants.ReasonFailedToGetSecret
		if serr := r.Status().Update(ctx, rwConn); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	connKey := rwclient.ConnectionKeyFrom(rwConn.Namespace, connRef.Host, port)
	connInfo := rwclient.DefaultConnectionInfo(connRef.Host, port, adminUser, adminPassword)

	db, err := r.ConnectionPool.Get(ctx, connKey, connInfo)
	if err != nil {
		logger.Error(err, "Failed to connect to RisingWave", "host", connRef.Host, "port", port)
		rwConn.Status.Phase = constants.PhaseNotReady
		rwConn.Status.Reason = constants.ReasonConnectionFailed
		if serr := r.Status().Update(ctx, rwConn); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	logger.Info("Connected to RisingWave", "host", connRef.Host, "port", port)

	// 4. Handle finalizer
	finalizerResult, finalizerErr := utils.HandleFinalizer(utils.FinalizerParams{
		Object:    rwConn,
		Context:   ctx,
		Client:    r.Client,
		Finalizer: risingWaveConnectionFinalizer,
		FinalizationFunc: func() error {
			return r.finalizeRisingWaveConnection(ctx, db, rwConn, connKey)
		},
		OnFailure: func(err error) error {
			rwConn.Status.Phase = constants.PhaseNotReady
			rwConn.Status.Reason = constants.ReasonFailedToFinalize
			if serr := r.Status().Update(ctx, rwConn); serr != nil {
				logger.Error(serr, "Failed to update finalization status")
				return serr
			}
			return nil
		},
	})

	if finalizerErr != nil || !rwConn.GetDeletionTimestamp().IsZero() {
		return finalizerResult, finalizerErr
	}

	// Effective connection name and database
	connName := rwConn.GetConnectionName()
	dbName := rwConn.Spec.DatabaseRef.Name

	// 5. Switch to target database
	useSQL := rwclient.BuildUseDatabaseSQL(dbName)
	if _, err := db.ExecContext(ctx, useSQL); err != nil {
		logger.Error(err, "Failed to switch to database", "database", dbName)
		rwConn.Status.Phase = constants.PhaseNotReady
		rwConn.Status.Reason = constants.ReasonFailedToCreateConnection
		if serr := r.Status().Update(ctx, rwConn); serr != nil {
			return ctrl.Result{RequeueAfter: time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// 6. Check if connection exists
	connExists, err := rwclient.CheckConnectionExists(ctx, db, connName)
	if err != nil {
		logger.Error(err, "Failed to check if connection exists", "connection", connName)
		return ctrl.Result{}, err
	}

	// 7. Create or update connection
	if !connExists {
		// Create new connection
		createSQL := rwclient.BuildCreateConnectionSQL(rwConn)
		logger.V(1).Info("Creating connection", "sql", createSQL)
		if _, err := db.ExecContext(ctx, createSQL); err != nil {
			logger.Error(err, "Failed to create connection", "connection", connName)
			rwConn.Status.Phase = constants.PhaseNotReady
			rwConn.Status.Reason = constants.ReasonFailedToCreateConnection
			if serr := r.Status().Update(ctx, rwConn); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		logger.Info("Created connection", "connection", connName, "type", rwConn.Spec.Type)
		rwConn.Status.ConnectionCreated = true
		metrics.RisingWaveConnectionCreatedTotal.Increment()
	} else if rwConn.Status.ObservedGeneration != rwConn.Generation {
		// Connection exists but spec changed - update with ALTER CONNECTION
		alterSQL := rwclient.BuildAlterConnectionSQL(rwConn)
		logger.V(1).Info("Updating connection properties", "sql", alterSQL)

		if _, err := db.ExecContext(ctx, alterSQL); err != nil {
			logger.Error(err, "Failed to update connection", "connection", connName)
			rwConn.Status.Phase = constants.PhaseNotReady
			rwConn.Status.Reason = constants.ReasonUpdateNotSupported
			if serr := r.Status().Update(ctx, rwConn); serr != nil {
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			// Some properties may not be alterable - return error to surface the issue
			return ctrl.Result{}, fmt.Errorf("failed to update connection: %w", err)
		}
		logger.Info("Updated connection properties", "connection", connName)
	} else {
		rwConn.Status.ConnectionCreated = true
	}

	// 8. Update connection owner if specified and changed
	if rwConn.Spec.Owner != "" {
		currentOwner, err := rwclient.GetConnectionOwner(ctx, db, connName)
		if err != nil {
			// Connection might not exist yet or query failed - log and continue
			logger.V(1).Info("Failed to get connection owner", "connection", connName, "error", err)
		} else if currentOwner != rwConn.Spec.Owner {
			alterOwnerSQL := rwclient.BuildAlterConnectionOwnerSQL(connName, rwConn.Spec.Owner)
			logger.V(1).Info("Updating connection owner", "sql", alterOwnerSQL)
			if _, err := db.ExecContext(ctx, alterOwnerSQL); err != nil {
				logger.Error(err, "Failed to update connection owner", "connection", connName, "owner", rwConn.Spec.Owner)
				// Non-fatal: connection is created/updated, just owner not changed
			} else {
				logger.Info("Updated connection owner", "connection", connName, "owner", rwConn.Spec.Owner)
			}
		}
	}

	// 9. Update status to Ready
	rwConn.Status.Phase = constants.PhaseReady
	rwConn.Status.Reason = constants.ReasonCompleted
	rwConn.Status.ObservedGeneration = rwConn.Generation
	if serr := r.Status().Update(ctx, rwConn); serr != nil {
		logger.Error(serr, "Failed to update RisingWaveConnection status")
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	return ctrl.Result{RequeueAfter: constants.ReconciliationPeriod}, nil
}

// resolveAdminCredentials resolves the admin username and password for connecting to RisingWave.
func (r *RisingWaveConnectionReconciler) resolveAdminCredentials(ctx context.Context, rwConn *v1alpha1.RisingWaveConnection) (string, string, error) {
	connRef := rwConn.Spec.ConnectionRef
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
				secretKey.Namespace = rwConn.Namespace
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

// finalizeRisingWaveConnection handles the finalization of a RisingWaveConnection.
// Safe-by-default: only drops the connection if deletion-policy is explicitly set to "delete".
func (r *RisingWaveConnectionReconciler) finalizeRisingWaveConnection(
	ctx context.Context, db *sql.DB, rwConn *v1alpha1.RisingWaveConnection, connKey rwclient.ConnectionKey,
) error {
	logger := log.FromContext(ctx).WithName("RisingWaveConnectionReconciler").WithValues("connection", rwConn.Name)

	// Safe-by-default: only drop if explicitly requested
	if rwConn.GetAnnotations()[constants.DeletionPolicyAnnotation] != constants.DeletionPolicyDelete {
		logger.Info("Deletion policy is not 'delete', retaining connection",
			"connection", rwConn.GetConnectionName(),
			"policy", rwConn.GetAnnotations()[constants.DeletionPolicyAnnotation])
		r.ConnectionPool.Remove(connKey)
		return nil
	}

	// Explicit delete requested - switch to database and drop connection
	dbName := rwConn.Spec.DatabaseRef.Name
	connName := rwConn.GetConnectionName()

	// Switch to database
	useSQL := rwclient.BuildUseDatabaseSQL(dbName)
	if _, err := db.ExecContext(ctx, useSQL); err != nil {
		return fmt.Errorf("failed to switch to database: %w", err)
	}

	// Drop connection
	dropSQL := rwclient.BuildDropConnectionSQL(connName)
	if _, err := db.ExecContext(ctx, dropSQL); err != nil {
		// Handle "does not exist" gracefully
		if strings.Contains(err.Error(), "does not exist") || strings.Contains(err.Error(), "not found") {
			logger.V(1).Info("Connection already does not exist")
		} else {
			return fmt.Errorf("failed to drop connection: %w", err)
		}
	}

	logger.Info("Dropped connection", "connection", connName, "database", dbName)
	metrics.RisingWaveConnectionDeletedTotal.Increment()
	r.ConnectionPool.Remove(connKey)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RisingWaveConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.RisingWaveConnection{}).
		Complete(r)
}
