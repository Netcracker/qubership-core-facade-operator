package templates

import (
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var labelsToPropagate = []string{
	"app.kubernetes.io/part-of",
	"app.kubernetes.io/version",
}

type RouterDeployment struct {
	ServiceName                string
	GatewayName                string
	NameSpace                  string
	CrLabels                   map[string]string
	InstanceLabel              string
	ArtifactDescriptionVersion string
	ImageName                  string
	Recourses                  corev1.ResourceRequirements
	TracingEnabled             string
	TracingHost                string
	IpStack                    string
	IpBind                     string
	MeshRouter                 bool
	Replicas                   int32
	CloudTopologies            []utils.CloudTopology
	CloudTopologyKey           string
	XdsClusterHost             string
	XdsClusterPort             string
	TLSSecretPath              string
	TLSPasswordSecretName      string
	TLSPasswordKey             string
	MasterCR                   string
	MasterCRName               string
	MasterCRVersion            string
	MasterCRKind               string
	MasterCRUID                types.UID
	GatewayPorts               []facade.GatewayPorts
	ReadOnlyContainerEnabled   bool
	GwTerminationGracePeriodS  int
	EnvoyConcurrency           int
	HostedBy                   string
	LastAppliedCR              string
}

func (d RouterDeployment) GetDeployment() *appsv1.Deployment {
	memLimit := d.Recourses.Limits[corev1.ResourceMemory]
	deployment := &appsv1.Deployment{
		ObjectMeta: d.getObjectMeta(),
		Spec: appsv1.DeploymentSpec{
			Replicas: &d.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": d.GatewayName,
				},
			},
			Template: d.getPodTemplateSpec(memLimit.String()),
		},
	}

	if d.MeshRouter {
		deployment.ObjectMeta.Labels[utils.MeshRouter] = "true"
		deployment.Spec.Template.ObjectMeta.Labels[utils.MeshRouter] = "true"
	}

	return deployment
}

func (d RouterDeployment) getObjectMeta() metav1.ObjectMeta {
	controller := false
	labelAppKuberName := d.GatewayName
	if labelVal, ok := d.CrLabels["app.kubernetes.io/name"]; ok && d.MasterCR != "" {
		labelAppKuberName = labelVal
	}
	objectMeta := metav1.ObjectMeta{
		Namespace: d.NameSpace,
		Name:      d.GatewayName,
		Labels: d.propagateLabels(map[string]string{
			"name":                                  d.GatewayName,
			"app.kubernetes.io/name":                labelAppKuberName,
			"app.kubernetes.io/instance":            d.InstanceLabel,
			"app.kubernetes.io/component":           "mesh-gateway",
			"app.kubernetes.io/technology":          "cpp",
			"app.kubernetes.io/managed-by":          "operator",
			"app.kubernetes.io/managed-by-operator": "facade-operator",
			utils.FacadeGateway:                     "true",
		}),
		Annotations: map[string]string{
			utils.LastAppliedCRAnnotation: d.LastAppliedCR,
		},
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion: d.MasterCRVersion,
				Kind:       d.MasterCRKind,
				Name:       d.MasterCRName,
				UID:        d.MasterCRUID,
				Controller: &controller,
			},
		},
	}
	d.propagateMasterCr(objectMeta)

	return objectMeta
}

func (d RouterDeployment) getPodTemplateSpec(memLimit string) corev1.PodTemplateSpec {
	labelAppKuberName := d.GatewayName
	if labelVal, ok := d.CrLabels["app.kubernetes.io/name"]; ok && d.MasterCR != "" {
		labelAppKuberName = labelVal
	}
	objectMeta := metav1.ObjectMeta{
		Labels: d.propagateLabels(map[string]string{
			"app":                          d.GatewayName,
			"name":                         d.GatewayName,
			"app.kubernetes.io/name":       labelAppKuberName,
			"app.kubernetes.io/instance":   d.InstanceLabel,
			"app.kubernetes.io/component":  "mesh-gateway",
			"app.kubernetes.io/technology": "cpp",
			utils.FacadeGateway:            "true",
		}),
	}
	d.propagateMasterCr(objectMeta)

	return corev1.PodTemplateSpec{
		ObjectMeta: objectMeta,
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            d.GatewayName,
					Image:           d.ImageName,
					Args:            []string{"/envoy/run.sh"},
					Ports:           d.getContainerPorts(),
					Env:             d.getEnvVariables(memLimit, d.GwTerminationGracePeriodS, d.EnvoyConcurrency),
					Resources:       d.Recourses,
					LivenessProbe:   d.getProbe(300, 30, 10, 1, 3),
					ReadinessProbe:  d.getProbe(1, 30, 2, 1, 15),
					SecurityContext: d.getSecurityContext(),
					VolumeMounts:    d.getVolumeMounts(),
				},
			},
			TopologySpreadConstraints:     d.getTopologySpreadConstraints(),
			Volumes:                       d.getVolumes(),
			TerminationGracePeriodSeconds: utils.GetPointer(int64(d.GwTerminationGracePeriodS)),
		},
	}
}

func (d RouterDeployment) propagateLabels(labels map[string]string) map[string]string {
	for _, label := range labelsToPropagate {
		if labelVal, ok := d.CrLabels[label]; ok {
			labels[label] = labelVal
		} else {
			labels[label] = utils.Unknown
		}
	}
	return labels
}

