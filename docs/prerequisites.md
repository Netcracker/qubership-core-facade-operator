This document describes necessary prerequisites to deploy facade-operator microservice.

Actions below are actual for Kubernetes and Openshift 3.11+. If installation is performed on Openshift 1.5- please ignore entire following step. Openshift 1.5 doesn't support CRD and Facade Gateway feature will be disabled. 
### Dependencies
Note that Role and Role binding names from scripts above are entities with shared contract names, thus- these objects could be provided by other Qubership components (Data Bases and Message queues) and could be already installed to cloud. 
CRD has unique name, but it is installed once per cloud and it also could be already installed by other Cloud Core installation.  

1. There are two deployment methods: Cluster Admin grants permission to get/create CRD or not  
    - If Cluster Admin grants permission to get/create CRD.  
        Cluster Admin must grants permissions to get/create CRD for a deployment user
    
        **ClusterRole definition example:**
        ```
        apiVersion: rbac.authorization.k8s.io/v1beta1
        kind: ClusterRole
        metadata:
          name: qubership-crd
        rules:
          - apiGroups: ["apiextensions.k8s.io/v1beta1"]
            resources: ["customresourcedefinition"]
            verbs: ["get", "create", "patch"]      
        ```
        
        **ClusterRoleBinding definition example:**
        ```
        apiVersion: rbac.authorization.k8s.io/v1beta1
        kind: ClusterRoleBinding
        metadata:
          name: qubership-crd
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: ClusterRole
          name: qubership-crd
        subjects:
        - kind: ServiceAccount
          name: <deploy user name>
          namespace: <namespace>
         ```  
    - If Cluster Admin does not grants permission to get/create CRD.
        Cluster Admin must create CRD:  
        + If you have Openshift 3.11
        ```
        apiVersion: apiextensions.k8s.io/v1beta1
        kind: CustomResourceDefinition
        metadata:
          name: facadeservices.netcracker.com
        spec:
          group: netcracker.com
          preserveUnknownFields: true
          version: v1alpha
          names:
            plural: facadeservices
            singular: facadeservice
            kind: FacadeService
          scope: Namespaced
          validation:
            openAPIV3Schema:
              type: object
              properties:
                spec:
                  type: object
                  required:
                    - port
                  properties:
                    port:
                      type: integer
                    gateway:
                      type: string
         ```  
       + If you have Openshift 4.5+ or kubernetes 1.16+  
        ```
        apiVersion: apiextensions.k8s.io/v1
        kind: CustomResourceDefinition
        metadata:
          name: facadeservices.netcracker.com
        spec:
          group: netcracker.com
          versions:
            - name: v1alpha
              served: true
              storage: true
              schema:
                openAPIV3Schema:
                  type: object
                  properties:
                    spec:
                      x-kubernetes-preserve-unknown-fields: true
                      type: object
                      required:
                        - port
                      properties:
                        port:
                          type: integer
                        gateway:
                          type: string
          names:
            plural: facadeservices
            singular: facadeservice
            kind: FacadeService
          scope: Namespaced
        ```
2. Cluster admin allow reading/writing resources in the "netcracker.com" and "core.netcracker.com" API groups for a deploy user.  
   Following Role and RoleBinding can be used to grant permissions:
    
    **Role description example:**
    ```
    apiVersion: rbac.authorization.k8s.io/v1beta1
    kind: Role
    metadata:
      name: qubership-cr
    rules:
    - apiGroups: ["netcracker.com", "core.netcracker.com"]
      resources: ["*"]
      verbs: ["*"]
    ```
    
     **RoleBinding description example:**
      ```
      apiVersion: rbac.authorization.k8s.io/v1beta1
      kind: RoleBinding
      metadata:
        name: qubership-cr
        namespace: <namespace>
      subjects:
      - kind: ServiceAccount
        name: <deploy user name>
        namespace: <namespace>
      roleRef:
        apiGroup: rbac.authorization.k8s.io
        kind: Role
        name: qubership-cr
     ```
