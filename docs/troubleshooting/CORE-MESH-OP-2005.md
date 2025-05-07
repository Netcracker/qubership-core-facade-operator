## CORE-MESH-OP-2007
### Message text
Unexpected error while tls processing.

### Scenario
The client added, changed or deleted a facade gateway custom resource.

### Reason
1) Kubernetes API either did not respond to the request or threw an error.
2) The certificate library responded with an error.

### Solution
First of all, you need to check the availability of the cloud.

If there are no visible problems with the cloud, you need to make sure that the service account facade-operator has been created.

In all other cases, please contact support.
