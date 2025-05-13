package templates

import (
	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/api/facade"
	customerrors "github.com/netcracker/qubership-core-facade-operator/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	utilsCore "github.com/netcracker/qubership-core-lib-go/v3/utils"
	"maps"
	"os"
	"strings"

	v1 "github.com/openshift/api/route/v1"
	"k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type IngressTemplateBuilder struct {
	gwIngressAnnotations map[string]string
	x509Enable           bool
	isSatellite          bool
	baselineNamespace    string
	ingressClassName     *string
}

func NewIngressTemplateBuilder(x509Enable bool, isSatellite bool, baselineNamespace string) *IngressTemplateBuilder {
	peerNamespace := os.Getenv("PEER_NAMESPACE")
	var ingressClassName *string = nil
	ingressClassNameString := facade.IngressClassName
	if peerNamespace == "" {
		ingressClassNameStringFromEnv := os.Getenv("INGRESS_CLASS")
		if ingressClassNameStringFromEnv != "" {
			ingressClassNameString = ingressClassNameStringFromEnv
			ingressClassName = &ingressClassNameString
		}
	} else {
		ingressClassName = &ingressClassNameString
	}
	return &IngressTemplateBuilder{gwIngressAnnotations: buildGwIngressAnnotations(), x509Enable: x509Enable, isSatellite: isSatellite, baselineNamespace: baselineNamespace, ingressClassName: ingressClassName}
}

func buildGwIngressAnnotations() map[string]string {
	envVal := os.Getenv("GW_INGRESS_ANNOTATIONS")
	if envVal == "" {
		return nil
	}
	annotations := strings.Split(strings.ReplaceAll(envVal, "\r\n", "\n"), "\n")
	result := make(map[string]string, len(annotations))
	for _, annotationStr := range annotations {
		result[extractName(annotationStr)] = extractValue(annotationStr)
	}
	return result
}

func extractName(annotationString string) string {
	endIdx := strings.Index(annotationString, ":")
	return annotationString[:endIdx]
}

func extractValue(annotationString string) string {
	startIdx := strings.Index(annotationString, "'")
	endIdx := strings.LastIndex(annotationString, "'")
	return annotationString[startIdx+1 : endIdx]
}

type Ingress struct {
	Name             string
	Namespace        string
	Labels           map[string]string
	Annotations      map[string]string
	Hostname         string
	ServiceName      string
	Port             int32
	IngressClassName *string
	MasterCR         string
	MasterCRVersion  string
	MasterCRKind     string
	MasterCRUID      types.UID
}

func (i Ingress) BuildOpenshiftRoute() *v1.Route {
	controller := false
	return &v1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        i.Name,
			Namespace:   i.Namespace,
			Annotations: i.Annotations,
			Labels:      i.Labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: i.MasterCRVersion,
					Kind:       i.MasterCRKind,
					Name:       i.MasterCR,
					UID:        i.MasterCRUID,
					Controller: &controller,
				},
			},
		},
		Spec: v1.RouteSpec{
			Host: i.Hostname,
			To: v1.RouteTargetReference{
				Kind: "Service",
				Name: i.ServiceName,
			},
			Port: &v1.RoutePort{TargetPort: intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: i.Port,
			}},
		},
	}
}

func (i Ingress) BuildK8sIngress() *networkingv1.Ingress {
	controller := false
	return &networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        i.Name,
			Namespace:   i.Namespace,
			Annotations: i.Annotations,
			Labels:      i.Labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: i.MasterCRVersion,
					Kind:       i.MasterCRKind,
					Name:       i.MasterCR,
					UID:        i.MasterCRUID,
					Controller: &controller,
				},
			},
		},
		Spec: networkingv1.IngressSpec{
			IngressClassName: i.IngressClassName,
			Rules: []networkingv1.IngressRule{{
				Host: i.Hostname,
				IngressRuleValue: networkingv1.IngressRuleValue{
					HTTP: &networkingv1.HTTPIngressRuleValue{
						Paths: []networkingv1.HTTPIngressPath{{
							Path:     "/",
							PathType: utils.GetPointer(networkingv1.PathTypePrefix),
							Backend: networkingv1.IngressBackend{
								Service: &networkingv1.IngressServiceBackend{
									Name: i.ServiceName,
									Port: networkingv1.ServiceBackendPort{
										Number: i.Port,
									},
								},
							},
						}},
					},
				},
			}},
		},
	}
}

func (i Ingress) BuildK8sBetaIngress() *v1beta1.Ingress {
	controller := false
	return &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        i.Name,
			Namespace:   i.Namespace,
			Annotations: i.Annotations,
			Labels:      i.Labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: i.MasterCRVersion,
					Kind:       i.MasterCRKind,
					Name:       i.MasterCR,
					UID:        i.MasterCRUID,
					Controller: &controller,
				},
			},
		},
		Spec: v1beta1.IngressSpec{
			IngressClassName: i.IngressClassName,
			Rules: []v1beta1.IngressRule{{
				Host: i.Hostname,
				IngressRuleValue: v1beta1.IngressRuleValue{
					HTTP: &v1beta1.HTTPIngressRuleValue{
						Paths: []v1beta1.HTTPIngressPath{{
							Path:     "/",
							PathType: utils.GetPointer(v1beta1.PathTypePrefix),
							Backend: v1beta1.IngressBackend{
								ServiceName: i.ServiceName,
								ServicePort: intstr.IntOrString{
									Type:   intstr.Int,
									IntVal: i.Port,
								},
							},
						}},
					},
				},
			}},
		},
	}
}

