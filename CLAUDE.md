# Risingwave Resource Operator

Kubernetes operator managing RisingWave logical resources: databases, schemas, connections, users, and privileges via four CRDs (`RisingWaveDatabase`, `RisingWaveSchema`, `RisingWaveConnection`, `RisingWaveUser`).

## Skills

- **writing-documentation** — Use when writing or updating documentation (README, guides, API docs) to ensure quality standards

## Commands

```bash
make generate     # regenerate deepcopy methods (run after types.go changes)
make manifests    # regenerate CRD YAML + RBAC (run after types.go changes)
make fmt          # format code
make vet          # static analysis
make build        # build bin/manager
make test         # all tests with coverage (runs manifests + generate + fmt + vet first)
make run          # run locally (requires kubeconfig)
make install      # install CRDs to cluster
make deploy       # deploy operator (IMG=<image> make deploy)
make uninstall    # uninstall CRDs from cluster
```

Unit tests only (no envtest required):
```bash
go test ./internal/rwclient/... ./internal/utils/... ./internal/constants/... ./internal/metrics/...
```

## Workflow

- After ANY change to `api/v1alpha1/*_types.go`, ALWAYS run both `make generate` and `make manifests`
- Never edit generated files directly: `api/v1alpha1/zz_generated.deepcopy.go`, `config/crd/bases/*.yaml`
- Run `make fmt` and `make vet` before finalizing changes
- Add `//+kubebuilder:` marker comments to source types, never to generated files
- `make test` downloads envtest binaries on first run; subsequent runs are fast

## Architecture

- **API**: `risingwave.risingwavelabs.com/v1alpha1` — see `api/v1alpha1/*_types.go` for all 4 CRDs; grants are hierarchical: databases → schemas → tables/views/MVs/sources/sinks/connections/secrets/functions; object name `"*"` means ALL in schema
- **Controllers**: `internal/controller/*_controller.go` — snapshot-based reconciliation for each resource type; fetches actual state, diffs against spec, executes CREATE/ALTER/DROP/GRANT/REVOKE; status updated at every error path
- **Connection pool**: `internal/rwclient/pool.go` — keyed by `namespace/host:port`; dead connections replaced transparently on next Get(); background health checker runs every 30s, removes stale connections idle >10min
- **Privilege engine**: `internal/rwclient/privilege_snapshot.go` + `privilege_diff.go` — two-phase snapshot fetch (database-level, then per-DB object-level), then set-difference diff
- **Finalizer**: `internal/utils/finalizer.go` — `HandleFinalizer` runs `FinalizationFunc` then removes finalizer; if finalization fails, finalizer stays and object is stuck in Terminating

## Key Gotchas

- **RisingWave has no fully-qualified object syntax**: GRANT/REVOKE statements cannot use `database.schema.object` form. The controller must issue `USE <database>` before executing object-level privilege statements — this is managed in `syncPrivileges()`. Never skip this when adding new privilege types.
- **Abandon deletion policy**: When annotation `risingwave.risingwavelabs.com/deletion-policy: "abandon"` is set, `FinalizationFunc` is skipped entirely — the finalizer is removed without running DROP USER. Check for this in any new finalization logic.
- **Orphaned object revokes fail silently**: If a DB object was deleted from RisingWave, the REVOKE for it will fail. The controller filters errors containing "does not exist"/"not found"/"must be owner" and logs at V(1) instead of failing — this is intentional for idempotency.
- **Status update failures return nil error**: Status update errors during reconciliation return `nil` (not the error), relying on `RequeueAfter` for retry. This is intentional — do not change this pattern.
- **Password management**: If `spec.password.secretRef` is set, the controller uses it as-is and does NOT create a managed Secret. If absent, it auto-generates and stores the password in Secret `risingwave-user-{name}`. OAuth/LDAP auth types skip password management entirely.

## Code Conventions

- Wrap errors with context using `fmt.Errorf("failed to <action>: %w", err)` — never bare `errors.New()` for errors crossing package boundaries
- Use table-driven tests with `t.Run()` for unit tests; use Ginkgo BDD style for controller integration tests
- Controller integration tests live in `internal/controller/` and require envtest; unit tests in other packages use standard `testing.T`
- Define interfaces for testability when a function needs a database connection (`DatabaseAccessor`, `Rows` in `privilege_snapshot.go`)
- Default constructors follow the `DefaultXxxConfig()` + nil-check pattern (see `pool.go`)
