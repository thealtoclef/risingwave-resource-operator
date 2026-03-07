---
name: writing-and-running-tests
description: Use when writing or running tests for this Kubernetes operator codebase - covers unit tests, integration tests with envtest, and codebase-specific testing patterns
---

# Writing and Running Tests

## Overview

This codebase uses two testing approaches: **standard Go tests** for unit tests and **Ginkgo BDD** for controller integration tests. Understanding which to use and how to run them is critical for efficient development.

## Test Types

| Type | Location | Framework | Requires envtest | Speed |
|------|----------|-----------|------------------|-------|
| Unit tests | `internal/rwclient/`, `internal/utils/`, `internal/constants/`, `internal/metrics/` | `testing.T` + table-driven | No | Fast |
| Integration tests | `internal/controller/` | Ginkgo BDD | Yes | Slower |

## Running Tests

### Unit Tests Only (Fast, No Dependencies)

```bash
go test ./internal/rwclient/... ./internal/utils/... ./internal/constants/... ./internal/metrics/... -v
```

### Full Test Suite (Includes Integration)

```bash
make test
```

This automatically:
1. Runs `make generate` and `make manifests`
2. Downloads envtest binaries on first run
3. Runs all tests with coverage

### Run Specific Package

```bash
go test ./internal/constants/... -v
go test ./internal/controller/... -v  # Requires envtest
```

## Unit Test Pattern

Use table-driven tests with `t.Run()`:

```go
func TestQuoteIdentifier(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "ordinary name",
            input:    "mytable",
            expected: `"mytable"`,
        },
        {
            name:     "wildcard passthrough",
            input:    "*",
            expected: "*",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := QuoteIdentifier(tt.input)
            if got != tt.expected {
                t.Errorf("QuoteIdentifier() = %q, want %q", got, tt.expected)
            }
        })
    }
}
```

## Integration Test Pattern (Ginkgo)

Controller tests use Ginkgo BDD style:

```go
var _ = Describe("Creating a RisingWaveSchema", func() {
    It("should be stored correctly", func() {
        ctx := context.Background()
        rwSchema := &v1alpha1.RisingWaveSchema{
            ObjectMeta: metav1.ObjectMeta{
                Name:      "test-schema",
                Namespace: "default",
            },
            Spec: v1alpha1.RisingWaveSchemaSpec{
                // ... spec fields
            },
        }
        Expect(k8sClient.Create(ctx, rwSchema)).To(Succeed())

        fetched := &v1alpha1.RisingWaveSchema{}
        Expect(k8sClient.Get(ctx, types.NamespacedName{
            Name:      rwSchema.Name,
            Namespace: rwSchema.Namespace,
        }, fetched)).To(Succeed())

        Expect(fetched.Spec.Name).To(Equal("expected-value"))
    })
})
```

## Key Gotchas

### Kubernetes Resource Names Must Be RFC 1123 Compliant

Resource names in tests must use lowercase alphanumeric characters, hyphens, or dots only:

```go
// ❌ BAD: Underscores are invalid
Name: "type-test-schema_registry"

// ✅ GOOD: Use hyphens instead
Name: "type-test-schema-registry"
```

If using a value that might contain underscores, sanitize it:

```go
sanitizedName := strings.ReplaceAll(string(connType), "_", "-")
```

### Ginkgo Version Mismatch Warnings

You may see warnings about Ginkgo CLI version mismatch. These can be ignored if tests pass. The `make test` target uses the correct version.

### envtest Binary Location

envtest binaries are downloaded to `~/Library/Application Support/io.kubebuilder.envtest/` on macOS. If tests fail with "no such file or directory", run `make test` once to download them.

## Test File Naming

- Test files: `*_test.go`
- Test functions: `TestXxx(t *testing.T)` for unit tests
- Ginkgo specs: `var _ = Describe(...)` in `*_test.go` files

## Best Practices

1. **Run unit tests first** - They're fast and catch most issues
2. **Use table-driven tests** - One test function, multiple cases
3. **Test edge cases** - Empty strings, special characters, nil values
4. **Use `t.Run()`** - Groups related tests and improves output
5. **Check error messages** - Use `%q` for strings to see exact values
6. **Sanitize K8s names** - Ensure test resource names are RFC 1123 compliant
