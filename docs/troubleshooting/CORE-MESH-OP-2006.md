## CORE-MESH-OP-2006
### Message text
Communication with control-plane failed.

### Scenario
FacadeService custom resource was applied and operator tries to register declared gateway in control-plane microservice. 

### Reason
Control-plane responds with error on gateway registration request sent by facade-operator. 

### Solution
First of all, check that control-plane is up and running. 

If control-plane is online, see details of this error in facade-operator logs - it should contain response message from control-plane. 

If facade-operator logs are not helpful, try to find corresponding error in control-plane logs. 

If it does not help, please contact support.
