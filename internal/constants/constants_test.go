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
	"testing"
)

func TestPauseReconcileAnnotation(t *testing.T) {
	expected := "risingwave.risingwavelabs.com/pause-reconcile"
	if PauseReconcileAnnotation != expected {
		t.Errorf("PauseReconcileAnnotation = %q, want %q", PauseReconcileAnnotation, expected)
	}
}

func TestDeletionPolicyAnnotation(t *testing.T) {
	expected := "risingwave.risingwavelabs.com/deletion-policy"
	if DeletionPolicyAnnotation != expected {
		t.Errorf("DeletionPolicyAnnotation = %q, want %q", DeletionPolicyAnnotation, expected)
	}
}

func TestRotatePasswordAnnotation(t *testing.T) {
	expected := "risingwave.risingwavelabs.com/rotate-password"
	if RotatePasswordAnnotation != expected {
		t.Errorf("RotatePasswordAnnotation = %q, want %q", RotatePasswordAnnotation, expected)
	}
}

func TestPhaseConstants(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"PhaseReady", PhaseReady, "Ready"},
		{"PhaseNotReady", PhaseNotReady, "NotReady"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

func TestReasonConstants(t *testing.T) {
	tests := []struct {
		name     string
		actual   string
		expected string
	}{
		{"ReasonCompleted", ReasonCompleted, "Successfully reconciled"},
		{"ReasonFailedToFinalize", ReasonFailedToFinalize, "Failed to finalize"},
		{"ReasonFailedToGetSecret", ReasonFailedToGetSecret, "Failed to get Secret"},
		{"ReasonConnectionFailed", ReasonConnectionFailed, "Failed to connect to RisingWave"},
		{"ReasonFailedToCreateUser", ReasonFailedToCreateUser, "Failed to create user"},
		{"ReasonFailedToUpdatePassword", ReasonFailedToUpdatePassword, "Failed to update password"},
		{"ReasonFailedToGrant", ReasonFailedToGrant, "Failed to reconcile privileges"},
		{"ReasonFailedToCreateSecret", ReasonFailedToCreateSecret, "Failed to create/update credential secret"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s = %q, want %q", tt.name, tt.actual, tt.expected)
			}
		})
	}
}

func TestReconciliationPeriodConstant(t *testing.T) {
	if DefaultReconciliationPeriodSeconds != 60 {
		t.Errorf("DefaultReconciliationPeriodSeconds = %d, want 60", DefaultReconciliationPeriodSeconds)
	}
}
