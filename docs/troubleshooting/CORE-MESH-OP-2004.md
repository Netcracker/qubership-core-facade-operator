## CORE-MESH-OP-2004
### Message text
Can not get gateway image.

### Scenario
1) The client added, changed or deleted a facade gateway custom resource.
2) Gateway image url in `core-gateway-image` config map has been updated.

### Reason
`core-gateway-image` config map is missing or contains invalid gateway docker image URL.
Normally this config map is created and filled during cloud-core deploy. 

### Solution
Please, check `core-gateway-image` config map in facade-operator namespace. It should contain valid URL to gateway image. 
If it does not, you can deploy Cloud-Core in rolling mode so variable will be filled automatically. 
