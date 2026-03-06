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

var _ = Describe("RisingWaveDatabase Controller", func() {
	ctx := context.Background()

	Describe("Creating a RisingWaveDatabase", func() {
		It("should be stored correctly", func() {
			rwDB := &v1alpha1.RisingWaveDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-database",
					Namespace: "default",
				},
				Spec: v1alpha1.RisingWaveDatabaseSpec{
					ConnectionRef: v1alpha1.ConnectionRef{
						Host: "risingwave-frontend.default.svc.cluster.local",
						Port: 4567,
					},
					Name:  "analytics",
					Owner: "admin_user",
				},
			}

			Expect(k8sClient.Create(ctx, rwDB)).To(Succeed())

			fetched := &v1alpha1.RisingWaveDatabase{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      rwDB.Name,
				Namespace: rwDB.Namespace,
			}, fetched)).To(Succeed())

			Expect(fetched.Spec.ConnectionRef.Host).To(Equal("risingwave-frontend.default.svc.cluster.local"))
			Expect(fetched.Spec.ConnectionRef.Port).To(Equal(int32(4567)))
			Expect(fetched.Spec.Name).To(Equal("analytics"))
			Expect(fetched.Spec.Owner).To(Equal("admin_user"))

			Expect(k8sClient.Delete(ctx, rwDB)).To(Succeed())
		})

		It("should use metadata.name as default database name", func() {
			rwDB := &v1alpha1.RisingWaveDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "default-db-name",
					Namespace: "default",
				},
				Spec: v1alpha1.RisingWaveDatabaseSpec{
					ConnectionRef: v1alpha1.ConnectionRef{
						Host: "risingwave-frontend.default.svc.cluster.local",
					},
				},
			}

			Expect(k8sClient.Create(ctx, rwDB)).To(Succeed())

			fetched := &v1alpha1.RisingWaveDatabase{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      rwDB.Name,
				Namespace: rwDB.Namespace,
			}, fetched)).To(Succeed())

			// GetDatabaseName() should return metadata.name when spec.name is empty
			Expect(fetched.GetDatabaseName()).To(Equal("default-db-name"))

			Expect(k8sClient.Delete(ctx, rwDB)).To(Succeed())
		})

		It("should store admin credentials correctly", func() {
			rwDB := &v1alpha1.RisingWaveDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-with-creds",
					Namespace: "default",
				},
				Spec: v1alpha1.RisingWaveDatabaseSpec{
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
				},
			}

			Expect(k8sClient.Create(ctx, rwDB)).To(Succeed())

			fetched := &v1alpha1.RisingWaveDatabase{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      rwDB.Name,
				Namespace: rwDB.Namespace,
			}, fetched)).To(Succeed())

			Expect(fetched.Spec.ConnectionRef.Credentials.Username).To(Equal("admin"))
			Expect(fetched.Spec.ConnectionRef.Credentials.PasswordSecretRef.Name).To(Equal("admin-secret"))
			Expect(fetched.Spec.ConnectionRef.Credentials.PasswordSecretRef.Key).To(Equal("password"))

			Expect(k8sClient.Delete(ctx, rwDB)).To(Succeed())
		})
	})

	Describe("RisingWaveDatabase status", func() {
		It("should allow status updates", func() {
			rwDB := &v1alpha1.RisingWaveDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "status-test",
					Namespace: "default",
				},
				Spec: v1alpha1.RisingWaveDatabaseSpec{
					ConnectionRef: v1alpha1.ConnectionRef{
						Host: "risingwave-frontend.default.svc.cluster.local",
					},
				},
			}

			Expect(k8sClient.Create(ctx, rwDB)).To(Succeed())

			// Update status
			rwDB.Status.Phase = "Ready"
			rwDB.Status.Reason = "Successfully reconciled"
			rwDB.Status.DatabaseCreated = true
			Expect(k8sClient.Status().Update(ctx, rwDB)).To(Succeed())

			// Fetch and verify
			fetched := &v1alpha1.RisingWaveDatabase{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      rwDB.Name,
				Namespace: rwDB.Namespace,
			}, fetched)).To(Succeed())

			Expect(fetched.Status.Phase).To(Equal("Ready"))
			Expect(fetched.Status.Reason).To(Equal("Successfully reconciled"))
			Expect(fetched.Status.DatabaseCreated).To(BeTrue())

			Expect(k8sClient.Delete(ctx, rwDB)).To(Succeed())
		})
	})

	Describe("RisingWaveDatabase conditions", func() {
		It("should allow condition updates", func() {
			rwDB := &v1alpha1.RisingWaveDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "condition-test",
					Namespace: "default",
				},
				Spec: v1alpha1.RisingWaveDatabaseSpec{
					ConnectionRef: v1alpha1.ConnectionRef{
						Host: "risingwave-frontend.default.svc.cluster.local",
					},
				},
			}

			Expect(k8sClient.Create(ctx, rwDB)).To(Succeed())

			// Update conditions
			rwDB.Status.Conditions = []metav1.Condition{
				{
					Type:               string(v1alpha1.RisingWaveDatabaseConditionReady),
					Status:             metav1.ConditionTrue,
					Reason:             "DatabaseCreated",
					LastTransitionTime: metav1.Now(),
				},
			}
			Expect(k8sClient.Status().Update(ctx, rwDB)).To(Succeed())

			// Fetch and verify
			fetched := &v1alpha1.RisingWaveDatabase{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      rwDB.Name,
				Namespace: rwDB.Namespace,
			}, fetched)).To(Succeed())

			Expect(fetched.Status.Conditions).To(HaveLen(1))
			Expect(fetched.Status.Conditions[0].Type).To(Equal(string(v1alpha1.RisingWaveDatabaseConditionReady)))

			Expect(k8sClient.Delete(ctx, rwDB)).To(Succeed())
		})
	})

	Describe("Deletion policy annotation", func() {
		It("should store deletion policy annotation", func() {
			rwDB := &v1alpha1.RisingWaveDatabase{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deletion-policy-test",
					Namespace: "default",
					Annotations: map[string]string{
						"risingwave.risingwavelabs.com/deletion-policy": "delete",
					},
				},
				Spec: v1alpha1.RisingWaveDatabaseSpec{
					ConnectionRef: v1alpha1.ConnectionRef{
						Host: "risingwave-frontend.default.svc.cluster.local",
					},
				},
			}

			Expect(k8sClient.Create(ctx, rwDB)).To(Succeed())

			fetched := &v1alpha1.RisingWaveDatabase{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      rwDB.Name,
				Namespace: rwDB.Namespace,
			}, fetched)).To(Succeed())

			Expect(fetched.Annotations["risingwave.risingwavelabs.com/deletion-policy"]).To(Equal("delete"))

			Expect(k8sClient.Delete(ctx, rwDB)).To(Succeed())
		})
	})
})
