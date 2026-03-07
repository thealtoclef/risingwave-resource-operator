# RisingWave Kubernetes Resource Operator

A Kubernetes operator for declaratively managing RisingWave resources including users, privileges, databases, schemas, and connections.

## Problem Statement

RisingWave provides SQL-based resource management via `CREATE USER`, `GRANT`, `CREATE DATABASE`, and other statements. Managing these resources across multiple RisingWave deployments, environments, or teams manually is error-prone and lacks:

- **Idempotency**: Retrying manual SQL scripts often fails if objects already exist.
- **Audit Trails**: Difficulty tracking who changed what and when in the cluster.
- **Consistency**: Hard to maintain same resource configurations across dev/staging/prod.
- **Automation**: Manual bottlenecks in user provisioning and environment setup.

This operator addresses these challenges by managing RisingWave resources as standard Kubernetes custom resources.

## Architecture

```text
┌──────────────────────────────────────────────────────────────┐
│                      Kubernetes Cluster                      │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────┐                            │
│  │             CRDs             │                            │
│  │  - User                      │                            │
│  │  - Connection                │                            │
│  │  - Database                  │                            │
│  │  - Schema                    │                            │
│  └──────────────────────────────┘                            │
│                │                                             │
│                ▼                                             │
│  ┌────────────────────────────────────────────────────────┐  │
│  │                 Operator Controller                    │  │
│  │                                                        │  │
│  │  ┌──────────────┐  ┌──────────────┐                    │  │
│  │  │ Reconciler   │  │ Reconciler   │                    │  │
│  │  │ (User)       │  │ (Connection) │                    │  │
│  │  └──────────────┘  └──────────────┘                    │  │
│  │                                                        │  │
│  │  ┌──────────────┐  ┌──────────────┐                    │  │
│  │  │ Reconciler   │  │ Reconciler   │                    │  │
│  │  │ (Database)   │  │ (Schema)     │                    │  │
│  │  └──────────────┘  └──────────────┘                    │  │
│  └────────────────────────────────────────────────────────┘  │
│                         │                                    │
│                         ▼                                    │
│                ┌──────────────────┐                          │
│                │  Connection Pool │                          │
│                └──────────────────┘                          │
│                         │                                    │
└─────────────────────────┼────────────────────────────────────┘
                          │ PostgreSQL Connection
                          ▼
┌──────────────────────────────────────────────────────────────┐
│                      RisingWave Instance                     │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│   Users        Connections        Databases                  │
│                                                              │
│   Schemas      Tables / Views / MVs                          │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Components

| Component                   | Description                                                               |
| ---------------------------- | ------------------------------------------------------------------------- |
| **RisingWaveUser CRD**       | `risingwave.risingwavelabs.com/v1alpha1` — declarative user specification |
| **RisingWaveConnection CRD** | `risingwave.risingwavelabs.com/v1alpha1` — connection configuration       |
| **RisingWaveDatabase CRD**   | `risingwave.risingwavelabs.com/v1alpha1` — database management            |
| **RisingWaveSchema CRD**     | `risingwave.risingwavelabs.com/v1alpha1` — schema-scoped resources        |
| **Controller**               | Reconciliation loop watching CRs and syncing to RisingWave                |
| **Connection Pool**          | PostgreSQL connection pool keyed by `namespace/host:port`                 |
| **Privilege Engine**         | Snapshot-based diff calculation for grant/revoke SQL generation           |
| **SQL Builder**              | SQL generation for CREATE/ALTER/DROP operations for all resource types     |

### Reconciliation Flow

The operator follows a consistent reconciliation pattern for all resource types:

1. **Existence Check**: Verify if the resource (user, database, schema, or connection) exists in RisingWave.
2. **State Snapshot**: Fetch the current configuration and metadata of the existing object.
3. **Diff Calculation**: Compare the actual state with the desired state defined in the CRD spec.
4. **Statement Execution**: Generate and execute the necessary SQL commands (`CREATE`, `ALTER`, `GRANT`, `REVOKE`) to reach the desired state.
5. **Status Update**: Reflect the results in `.status.phase`, `.status.conditions[]`, and observability fields (e.g., `.status.userCreated`).

### Failure Handling

| Failure Mode             | Behavior                                                                        |
| ------------------------ | ------------------------------------------------------------------------------- |
| **Connection Refused**   | Retry with exponential backoff, status: `ConnectionFailed`                      |
| **SQL Syntax Error**     | Log error, mark `Failed`, retry on next sync                                    |
| **Secret Missing**       | Mark `NotReady` with `SecretNotFound` reason                                    |
| **Privilege Conflict**   | Log warning, continue with remaining grants if possible                         |
| **Dependency Missing**   | (e.g., Schema for a Connection) Mark `NotReady` until dependency is satisfied |
| **Update Not Supported** | Some properties cannot be changed; status updated with `UpdateNotSupported`    |

## Installation

### Prerequisites

- Kubernetes 1.28+
- RisingWave instance accessible from operator pod
- Admin credentials for RisingWave (user with privilege grant permissions)

### Deploy Operator

```bash
# Install CRDs
make install

