# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Kubernetes operator (using controller-runtime) that manages facade gateways for the Qubership platform. It watches `FacadeService` and `Gateway` custom resources and creates/manages Kubernetes resources (Deployments, Services, HTTPRoutes, ConfigMaps, HPAs, PodMonitors) based on them.

All application code lives in the `facade-operator-service/` subdirectory.

## Build & Test Commands

All commands run from `facade-operator-service/`:

```bash
# Build
go build ./...

# Run all tests
go test ./...

# Run tests in a specific package
go test ./pkg/utils/... -v

# Run a single test by name
go test ./controllers/... -v -run TestReconcile_shouldApplyMeshRouterFailed_whenUnknownError

# Run with coverage
go test ./... -cover
```

There is no Makefile. CI runs via `.github/workflows/go-build.yml` using a reusable workflow.

## Architecture

### Reconciliation Flow

```
FacadeService CR created/updated
        ↓
FacadeServiceReconciler.Reconcile()
        ↓
FacadeCommonReconciler (shared logic for both FacadeService and Gateway)
        ├─→ ControlPlaneClient  — registers gateway routes with external Control Plane
        ├─→ DeploymentClient   — creates/updates the gateway Deployment
        ├─→ ServiceClient      — creates/updates the K8s Service
        ├─→ HTTPRouteClient    — creates/updates Gateway API HTTPRoutes (replaced Ingress)
        ├─→ ConfigMapClient    — writes gateway configuration
        ├─→ PodMonitorClient   — sets up Prometheus scraping
        ├─→ HPAClient          — configures autoscaling
        └─→ StatusUpdater      — tracks readiness in CR status
```

### Key Packages

| Path | Role |
|------|------|
| `lib/runner.go` | Initializes the controller-runtime manager and registers all reconcilers |
| `controllers/` | Reconciliation loops for FacadeService, Gateway, and ConfigMap CRs |
| `controllers/facade_common_controller.go` | Shared reconciliation logic used by both controllers |
| `pkg/services/` | Thin wrappers around the K8s client for each resource type |
| `pkg/templates/` | Builders that generate K8s resource manifests from CR specs |
| `pkg/restclient/` | HTTP client for the external Control Plane API |
| `pkg/errors/` | Custom error types and codes used across reconciliation |
| `pkg/predicates/` | Event filters to avoid reconciling on status-only updates |
| `pkg/indexes/` | K8s cache field indexes for efficient list queries |
| `api/facade/v1alpha/` | `FacadeService` CRD type definitions |
| `api/facade/v1/` | `Gateway` CRD type definitions |

### CRDs Managed

- `FacadeService` (`netcracker.com/v1alpha`) — primary resource
- `Gateway` (`core.netcracker.com/v1`) — alternative gateway type
- Also integrates with `Gateway` and `HTTPRoute` from `gateway.networking.k8s.io/v1`

### Testing Patterns

Tests use `testify/assert` for assertions and `go.uber.org/mock/gomock` for mocking K8s clients. Mock interfaces are generated — regenerate them with `go generate ./...` if interfaces change. Controller tests (in `controllers/*_test.go`) are integration-style with full mock setup; utility tests (in `pkg/utils/`) are simple unit tests.

### Configuration

The operator is configured via environment variables (see `pkg/utils/` for parsing helpers) and an `application.yaml` file loaded via `configloader`. Key variables: `CLOUD_NAMESPACE`, `MAX_CONCURRENT_RECONCILES`, `X509_AUTHENTICATION_ENABLED`, `COMPOSITE_PLATFORM`, `FACADE_GATEWAY_REPLICAS`, `FACADE_GATEWAY_MEMORY_LIMIT`.

Health/ready probes: port `8081`. Metrics: port `8082`. Webhook server: port `9443`.
