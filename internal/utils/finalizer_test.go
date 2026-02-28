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

package utils

import (
	"context"
	"errors"
	"testing"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

func init() {
	_ = v1alpha1.AddToScheme(scheme.Scheme)
}

func TestHandleFinalizer_AddsFinalizer(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()

	const testFinalizer = "test.example.com/finalizer"

	obj := &v1alpha1.RisingWaveUser{
		ObjectMeta: v1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
		},
		Spec: v1alpha1.RisingWaveUserSpec{
			ConnectionRef: v1alpha1.ConnectionRef{
				Host: "localhost",
				Port: 4567,
			},
		},
	}

	// Create the object
	if err := fakeClient.Create(ctx, obj); err != nil {
		t.Fatalf("failed to create object: %v", err)
	}

	// Handle finalizer on non-deleted object
	result, err := HandleFinalizer(FinalizerParams{
		Object:    obj,
		Context:   ctx,
		Client:    fakeClient,
		Finalizer: testFinalizer,
		FinalizationFunc: func() error {
			return nil
		},
	})

	if err != nil {
		t.Errorf("HandleFinalizer() error = %v, wantErr false", err)
	}

	if result != (ctrl.Result{}) {
		t.Errorf("HandleFinalizer() result = %v, want empty Result", result)
	}

	// Verify finalizer was added
	if err := fakeClient.Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		t.Fatalf("failed to get object: %v", err)
	}

	if len(obj.GetFinalizers()) == 0 {
		t.Error("HandleFinalizer() did not add finalizer")
	}

	if !containsFinalizer(obj.GetFinalizers(), testFinalizer) {
		t.Errorf("HandleFinalizer() finalizer not found in %v", obj.GetFinalizers())
	}
}

func TestHandleFinalizer_SkipsWhenFinalizerExists(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()

	const testFinalizer = "test.example.com/finalizer"

	obj := &v1alpha1.RisingWaveUser{
		ObjectMeta: v1.ObjectMeta{
			Name:       "test-user",
			Namespace:  "default",
			Finalizers: []string{testFinalizer},
		},
		Spec: v1alpha1.RisingWaveUserSpec{
			ConnectionRef: v1alpha1.ConnectionRef{
				Host: "localhost",
				Port: 4567,
			},
		},
	}

	// Create the object
	if err := fakeClient.Create(ctx, obj); err != nil {
		t.Fatalf("failed to create object: %v", err)
	}

	// Handle finalizer on non-deleted object (finalizer already exists)
	result, err := HandleFinalizer(FinalizerParams{
		Object:    obj,
		Context:   ctx,
		Client:    fakeClient,
		Finalizer: testFinalizer,
		FinalizationFunc: func() error {
			return nil
		},
	})

	if err != nil {
		t.Errorf("HandleFinalizer() error = %v, wantErr false", err)
	}

	if result != (ctrl.Result{}) {
		t.Errorf("HandleFinalizer() result = %v, want empty Result", result)
	}
}

func TestHandleFinalizer_DeletedObjectWithFinalizer(t *testing.T) {
	const testFinalizer = "test.example.com/finalizer"

	finalizationCalled := false
	expectedErr := errors.New("test finalization")

	// Create an object with deletion timestamp and finalizer
	now := v1.Now()
	obj := &v1alpha1.RisingWaveUser{
		ObjectMeta: v1.ObjectMeta{
			Name:              "test-user",
			Namespace:         "default",
			Finalizers:        []string{testFinalizer},
			DeletionTimestamp: &now,
		},
		Spec: v1alpha1.RisingWaveUserSpec{
			ConnectionRef: v1alpha1.ConnectionRef{
				Host: "localhost",
				Port: 4567,
			},
		},
	}

	// With a fake client that doesn't exist, we test the logic
	// by checking that the finalization function gets called
	ctx := context.Background()
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()

	result, err := HandleFinalizer(FinalizerParams{
		Object:    obj,
		Context:   ctx,
		Client:    fakeClient,
		Finalizer: testFinalizer,
		FinalizationFunc: func() error {
			finalizationCalled = true
			return expectedErr
		},
	})

	if !finalizationCalled {
		t.Error("HandleFinalizer() did not call FinalizationFunc for deleted object")
	}

	if err != expectedErr {
		t.Errorf("HandleFinalizer() error = %v, want %v", err, expectedErr)
	}

	if result != (ctrl.Result{}) {
		t.Errorf("HandleFinalizer() result = %v, want empty Result", result)
	}
}

func TestHandleFinalizer_AbandonPolicy(t *testing.T) {
	const testFinalizer = "test.example.com/finalizer"

	finalizationCalled := false

	// Create an object with abandon policy and deletion timestamp
	now := v1.Now()
	obj := &v1alpha1.RisingWaveUser{
		ObjectMeta: v1.ObjectMeta{
			Name:              "test-user",
			Namespace:         "default",
			Finalizers:        []string{testFinalizer},
			DeletionTimestamp: &now,
			Annotations: map[string]string{
				"risingwave.risingwavelabs.com/deletion-policy": "abandon",
			},
		},
		Spec: v1alpha1.RisingWaveUserSpec{
			ConnectionRef: v1alpha1.ConnectionRef{
				Host: "localhost",
				Port: 4567,
			},
		},
	}

	ctx := context.Background()
	// Create the object first
	fakeClient := fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	if err := fakeClient.Create(ctx, obj); err != nil {
		t.Fatalf("failed to create object: %v", err)
	}

	result, err := HandleFinalizer(FinalizerParams{
		Object:    obj,
		Context:   ctx,
		Client:    fakeClient,
		Finalizer: testFinalizer,
		FinalizationFunc: func() error {
			finalizationCalled = true
			return nil
		},
	})

	if finalizationCalled {
		t.Error("HandleFinalizer() called FinalizationFunc with abandon policy")
	}

	if err != nil {
		t.Errorf("HandleFinalizer() error = %v, wantErr false", err)
	}

	if result != (ctrl.Result{}) {
		t.Errorf("HandleFinalizer() result = %v, want empty Result", result)
	}
}

func TestHandleFinalizer_NoFinalizerOnDeletion(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme.Scheme).
		Build()

	const testFinalizer = "test.example.com/finalizer"

	now := v1.Now()
	obj := &v1alpha1.RisingWaveUser{
		ObjectMeta: v1.ObjectMeta{
			Name:              "test-user",
			Namespace:         "default",
			DeletionTimestamp: &now,
		},
		Spec: v1alpha1.RisingWaveUserSpec{
			ConnectionRef: v1alpha1.ConnectionRef{
				Host: "localhost",
				Port: 4567,
			},
		},
	}

	// Create the object
	if err := fakeClient.Create(ctx, obj); err != nil {
		t.Fatalf("failed to create object: %v", err)
	}

	finalizationCalled := false
	result, err := HandleFinalizer(FinalizerParams{
		Object:    obj,
		Context:   ctx,
		Client:    fakeClient,
		Finalizer: testFinalizer,
		FinalizationFunc: func() error {
			finalizationCalled = true
			return nil
		},
	})

	if err != nil {
		t.Errorf("HandleFinalizer() error = %v, wantErr false", err)
	}

	if result != (ctrl.Result{}) {
		t.Errorf("HandleFinalizer() result = %v, want empty Result", result)
	}

	// Finalization should not be called if no finalizer exists
	if finalizationCalled {
		t.Error("HandleFinalizer() called FinalizationFunc when no finalizer present")
	}
}

func containsFinalizer(finalizers []string, finalizer string) bool {
	for _, f := range finalizers {
		if f == finalizer {
			return true
		}
	}
	return false
}