# Deploy with default image
make deploy

# Deploy with custom image
IMG=ghcr.io/yourorg/risingwave-resource-operator:v1.0.0 make deploy
```

### Verify

```bash
kubectl get pods -n risingwave-resource-operator-system
kubectl get crd risingwaveusers.risingwave.risingwavelabs.com
```

## Usage & CRD Reference

### 1. RisingWaveDatabase

Manage databases and their ownership at the cluster level.

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveDatabase
metadata:
  name: analytics-db
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
    port: 4567
  name: analytics             # Optional: RisingWave database name (defaults to metadata.name)
  owner: "analytics_admin"    # Optional: Initial owner of the database
```

### 2. RisingWaveSchema

Manage schemas within a specific database.

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveSchema
metadata:
  name: reports-schema
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
  databaseRef:
    name: analytics           # Required: Target database name
  name: reports               # Optional: RisingWave schema name
  owner: "reports_owner"      # Optional: Initial owner of the schema
```

### 3. RisingWaveConnection

Reusable connection objects for sources, sinks, and tables. Supports literal values and `SECRET` references.

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveConnection
metadata:
  name: kafka-connection
spec:
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
  databaseRef:
    name: analytics
  schemaRef:                  # Optional: Target schema name (defaults to "public")
    name: public
  name: kafka_prod            # Optional: Connection name in RisingWave
  type: kafka
  properties:
    properties.bootstrap.server: "kafka-broker:9092"
    properties.sasl.password: "SECRET kafka_credentials"  # Reference a RisingWave secret
```

### 4. RisingWaveUser

Manage users, authentication methods, and hierarchical privilege grants.

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: alice
spec:
  name: alice
  connectionRef:
    host: risingwave.risingwave.svc.cluster.local
  password:
    generateRandomLength: 16  # Auto-generates secret: risingwave-user-alice
  grants:
    databases:
      - name: dev
        privileges: [CONNECT]
        schemas:
          - name: public
            privileges: [USAGE]
            tables:
              - name: "*"     # Wildcard for all tables in schema
                privileges: [SELECT]
```

**Secret Reference**: Prefix value with `SECRET ` in a `RisingWaveConnection` to reference a RisingWave secret:

- `"SECRET my_secret"` → renders as `SECRET my_secret` in SQL
- `"literal_value"` → renders as `'literal_value'` in SQL

## Supported Object Types

| Object Type           | Privileges                                                         |
| --------------------- | ------------------------------------------------------------------ |
| **Database**          | CONNECT, CREATE, ALL                                               |
| **Schema**            | USAGE, CREATE, ALL                                                 |
| **Table**             | SELECT, INSERT, UPDATE, DELETE, TRUNCATE, REFERENCES, TRIGGER, ALL |
| **View**              | SELECT, INSERT, DELETE, UPDATE, TRIGGER, ALL                       |
| **Materialized View** | SELECT, ALL                                                        |
| **Source**            | SELECT, ALL                                                        |
| **Sink**              | SELECT, ALL                                                        |
| **Connection**        | USAGE, ALL                                                         |
| **Secret**            | USAGE, ALL                                                         |
| **Function**          | EXECUTE, ALL                                                       |

## Annotations

| Annotation                                                 | Effect                                   | Applies To |
| ---------------------------------------------------------- | ---------------------------------------- | ---------- |
| `risingwave.risingwavelabs.com/pause-reconcile: "true"`    | Skip reconciliation for this resource    | All CRDs    |
| `risingwave.risingwavelabs.com/deletion-policy: "abandon"` | Skip `DROP` on resource deletion         | All CRDs    |
| `risingwave.risingwavelabs.com/rotate-password: "true"`    | Trigger password rotation (auto-cleared) | RisingWaveUser |

**Deletion Policy**: `abandon` (default) - Resource is retained. Use `delete` to remove it from RisingWave.

> **⚠️ Destructive Operations Warning**
>
> The operator's `deletion-policy: "delete"` annotation triggers **irreversible operations** that can delete production data:
>
> - **DROP DATABASE**: Deletes the entire database and ALL data within it. Cannot be undone.
> - **DROP SCHEMA CASCADE**: Automatically deletes all objects in the schema (tables, views, materialized views, etc.) and any objects that depend on those objects. Cannot be undone.
>
> **Always use `deletion-policy: "abandon"`** for safety by default unless you explicitly intend to delete the resource and its data.

## Authentication Types

### Password (Default)

```yaml
spec:
  name: myuser
  password:
    generateRandomLength: 16    # 8-32 characters
