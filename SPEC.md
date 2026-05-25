# Kubernetes Resource Attribution Operator — Implementation Spec

## Overview

An operator that attributes Kubernetes infrastructure usage (CPU, memory) to logical owners (teams, applications, namespaces) using label selectors and the Kubernetes metrics server.

---

## MVP Scope

- CPU and memory aggregation per label selector
- Kubernetes-native reporting via CRD status
- Periodic reconciliation (every 30s)
- Namespace-scoped or cluster-wide attribution

---

## Custom Resources

### ResourceAttribution

Defines a logical ownership boundary and holds the attribution result.

**Spec fields:**

| Field | Type | Required | Description |
|---|---|---|---|
| `selector` | `LabelSelector` | yes | Matches pods to attribute |
| `namespace` | `string` | no | Scope to a namespace. Empty = all |

**Status fields:**

| Field | Type | Description |
|---|---|---|
| `cpuUsage` | `string` | Total CPU cores used (e.g. `"2.3"`) |
| `memoryUsage` | `string` | Total memory used (e.g. `"512Mi"`) |
| `podCount` | `int` | Number of matched pods |
| `lastUpdated` | `Time` | When metrics were last collected |
| `conditions` | `[]Condition` | Standard K8s conditions |

---

## Implementation Steps

### Step 1 — API Types
File: `api/v1alpha1/resourceattribution_types.go`

- Remove placeholder `Foo` field
- Add `Spec` fields: `selector`, `namespace`
- Add `Status` fields: `cpuUsage`, `memoryUsage`, `podCount`, `lastUpdated`, `conditions`

After editing:
```
make manifests generate
```

---

### Step 2 — Add Metrics Client
```
go get k8s.io/metrics@v0.35.0
```

Required to read `PodMetrics` from the Kubernetes metrics server.

---

### Step 3 — Controller Reconcile Logic
File: `internal/controller/resourceattribution_controller.go`

Reconcile steps:
1. Fetch the `ResourceAttribution` CR
2. Convert `spec.selector` to a label selector
3. List all matching pods (scoped to `spec.namespace` if set)
4. For each pod, fetch `PodMetrics` from the metrics server
5. Sum CPU and memory across all matched pod containers
6. Write totals to `status`
7. Requeue after 30 seconds

RBAC required:
- `pods` — get, list, watch
- `metrics.k8s.io/pods` — get, list
- `resourceattributions/status` — get, update, patch

---

### Step 4 — Register Metrics Scheme
File: `cmd/main.go`

Register `metricsv1beta1` in the scheme inside `init()` so the client can decode `PodMetrics` objects from the API server.

---

### Step 5 — Sample CR
File: `config/samples/attribution_v1alpha1_resourceattribution.yaml`

Example that selects all pods with `team: payments` in the `payments` namespace.

---

## Data Flow

```
User applies ResourceAttribution CR
           │
           ▼
Controller reconciles (triggered + every 30s)
           │
           ├── List pods matching spec.selector
           │
           ├── Fetch PodMetrics for each pod
           │           (from metrics-server via metrics.k8s.io API)
           │
           ├── Aggregate CPU + Memory
           │
           └── Update status.cpuUsage / memoryUsage / podCount
```

---

## RBAC Summary

```
Group                     Resource                  Verbs
attribution.krato.io      resourceattributions      get list watch create update patch delete
attribution.krato.io      resourceattributions/status  get update patch
attribution.krato.io      resourceattributions/finalizers  update
""  (core)                pods                      get list watch
metrics.k8s.io            pods                      get list
```

---

## Testing Plan

| Test | Type | Description |
|---|---|---|
| Types compile | Unit | `make generate` succeeds |
| Reconcile with no pods | Unit | Status shows 0 pods, empty usage |
| Reconcile with matched pods | Integration | Status reflects summed metrics |
| Selector filters correctly | Integration | Only matching pods counted |
| Metrics server unavailable | Unit | Controller retries gracefully |

---

## Future Extensions (Post-MVP)

| Feature | Description |
|---|---|
| Storage attribution | Aggregate PVC usage per owner |
| Network attribution | Ingress/egress bytes per workload |
| Cost estimation | Multiply usage by node pricing |
| Budget policies | Alert when usage exceeds threshold |
| `CostTarget` + `CostReport` split | Separate ownership definition from report |
| Prometheus metrics export | Expose attribution as Prometheus gauges |
| Multi-cluster aggregation | Federate reports across clusters |

---

## File Checklist

- [ ] `api/v1alpha1/resourceattribution_types.go` — Spec/Status fields defined
- [ ] `go.mod` — `k8s.io/metrics` added
- [ ] `internal/controller/resourceattribution_controller.go` — Reconcile implemented
- [ ] `cmd/main.go` — metrics scheme registered
- [ ] `config/samples/attribution_v1alpha1_resourceattribution.yaml` — sample CR created
- [ ] `make manifests generate` — CRD regenerated
- [ ] `make test` — unit tests pass
