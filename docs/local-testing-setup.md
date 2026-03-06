# Local Testing Setup Guide

This guide walks you through setting up a complete local Kubernetes testing environment for the RisingWave Resource Operator.

## Prerequisites

- **Go**: 1.21+
- **kind**: v0.20+ ([install](https://kind.sigs.k8s.io/docs/user/quick-start/#installation))
- **kubectl**: v1.28+ ([install](https://kubernetes.io/docs/tasks/tools/))
- **Docker**: running (kind uses Docker containers as nodes)
- **psql**: PostgreSQL client ([install](https://www.postgresql.org/download/))

Verify installation:

```bash
go version
kind version
kubectl version --client
docker version
psql --version
```

## Step 1: Create a Kind Cluster

Create a kind cluster with port mappings:

```bash
cat > /tmp/kind-config.yaml <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: risingwave-test
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 4567
    hostPort: 4567
    protocol: TCP
EOF
```

Create the cluster:

```bash
kind create cluster --config /tmp/kind-config.yaml
```

Verify:

```bash
kubectl cluster-info
kubectl get nodes
```

## Step 2: Deploy RisingWave

### Option A: Using Helm (Recommended)

```bash
# Add Helm repository
helm repo add risingwavelabs https://risingwavelabs.github.io/helm-charts
helm repo update

# Create namespace
kubectl create namespace risingwave

# Deploy RisingWave (standalone with bundled MinIO state store and PostgreSQL meta store)
helm install risingwave risingwavelabs/risingwave \
  --namespace risingwave \
  --set standalone.enabled=true \
  --set tags.minio=true \
  --set tags.postgresql=true \
  --wait \
  --timeout 8m

# Verify deployment
kubectl get pods -n risingwave
kubectl get svc -n risingwave
```

### Option B: Manual YAML Deployment

```bash
kubectl create namespace risingwave
kubectl apply -f https://raw.githubusercontent.com/risingwavelabs/risingwave/main/k8s/standalone-manifests.yaml -n risingwave
kubectl wait --for=condition=ready pod -l app=risingwave -n risingwave --timeout=300s
```

## Step 3: Build and Deploy the Operator

### Generate Code and CRDs

```bash
cd /path/to/risingwave-resource-operator
make generate
make manifests
```

### Build and Load Image

```bash
# Build operator
make build

# Build Docker image
docker build -t risingwave-resource-operator:dev .

# Load into kind cluster
kind load docker-image risingwave-resource-operator:dev --name risingwave-test
```

### Deploy Operator

```bash
# Install CRDs
make install

# Deploy operator with local image
IMG=risingwave-resource-operator:dev make deploy

# Verify deployment
kubectl get pods -n risingwave-resource-operator-system
```

## Step 4: Verify RisingWave Connection

```bash
# Port-forward to RisingWave
kubectl port-forward -n risingwave svc/risingwave 4567:4567 &
PF_PID=$!

# Test connection
PGPASSWORD="" psql -h localhost -p 4567 -U root -d dev -c "SELECT 1"

# Expected output: 1
```

## Step 5: Create Test Users

### 5.1: Create Test Namespace

```bash
kubectl create namespace test-users
```

### 5.2: Basic User with Table Privileges

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: alice
  namespace: test-users
spec:
  name: alice
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    generateRandomLength: 16
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
        schemas:
          - name: public
            privileges:
              - USAGE
            tables:
              - name: users
                privileges:
                  - SELECT
                  - INSERT
```

```bash
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: alice
  namespace: test-users
spec:
  name: alice
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    generateRandomLength: 16
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
        schemas:
          - name: public
            privileges:
              - USAGE
            tables:
              - name: users
                privileges:
                  - SELECT
                  - INSERT
EOF
```

### 5.3: Multi-Database User

```bash
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: bob
  namespace: test-users
spec:
  name: bob
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    generateRandomLength: 16
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
        schemas:
          - name: public
            tables:
              - name: users
                privileges:
                  - SELECT
      - name: test
        privileges:
          - CONNECT
          - CREATE
        schemas:
          - name: analytics
            tables:
              - name: events
                privileges:
                  - SELECT
                  - INSERT
EOF
```

### 5.4: Wildcard Privileges User

```bash
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: charlie
  namespace: test-users
spec:
  name: charlie
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    generateRandomLength: 16
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
        schemas:
          - name: public
            tables:
              - name: "*"
                privileges:
                  - SELECT
EOF
```

### 5.5: OAuth Authentication User

```bash
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: oauth-user
  namespace: test-users
spec:
  name: oauth_user
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  auth:
    type: oauth
    oauth:
      jwksUrl: "https://example.com/.well-known/jwks.json"
      issuer: "https://example.com"
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
        schemas:
          - name: public
            privileges:
              - USAGE
EOF
```

### 5.6: LDAP Authentication User

```bash
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: ldap-user
  namespace: test-users
spec:
  name: ldap_user
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  auth:
    type: ldap
    ldap:
      host: "ldap-server.local"
      port: 389
      baseDN: "dc=local"
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
        schemas:
          - name: public
            privileges:
              - USAGE
EOF
```

## Step 6: Monitor and Verify

### Check User Status

```bash
# List all users
kubectl get risingwaveuser -n test-users

# Get detailed status
kubectl get risingwaveuser -n test-users alice -o yaml

# Watch operator logs
kubectl logs -f -n risingwave-resource-operator-system \
  deployment/risingwave-resource-operator-controller-manager
```

### Retrieve Generated Password

```bash
# Get password for alice
kubectl get secret -n test-users risingwave-user-alice \
  -o jsonpath='{.data.password}' | base64 -d
```

### Verify User in RisingWave

```bash
# Port-forward (if not already running)
kubectl port-forward -n risingwave svc/risingwave 4567:4567 &
PF_PID=$!

# Connect as root
PGPASSWORD="" psql -h localhost -p 4567 -U root -d dev

# List users
\du

# Check table privileges
SELECT * FROM pg_table_privileges WHERE grantee = 'alice';

# Clean up port-forward when done
kill $PF_PID
```

## Step 7: Test Operator Features

### Password Rotation

```bash
kubectl annotate risingwaveuser alice \
  risingwave.risingwavelabs.com/rotate-password="true" \
  -n test-users --overwrite
```

### Pause Reconciliation

```bash
kubectl annotate risingwaveuser alice \
  risingwave.risingwavelabs.com/pause-reconcile="true" \
  -n test-users --overwrite
```

### Abandon Deletion Policy

```bash
kubectl annotate risingwaveuser bob \
  risingwave.risingwavelabs.com/deletion-policy="abandon" \
  -n test-users --overwrite
```

## Step 8: Test All Object Types

Comprehensive test covering all 8+ object types:

```bash
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: comprehensive-user
  namespace: test-users
spec:
  name: comprehensive_user
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  password:
    generateRandomLength: 16
  grants:
    databases:
      - name: dev
        privileges:
          - CONNECT
          - CREATE
        schemas:
          - name: public
            privileges:
              - USAGE
              - CREATE
            tables:
              - name: users
                privileges:
                  - SELECT
                  - INSERT
                  - UPDATE
                  - DELETE
            views:
              - name: user_summary
                privileges:
                  - SELECT
                  - INSERT
            materializedViews:
              - name: user_metrics
                privileges:
                  - SELECT
            sources:
              - name: events_source
                privileges:
                  - SELECT
            sinks:
              - name: results_sink
                privileges:
                  - SELECT
            connections:
              - name: kafka_connection
                privileges:
                  - USAGE
            secrets:
              - name: aws_credentials
                privileges:
                  - USAGE
            functions:
              - name: process_data
                privileges:
                  - EXECUTE
EOF
```

## Step 9: Test New CRDs

### 9.1: Test RisingWaveDatabase

```bash
kubectl create namespace test-databases

kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveDatabase
metadata:
  name: analytics-db
  namespace: test-databases
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  name: analytics
  owner: "analytics_admin"
EOF

# Check status
kubectl get risingwavedatabase analytics-db -n test-databases
kubectl describe risingwavedatabase analytics-db -n test-databases
```

### 9.2: Test RisingWaveSchema

```bash
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveSchema
metadata:
  name: reports-schema
  namespace: test-databases
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  databaseRef:
    name: analytics
  name: reports
EOF

# Check status
kubectl get risingwaveschema reports-schema -n test-databases
```

### 9.3: Test RisingWaveConnection (Kafka)

```bash
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveConnection
metadata:
  name: kafka-test
  namespace: test-databases
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  databaseRef:
    name: analytics
  name: kafka_test
  type: kafka
  properties:
    properties.bootstrap.server: "kafka-broker:9092"
    properties.security.protocol: "PLAINTEXT"
EOF

# Check status
kubectl get risingwaveconnection kafka-test -n test-databases
```

### 9.4: Test Deletion Policy

```bash
# Test abandon (default)
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveDatabase
metadata:
  name: db-abandon
  namespace: test-databases
  annotations:
    risingwave.risingwavelabs.com/deletion-policy: "abandon"
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  name: db_abandon
EOF

# Delete CR - database should be retained
kubectl delete risingwavedatabase db-abandon -n test-databases
kubectl get risingwavedatabase db-abandon -n test-databases  # Should still exist

# Test delete
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveDatabase
metadata:
  name: db-delete
  namespace: test-databases
  annotations:
    risingwave.risingwavelabs.com/deletion-policy: "delete"
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  name: db_delete
EOF

# Delete CR - database should be dropped
kubectl delete risingwavedatabase db-delete -n test-databases
kubectl get risingwavedatabase db-delete -n test-databases  # Should not exist
```

### 9.5: Test Secret Reference

```bash
# First, create a RisingWave secret (requires SecretManagement feature)
# This is a premium feature and requires proper licensing

# Test with literal values
kubectl apply -f - <<EOF
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveConnection
metadata:
  name: iceberg-literal
  namespace: test-databases
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
    credentials:
      username: root
      password: ""
  databaseRef:
    name: analytics
  name: iceberg_literal
  type: iceberg
  properties:
    catalog.type: "storage"
    catalog.name: "demo"
    warehouse.path: "s3a://test-bucket/"
    s3.endpoint: "http://minio:9000"
    s3.access.key: "minioadmin"
    s3.secret.key: "minioadmin"
EOF

# Check status
kubectl get risingwaveconnection iceberg-literal -n test-databases
```

## Step 10: Run Tests

```bash
# Run all unit tests
make test

# Run specific package tests
go test ./internal/rwclient/... -v
go test ./internal/controller/... -v

# Run with coverage
go test -cover ./internal/...
```

## Troubleshooting

### Operator Pod Issues

```bash
# Check pod status
kubectl get pods -n risingwave-resource-operator-system

# View logs
kubectl logs -n risingwave-resource-operator-system \
  deployment/risingwave-resource-operator-controller-manager

# Describe pod for events
kubectl describe pod -n risingwave-resource-operator-system \
  <pod-name>
```

### RisingWave Connection Issues

```bash
# Check RisingWave pods
kubectl get pods -n risingwave

# Check service
kubectl get svc -n risingwave

# Test from operator pod
kubectl exec -n risingwave-resource-operator-system \
  <pod-name> -- nc -zv risingwave.risingwave.svc.cluster.local 4567
```

### Common Issues

| Issue | Solution |
|-------|----------|
| User not created | Check logs for SQL errors, verify connection to RisingWave |
| Secret not created | Check RBAC, ensure operator has permissions to create secrets |
| Privileges not granted | Check `status.phase` is `Ready`, verify database/schema exists |
| Password mismatch | Use rotation annotation to generate new password |
| Connection refused | Verify service name, port, and network policies |

## Cleanup

```bash
# Delete test users
kubectl delete risingwaveuser -n test-users --all
kubectl delete namespace test-users

# Uninstall operator
make uninstall

# Delete RisingWave
helm uninstall risingwave -n risingwave
kubectl delete namespace risingwave

# Delete kind cluster
kind delete cluster --name risingwave-test
```

## API Reference

### RisingWaveUser Spec Structure

```yaml
spec:
  name: string                          # RisingWave username
  connectionRef:
    host: string                        # RisingWave service host
    port: int32                          # Port (default: 4567)
    credentials:
      username: string
      password: string
      passwordSecretRef:
        name: string
        namespace: string
        key: string
  password:
    secretRef:
      name: string
      namespace: string
      key: string
    generateRandomLength: int32          # 8-32
  auth:
    type: string                        # "password", "oauth", "ldap"
    oauth:
      jwksUrl: string
      issuer: string
    ldap:
      host: string
      port: int32
      baseDN: string
  permissions: [string]                  # User-level permissions
  grants:
    databases:
      - name: string
        privileges: [DatabasePrivilegeType]  # CONNECT, CREATE, ALL
        withGrantOption: bool
        schemas:
          - name: string
            privileges: [SchemaPrivilegeType]    # USAGE, CREATE, ALL
            withGrantOption: bool
            tables:
              - name: string              # or "*" for wildcard
                privileges: [TablePrivilegeType]
                withGrantOption: bool
            views:
              - name: string
                privileges: [ViewPrivilegeType]
            materializedViews:
              - name: string
                privileges: [MaterializedViewPrivilegeType]
            sources:
              - name: string
                privileges: [SourcePrivilegeType]
            sinks:
              - name: string
                privileges: [SinkPrivilegeType]
            connections:
              - name: string
                privileges: [ConnectionPrivilegeType]
            secrets:
              - name: string
                privileges: [SecretPrivilegeType]
            functions:
              - name: string
                privileges: [FunctionPrivilegeType]
```

## Annotations

| Annotation | Description |
|------------|-------------|
| `risingwave.risingwavelabs.com/pause-reconcile` | Pause reconciliation (value: "true") |
| `risingwave.risingwavelabs.com/deletion-policy` | Deletion policy (value: "abandon") |
| `risingwave.risingwavelabs.com/rotate-password` | Rotate password (value: "true") |

## Supported Object Types

| Object Type | Supported Privileges |
|-------------|---------------------|
| **Database** | CONNECT, CREATE, ALL |
| **Schema** | USAGE, CREATE, ALL |
| **Table** | SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER, ALL |
| **View** | SELECT, INSERT, DELETE, UPDATE, TRIGGER, ALL |
| **Materialized View** | SELECT, ALL |
| **Source** | SELECT, ALL |
| **Sink** | SELECT, ALL |
| **Connection** | USAGE, ALL |
| **Secret** | USAGE, ALL |
| **Function** | EXECUTE, ALL |

---

Last updated: 2026-02-28
