# Helm Chart Values for Cascader

This document provides an overview of the configurable values for the Cascader Helm chart. You can customize these values in your `values.yaml` file when deploying the chart.

## Adding this Repository to Helm

Add the repository to Helm:

```bash
helm repo add tkb https://charts.tkb.ch
```

---

## General Configuration

| Key                | Description                         | Default Value                  |
| ------------------ | ----------------------------------- | ------------------------------ |
| `image.repository` | The container image repository.     | `ghcr.io/thurgauerkb/cascader` |
| `image.tag`        | Overrides the default image tag.    | Chart `appVersion`             |
| `image.pullPolicy` | The image pull policy.              | `IfNotPresent`                 |
| `imagePullSecrets` | Secrets for pulling private images. | `[]`                           |

---

## Pod Configuration

| Key              | Description                            | Default Value |
| ---------------- | -------------------------------------- | ------------- |
| `replicas`       | Number of replicas for the deployment. | `1`           |
| `sidecars`       | Additional containers for the pod.     | `[]`          |
| `podAnnotations` | Annotations for the pod.               | `{}`          |
| `podLabels`      | Labels for the pod.                    | `{}`          |
| `nodeSelector`   | Node selector for pod placement.       | `{}`          |
| `tolerations`    | Tolerations for pod scheduling.        | `[]`          |
| `affinity`       | Affinity rules for pod placement.      | `{}`          |

---

## Probes

| Key                      | Description                            | Default Value       |
| ------------------------ | -------------------------------------- | ------------------- |
| `startupProbe.enabled`   | Enable startup probe.                  | `true`              |
| `startupProbe.spec`      | Configuration for the startup probe.   | See default values. |
| `livenessProbe.enabled`  | Enable liveness probe.                 | `true`              |
| `livenessProbe.spec`     | Configuration for the liveness probe.  | See default values. |
| `readinessProbe.enabled` | Enable readiness probe.                | `true`              |
| `readinessProbe.spec`    | Configuration for the readiness probe. | See default values. |

---

## Security

| Key                  | Description                          | Default Value |
| -------------------- | ------------------------------------ | ------------- |
| `podSecurityContext` | Security context for the pod.        | `{}`          |
| `securityContext`    | Security context for the containers. | `{}`          |

---

## Resource Configuration

| Key                         | Description                       | Default Value |
| --------------------------- | --------------------------------- | ------------- |
| `resources.limits.cpu`      | CPU limit for the container.      | `100m`        |
| `resources.limits.memory`   | Memory limit for the container.   | `200Mi`       |
| `resources.requests.cpu`    | CPU request for the container.    | `100m`        |
| `resources.requests.memory` | Memory request for the container. | `200Mi`       |

---

## Requeue Interval

| Key                   | Description                               | Default Value |
| --------------------- | ----------------------------------------- | ------------- |
| `requeueAfterDefault` | Default interval between reconciliations. | `5s`          |

---

## Annotations

| Key                           | Description                                  | Default Value                   |
| ----------------------------- | -------------------------------------------- | ------------------------------- |
| `annotationKeys.deployment`   | Annotation key for deployments.              | `cascader.tkb.ch/deployment`    |
| `annotationKeys.statefulset`  | Annotation key for statefulsets.             | `cascader.tkb.ch/statefulset`   |
| `annotationKeys.daemonset`    | Annotation key for daemonsets.               | `cascader.tkb.ch/daemonset`     |
| `annotationKeys.requeueAfter` | Annotation key for custom requeue intervals. | `cascader.tkb.ch/requeue-after` |

---

## Metrics Configuration

| Key                                       | Description                             | Default Value       |
| ----------------------------------------- | --------------------------------------- | ------------------- |
| `metrics.enabled`                         | Enable metrics collection.              | `true`              |
| `metrics.service.type`                    | Metrics service type.                   | `ClusterIP`         |
| `metrics.service.ports`                   | Ports for the metrics service.          | See default values. |
| `metrics.reader.enabled`                  | Enable metrics-reader role and binding. | `true`              |
| `metrics.prometheusRule.enabled`          | Enable Prometheus rules for alerts.     | `true`              |
| `metrics.prometheusRule.namespace`        | Namespace for Prometheus rules.         | `monitoring`        |
| `metrics.prometheusRule.severity`         | Severity of alerts.                     | `critical`          |
| `metrics.prometheusRule.additionalLabels` | Additional labels for Prometheus rules. | `{}`                |

---

## RBAC and Service Account

| Key                          | Description                                | Default Value |
| ---------------------------- | ------------------------------------------ | ------------- |
| `clusterRole.create`         | Create a ClusterRole and binding.          | `true`        |
| `clusterRole.name`           | Custom name for the ClusterRole.           | `""`          |
| `clusterRole.extraRules`     | Additional RBAC rules for the ClusterRole. | `[]`          |
| `serviceAccount.create`      | Create a ServiceAccount.                   | `true`        |
| `serviceAccount.annotations` | Annotations for the ServiceAccount.        | `{}`          |
| `serviceAccount.name`        | Custom name for the ServiceAccount.        | `""`          |

---

## Leader Election

| Key                      | Description             | Default Value |
| ------------------------ | ----------------------- | ------------- |
| `leaderElection.enabled` | Enable leader election. | `true`        |

---

## Logging Configuration

| Key              | Description     | Default Value |
| ---------------- | --------------- | ------------- |
| `logging.level`  | Logging level.  | `info`        |
| `logging.format` | Logging format. | `json`        |

---

## Environment Variables

| Key   | Description                        | Default Value                        |
| ----- | ---------------------------------- | ------------------------------------ |
| `env` | Environment variables for the pod. | `[{name: TZ, value: Europe/Zurich}]` |

---

## Arguments

| Key         | Description                  | Default Value |
| ----------- | ---------------------------- | ------------- |
| `extraArgs` | Extra arguments for the pod. | `[]`          |

---

## Extra Configuration

| Key            | Description                         | Default Value |
| -------------- | ----------------------------------- | ------------- |
| `extraObjects` | Extra Kubernetes objects to deploy. | `[]`          |
