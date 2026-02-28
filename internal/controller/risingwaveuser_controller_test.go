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

var _ = Describe("RisingWaveUser Controller", func() {
	ctx := context.Background()

	Describe("Creating a RisingWaveUser", func() {
		It("should be stored correctly", func() {
			rwUser := &v1alpha1.RisingWaveUser{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "default",
				},
				Spec: v1alpha1.RisingWaveUserSpec{
					ConnectionRef: v1alpha1.ConnectionRef{
						Host: "risingwave-frontend.default.svc.cluster.local",
						Port: 4567,
					},
				},
			}

			Expect(k8sClient.Create(ctx, rwUser)).To(Succeed())

			fetched := &v1alpha1.RisingWaveUser{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      rwUser.Name,
				Namespace: rwUser.Namespace,
			}, fetched)).To(Succeed())

			Expect(fetched.Spec.ConnectionRef.Host).To(Equal("risingwave-frontend.default.svc.cluster.local"))
			Expect(fetched.Spec.ConnectionRef.Port).To(Equal(int32(4567)))

			Expect(k8sClient.Delete(ctx, rwUser)).To(Succeed())
		})
	})
})
