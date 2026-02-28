# RisingWave Kubernetes Resource Operator

A Kubernetes operator for managing RisingWave users and privileges through custom resources.

## Problem Statement

RisingWave provides SQL-based user and privilege management via `CREATE USER`, `GRANT`, and `REVOKE` statements. Managing users across multiple RisingWave deployments, environments, or teams requires:

- Manual SQL execution against each database
- Audit trails for privilege changes
- Consistent privilege templates across environments
- Automated user provisioning/deprovisioning

This operator addresses these challenges by declaratively managing RisingWave users as Kubernetes custom resources.

## Architecture

```
┌─────────────────┐     ┌──────────────────────────┐     ┌─────────────────┐
│ RisingWaveUser  │────▶│ risingwave-resource-     │────▶│  RisingWave     │
│   CR (K8s)      │     │    operator controller   │     │  Database       │
└─────────────────┘     └──────────────────────────┘     └─────────────────┘
                              │                   ▲
                              │                   │
                              ▼                   │
                       ┌─────────────┐            │
                       │ K8s Secrets │            │
                       │ (passwords) │────────────┘
                       └─────────────┘
```

### Components

| Component | Description |
|-----------|-------------|
| **RisingWaveUser CRD** | `risingwave.risingwavelabs.com/v1alpha1` — declarative user specification |
| **Controller** | Reconciliation loop watching CRs and syncing to RisingWave |
| **Connection Pool** | PostgreSQL connection pool keyed by `namespace/host:port` |
| **Privilege Engine** | Snapshot-based diff calculation for grant/revoke SQL generation |

### Reconciliation Flow

1. **User Creation**: Execute `CREATE USER` with password (auto-generated or from secret)
2. **Privilege Snapshot**: Fetch current privileges from `pg_user_privileges` views
3. **Diff Calculation**: Compare actual vs. desired state, generate GRANT/REVOKE statements
4. **Statement Execution**: Execute REVOKEs first, then GRANTs (per database context)
5. **Status Update**: Reflect sync state in `.status.phase` and `.status.conditions[]`

### Failure Handling

| Failure Mode | Behavior |
|--------------|----------|
| Connection refused | Retry with exponential backoff, update status with `ConnectionFailed` reason |
| Invalid privilege | Log SQL error, continue with remaining statements |
| Object doesn't exist | Skip GRANT, log warning (idempotent on subsequent reconciles) |
| Secret missing | Mark `NotReady` with `SecretNotFound` reason |

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

## Usage

### Basic User with Table Privileges

```yaml
apiVersion: risingwave.risingwavelabs.com/v1alpha1
kind: RisingWaveUser
metadata:
  name: alice
  namespace: default
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
kubectl apply -f alice-user.yaml
kubectl get risingwaveuser alice
kubectl get secret risingwave-user-alice -o jsonpath='{.data.password}' | base64 -d
```

### Wildcard Privileges

```yaml
grants:
  databases:
    - name: dev
      privileges:
        - CONNECT
      schemas:
        - name: public
          tables:
            - name: "*"              # All tables in schema
              privileges:
                - SELECT
```

Generates: `GRANT SELECT ON ALL TABLES IN SCHEMA "public" TO "alice"`

### Multi-Database Support

```yaml
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
    - name: analytics
      privileges:
        - CONNECT
        - CREATE
      schemas:
        - name: reporting
          tables:
            - name: reports
              privileges:
                - SELECT
                - INSERT
```

Each database's privileges are executed with proper `USE <database>` context switching.

## Supported Object Types

| Object Type | Privileges |
|-------------|------------|
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

## Annotations

| Annotation | Effect |
|------------|--------|
| `risingwave.risingwavelabs.com/pause-reconcile: "true"` | Skip reconciliation for this resource |
| `risingwave.risingwavelabs.com/deletion-policy: "abandon"` | Skip `DROP USER` on resource deletion |
| `risingwave.risingwavelabs.com/rotate-password: "true"` | Trigger password rotation (auto-cleared) |

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

| Metric | Type | Description |
|--------|------|-------------|
| `risingwave_user_reconcile_total` | Counter | Total reconcile operations |
| `risingwave_user_reconcile_errors_total` | Counter | Failed reconciles |
| `risingwave_user_privilege_grants_total` | Counter | GRANT statements executed |
| `risingwave_user_privilege_revokes_total` | Counter | REVOKE statements executed |

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

```bash
# Delete all RisingWaveUser resources first
kubectl delete risingwaveusers --all --all-namespaces

# Undeploy operator
make undeploy

# Uninstall CRDs
make uninstall
```

## Documentation

- [Getting Started Guide](docs/getting-started.md) — Installation and usage examples
- [Developer Guide](docs/developer-guide.md) — Architecture and contributing
- [Local Testing Setup](docs/local-testing-setup.md) — Complete local development walkthrough
- [Implementation Review](docs/implementation-review.md) — Technical implementation details

## License

Apache License 2.0
