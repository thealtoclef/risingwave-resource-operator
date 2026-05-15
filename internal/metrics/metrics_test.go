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

package metrics

import (
	"reflect"
	"testing"
)

func TestMetricsNamespace(t *testing.T) {
	expected := "risingwave_resource_operator"
	if MetricsNamespace != expected {
		t.Errorf("MetricsNamespace = %q, want %q", MetricsNamespace, expected)
	}
}

func TestRisingWaveUserTotalAdaptorIncrement(t *testing.T) {
	adaptor := &RisingWaveUserTotalAdaptor{
		metric: userCreatedTotal,
	}

	// Test that Increment() method exists and is callable
	// We can't directly verify the counter value due to Prometheus registry,
	// but we can verify the method doesn't panic
	adaptor.Increment()
	adaptor.Increment()
	adaptor.Increment()
}

func TestMetricsInitialization(t *testing.T) {
	// Verify that metrics are initialized
	if RisingWaveUserCreatedTotal == nil {
		t.Error("RisingWaveUserCreatedTotal is nil")
	}

	if RisingWaveUserDeletedTotal == nil {
		t.Error("RisingWaveUserDeletedTotal is nil")
	}

	// Verify that the adaptors have non-nil metrics
	if RisingWaveUserCreatedTotal.metric == nil {
		t.Error("RisingWaveUserCreatedTotal.metric is nil")
	}

	if RisingWaveUserDeletedTotal.metric == nil {
		t.Error("RisingWaveUserDeletedTotal.metric is nil")
	}
}

func TestRisingWaveUserCreatedTotal(t *testing.T) {
	if RisingWaveUserCreatedTotal == nil {
		t.Fatal("RisingWaveUserCreatedTotal not initialized")
	}

	// Test type
	if reflect.TypeOf(RisingWaveUserCreatedTotal).String() != "*metrics.RisingWaveUserTotalAdaptor" {
		t.Errorf("RisingWaveUserCreatedTotal type = %T, want *RisingWaveUserTotalAdaptor", RisingWaveUserCreatedTotal)
	}
}

func TestRisingWaveUserDeletedTotal(t *testing.T) {
	if RisingWaveUserDeletedTotal == nil {
		t.Fatal("RisingWaveUserDeletedTotal not initialized")
	}

	// Test type
	if reflect.TypeOf(RisingWaveUserDeletedTotal).String() != "*metrics.RisingWaveUserTotalAdaptor" {
		t.Errorf("RisingWaveUserDeletedTotal type = %T, want *RisingWaveUserTotalAdaptor", RisingWaveUserDeletedTotal)
	}
}

func TestRisingWaveDatabaseCreatedTotal(t *testing.T) {
	if RisingWaveDatabaseCreatedTotal == nil {
		t.Fatal("RisingWaveDatabaseCreatedTotal not initialized")
	}

	// Test type
	if reflect.TypeOf(RisingWaveDatabaseCreatedTotal).String() != "*metrics.RisingWaveUserTotalAdaptor" {
		t.Errorf("RisingWaveDatabaseCreatedTotal type = %T, want *RisingWaveUserTotalAdaptor", RisingWaveDatabaseCreatedTotal)
	}

	// Test that Increment() method exists and is callable
	RisingWaveDatabaseCreatedTotal.Increment()
}

func TestRisingWaveDatabaseDeletedTotal(t *testing.T) {
	if RisingWaveDatabaseDeletedTotal == nil {
		t.Fatal("RisingWaveDatabaseDeletedTotal not initialized")
	}

	// Test type
	if reflect.TypeOf(RisingWaveDatabaseDeletedTotal).String() != "*metrics.RisingWaveUserTotalAdaptor" {
		t.Errorf("RisingWaveDatabaseDeletedTotal type = %T, want *RisingWaveUserTotalAdaptor", RisingWaveDatabaseDeletedTotal)
	}

	// Test that Increment() method exists and is callable
	RisingWaveDatabaseDeletedTotal.Increment()
}

func TestRisingWaveSchemaCreatedTotal(t *testing.T) {
	if RisingWaveSchemaCreatedTotal == nil {
		t.Fatal("RisingWaveSchemaCreatedTotal not initialized")
	}

	// Test type
	if reflect.TypeOf(RisingWaveSchemaCreatedTotal).String() != "*metrics.RisingWaveUserTotalAdaptor" {
		t.Errorf("RisingWaveSchemaCreatedTotal type = %T, want *RisingWaveUserTotalAdaptor", RisingWaveSchemaCreatedTotal)
	}

	// Test that Increment() method exists and is callable
	RisingWaveSchemaCreatedTotal.Increment()
}

func TestRisingWaveSchemaDeletedTotal(t *testing.T) {
	if RisingWaveSchemaDeletedTotal == nil {
		t.Fatal("RisingWaveSchemaDeletedTotal not initialized")
	}

	// Test type
	if reflect.TypeOf(RisingWaveSchemaDeletedTotal).String() != "*metrics.RisingWaveUserTotalAdaptor" {
		t.Errorf("RisingWaveSchemaDeletedTotal type = %T, want *RisingWaveUserTotalAdaptor", RisingWaveSchemaDeletedTotal)
	}

	// Test that Increment() method exists and is callable
	RisingWaveSchemaDeletedTotal.Increment()
}
