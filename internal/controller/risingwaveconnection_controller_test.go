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

var _ = Describe("RisingWaveConnection Controller", func() {
})

var _ = Describe("Creating a RisingWaveConnection", func() {
	It("should be stored correctly", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-kafka-connection",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
					Port: 4567,
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				SchemaRef: &v1alpha1.SchemaRef{
					Name: "public",
				},
				Name: "kafka_prod",
				Type: v1alpha1.ConnectionTypeKafka,
				Properties: map[string]string{
					"properties.bootstrap.server":  "kafka-broker-1:9092",
					"properties.security.protocol": "SASL_SSL",
				},
				Owner: "admin_user",
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())
		Expect(fetched.Spec.ConnectionRef.Host).To(Equal("risingwave-frontend.default.svc.cluster.local"))
		Expect(fetched.Spec.ConnectionRef.Port).To(Equal(int32(4567)))
		Expect(fetched.Spec.DatabaseRef.Name).To(Equal("analytics"))
		Expect(fetched.Spec.SchemaRef.Name).To(Equal("public"))
		Expect(fetched.Spec.Name).To(Equal("kafka_prod"))
		Expect(fetched.Spec.Type).To(Equal(v1alpha1.ConnectionTypeKafka))
		Expect(fetched.Spec.Owner).To(Equal("admin_user"))
		Expect(fetched.Spec.Properties).To(HaveKey("properties.bootstrap.server"))
	})

	It("should use metadata.name as default connection name", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-name-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Type: v1alpha1.ConnectionTypeKafka,
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.GetConnectionName()).To(Equal("default-name-test"))
	})

	It("should use 'public' as default schema", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "default-schema-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Type: v1alpha1.ConnectionTypeKafka,
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.GetSchemaName()).To(Equal("public"))
	})

	It("should store admin credentials correctly", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "credentials-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
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
				Type: v1alpha1.ConnectionTypeKafka,
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Spec.ConnectionRef.Credentials.Username).To(Equal("admin"))
		Expect(fetched.Spec.ConnectionRef.Credentials.PasswordSecretRef.Name).To(Equal("admin-secret"))
		Expect(fetched.Spec.ConnectionRef.Credentials.PasswordSecretRef.Key).To(Equal("password"))
	})

	It("should allow status updates", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "status-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Type: v1alpha1.ConnectionTypeKafka,
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		// Update status
		rwConn.Status.Phase = "Ready"
		rwConn.Status.Reason = "Successfully reconciled"
		rwConn.Status.ConnectionCreated = true
		Expect(k8sClient.Status().Update(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Status.Phase).To(Equal("Ready"))
		Expect(fetched.Status.Reason).To(Equal("Successfully reconciled"))
		Expect(fetched.Status.ConnectionCreated).To(BeTrue())
	})

	It("should allow condition updates", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "condition-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Type: v1alpha1.ConnectionTypeKafka,
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		conditions := []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "SuccessfullyReconciled",
				LastTransitionTime: metav1.Now(),
			},
		}
		rwConn.SetConditions(conditions)
		Expect(k8sClient.Status().Update(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.GetConditions()).To(HaveLen(1))
		Expect(fetched.GetConditions()[0].Type).To(Equal("Ready"))
	})

	It("should store deletion policy annotation", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "deletion-policy-test",
				Namespace: "default",
				Annotations: map[string]string{
					"risingwave.risingwavelabs.com/deletion-policy": "delete",
				},
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
					Port: 4567,
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Name: "kafka_prod",
				Type: v1alpha1.ConnectionTypeKafka,
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Annotations["risingwave.risingwavelabs.com/deletion-policy"]).To(Equal("delete"))
	})

	It("should support all connection types", func() {
		ctx := context.Background()

		connectionTypes := []v1alpha1.ConnectionType{
			v1alpha1.ConnectionTypeKafka,
			v1alpha1.ConnectionTypeSchemaRegistry,
			v1alpha1.ConnectionTypeIceberg,
		}

		for _, connType := range connectionTypes {
			rwConn := &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "type-test-" + string(connType),
					Namespace: "default",
				},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					ConnectionRef: v1alpha1.ConnectionRef{
						Host: "risingwave-frontend.default.svc.cluster.local",
					},
					DatabaseRef: v1alpha1.DatabaseRef{
						Name: "analytics",
					},
					Type: connType,
				},
			}
			Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

			fetched := &v1alpha1.RisingWaveConnection{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      rwConn.Name,
				Namespace: rwConn.Namespace,
			}, fetched)).To(Succeed())

			Expect(fetched.Spec.Type).To(Equal(connType))
		}
	})

	It("should store properties with SECRET prefix", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Type: v1alpha1.ConnectionTypeKafka,
				Properties: map[string]string{
					"properties.bootstrap.server":  "kafka:9092",
					"properties.sasl.password":     "SECRET kafka_credentials",
					"properties.security.protocol": "SASL_SSL",
				},
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Spec.Properties["properties.sasl.password"]).To(Equal("SECRET kafka_credentials"))
	})

	It("should support schema registry connection type", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "schema-registry-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Type: v1alpha1.ConnectionTypeSchemaRegistry,
				Properties: map[string]string{
					"schema.registry": "https://schema-registry:8081",
				},
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Spec.Type).To(Equal(v1alpha1.ConnectionTypeSchemaRegistry))
		Expect(fetched.Spec.Properties["schema.registry"]).To(Equal("https://schema-registry:8081"))
	})

	It("should support iceberg connection type", func() {
		ctx := context.Background()
		rwConn := &v1alpha1.RisingWaveConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "iceberg-test",
				Namespace: "default",
			},
			Spec: v1alpha1.RisingWaveConnectionSpec{
				ConnectionRef: v1alpha1.ConnectionRef{
					Host: "risingwave-frontend.default.svc.cluster.local",
				},
				DatabaseRef: v1alpha1.DatabaseRef{
					Name: "analytics",
				},
				Type: v1alpha1.ConnectionTypeIceberg,
				Properties: map[string]string{
					"catalog.name":   "demo",
					"catalog.type":   "storage",
					"s3.access.key":  "hummockadmin",
					"s3.secret.key":  "SECRET iceberg_credentials",
					"s3.endpoint":    "http://127.0.0.1:9301",
					"warehouse.path": "s3a://hummock001/iceberg-data",
				},
			},
		}
		Expect(k8sClient.Create(ctx, rwConn)).To(Succeed())

		fetched := &v1alpha1.RisingWaveConnection{}
		Expect(k8sClient.Get(ctx, types.NamespacedName{
			Name:      rwConn.Name,
			Namespace: rwConn.Namespace,
		}, fetched)).To(Succeed())

		Expect(fetched.Spec.Type).To(Equal(v1alpha1.ConnectionTypeIceberg))
		Expect(fetched.Spec.Properties["catalog.name"]).To(Equal("demo"))
		Expect(fetched.Spec.Properties["s3.secret.key"]).To(Equal("SECRET iceberg_credentials"))
	})
})