```

### External Secret

```yaml
spec:
  name: myuser
  password:
    secretRef:
      name: my-secret
      key: password
```

### OAuth

```yaml
spec:
  name: oauth_user
  auth:
    type: oauth
    oauth:
      jwksUrl: "https://example.com/.well-known/jwks.json"
      issuer: "https://example.com"
```

### LDAP

```yaml
spec:
  name: ldap_user
  auth:
    type: ldap
    ldap:
      host: "ldap.example.com"
      port: 389
      baseDN: "dc=example,dc=com"
```

## Observability

### Status Fields

```yaml
status:
  phase: Ready                       # Ready | NotReady
  reason: PrivilegesSynced           # Human-readable reason
  userCreated: true
  privilegesSynced: true
  secretName: risingwave-user-alice
  observedGeneration: 1
```

### Metrics

Exposed on `:8080/metrics` (proxy via `:8443` for Prometheus):

| Metric                                                          | Type    | Description                     |
| --------------------------------------------------------------- | ------- | ------------------------------- |
| `risingwave_user_reconcile_total`                               | Counter | Total reconcile operations      |
| `risingwave_user_reconcile_errors_total`                        | Counter | Failed reconciles               |
| `risingwave_user_privilege_grants_total`                        | Counter | GRANT statements executed       |
| `risingwave_user_privilege_revokes_total`                       | Counter | REVOKE statements executed      |
| `risingwave_connection_created_total`                           | Counter | Connections successfully created |
| `risingwave_connection_deleted_total`                           | Counter | Connections successfully dropped |
| `risingwave_database_created_total`                             | Counter | Databases successfully created  |
| `risingwave_database_deleted_total`                             | Counter | Databases successfully dropped  |
| `risingwave_schema_created_total`                               | Counter | Schemas successfully created    |
| `risingwave_schema_deleted_total`                               | Counter | Schemas successfully dropped    |

### Logging

Controller logs include:

- SQL statements executed (with `GRANT`/`REVOKE` prefix and index)
- Connection pool events (connect, disconnect, errors)
- Reconciliation loop status

Enable verbose logging:

```bash
kubectl logs -n risingwave-resource-operator-system deployment/risingwave-resource-operator-controller-manager
```

## Local Development

### Prerequisites

- Go 1.21+
- kind v0.20+ or minikube
- kubectl v1.28+
- Docker (running)

### Setup

```bash
# Create kind cluster
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
kind create cluster --config /tmp/kind-config.yaml

# Deploy RisingWave
helm repo add risingwavelabs https://risingwavelabs.github.io/helm-charts
helm install risingwave risingwavelabs/risingwave \
  --namespace risingwave \
  --set standalone.enabled=true \
  --set tags.minio=true \
  --set tags.postgresql=true \
  --wait

# Generate code and build
make generate
make manifests
make build

# Load image into kind
docker build -t risingwave-resource-operator:dev .
kind load docker-image risingwave-resource-operator:dev --name risingwave-test

# Deploy operator
make install
IMG=risingwave-resource-operator:dev make deploy
```

### Run Tests

```bash
# All tests
make test

# Unit tests only
go test ./internal/...

# Controller envtest
go test ./internal/controller/... -v

# With coverage
go test -cover ./internal/...
```

### Run Locally (outside cluster)

```bash
# Requires kubeconfig pointing to cluster
make run
```

## Development Workflow

```bash
# 1. Modify API types
vim api/v1alpha1/risingwaveuser_types.go

# 2. Generate deepcopy methods
make generate

# 3. Generate CRD and RBAC manifests
make manifests

# 4. Build binary
make build

# 5. Run unit tests
make test

# 6. Deploy to cluster for testing
make install
IMG=risingwave-resource-operator:dev make deploy
```

## Uninstall

To remove the operator and all its resources from the cluster:

```bash
# 1. Delete all custom resources across all namespaces
kubectl delete risingwaveusers --all -A
kubectl delete risingwaveconnections --all -A
kubectl delete risingwaveschemas --all -A
kubectl delete risingwavedatabases --all -A

# 2. Undeploy operator
make undeploy

# 3. Uninstall CRDs
make uninstall
```

## Documentation

- [Getting Started Guide](docs/getting-started.md) — Installation and comprehensive usage examples
- [Developer Guide](docs/developer-guide.md) — Architecture, local development, and contributing
- [Local Testing Setup](docs/local-testing-setup.md) — Step-by-step guide for a local `kind` environment

## License

Apache License 2.0
