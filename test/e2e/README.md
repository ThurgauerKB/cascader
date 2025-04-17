# End-to-End (E2E) Tests for Cascader

This directory contains all E2E tests for Cascader, organized by workload type and test focus.

## Structure

| File                  | Purpose                                                                                                     |
| --------------------- | ----------------------------------------------------------------------------------------------------------- |
| `deployment_test.go`  | Tests for Deployment-specific behavior and edge cases.                                                      |
| `statefulset_test.go` | Tests for StatefulSet-specific behavior and edge cases.                                                     |
| `daemonset_test.go`   | Tests for DaemonSet-specific behavior and strategies.                                                       |
| `mixed_test.go`       | Tests for mixed workload chains and cross-resource scenarios.                                               |
| `cycle_test.go`       | Tests for direct and indirect dependency cycle detection.                                                   |
| `namespace_test.go`   | Tests for multi-namespace watching and filtering.                                                           |
| `edgecases_test.go`   | Tests for special edge cases like invalid config, overlapping dependencies, long chains, and HTTP2 toggles. |

---

## Running all E2E Tests

```bash
make e2e
```

Runs the complete test suite against an existing Kubernetes cluster.

---

## Running a specific test group

Use the `FOCUS` parameter to target specific tests (regex match against the `Describe` block names):

Examples:

```bash
make e2e FOCUS=Deployment
make e2e FOCUS=StatefulSet
make e2e FOCUS=Cycle
make e2e FOCUS=Namespace
make e2e FOCUS=DaemonSet
make e2e FOCUS=Mixed
make e2e FOCUS=Edge
```

This allows faster iteration and debugging when working on specific workloads or features.

## Requirements

- An existing Kubernetes cluster accessible via `kubectl`
- `USE_EXISTING_CLUSTER=true` is automatically set in the Makefile for E2E tests
- Go installed
- Ginkgo integrated via `go test`

## Example: Running only Deployment tests

```bash
make e2e FOCUS=Deployment
```

## Notes

- All namespaces created during tests are cleaned up automatically.
- The operator is started and stopped dynamically within each test or test group.
- Logs are asserted via Ginkgo's `ContainsLogs` helpers.
