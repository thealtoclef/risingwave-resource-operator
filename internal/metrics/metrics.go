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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

const MetricsNamespace = "risingwave_resource_operator"

// RisingWaveUserTotalAdaptor is a wrapper for prometheus Counter metrics.
type RisingWaveUserTotalAdaptor struct {
	metric prometheus.Counter
}

// Increment increments the counter.
func (m RisingWaveUserTotalAdaptor) Increment() {
	m.metric.Inc()
}

var (
	userCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "risingwave_user_created_total",
			Help:      "Number of created RisingWave users",
		},
	)

	userDeletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "risingwave_user_deleted_total",
			Help:      "Number of deleted RisingWave users",
		},
	)

	databaseCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "risingwave_database_created_total",
			Help:      "Number of created RisingWave databases",
		},
	)

	databaseDeletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "risingwave_database_deleted_total",
			Help:      "Number of deleted RisingWave databases",
		},
	)

	schemaCreatedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "risingwave_schema_created_total",
			Help:      "Number of created RisingWave schemas",
		},
	)

	schemaDeletedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: MetricsNamespace,
			Name:      "risingwave_schema_deleted_total",
			Help:      "Number of deleted RisingWave schemas",
		},
	)

	RisingWaveUserCreatedTotal     = &RisingWaveUserTotalAdaptor{metric: userCreatedTotal}
	RisingWaveUserDeletedTotal     = &RisingWaveUserTotalAdaptor{metric: userDeletedTotal}
	RisingWaveDatabaseCreatedTotal = &RisingWaveUserTotalAdaptor{metric: databaseCreatedTotal}
	RisingWaveDatabaseDeletedTotal = &RisingWaveUserTotalAdaptor{metric: databaseDeletedTotal}
	RisingWaveSchemaCreatedTotal   = &RisingWaveUserTotalAdaptor{metric: schemaCreatedTotal}
	RisingWaveSchemaDeletedTotal   = &RisingWaveUserTotalAdaptor{metric: schemaDeletedTotal}
)

func init() {
	metrics.Registry.MustRegister(
		userCreatedTotal,
		userDeletedTotal,
		databaseCreatedTotal,
		databaseDeletedTotal,
		schemaCreatedTotal,
		schemaDeletedTotal,
	)
}
