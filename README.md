# Krato — Kubernetes Resource Attribution Operator

[![Go Version](https://img.shields.io/badge/go-1.26-blue)](https://golang.org)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

Krato is a Kubernetes-native operator that attributes infrastructure resource consumption (CPU, memory) to logical application owners — teams, services, namespaces, or GitOps applications.

It connects cluster telemetry with ownership models, making resource usage transparent and actionable directly inside the Kubernetes control plane.

---

## Overview

Kubernetes exposes raw metrics per pod or node, but organizations struggle to answer higher-level questions:

- Which team is consuming the most CPU?
- Which deployment increased cluster usage after a rollout?
- Which services are over-provisioned?

Krato answers these by introducing a `ResourceAttribution` Custom Resource that maps workloads to logical owners and periodically aggregates their resource usage.

---

## Architecture

```
User applies ResourceAttribution CR
           │
           ▼
Controller reconciles (triggered + periodic)
           │
           ├── List pods matching spec.selector
           │
           ├── Group pods by workload owner (Deployment / StatefulSet / Job)
           │
           ├── Collect metrics (requests fallback / metrics-server / prometheus)
           │
           ├── Apply pricing (noop / AWS EC2 — future)
           │
           └── Write grouped results to status
```

### Internal Packages

```
internal/
├── controller/         # Thin reconcile loop — orchestrates everything
├── metrics/            # Pluggable metrics collection backends
│   ├── collector.go    # MetricsCollector interface
│   ├── requests.go     # Fallback: pod spec resource requests
│   ├── metricsserver.go# Kubernetes metrics-server backend
│   └── prometheus.go   # Prometheus backend (future)
├── attribution/        # Engine: groups pods by workload, applies pricing
│   └── engine.go
└── pricing/            # Pluggable pricing backends
    ├── provider.go     # PricingProvider interface
    ├── noop.go         # Returns $0 (default)
    └── aws.go          # AWS EC2 pricing API (future)
```

---

## Custom Resource

### ResourceAttribution

Defines a logical ownership boundary and holds attribution results.

```yaml
apiVersion: attribution.krato.io/v1alpha1
kind: ResourceAttribution
metadata:
  name: payments-team
spec:
  selector:
    matchLabels:
      team: payments
  namespace: payments          # optional — omit for all namespaces
  updateInterval: "30s"        # optional — defaults to 30s
```

After reconciliation, the status is populated:

```yaml
status:
  podCount: 7
  totalCpu: "2300m"
  totalMemory: "1536Mi"
  workloads:
    - name: payments-api
      kind: Deployment
      namespace: payments
      podCount: 4
      cpuMilli: 1600
      memBytes: 1073741824
    - name: payments-worker
      kind: StatefulSet
      namespace: payments
      podCount: 3
      cpuMilli: 700
      memBytes: 536870912
  lastUpdated: "2026-06-06T10:00:00Z"
```

---

## Spec Fields

| Field | Type | Required | Default | Description |
|---|---|---|---|---|
| `selector` | `LabelSelector` | yes | — | Matches pods to attribute |
| `namespace` | `string` | no | all | Scope to a specific namespace |
| `updateInterval` | `string` | no | `"30s"` | How often to refresh metrics |

---

## Configuration Flags

| Flag | Default | Description |
|---|---|---|
| `--metrics-bind-address` | `:8080` | Prometheus metrics endpoint |
| `--health-probe-bind-address` | `:8081` | Liveness/readiness probe endpoint |
| `--leader-elect` | `false` | Enable leader election for HA |
| `--max-concurrent-reconciles` | `1` | Concurrent reconcilers per controller |
| `--metrics-backend` | `none` | Metrics source: `none`, `metrics-server`, `prometheus` |
| `--metrics-backend-address` | `""` | Endpoint for prometheus or custom backend |

---

## Metrics Backends

| Backend | Flag value | Accuracy | Requirement |
|---|---|---|---|
| Pod spec requests | `none` | Requested capacity | Always available |
| Kubernetes metrics-server | `metrics-server` | Actual usage | metrics-server installed |
| Prometheus | `prometheus` | Actual usage | `--metrics-backend-address` required |

---

## Getting Started

### Prerequisites

- Go 1.26+
- kubectl
- kubebuilder v4
- A running Kubernetes cluster (local: kind or minikube)

### Local Development

```bash
# Clone the repo
git clone https://github.com/sanoguzhan/krato
cd krato

# Install CRDs into the cluster
make install

# Run the operator locally (uses current kubeconfig)
make run

# Apply a sample CR
kubectl apply -f config/samples/attribution_v1alpha1_resourceattribution.yaml

# Watch status updates
kubectl get resourceattribution payments-team -w
```

### Build and Deploy

```bash
export IMG=ghcr.io/sanoguzhan/krato:latest

make docker-build docker-push IMG=$IMG
make deploy IMG=$IMG
```

---

## Development

```bash
# Regenerate CRDs and RBAC from types and markers
make manifests

# Regenerate DeepCopy methods
make generate

# Run linter
make lint

# Run unit tests
make test

# Run end-to-end tests (requires kind cluster)
make test-e2e
```

---

## Roadmap

| Feature | Status |
|---|---|
| CPU + memory aggregation via pod requests | MVP |
| Label-based ownership | MVP |
| Workload grouping (Deployment / StatefulSet / Job) | In progress |
| Kubernetes metrics-server backend | In progress |
| Prometheus backend | Planned |
| AWS EC2 cost estimation | Planned |
| Budget policies per attribution | Planned |
| Storage (PVC) attribution | Planned |
| Network ingress/egress attribution | Planned |
| Multi-cluster aggregation | Planned |

---

## License

Apache 2.0 — see [LICENSE](LICENSE).
