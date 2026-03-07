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

package constants

import (
	"os"
	"strconv"
	"time"
)

// DefaultReconciliationPeriodSeconds is the default reconciliation period in seconds.
const DefaultReconciliationPeriodSeconds = 60

// GetReconciliationPeriod returns the reconciliation period from the RECONCILIATION_PERIOD_SECONDS
// environment variable or defaults to 60 seconds.
func GetReconciliationPeriod() time.Duration {
	envPeriod := os.Getenv("RECONCILIATION_PERIOD_SECONDS")
	if envPeriod != "" {
		if seconds, err := strconv.Atoi(envPeriod); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
	}
	return DefaultReconciliationPeriodSeconds * time.Second
}

// ReconciliationPeriod is the time between controller reconciliations.
var ReconciliationPeriod = GetReconciliationPeriod()

// Status phases.
const (
	PhaseReady    = "Ready"
	PhaseNotReady = "NotReady"
)

// Reason constants.
const (
	// Success
	ReasonCompleted = "Successfully reconciled"

	// Errors
	ReasonFailedToFinalize         = "Failed to finalize"
	ReasonFailedToGetSecret        = "Failed to get Secret"
	ReasonConnectionFailed         = "Failed to connect to RisingWave"
	ReasonFailedToCreateUser       = "Failed to create user"
	ReasonFailedToUpdatePassword   = "Failed to update password"
	ReasonFailedToGrant            = "Failed to reconcile privileges"
	ReasonFailedToCreateSecret     = "Failed to create/update credential secret"
	ReasonFailedToCreateDatabase   = "Failed to create database"
	ReasonFailedToUpdateOwner      = "Failed to update owner"
	ReasonFailedToSyncSchemas      = "Failed to sync schemas"
	ReasonFailedToCreateSchema     = "Failed to create schema"
	ReasonFailedToCreateConnection = "Failed to create connection"
	ReasonUpdateNotSupported       = "Update not supported"
)

// Deletion policy values for annotation.
const (
	DeletionPolicyDelete = "delete"
)

// Annotation constants.
const (
	DeletionPolicyAnnotation = "risingwave.risingwavelabs.com/deletion-policy"
	PauseReconcileAnnotation = "risingwave.risingwavelabs.com/pause-reconcile"
	RotatePasswordAnnotation = "risingwave.risingwavelabs.com/rotate-password"
)

// Schema name constants.
const (
	// DefaultSchemaName is the default schema created in every RisingWave database.
	// This schema should never be deleted.
	DefaultSchemaName = "public"
)
