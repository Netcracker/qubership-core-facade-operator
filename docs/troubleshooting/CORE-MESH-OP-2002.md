## CORE-MESH-OP-2002
### Message text
Kubernetes API call failed while update image processing.

### Scenario
Public gateway image has been updated.

### Reason
Kubernetes API either did not respond to the request or threw an error.

### Solution
First of all, you need to check the availability of the cloud.

If there are no visible problems with the cloud, you need to make sure that the service account facade-operator has been created.

In all other cases, please contact support.