func (b *IngressTemplateBuilder) BuildNameAndPort(ingressSpec facade.IngressSpec, cr facade.MeshGateway, gatewayServiceName string) (string, int32, error) {
	gwPort, err := b.resolveGatewayPort(ingressSpec, cr)
	if err != nil {
		return "", 0, err
	}
	ingressName := buildIngressName(gatewayServiceName, ingressSpec.IsGrpc, b.resolvePortName(gwPort, cr))
	return ingressName, gwPort, nil
}

func (b *IngressTemplateBuilder) BuildIngressTemplate(ingressSpec facade.IngressSpec, cr facade.MeshGateway, gatewayServiceName string) (Ingress, error) {
	ingressName, gwPort, err := b.BuildNameAndPort(ingressSpec, cr, gatewayServiceName)
	if err != nil {
		return Ingress{}, err
	}
	return Ingress{
		Name:             ingressName,
		Namespace:        cr.GetNamespace(),
		Labels:           b.buildIngressLabels(cr.GetLabels()["app.kubernetes.io/part-of"]),
		Annotations:      b.buildIngressAnnotations(gatewayServiceName, cr.GetNamespace(), ingressSpec.IsGrpc),
		Hostname:         ingressSpec.Hostname,
		ServiceName:      gatewayServiceName,
		Port:             gwPort,
		IngressClassName: b.ingressClassName,
		MasterCR:         cr.GetName(),
		MasterCRVersion:  cr.GetAPIVersion(),
		MasterCRKind:     cr.GetKind(),
		MasterCRUID:      cr.GetUID(),
	}, nil
}

func (b *IngressTemplateBuilder) buildIngressLabels(partOfLabel string) map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/managed-by":          "operator",
		"app.kubernetes.io/managed-by-operator": "facade-operator",
	}
	if partOfLabel != "" {
		labels["app.kubernetes.io/part-of"] = partOfLabel
	} else {
		labels["app.kubernetes.io/part-of"] = utils.Unknown
	}
	return labels
}

func (b *IngressTemplateBuilder) buildIngressAnnotations(gatewayServiceName, namespace string, isGrpc bool) map[string]string {
	annotations := make(map[string]string)
	annotations["app.kubernetes.io/managed-by"] = "facade-operator"

	mapper := serviceloader.MustLoad[utilsCore.AnnotationMapper]()
	maps.Copy(annotations, mapper.AddPrefix(map[string]string{"start.stage": "1"}))
	if gatewayServiceName == facade.PublicGatewayService {
		maps.Copy(annotations, mapper.AddPrefix(map[string]string{
			"tenant.service.tenant.id":        "GENERAL",
			"tenant.service.show.name":        "Public Gateway",
			"tenant.service.show.description": "Api Gateway to access public API",
		}))
		maps.Copy(annotations, b.gwIngressAnnotations)
	} else if gatewayServiceName == facade.PrivateGatewayService {
		maps.Copy(annotations, b.gwIngressAnnotations)
	}

	if utils.GetPlatform() == utils.Openshift {
		return annotations
	}

	if isGrpc {
		annotations["nginx.ingress.kubernetes.io/ssl-redirect"] = "true"
		annotations["nginx.ingress.kubernetes.io/backend-protocol"] = "GRPC"
	}
	if b.x509Enable {
		annotations["nginx.ingress.kubernetes.io/auth-tls-pass-certificate-to-upstream"] = "true"
		annotations["nginx.ingress.kubernetes.io/auth-tls-verify-client"] = "optional_no_ca"
		if b.isSatellite {
			annotations["nginx.ingress.kubernetes.io/auth-tls-secret"] = fmt.Sprintf("%s/x509", b.baselineNamespace)
		} else {
			annotations["nginx.ingress.kubernetes.io/auth-tls-secret"] = fmt.Sprintf("%s/x509", namespace)
		}
	}
	return annotations
}

func addGrpcSuffixForIngress(ingressName string, isGrpc bool) string {
	if isGrpc {
		return ingressName + "-grpc"
	}
	return ingressName
}

func buildIngressName(gatewayServiceName string, isGrpc bool, portName string) string {
	switch gatewayServiceName {
	case facade.PublicGatewayService:
		return addGrpcSuffixForIngress("public-gateway", isGrpc)
	case facade.PrivateGatewayService:
		return addGrpcSuffixForIngress("private-gateway", isGrpc)
	default:
		return addGrpcSuffixForIngress(fmt.Sprintf("%s-%s", gatewayServiceName, portName), isGrpc)
	}
}

func (b *IngressTemplateBuilder) resolveGatewayPort(ingressSpec facade.IngressSpec, cr facade.MeshGateway) (int32, error) {
	// if ingress spec already has port - return it
	gatewayPort := ingressSpec.GatewayPort
	if gatewayPort > 0 {
		return gatewayPort, nil
	}

	// try to get port from facadeService CR GatewayPorts collection
	if len(cr.GetSpec().GatewayPorts) > 0 {
		if len(cr.GetSpec().GatewayPorts) > 1 {
			return 0, errs.NewError(customerrors.InvalidFacadeServiceCRError, fmt.Sprintf("could not resolve gateway port while building Ingress by CR %s: there are more than one port configured for the gateway service", cr.GetName()), nil)
		}
		gatewayPort = cr.GetSpec().GatewayPorts[0].Port
	}

	if gatewayPort <= 0 {
		// still no luck, try to get port from facadeService CR port field
		gatewayPort = cr.GetSpec().Port
	}

	if gatewayPort > 0 {
		return gatewayPort, nil
	}

	return 8080, nil
}

func (b *IngressTemplateBuilder) resolvePortName(port int32, cr facade.MeshGateway) string {
	for _, gwPort := range cr.GetSpec().GatewayPorts {
		if gwPort.Port == port {
			return gwPort.Name
		}
	}

	return "web"
}
