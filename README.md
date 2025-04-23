# ![Cascader-Logo](.github/assets/cascader_100x100.png) Cascader

[![Go Report Card](https://goreportcard.com/badge/github.com/thurgauerkb/cascader?style=flat-square)](https://goreportcard.com/report/github.com/thurgauerkb/cascader)
[![Go Doc](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/thurgauerkb/cascader)
[![Release](https://img.shields.io/github/release/thurgauerkb/cascader.svg?style=flat-square)](https://github.com/thurgauerkb/cascader/releases/latest)
[![GitHub tag](https://img.shields.io/github/tag/thurgauerkb/cascader.svg?style=flat-square)](https://github.com/thurgauerkb/cascader/releases/latest)
[![license](https://img.shields.io/github/license/thurgauerkb/cascader.svg?style=flat-square)](LICENSE)

`Cascader` is a Kubernetes operator designed to simplify the orchestration of dependent workloads within your cluster. By leveraging custom annotations, `Cascader` monitors specified Kubernetes resources and automatically triggers reloads of dependent workloads when the primary workload becomes stable.

## Problem

Managing dependencies between Kubernetes workloads can be challenging, particularly when updates to one workload require coordinated restarts of dependent workloads.

## Solution

`Cascader` automates this process by:

- Monitoring resources like `Deployments`, `StatefulSets`, and `DaemonSets` for changes.
- Using custom annotations to define dependencies.
- Triggering restarts of dependent workloads when the monitored resource becomes stable.
- Preventing cyclic dependencies through built-in cycle detection.

This reduces manual intervention, ensures consistency, and keeps your cluster in a reliable state.

## Features

- **Dependency Management**: Define workload dependencies via annotations.
- **Dependent Restarts**: Triggers restarts of dependent kubernetes workloads.
- **Customizable Intervals**: Configure requeue intervals unitl workload is stable, per workload or globally.
- **Cycle Detection**: Avoid cyclic dependencies in your workload graph.
- **Scoped Namespace Watching**: Limit `Cascader` to specific namespaces with the `--watch-namespace` flag.
- **Prometheus Metrics**: Gain insights into dependency cycles, workloads managed, and restarts performed.

> Note: `Cascader` follows a best-effort restart approach. It updates the `kubectl.kubernetes.io/restartedAt` annotation to trigger dependent workload reloads but does not verify if the restart was successful. For reliability checks, use an external monitoring tool like Prometheus.

## Installation and Usage

### Installation with Vanilla Manifests

Apply the default manifests from the repository to deploy `Cascader`:

```bash
kubectl apply -f https://raw.githubusercontent.com/thurgauerkb/cascader/main/deploy/kubernetes/cascader.yaml
```

This will deploy `Cascader` in the `cascader-system` namespace.

### Namespaced Mode

By default, `Cascader` watches all namespaces. To restrict it to specific namespaces, pass the `--watch-namespace` flag. This flag can be repeated or comma-separated to specify multiple namespaces. When set, `Cascader` will only monitor workloads and trigger dependent restarts within those namespaces.

If running in namespaced mode, ensure the associated `Role` and `RoleBinding` are configured accordingly. You can use `deploy/manifests/role.template` and `deploy/manifests/rolebinding.template` as starting points for custom RBAC definitions.

### Kustomize

You can create your own `kustomization.yaml` file by referencing our manifests as a base and adding patches to customize the configuration:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: custom-namespace
resources:
  - https://github.com/thurgauerkb/cascader/deploy/kubernetes/
```

### Helm

Coming soon.

## How to Use Cascader

Define dependencies using annotations on your Kubernetes resources. `Cascader` will monitor these resources and trigger dependent restarts when the primary resource becomes stable.

### Example: Single Dependency

You have a Deployment called `api-service` in the namespace `staging` that should restart after `database` in the namespace `production`:

- `production/database` → `staging/api-service`.

Add the annotation `cascader.tkb.ch/deployment: staging/api-service` to the `database` deployment:

```bash
kubectl annotate deployment database -n production cascader.tkb.ch/deployment='staging/api-service'
```

Restart the `database` deployment:

```bash
kubectl rollout restart deployment database -n production
```

When `Cascader` detects the `database` deployment restarting, it waits until the resource becomes stable. Once stable, it triggers a restart of the `api-service` deployment in the `staging` namespace.

### Example: Multiple Dependencies

You have four workloads:

1. A Deployment named `backend-service` in the `backend` namespace (this is the root of the chain).
2. A StatefulSet named `frontend-cache` in the `frontend` namespace.
3. A Deployment named `cache-service` in the `backend` namespace.
4. A Deployment named `web-app` in the `frontend` namespace.

There is a _chain_ of dependencies, like this:

- `backend/backend-service` → `frontend/frontend-cache`
- `backend/backend-service` → `backend/cache-service` → `frontend/web-app`

In other words:

- As soon as `backend-service` restarts and becomes stable, `frontend-cache` **and** `cache-service` get a restart (in parallel).
- Once `cache-service` finishes rolling out and is stable, `web-app` gets restarted.

You can achieve this by adding annotations that link each dependency in turn.

#### Step 1: Annotate the `backend-service` Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-service
  namespace: backend
  annotations:
    cascader.tkb.ch/statefulset: "frontend-cache"
    cascader.tkb.ch/deployment: "cache-service"
spec:
  replicas: 2
  # ...
```

#### Step 2: Annotate the `cache-service` Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cache-service
  namespace: backend
  annotations:
    cascader.tkb.ch/deployment: "frontend/web-app"
spec:
  replicas: 2
  # ...
```

With these annotations:

1. **Restart `backend-service`** (the root of your chain):
   ```bash
   kubectl rollout restart deployment backend-service -n backend
   ```
2. `cascader` sees that `backend-service` depends on the `frontend-cache` StatefulSet and the `cache-service` Deployment.
3. It waits for `backend-service` to stabilize, then restarts **both** `frontend-cache` and `cache-service`.
4. Once `cache-service` becomes stable, `cascader` sees that it depends on `frontend/web-app`, so it restarts `web-app`.
5. `cascader` will only restart the next workload once the previous one is stable, ensuring a safe rollout sequence.

This chaining of dependencies allows you to orchestrate multi-step rollouts automatically, with each step waiting for the previous workload to become stable.

## Key Concepts

### Supported Workloads

- `Cascader` supports the following Kubernetes workloads:
  - Deployments
  - StatefulSets
  - DaemonSets

### Best-Effort Restarts

- `Cascader` triggers restarts by updating the `kubectl.kubernetes.io/restartedAt` annotation in `.Spec.Template.Annotations`.
- It does not confirm whether dependent workloads successfully restarted.
- Use external monitoring tools for verification and reliability checks.

### Restart Detection

`Cascader` tracks restart events of source workloads and coordinates dependent restarts accordingly. To do this, it monitors for meaningful changes to the workload that indicate a restart has occurred or is underway.

When such a change is detected, `Cascader` sets the annotation:

```yaml
cascader.tkb.ch/last-observed-restart: "<timestamp>"
```

in the `.metadata.annotations` of the source workload. This annotation acts as a **signal** that a restart has been observed and dependent workloads may need to be reloaded. Once the source workload is verified to be **stable**, the annotation is automatically removed.

#### Conditions that trigger restart detection include:

- **A change to the Pod template specification (`.spec.template`)**, including:
  - Image changes
  - Command or environment updates
  - **Annotations inside `.spec.template.metadata.annotations`**, such as those set by `kubectl rollout restart`
- **A change to the restart-specific annotation**, typically:
  - `kubectl.kubernetes.io/restartedAt`
- **Scaling events**, including:
  - Scaling from zero (workload was previously inactive)
  - Scaling to zero (resetting all Pods)
- **For single-replica workloads**:
  - If the sole Pod is deleted and no longer ready or available
- **For DaemonSets**:
  - If not all desired Pods are updated or available (indicating an update is rolling out)

#### Notes

- `Cascader` does **not** respond to arbitrary Pod restarts (e.g., if 1 Pod out of 5 is restarted due to node eviction or OOM).
- This detection is **best-effort**: it assumes well-behaved workloads and does not verify Pod-level success.
- External monitoring tools should be used to ensure full reliability and correctness of dependent restarts.

### Cycle Detection

Dependency cycles are automatically detected, and processing halts if a cycle is found. For example:

- **Direct Cycle:** A resource depends on itself (`A → A`).
- **Indirect Cycle:** A resource indirectly depends on itself through others (`A → B → C → A`).

### Custom Annotations

If you do not want to use the default annotations, you can customize them by passing the `--deployment-annotation`, `--statefulset-annotation`, `--daemonset-annotation`, `--last-observed-restart-annotation`, and `--requeue-after-annotation` flags to `cascader`.

### Start Parameters

| Parameter                                   | Description                                                                     | Default                                 |
| :------------------------------------------ | :------------------------------------------------------------------------------ | :-------------------------------------- |
| `--deployment-annotation` string            | Annotation key for monitored Deployments                                        | `cascader.tkb.ch/deployment`            |
| `--statefulset-annotation` string           | Annotation key for monitored StatefulSets                                       | `cascader.tkb.ch/statefulset`           |
| `--daemonset-annotation` string             | Annotation key for monitored DaemonSets                                         | `cascader.tkb.ch/daemonset`             |
| `--last-observed-restart-annotation` string | Annotation key for last observed restart                                        | `cascader.tkb.ch/last-observed-restart` |
| `--requeue-after-annotation` string         | Annotation key for requeue interval override                                    | `cascader.tkb.ch/requeue-after`         |
| `--requeue-after-default` duration          | Default requeue interval                                                        | `5s`                                    |
| `--watch-namespace` stringSlice             | Namespaces to watch (can be repeated or comma-separated). Watches all if unset. | ``                                      |
| `--metrics-enabled`                         | Enable or disable the metrics endpoint                                          | `true`                                  |
| `--metrics-bind-address` string             | Metrics server address (e.g., `:8080` for HTTP, `:8443` for HTTPS)              | `:8443`                                 |
| `--metrics-secure`                          | Serve metrics over HTTPS                                                        | `true`                                  |
| `--enable-http2`                            | Enable HTTP/2 for servers                                                       | `false`                                 |
| `--health-probe-bind-address` string        | Health and readiness probe address                                              | `:8081`                                 |
| `--leader-elect`                            | Enable leader election                                                          | `true`                                  |
| `--log-encoder` string                      | Log format (`json`, `console`)                                                  | `json`                                  |
| `--log-stacktrace-level` string             | Stacktrace log level (`info`, `error`, `panic`)                                 | `panic`                                 |
| `--log-devel`                               | Enable development mode logging                                                 | `false`                                 |
| `--version`                                 | Show version and exit                                                           |
| `-h`, `--help`                              | Show help and exit                                                              |

## Prometheus Metrics

Cascader exposes a set of Prometheus metrics to monitor dependency cycles, target relationships, and workload restarts.
**Metrics are only emitted for workloads that have been processed by Cascader, typically after a restart has been detected.**
This means metrics may not appear for all workloads immediately, but only for those that trigger reconciliation through annotated restart events.

### Available Metrics

1. **Dependency Cycles Detected**

   - **Metric:** `cascader_dependency_cycles_detected`
   - **Description:** Indicates whether a dependency cycle is currently detected for a specific workload (1 = cycle detected, 0 = no cycle).
   - **Labels:** `namespace`, `name`, `resource_kind`.

2. **Workloads Targets**

   - **Metric:** `cascader_workload_targets`
   - **Description:** Number of dependency targets extracted from a workload's annotations by Cascader.
   - **Labels:** `namespace`, `name`, `resource_kind`.

3. **Restarts Performed**
   - **Metric:** `cascader_restarts_performed_total`
   - **Description:** Total number of restarts performed by Cascader.
   - **Labels:** `namespace`, `name`, `resource_kind`.

## Contributing

We welcome contributions of all kinds! Please refer to our [CONTRIBUTING.md](.github/CONTRIBUTING.md) file for detailed guidelines on how to contribute, report issues, and improve Cascader.

To summarize:

- **Fork the Repository**: Create your own copy and work on a feature branch.
- **Set Up the Development Environment**: Install dependencies and verify your setup.
- **Run Tests**: Ensure all tests pass before submitting your changes.
- **Submit a Pull Request**: Clearly describe your changes and their purpose.

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for details.