func (d RouterDeployment) propagateMasterCr(objectMeta metav1.ObjectMeta) {
	if d.MasterCR != "" {
		objectMeta.Labels[utils.MasterCR] = d.MasterCR
		if d.HostedBy != "" {
			objectMeta.Labels[utils.HostedByLabel] = d.HostedBy
		}
	}
}

func (d RouterDeployment) getSecurityContext() *corev1.SecurityContext {
	context := &corev1.SecurityContext{
		ReadOnlyRootFilesystem:   utils.GetPointer(d.ReadOnlyContainerEnabled),
		RunAsNonRoot:             utils.GetPointer(true),
		SeccompProfile:           &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
		AllowPrivilegeEscalation: utils.GetPointer(false),
		Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
	}
	if utils.GetPlatform() == utils.Kubernetes {
		context.RunAsGroup = utils.GetPointer(int64(10001))
	}
	return context
}

func (d RouterDeployment) getTopologySpreadConstraints() []corev1.TopologySpreadConstraint {
	if d.CloudTopologies != nil {
		var topologySpreadConstraints []corev1.TopologySpreadConstraint
		for _, topology := range d.CloudTopologies {
			topologySpreadConstraints = append(topologySpreadConstraints, corev1.TopologySpreadConstraint{
				MaxSkew:           *topology.MaxSkew,
				TopologyKey:       topology.TopologyKey,
				WhenUnsatisfiable: corev1.UnsatisfiableConstraintAction(*topology.WhenUnsatisfiable),
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"name": d.GatewayName},
				},
			})
		}
		return topologySpreadConstraints
	}

	return []corev1.TopologySpreadConstraint{{
		MaxSkew:           1,
		TopologyKey:       d.CloudTopologyKey,
		WhenUnsatisfiable: corev1.ScheduleAnyway,
		LabelSelector: &metav1.LabelSelector{
			MatchLabels: map[string]string{"name": d.GatewayName},
		},
	}}
}

func (d RouterDeployment) getProbe(initialDelaySeconds, timeoutSeconds, periodSeconds, successThreshold, failureThreshold int32) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path:   "/ready",
				Port:   intstr.FromInt32(9901),
				Scheme: corev1.URISchemeHTTP,
			},
		},
		InitialDelaySeconds: initialDelaySeconds,
		TimeoutSeconds:      timeoutSeconds,
		PeriodSeconds:       periodSeconds,
		SuccessThreshold:    successThreshold,
		FailureThreshold:    failureThreshold,
	}
}

func (d RouterDeployment) getEnvVariables(memLimit string, gwTerminationGracePeriodS int, envoyConcurrency int) []corev1.EnvVar {
	envVars := []corev1.EnvVar{
		{
			Name:  "SERVICE_NAME_VARIABLE",
			Value: d.ServiceName,
		},
		{
			Name: "CLOUD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: "POD_HOSTNAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		{
			Name:  "GW_MEMORY_LIMIT",
			Value: memLimit,
		},
		{
			Name:  "GW_TERMINATION_GRACE_PERIOD_S",
			Value: strconv.Itoa(gwTerminationGracePeriodS),
		},
		{
			Name:  "TRACING_ENABLED",
			Value: d.TracingEnabled,
		},
		{
			Name:  "TRACING_HOST",
			Value: d.TracingHost,
		},
		{
			Name:  "IP_STACK",
			Value: d.IpStack,
		},
		{
			Name:  "IP_BIND",
			Value: d.IpBind,
		},
		{
			Name:  "ENVOY_UID",
			Value: "0",
		},
		{
			Name:  "XDS_CLUSTER_HOST",
			Value: d.XdsClusterHost,
		},
		{
			Name:  "XDS_CLUSTER_PORT",
			Value: d.XdsClusterPort,
		},
		{
			Name:  "LOG_LEVEL",
			Value: configloader.GetOrDefaultString("log.level", "info"),
		},
		{
			Name:  "ENVOY_CONCURRENCY",
			Value: strconv.Itoa(envoyConcurrency),
		},
	}

	return envVars
}

func (d RouterDeployment) getContainerPorts() []corev1.ContainerPort {
	ports := []corev1.ContainerPort{
		{
			Name:          "admin",
			ContainerPort: 9901,
			Protocol:      "TCP",
		},
	}

	if d.GatewayPorts != nil && len(d.GatewayPorts) > 0 {
		for _, port := range d.GatewayPorts {
			if port.Port == 9901 {
				continue
			}
			protocol := "TCP"
			if port.Protocol != "" {
				protocol = strings.ToUpper(port.Protocol)
			}
			ports = append(ports, corev1.ContainerPort{
				Name:          port.Name,
				ContainerPort: port.Port,
				Protocol:      corev1.Protocol(protocol),
			})
		}
		return ports
	}

	ports = append(ports, corev1.ContainerPort{
		Name:          "web",
		ContainerPort: 8080,
		Protocol:      corev1.ProtocolTCP,
	})

	return ports
}

func (d RouterDeployment) getVolumes() []corev1.Volume {
	var volumes []corev1.Volume
	if d.ReadOnlyContainerEnabled {
		volumes = append(volumes, corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})
	}
	return volumes
}

func (d RouterDeployment) getVolumeMounts() []corev1.VolumeMount {
	var volumeMounts []corev1.VolumeMount
	if d.ReadOnlyContainerEnabled {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "config",
			MountPath: "/envoy/config",
		})
	}

	return volumeMounts
}
