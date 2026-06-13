# Core Egress Gateway

## Overview

The `core-egress-gateway` is a special CR name that creates an egress gateway deployment with specific behaviors that differ from standard facade gateways. `core-egress-gateway` comes with Cloud-Core and should not be created by other applications. This feature is necessary to resolve conflict between `egress-gateway` CRs coming from different applications causing issues for Istio migration. 

## Configuration

A `core-egress-gateway` CR must be configured with:

```yaml
apiVersion: netcracker.com/v1alpha
kind: FacadeService
metadata:
  name: core-egress-gateway
spec:
  gateway: egress-gateway-gateway
  gatewayType: egress
  # ... other configuration
```

### Key Fields

- **`metadata.name`**: Must be exactly `core-egress-gateway`
- **`spec.gateway`**: Must be set to `egress-gateway-gateway`
- **`spec.gatewayType`**: Should be explicitly set to `egress`

## Behavior

### Resources Created

The operator creates the following resources for `core-egress-gateway`:

1. **Deployment**: Named `egress-gateway-gateway`
   - Container environment variable `SERVICE_NAME_VARIABLE` is set to `egress-gateway`
   - Minimum memory requirements: 64Mi limit and request (same as standard `egress-gateway`)
2. **ConfigMap**: Named `egress-gateway-gateway.monitoring-config`
3. **HPA**: Named `egress-gateway-gateway`
4. **PodMonitor**: Named `egress-gateway-gateway-pod-monitor`
4. **Service**: Named `core-egress-gateway` - not really needed

### Cleanup

All created resources have `OwnerReferences` set to the `core-egress-gateway` CR. When the CR is deleted, Kubernetes garbage collection automatically removes all associated resources.

## Coexistence with `egress-gateway`

Both `egress-gateway` and `core-egress-gateway` CRs can exist in the same namespace. When `core-egress-gateway` CR exists, it **always wins** and controls the egress gateway resources regardless of any `masterCR` settings:

- Any reconciliation event for `egress-gateway` CR is a complete no-op — no resources are created, updated, or deleted.
- Deleting `egress-gateway` CR while `core-egress-gateway` exists has no effect on the egress gateway resources (deployment, service, etc.).
- `core-egress-gateway` is the authoritative owner of all egress gateway resources and takes precedence over any other `egress-gateway` CR.
