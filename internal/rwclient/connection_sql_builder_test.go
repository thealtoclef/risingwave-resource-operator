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

package rwclient

import (
	"testing"

	v1alpha1 "github.com/risingwavelabs/risingwave-resource-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildCreateConnectionSQL(t *testing.T) {
	tests := []struct {
		name     string
		conn     *v1alpha1.RisingWaveConnection
		expected string
	}{
		{
			name: "kafka connection with literal values",
			conn: &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{Name: "kafka-conn"},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					Name: "kafka_prod",
					Type: v1alpha1.ConnectionTypeKafka,
					Properties: map[string]string{
						"properties.bootstrap.server":  "kafka-broker-1:9092",
						"properties.security.protocol": "SASL_SSL",
					},
				},
			},
			expected: `CREATE CONNECTION "kafka_prod"
WITH (
    type = 'kafka',
    properties.bootstrap.server = 'kafka-broker-1:9092',
    properties.security.protocol = 'SASL_SSL'
)`,
		},
		{
			name: "kafka connection with secret reference",
			conn: &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{Name: "kafka-conn"},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					Name: "kafka_with_secret",
					Type: v1alpha1.ConnectionTypeKafka,
					Properties: map[string]string{
						"properties.bootstrap.server": "kafka-broker-1:9092",
						"properties.sasl.password":    "SECRET kafka_credentials",
					},
				},
			},
			expected: `CREATE CONNECTION "kafka_with_secret"
WITH (
    type = 'kafka',
    properties.bootstrap.server = 'kafka-broker-1:9092',
    properties.sasl.password = SECRET "kafka_credentials"
)`,
		},
		{
			name: "schema registry connection",
			conn: &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{Name: "sr-conn"},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					Type: v1alpha1.ConnectionTypeSchemaRegistry,
					Properties: map[string]string{
						"schema.registry": "https://schema-registry:8081",
					},
				},
			},
			expected: `CREATE CONNECTION "sr-conn"
WITH (
    type = 'schema_registry',
    schema.registry = 'https://schema-registry:8081'
)`,
		},
		{
			name: "iceberg connection with mixed properties",
			conn: &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{Name: "iceberg-conn"},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					Type: v1alpha1.ConnectionTypeIceberg,
					Properties: map[string]string{
						"catalog.name":   "demo",
						"catalog.type":   "storage",
						"s3.access.key":  "hummockadmin",
						"s3.secret.key":  "SECRET iceberg_s3_credentials",
						"s3.endpoint":    "http://127.0.0.1:9301",
						"warehouse.path": "s3a://hummock001/iceberg-data",
					},
				},
			},
			expected: `CREATE CONNECTION "iceberg-conn"
WITH (
    type = 'iceberg',
    catalog.name = 'demo',
    catalog.type = 'storage',
    s3.access.key = 'hummockadmin',
    s3.endpoint = 'http://127.0.0.1:9301',
    s3.secret.key = SECRET "iceberg_s3_credentials",
    warehouse.path = 's3a://hummock001/iceberg-data'
)`,
		},
		{
			name: "connection with special characters in value",
			conn: &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{Name: "special-conn"},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					Type: v1alpha1.ConnectionTypeKafka,
					Properties: map[string]string{
						"properties.bootstrap.server": "kafka's-broker:9092",
					},
				},
			},
			expected: `CREATE CONNECTION "special-conn"
WITH (
    type = 'kafka',
    properties.bootstrap.server = 'kafka''s-broker:9092'
)`,
		},
		{
			name: "connection with no properties",
			conn: &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{Name: "minimal-conn"},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					Name: "minimal",
					Type: v1alpha1.ConnectionTypeKafka,
				},
			},
			expected: `CREATE CONNECTION "minimal"
WITH (
    type = 'kafka'
)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildCreateConnectionSQL(tt.conn)
			if got != tt.expected {
				t.Errorf("BuildCreateConnectionSQL() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}

func TestBuildDropConnectionSQL(t *testing.T) {
	tests := []struct {
		name     string
		connName string
		expected string
	}{
		{
			name:     "basic drop connection",
			connName: "kafka_prod",
			expected: `DROP CONNECTION IF EXISTS "kafka_prod"`,
		},
		{
			name:     "connection with special characters",
			connName: "my-connection.v1",
			expected: `DROP CONNECTION IF EXISTS "my-connection.v1"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildDropConnectionSQL(tt.connName)
			if got != tt.expected {
				t.Errorf("BuildDropConnectionSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestBuildAlterConnectionSQL(t *testing.T) {
	tests := []struct {
		name     string
		conn     *v1alpha1.RisingWaveConnection
		expected string
	}{
		{
			name: "alter connection with literal values",
			conn: &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{Name: "kafka-conn"},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					Name: "kafka_prod",
					Type: v1alpha1.ConnectionTypeKafka,
					Properties: map[string]string{
						"properties.bootstrap.server":  "new-broker:9092",
						"properties.security.protocol": "SASL_SSL",
					},
				},
			},
			expected: `ALTER CONNECTION "kafka_prod"
CONNECTOR WITH (
    properties.bootstrap.server = 'new-broker:9092',
    properties.security.protocol = 'SASL_SSL'
)`,
		},
		{
			name: "alter connection with secret reference",
			conn: &v1alpha1.RisingWaveConnection{
				ObjectMeta: metav1.ObjectMeta{Name: "kafka-conn"},
				Spec: v1alpha1.RisingWaveConnectionSpec{
					Name: "kafka_with_secret",
					Type: v1alpha1.ConnectionTypeKafka,
					Properties: map[string]string{
						"properties.sasl.password": "SECRET new_credentials",
					},
				},
			},
			expected: `ALTER CONNECTION "kafka_with_secret"
CONNECTOR WITH (
    properties.sasl.password = SECRET "new_credentials"
)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAlterConnectionSQL(tt.conn)
			if got != tt.expected {
				t.Errorf("BuildAlterConnectionSQL() =\n%q\nwant\n%q", got, tt.expected)
			}
		})
	}
}

func TestBuildAlterConnectionOwnerSQL(t *testing.T) {
	tests := []struct {
		name     string
		connName string
		owner    string
		expected string
	}{
		{
			name:     "basic alter owner",
			connName: "kafka_prod",
			owner:    "alice",
			expected: `ALTER CONNECTION "kafka_prod" OWNER TO "alice"`,
		},
		{
			name:     "owner with special characters",
			connName: "my-conn",
			owner:    `user"name`,
			expected: `ALTER CONNECTION "my-conn" OWNER TO "user""name"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildAlterConnectionOwnerSQL(tt.connName, tt.owner)
			if got != tt.expected {
				t.Errorf("BuildAlterConnectionOwnerSQL() = %q, want %q", got, tt.expected)
			}
		})
	}
}
