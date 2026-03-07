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

package controller_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
)

var _ = Describe("RisingWaveSchema Controller", func() {
})

var _ = Describe("Creating a RisingWaveSchema", func() {
	It("should be stored correctly", func() {
		ctx := context.Background()
		rwSchema := &v1alpha1.RisingWaveSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-schema",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveSchemaSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
					Port: 4567,
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Name:  "staging",
				Owner: "admin_user",
			},
		}
		Expect(k8sClient.Create(ctx, rwSchema)).To(Succeed())

		fetched := &v1alpha1.RisingWaveSchema{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwSchema.Name,
			Namespace: rwSchema.Namespace,
		}, fetched)).To(Succeed())
		Expect(fetched.Spec.ConnectionRef.Host).To(Equal("risingwave-frontend.default.svc.cluster.local"))
		Expect(fetched.Spec.ConnectionRef.Port).To(Equal(int32(4567)))
		Expect(fetched.Spec.DatabaseRef.Name).To(Equal("analytics"))
		Expect(fetched.Spec.Name).To(Equal("staging"))
		Expect(fetched.Spec.Owner).To(Equal("admin_user"))
	})

	It("should use metadata.name as default schema name", func() {
		ctx := context.Background()
		rwSchema := &v1alpha1.RisingWaveSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-name-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveSchemaSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
			},
		}
		Expect(k8sClient.Create(ctx, rwSchema)).To(Succeed())

		fetched := &v1alpha1.RisingWaveSchema{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwSchema.Name,
			Namespace: rwSchema.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.GetSchemaName()).To(Equal("default-name-test"))
	})

	It("should store admin credentials correctly", func() {
		ctx := context.Background()
		rwSchema := &v1alpha1.RisingWaveSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credentials-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveSchemaSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
					Port: 4567,
					Credentials: &v1alpha1.AdminCredentials{
						Username: "admin",
						PasswordSecretRef: &v1alpha1.SecretReference{
							Name: "admin-secret",
							Key:  "password",
						},
					},
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
			},
		}
		Expect(k8sClient.Create(ctx, rwSchema)).To(Succeed())

		fetched := &v1alpha1.RisingWaveSchema{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwSchema.Name,
			Namespace: rwSchema.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Spec.ConnectionRef.Credentials.Username).To(Equal("admin"))
		Expect(fetched.Spec.ConnectionRef.Credentials.PasswordSecretRef.Name).To(Equal("admin-secret"))
		Expect(fetched.Spec.ConnectionRef.Credentials.PasswordSecretRef.Key).To(Equal("password"))
	})

	It("should allow status updates", func() {
		ctx := context.Background()
		rwSchema := &v1alpha1.RisingWaveSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "status-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveSchemaSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
			},
		}
		Expect(k8sClient.Create(ctx, rwSchema)).To(Succeed())

		// Update status
		rwSchema.Status.Phase = "Ready"
		rwSchema.Status.Reason = "Successfully reconciled"
		rwSchema.Status.SchemaCreated = true
		Expect(k8sClient.Status().Update(ctx, rwSchema)).To(Succeed())

		fetched := &v1alpha1.RisingWaveSchema{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwSchema.Name,
			Namespace: rwSchema.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Status.Phase).To(Equal("Ready"))
		Expect(fetched.Status.Reason).To(Equal("Successfully reconciled"))
		Expect(fetched.Status.SchemaCreated).To(BeTrue())
	})

	It("should allow condition updates", func() {
		ctx := context.Background()
		rwSchema := &v1alpha1.RisingWaveSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "condition-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveSchemaSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
			},
		}
		Expect(k8sClient.Create(ctx, rwSchema)).To(Succeed())

		conditions := []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "SuccessfullyReconciled",
				LastTransitionTime: metav1.Now(),
			},
		}
		rwSchema.SetConditions(conditions)
		Expect(k8sClient.Status().Update(ctx, rwSchema)).To(Succeed())

		fetched := &v1alpha1.RisingWaveSchema{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwSchema.Name,
			Namespace: rwSchema.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.GetConditions()).To(HaveLen(1))
		Expect(fetched.GetConditions()[0].Type).To(Equal("Ready"))
	})

	It("should store deletion policy annotation", func() {
		ctx := context.Background()
		rwSchema := &v1alpha1.RisingWaveSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deletion-policy-test",
				Namespace: "default",
				Annotations: map[string]string{
					"risingwave.risingwavelabs.com/deletion-policy": "delete",
				},
			},
			Spec: v1alpha1.RisingWaveSchemaSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
					Port: 4567,
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Name: "staging",
			},
		}
		Expect(k8sClient.Create(ctx, rwSchema)).To(Succeed())

		fetched := &v1alpha1.RisingWaveSchema{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwSchema.Name,
			Namespace: rwSchema.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Annotations["risingwave.risingwavelabs.com/deletion-policy"]).To(Equal("delete"))
	})

	It("should store deletion policy annotation for public schema", func() {
		ctx := context.Background()
		rwSchema := &v1alpha1.RisingWaveSchema{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "public-schema-test",
				Namespace: "default",
				Annotations: map[string]string{
					"risingwave.risingwavelabs.com/deletion-policy": "delete",
				},
			},
			Spec: v1alpha1.RisingWaveSchemaSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Name: "public", // Testing with public schema name
			},
		}
		Expect(k8sClient.Create(ctx, rwSchema)).To(Succeed())

		fetched := &v1alpha1.RisingWaveSchema{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwSchema.Name,
			Namespace: rwSchema.Namespace,
		}, fetched)).To(Succeed())

		// Verify the schema name is "public"
		Expect(fetched.GetSchemaName()).To(Equal("public"))
		// Verify deletion policy is set
		Expect(fetched.Annotations["risingwave.risingwavelabs.com/deletion-policy"]).To(Equal("delete"))
	})
})
