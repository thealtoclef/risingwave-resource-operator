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
)

func TestConnectionKeyString(t *testing.T) {
	tests := []struct {
		name     string
		key      ConnectionKey
		expected string
	}{
		{
			name: "default namespace and port",
			key: ConnectionKey{
				Namespace: "default",
				Host:      "risingwave.example.com",
				Port:      4567,
			},
			expected: "default/risingwave.example.com:4567",
		},
		{
			name: "custom namespace and port",
			key: ConnectionKey{
				Namespace: "prod",
				Host:      "localhost",
				Port:      5432,
			},
			expected: "prod/localhost:5432",
		},
		{
			name: "ip address",
			key: ConnectionKey{
				Namespace: "default",
				Host:      "192.168.1.100",
				Port:      4567,
			},
			expected: "default/192.168.1.100:4567",
		},
		{
			name: "service dns name",
			key: ConnectionKey{
				Namespace: "risingwave",
				Host:      "risingwave-frontend.risingwave.svc.cluster.local",
				Port:      4567,
			},
			expected: "risingwave/risingwave-frontend.risingwave.svc.cluster.local:4567",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.key.String()
			if got != tt.expected {
				t.Errorf("ConnectionKey.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestConnectionKeyFormatConsistency(t *testing.T) {
	// Test that the same inputs always produce the same string
	key1 := ConnectionKey{Namespace: "default", Host: "localhost", Port: 4567}
	key2 := ConnectionKey{Namespace: "default", Host: "localhost", Port: 4567}

	if key1.String() != key2.String() {
		t.Errorf("ConnectionKey.String() not consistent: %q != %q", key1.String(), key2.String())
	}
}

func TestConnectionKeyDifferentInputs(t *testing.T) {
	key1 := ConnectionKey{Namespace: "default", Host: "localhost", Port: 4567}
	key2 := ConnectionKey{Namespace: "prod", Host: "localhost", Port: 4567}
	key3 := ConnectionKey{Namespace: "default", Host: "other-host", Port: 4567}
	key4 := ConnectionKey{Namespace: "default", Host: "localhost", Port: 5432}

	str1 := key1.String()
	str2 := key2.String()
	str3 := key3.String()
	str4 := key4.String()

	if str1 == str2 {
		t.Errorf("Different namespaces should produce different strings: %q == %q", str1, str2)
	}

	if str1 == str3 {
		t.Errorf("Different hosts should produce different strings: %q == %q", str1, str3)
	}

	if str1 == str4 {
		t.Errorf("Different ports should produce different strings: %q == %q", str1, str4)
	}
}
