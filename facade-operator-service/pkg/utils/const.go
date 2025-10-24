package utils

import "os"

var DefaultFacadeGatewayReplicas = os.Getenv("FACADE_GATEWAY_REPLICAS")
var DefaultFacadeGatewayMemoryLimit = os.Getenv("FACADE_GATEWAY_MEMORY_LIMIT")
var DefaultFacadeGatewayMemoryRequest = os.Getenv("FACADE_GATEWAY_MEMORY_REQUEST")
var DefaultFacadeGatewayCpuLimit = os.Getenv("FACADE_GATEWAY_CPU_LIMIT")
var DefaultFacadeGatewayCpuRequest = os.Getenv("FACADE_GATEWAY_CPU_REQUEST")
var DefaultFacadeGatewayConcurrency = os.Getenv("FACADE_GATEWAY_CONCURRENCY")
var FacadeGatewayTerminationGracePeriodS = os.Getenv("FACADE_GATEWAY_TERMINATION_GRACE_PERIOD_S")
var MonitoringEnabled = os.Getenv("MONITORING_ENABLED")
var ArtifactDescriptorVersion = os.Getenv("ARTIFACT_DESCRIPTOR_VERSION")
var TracingEnabled = os.Getenv("TRACING_ENABLED")
var TracingHost = os.Getenv("TRACING_HOST")
var IpStack = os.Getenv("IP_STACK")
var IpBind = os.Getenv("IP_BIND")
var LabelsDelimiter = "-"
var CloudTopologiesJsonBase64 = os.Getenv("CLOUD_TOPOLOGIES_JSON_BASE64")
var CloudTopologyKey = os.Getenv("CLOUD_TOPOLOGY_KEY")
var XDSClusterHost = os.Getenv("XDS_CLUSTER_HOST")
var XDSClusterPort = os.Getenv("XDS_CLUSTER_PORT")
var TlsSecretPath = os.Getenv("TLS_SECRET")
var TlsPasswordSecretName = os.Getenv("TLS_PASSWORD_SECRET_NAME")
var TlsPasswordKey = os.Getenv("TLS_PASSWORD_KEY")

// gateway hpa
var HpaDefaultMinReplicasEnvName = "GATEWAY_HPA_MIN_REPLICAS"
var HpaDefaultMaxReplicasEnvName = "GATEWAY_HPA_MAX_REPLICAS"
var HpaDefaultAverageUtilizationEnvName = "GATEWAY_HPA_AVERAGE_UTILIZATION_TARGET_PERCENT"
var HpaDefaultSelectPolicyEnvName = "GATEWAY_HPA_SELECT_POLICY"

// gateway hpa - scale up
var HpaDefaultScaleUpStabilizationWindowSecondsEnvName = "GATEWAY_HPA_SCALE_UP_STABILIZATION_WINDOW_SECONDS"
var HpaDefaultScaleUpPercentValueEnvName = "GATEWAY_HPA_SCALE_UP_PERCENT_VALUE"
var HpaDefaultScaleUpPercentPeriodSecondsEnvName = "GATEWAY_HPA_SCALE_UP_PERCENT_PERIOD_SECONDS"
var HpaDefaultScaleUpPodsValueEnvName = "GATEWAY_HPA_SCALE_UP_PODS_VALUE"
var HpaDefaultScaleUpPodsPeriodSecondsEnvName = "GATEWAY_HPA_SCALE_UP_PODS_PERIOD_SECONDS"

// gateway hpa - scale down
var HpaDefaultScaleDownStabilizationWindowSecondsEnvName = "GATEWAY_HPA_SCALE_DOWN_STABILIZATION_WINDOW_SECONDS"
var HpaDefaultScaleDownPercentValueEnvName = "GATEWAY_HPA_SCALE_DOWN_PERCENT_VALUE"
var HpaDefaultScaleDownPercentPeriodSecondsEnvName = "GATEWAY_HPA_SCALE_DOWN_PERCENT_PERIOD_SECONDS"
var HpaDefaultScaleDownPodsValueEnvName = "GATEWAY_HPA_SCALE_DOWN_PODS_VALUE"
var HpaDefaultScaleDownPodsPeriodSecondsEnvName = "GATEWAY_HPA_SCALE_DOWN_PODS_PERIOD_SECONDS"

var FacadeGateway = "facadeGateway"
var MasterCR = "masterCR"
var HostedByLabel = "mesh.netcracker.com/hosted.by"
var MeshRouter = "mesh-router"
var SpecGatewayField = "spec.gateway"
var SpecDnsNamesField = "spec.dnsNames"
var TlsCertificateVersion = "tlsCertificateVersion"
var ReadOnlyContainerEnabled = "READONLY_CONTAINER_FILE_SYSTEM_ENABLED"
var GatewaySuffix = "-gateway"
var LastAppliedCRAnnotation = "netcracker.cloud/last-applied-cr"

const Unknown = "unknown"

const MinimumEgressGatewayMemoryLimitInt = 64
const MinimumEgressGatewayMemoryRequestInt = 64
