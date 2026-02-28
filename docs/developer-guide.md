# Developer Guide for risingwave-resource-operator

This guide covers local development, testing, and deployment of the RisingWave resource operator.

## Architecture Overview

### Reconciliation Flow

1. **Watch** — Operator watches for changes to `RisingWaveUser` resources
2. **Fetch snapshot** — Read current privileges from RisingWave database
3. **Calculate diff** — Compare desired state (spec) with actual state (database)
4. **Apply changes** — Execute GRANT/REVOKE/CREATE/ALTER/DROP statements
5. **Update status** — Set phase and conditions on the resource
6. **Cleanup** — Remove finalizer and delete associated secrets

### Connection Pool

The operator maintains a connection pool (`internal/rwclient.Pool`) keyed by `namespace/host:port`:

- Reuses connections across multiple users in the same RisingWave instance
- Handles PostgreSQL wire protocol (default port: 4567)
- Closes connections when not in use

### Key Components

| Component | Purpose |
|-----------|---------|
| `internal/rwclient.Pool` | PostgreSQL connection management |
| `internal/rwclient.sql_builder.go` | SQL statement generation (CREATE USER, GRANT, etc.) |
| `internal/rwclient.privilege_snapshot.go` | Fetch privileges from database |
| `internal/rwclient.privilege_diff.go` | Calculate diffs between desired and actual |
| `internal/rwclient.acl_parser.go` | Parse RisingWave ACL format |
| `internal/utils.password.go` | Generate random passwords |
| `internal/controller.risingwaveuser_controller.go` | Main reconciler loop |

## Local Development

### Prerequisites

- Go 1.21+
- Kubebuilder v3+
- A local Kubernetes cluster (e.g., kind, minikube, or Docker Desktop)
- A running RisingWave instance

### Generate and Build

```bash
# Generate deepcopy methods
make generate

# Generate CRD YAML
make manifests

# Format code
make fmt

# Vet code
make vet

# Build binary
make build
```

### Run Locally

```bash
# Install CRDs into cluster
make install

# Run controller locally against cluster
make run
```

Logs will print to stdout.

### Run Tests

```bash
# Run all tests (unit + integration)
make test

# Run only unit tests (no database/cluster needed)
go test ./internal/rwclient/... ./internal/utils/...

# Run with coverage
go test -cover ./...
```

## Testing Strategy

### Unit Tests (No Database Required)

Located in `internal/rwclient/` and `internal/utils/`:

- **sql_builder_test.go** — SQL statement generation (CREATE USER, GRANT, REVOKE, etc.)
- **acl_parser_test.go** — ACL string parsing and privilege mapping
- **privilege_diff_test.go** — Privilege diff calculation
- **password_test.go** — Random password generation and properties

Run unit tests:

```bash
go test -v ./internal/rwclient/... ./internal/utils/...
```

### Integration Tests

Located in `internal/controller/`:

- Require a Kubernetes API server (provided by envtest)
- Use Ginkgo BDD framework
- Mock RisingWave database calls

Run integration tests:

```bash
make test
```

## Package Overview

| Package | Files | Responsibility |
|---------|-------|-----------------|
| `api/v1alpha1/` | `*_types.go`, `*_webhook.go` | CRD definitions and webhooks |
| `internal/rwclient/` | `pool.go`, `sql_builder.go`, `acl_parser.go`, etc. | RisingWave database interaction |
| `internal/controller/` | `risingwaveuser_controller.go` | Reconciliation logic |
| `internal/utils/` | `finalizer.go`, `password.go` | Utility functions |
| `internal/constants/` | `constants.go` | Phase names, reasons, annotations |
| `config/` | `crd/`, `rbac/`, `manager/`, `default/` | Kubernetes manifests |

## Deploy to Cluster

### Using Kustomize

```bash
# Deploy with default image
make deploy

# Deploy with custom image
make deploy IMG=ghcr.io/myregistry/risingwave-resource-operator:v1.0.0

# Verify
kubectl get pods -n risingwave-resource-operator-system
kubectl logs -n risingwave-resource-operator-system -l control-plane=controller-manager -f
```

### Manual Steps (Advanced)

```bash
# Build and push image
docker build -t ghcr.io/myregistry/risingwave-resource-operator:v1.0.0 .
docker push ghcr.io/myregistry/risingwave-resource-operator:v1.0.0

# Apply manifests
kustomize build config/crd | kubectl apply -f -
kustomize build config/rbac | kubectl apply -f -
kustomize build config/manager | kubectl set image deployment/controller-manager controller=ghcr.io/myregistry/risingwave-resource-operator:v1.0.0 -n system | kubectl apply -f -
```

## Enable Prometheus Monitoring

The operator exposes metrics on `https://0.0.0.0:8443/metrics` via kube-rbac-proxy.

### Add ServiceMonitor (requires Prometheus Operator)

Uncomment the prometheus config in `config/default/kustomization.yaml`:

```yaml
patchesStrategicMerge:
- manager_auth_proxy_patch.yaml
resources:
- ../prometheus  # Uncomment this line
```

Deploy:

```bash
make deploy
```

### Scrape Metrics

Configure Prometheus to scrape the operator:

```yaml
global:
  scrape_interval: 30s
scrape_configs:
- job_name: risingwave-resource-operator
  kubernetes_sd_configs:
  - role: endpoints
    namespaces:
      names:
      - risingwave-resource-operator-system
  relabel_configs:
  - source_labels: [__meta_kubernetes_service_label_control_plane]
    action: keep
    regex: controller-manager
```

## Code Style and Standards

- **Formatting**: Use `make fmt`
- **Linting**: Use `make lint` (requires golangci-lint)
- **Imports**: Organize with blank lines (standard library, Kubernetes, others)
- **Comments**: Use clear, concise English
- **Testing**: Write table-driven tests with descriptive names

## Making Changes

### Adding a New Field to RisingWaveUser

1. Add the field to `api/v1alpha1/risingwaveuser_types.go`
2. Regenerate code: `make generate`
3. Regenerate manifests: `make manifests`
4. Update the controller if needed: `internal/controller/risingwaveuser_controller.go`
5. Add unit tests for new logic
6. Run `make test` to verify

### Adding a New Reconciliation Step

1. Add logic to `internal/controller/risingwaveuser_controller.go`
2. Add unit tests in `internal/controller/risingwaveuser_controller_test.go`
3. Run `make test`
4. Test manually with a sample resource

## Common Commands

```bash
# Development workflow
make generate manifests fmt vet build

# Test workflow
make test

# Deploy workflow
make docker-build docker-push IMG=<your-image> && make deploy IMG=<your-image>

# Clean up
make undeploy uninstall
```

## References

- [Kubebuilder Book](https://book.kubebuilder.io/)
- [Controller-Runtime Documentation](https://pkg.go.dev/sigs.k8s.io/controller-runtime)
- [RisingWave SQL Documentation](https://docs.risingwave.com/)
- [Kubernetes API Conventions](https://kubernetes.io/docs/concepts/overview/kubernetes-api/)
