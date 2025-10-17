package controllers

import (
	"context"
	"regexp"
	"runtime/debug"

	"fmt"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/restclient"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/services"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates/builder"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"strconv"
	"time"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	labelRegexp            = regexp.MustCompile("(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?")
	monitoringConfigSuffix = ".monitoring-config"
	podMonitorSuffix       = "-pod-monitor"
)

type FacadeCommonReconciler struct {
	client             client.Client
	logger             logging.Logger
	serviceClient      services.ServiceClient
	deploymentsClient  services.DeploymentClient
	configMapClient    services.ConfigMapClient
	podMonitorClient   services.PodMonitorClient
	hpaClient          services.HPAClient
	ingressClient      services.IngressClientAggregator
	ingressBuilder     *templates.IngressTemplateBuilder
	namedLock          *utils.NamedResourceLock
	controlPlaneClient restclient.ControlPlaneClient
	statusUpdater      services.StatusUpdater
	readyService       services.ReadyService
	commonCRClient     services.CommonCRClient
	crPriorityService  services.CRPriorityService
	hpaBuilder         builder.HPATemplateBuilder
}

func NewFacadeCommonReconciler(
	client client.Client,
	serviceClient services.ServiceClient,
	deploymentsClient services.DeploymentClient,
	configMapClient services.ConfigMapClient,
	podMonitorClient services.PodMonitorClient,
	hpaClient services.HPAClient,
	ingressClient services.IngressClientAggregator,
	controlPlaneClient restclient.ControlPlaneClient,
	ingressBuilder *templates.IngressTemplateBuilder,
	statusUpdater services.StatusUpdater,
	readyService services.ReadyService,
	commonCRClient services.CommonCRClient,
	crPriorityService services.CRPriorityService,
) *FacadeCommonReconciler {
	return &FacadeCommonReconciler{
		client:             client,
		logger:             logging.GetLogger("FacadeReconciler"),
		serviceClient:      serviceClient,
		deploymentsClient:  deploymentsClient,
		configMapClient:    configMapClient,
		podMonitorClient:   podMonitorClient,
		hpaClient:          hpaClient,
		ingressClient:      ingressClient,
		ingressBuilder:     ingressBuilder,
		controlPlaneClient: controlPlaneClient,
		namedLock:          utils.NewNamedResourceLock(),
		statusUpdater:      statusUpdater,
		readyService:       readyService,
		commonCRClient:     commonCRClient,
		crPriorityService:  crPriorityService,
		hpaBuilder:         builder.NewHPATemplateBuilder(),
	}
}

func (r *FacadeCommonReconciler) Reconcile(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) (result ctrl.Result, err error) {
	r.logger.InfoC(ctx, "[%v] Start reconcile", req.NamespacedName)
	r.namedLock.Lock(req.NamespacedName.String())
	defer r.namedLock.Unlock(req.NamespacedName.String())
	defer func() {
		recErr := recover()
		if recErr != nil {
			if statusErr := r.statusUpdater.SetFail(ctx, cr); statusErr != nil {
				r.logger.ErrorC(ctx, "[%v] Can not update status on CR. Error: %s", req.NamespacedName, statusErr.Error())
			}
			r.logger.ErrorC(ctx, "Found panic. Err: %s", recErr)
			r.logger.ErrorC(ctx, string(debug.Stack()))
			result = ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}
		}
	}()

	result, err = r.reconcile(ctx, req, cr)
	if err == nil {
		if result.Requeue {
			r.logger.InfoC(ctx, "[%v] Reconcile requeue", req.NamespacedName)
		} else {
			r.logger.InfoC(ctx, "[%v] Reconcile done", req.NamespacedName)
		}
		return result, err
	}

	switch e := err.(type) {
	case *customerrors.ExpectedError:
		r.logger.WarnC(ctx, "[%v] Found expected error. %v", req.NamespacedName, err)
		// It is necessary to mute the errors associated with the race
		// For example, when creating multiple CRs at the same time with 1 composite gateway
		return ctrl.Result{Requeue: true}, nil
	case *errs.ErrCodeError:
		r.logger.ErrorC(ctx, "[%v] %v", req.NamespacedName, errs.ToLogFormat(e))
		if statusErr := r.statusUpdater.SetFail(ctx, cr); statusErr != nil {
			r.logger.ErrorC(ctx, "[%v] Can not update status on CR. Error: %s", req.NamespacedName, statusErr.Error())
		}
		return ctrl.Result{}, e
	default:
		errorCode := errs.NewError(customerrors.UnknownErrorCode, "Unknown error", err)
		if statusErr := r.statusUpdater.SetFail(ctx, cr); statusErr != nil {
			r.logger.ErrorC(ctx, "[%v] Can not update status on CR. Error: %s", req.NamespacedName, statusErr.Error())
		}
		return ctrl.Result{}, errorCode
	}
}

func (r *FacadeCommonReconciler) reconcile(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) (ctrl.Result, error) {
	if err := r.deleteNotUsedMeshRouters(ctx, req); err != nil {
		return ctrl.Result{}, err
	}

	if cr != nil {
		if !r.readyService.IsUpdatingPhase(ctx, req, cr) {
			if err := r.statusUpdater.SetUpdating(ctx, cr); err != nil {
				return ctrl.Result{}, err
			}
		}
		if err := r.applyFacadeService(ctx, req, cr); err != nil {
			return ctrl.Result{}, err
		}
		return r.getCtrlResultForApply(ctx, req, cr)
	} else {
		return ctrl.Result{}, r.deleteFacadeService(ctx, req)
	}
}

func (r *FacadeCommonReconciler) getCtrlResultForApply(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) (ctrl.Result, error) {
	_, isV1Alpha := cr.(*facadeV1Alpha.FacadeService)
	if isV1Alpha {
		return ctrl.Result{}, nil
	}

	return r.readyService.CheckDeploymentReady(ctx, req, cr)
}

func (r *FacadeCommonReconciler) deleteFacadeService(ctx context.Context, req ctrl.Request) error {
	r.logger.InfoC(ctx, "[%v] Start delete facade service", req.NamespacedName)
	gatewayName := req.Name + utils.GatewaySuffix

	if err := r.deleteService(ctx, req, gatewayName); err != nil {
		return err
	}

	if err := r.deleteFacadeGateway(ctx, req, gatewayName); err != nil {
		return err
	}

	if err := r.ingressClient.DeleteOrphaned(ctx, req); err != nil {
		return err
	}

	if err := r.controlPlaneClient.DropGateway(ctx, req.Name); err != nil {
		return err
	}

	r.logger.InfoC(ctx, "[%v] Facade service deleted", req.NamespacedName)
	return nil
}

func (r *FacadeCommonReconciler) deleteService(ctx context.Context, req ctrl.Request, gatewayName string) error {
	foundService := &corev1.Service{}
	nameSpacedRequest := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}
	err := r.client.Get(ctx, nameSpacedRequest, foundService, &client.GetOptions{})
	if err != nil {
		if k8sErrors.IsNotFound(err) {
			r.logger.Debugf("[%v] Facade service %v not found", req.NamespacedName, req.Name)
			return nil
		}
		return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to get service %v", req.Name), err)
	}

	serviceSelector := foundService.Spec.Selector["app"]
	r.logger.InfoC(ctx, "[%v] Facade service selector.app '%v'. Gateway name: '%v'", req.NamespacedName, serviceSelector, gatewayName)
	if serviceSelector == gatewayName {
		err = r.client.Delete(ctx, foundService)
		if err != nil {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete service %v", req.Name), err)
		}
	} else {
		return r.deleteServiceBySelector(ctx, req, foundService, serviceSelector)
	}

	return nil
}

func (r *FacadeCommonReconciler) deleteServiceBySelector(ctx context.Context, req ctrl.Request, foundService *corev1.Service, serviceSelector string) error {
	r.logger.InfoC(ctx, "[%v] Try to find mesh router for service", req.NamespacedName)
	foundDeployment, err := r.deploymentsClient.Get(ctx, req, serviceSelector)
	if err != nil {
		return err
	}

	if r.serviceShouldBeDeleted(ctx, req, foundDeployment, foundService.Name, serviceSelector) {
		err = r.client.Delete(ctx, foundService)
		if err != nil {
			return errs.NewError(customerrors.UnexpectedKubernetesError, fmt.Sprintf("Failed to delete service %v", req.Name), err)
		}
	} else {
		r.logger.InfoC(ctx, "[%v] Found service %v but it is not mesh router", req.NamespacedName, serviceSelector)
	}

	return nil
}

func (r *FacadeCommonReconciler) serviceShouldBeDeleted(ctx context.Context, req ctrl.Request, foundDeployment *v1.Deployment, serviceName string, serviceSelector string) bool {
	if foundDeployment == nil {
		r.logger.InfoC(ctx, "[%v] Mesh router %v already deleted. Delete service %v", req.NamespacedName, serviceSelector, serviceName)
		return true
	} else if foundDeployment.GetLabels()[utils.FacadeGateway] == "true" && foundDeployment.GetLabels()[utils.MeshRouter] == "true" {
		r.logger.InfoC(ctx, "[%v] Found mesh router. Delete service %v", req.NamespacedName, serviceName)
		return true
	}

	return false
}

func (r *FacadeCommonReconciler) deleteFacadeGateway(ctx context.Context, req ctrl.Request, name string) error {
	r.logger.InfoC(ctx, "[%v] Start delete facade gateway %v", req.NamespacedName, name)
	isFacade, err := r.deploymentsClient.IsFacadeGateway(ctx, req, name)
	if err != nil || !isFacade {
		return err
	}

	if err = r.deploymentsClient.Delete(ctx, req, name); err != nil {
		return err
	}
	if err = r.deleteConfigMap(ctx, req, name); err != nil {
		return err
	}
	if err = r.deletePodMonitor(ctx, req, name); err != nil {
		return err
	}
	if err = r.hpaClient.Delete(ctx, req, name); err != nil {
		return err
	}

	r.logger.InfoC(ctx, "[%v] Facade gateway %v deleted", req.NamespacedName, name)
	return nil
}

func (r *FacadeCommonReconciler) applyFacadeService(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) error {
	r.logger.InfoC(ctx, "[%v] Start apply facade service", req.NamespacedName)
	virtualServiceMode := false
	gatewayDeploymentName := req.Name
	gatewayServiceName := utils.ResolveGatewayServiceName(req.Name, cr)
	if cr.GetSpec().Gateway != "null" && cr.GetSpec().Gateway != "" {
		virtualServiceMode = true
		gatewayDeploymentName = cr.GetSpec().Gateway
		r.namedLock.Lock(gatewayDeploymentName)
		defer r.namedLock.Unlock(gatewayDeploymentName)
	} else {
		gatewayDeploymentName = req.Name + utils.GatewaySuffix
	}
	r.logger.InfoC(ctx, "[%v] Virtual service mode: %v", req.NamespacedName, virtualServiceMode)

	if err := r.controlPlaneClient.RegisterGateway(ctx, gatewayServiceName, cr); err != nil {
		return err
	}

	gatewayImage, err := r.configMapClient.GetGatewayImage(ctx, req)
	if gatewayImage == "" || gatewayImage == "null" || err != nil {
		return errs.NewError(customerrors.GatewayImageError, "gateway image is empty", err)
	}
	r.logger.InfoC(ctx, "[%v] frontend gateway image: %v", req.NamespacedName, gatewayImage)

	if virtualServiceMode {
		if err = r.applyMeshRouter(ctx, req, gatewayDeploymentName, gatewayImage, cr); err != nil {
			return err
		}

		facadeGatewayName := req.Name + utils.GatewaySuffix
		if err = r.deleteFacadeGateway(ctx, req, facadeGatewayName); err != nil {
			return err
		}
	} else {
		if err = r.applyFacadeGateway(ctx, req, gatewayDeploymentName, gatewayImage, cr); err != nil {
			return err
		}
	}

	return r.applyIngresses(ctx, req, gatewayServiceName, cr)
}

func (r *FacadeCommonReconciler) applyIngresses(ctx context.Context, req ctrl.Request, serviceName string, cr facade.MeshGateway) error {
	if err := r.ingressClient.DeleteOrphaned(ctx, req); err != nil {
		return err
	}
	if cr.GetGatewayType() != facade.Ingress {
		return nil
	}
	for _, ingressSpec := range cr.GetSpec().Ingresses {
		ingressTemplate, err := r.ingressBuilder.BuildIngressTemplate(ingressSpec, cr, serviceName)
		if err != nil {
			return err
		}
		r.logger.InfoC(ctx, "[%v] Applying ingress %+v", req.NamespacedName, ingressSpec)
		if err = r.ingressClient.Apply(ctx, req, ingressTemplate); err != nil {
			return err
		}
	}
	return nil
}

func (r *FacadeCommonReconciler) getFacadeResources(ctx context.Context, req ctrl.Request, cr facade.MeshGateway) corev1.ResourceRequirements {
	resources := utils.GetResourceRequirements(ctx, cr)
	r.logger.InfoC(ctx, "[%v] Calculated memoryLimit: %v, cpuLimit: %v, cpuRequest: %v",
		req.NamespacedName,
		resources.Limits.Memory().String(),
		resources.Limits.Cpu().String(),
		resources.Requests.Cpu().String())

	return resources
}

func (r *FacadeCommonReconciler) getFacadeReplicas(ctx context.Context, cr facade.MeshGateway) int32 {
	value := cr.GetSpec().Replicas
	if value == nil {
		return getDefaultReplicasValue()
	}
	switch value.(type) {
	case int:
		return getFromIntReplicas(int32(value.(int)))
	case int32:
		return getFromIntReplicas(value.(int32))
	case int64:
		return getFromIntReplicas(int32(value.(int64)))
	case string:
		return getFromStrReplicas(value.(string))
	default:
		defaultValue := getDefaultReplicasValue()
		r.logger.WarnC(ctx, "Not supported value for replica %T. Using default value %v", value, defaultValue)
		return defaultValue
	}
}

func (r *FacadeCommonReconciler) getFacadeGatewayConcurrency(ctx context.Context, cr facade.MeshGateway) int {
	value := cr.GetSpec().Env.FacadeGatewayConcurrency
	defaultValue := getDefaultFacadeGatewayConcurrencyValue()
	if value == nil {
		r.logger.WarnC(ctx, "Not supported value for FacadeGatewayConcurrency %T. Using default value %v", value, defaultValue)
		return defaultValue
	}
	switch value.(type) {
	case int:
		return getValueFromInt(value.(int), defaultValue)
	case int32:
		return getValueFromInt(int(value.(int32)), defaultValue)
	case int64:
		return getValueFromInt(int(value.(int64)), defaultValue)
	case string:
		return getIntValueFromStr(value.(string), defaultValue)
	default:
		r.logger.WarnC(ctx, "Not supported value for FacadeGatewayConcurrency %T. Using default value %v", value, defaultValue)
		return defaultValue
	}
}

func getFromStrReplicas(value string) int32 {
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return getDefaultReplicasValue()
	}
	return int32(intValue)
}

func getFromIntReplicas(value int32) int32 {
	if value == 0 {
		return getDefaultReplicasValue()
	}
	return value
}

func getIntValueFromStr(value string, defaultValue int) int {
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}

func getValueFromInt(value int, defaultValue int) int {
	if value <= 0 {
		return defaultValue
	}
	return value
}

func getDefaultReplicasValue() int32 {
	intDefaultValue, err := strconv.Atoi(utils.DefaultFacadeGatewayReplicas)
	if err != nil {
		return 1
	}
	return int32(intDefaultValue)
}

func getDefaultFacadeGatewayConcurrencyValue() int {
	intDefaultValue, err := strconv.Atoi(utils.DefaultFacadeGatewayConcurrency)
	if err != nil {
		return 0
	}
	return intDefaultValue
}

func (r *FacadeCommonReconciler) deleteNotUsedMeshRouters(ctx context.Context, req ctrl.Request) error {
	r.logger.InfoC(ctx, "[%v] Start delete not used mesh routers", req.NamespacedName)
	deployments, err := r.deploymentsClient.GetMeshRouterDeployments(ctx, req)
	if err != nil {
		return err
	}
	if deployments == nil || len(deployments.Items) < 1 {
		r.logger.Debugf("[%v] Can not found mesh routers deployments", req.NamespacedName)
		return nil
	}
	allCRs, err := r.commonCRClient.GetAll(ctx, req)
	if err != nil {
		return err
	}

	return r.cleanupDeployments(ctx, req, deployments.Items, allCRs)
}

func (r *FacadeCommonReconciler) cleanupDeployments(ctx context.Context, req ctrl.Request, deployments []v1.Deployment, crs []facade.MeshGateway) error {
	var deploymentsForDelete []string
	var deploymentsForDeleteMasterCR []string
	for _, deployment := range deployments {
		masterCRExist := false
		crExist := false
		for _, cr := range crs {
			if deployment.GetName() == cr.GetSpec().Gateway {
				crExist = true
			}

			masterCRLabel := r.getMasterCR(deployment)
			if cr.GetSpec().MasterConfiguration && masterCRLabel == cr.GetName() {
				masterCRExist = true
			}
		}

		if !crExist {
			deploymentsForDelete = append(deploymentsForDelete, deployment.GetName())
		} else {
			if err := r.cleanupLastApplied(ctx, req, deployment); err != nil {
				return err
			}
		}

		if crExist && !masterCRExist {
			deploymentsForDeleteMasterCR = append(deploymentsForDeleteMasterCR, deployment.GetName())
		}
	}

	for _, routerForDelete := range deploymentsForDelete {
		if err := r.deleteMeshRouter(ctx, req, routerForDelete); err != nil {
			return err
		}
	}

	for _, routerForDeleteMasterCR := range deploymentsForDeleteMasterCR {
		if err := r.deploymentsClient.DeleteMasterCRLabel(ctx, req, routerForDeleteMasterCR); err != nil {
			return err
		}
	}

	r.logger.InfoC(ctx, "[%v] Done delete not used mesh routers", req.NamespacedName)
	return nil
}

func (r *FacadeCommonReconciler) cleanupLastApplied(ctx context.Context, req ctrl.Request, deployment v1.Deployment) error {
	if deployment.Annotations == nil {
		r.logger.InfoC(ctx, "[%v] Annotations not found on deployment '%s'", req.NamespacedName, deployment.GetName())
		return nil
	}
	lastAppliedCRAnnotation := deployment.Annotations[utils.LastAppliedCRAnnotation]
	if lastAppliedCRAnnotation == "" {
		r.logger.InfoC(ctx, "[%v] %s annotation not found on deployment '%s'", req.NamespacedName, utils.LastAppliedCRAnnotation, deployment.GetName())
		return nil
	}

	lastAppliedCr, err := utils.JsonUnmarshal[utils.LastAppliedCr](lastAppliedCRAnnotation)
	if err != nil {
		r.logger.ErrorC(ctx, "[%v] Can not unmarshal '%s' annotation with value: %s", req.NamespacedName, utils.LastAppliedCRAnnotation, lastAppliedCRAnnotation)
		return err
	}
	foundAppliedCr, err := r.commonCRClient.GetByLastAppliedCr(ctx, req, lastAppliedCr)
	if err != nil {
		return err
	}

	if foundAppliedCr == nil {
		r.logger.InfoC(ctx, "[%v] Last applied CR %+v not found. Update deployment annotation", req.NamespacedName, lastAppliedCr)
		lastAppliedCr.Deleted = true
		return r.deploymentsClient.SetLastAppliedCR(ctx, req, deployment.Name, lastAppliedCr)
	}

	return nil
}

func (r *FacadeCommonReconciler) getMasterCR(deployment v1.Deployment) string {
	label := deployment.Labels[utils.MasterCR]
	if label != "" {
		return label
	}
	return deployment.Spec.Template.ObjectMeta.Labels[utils.MasterCR]
}

func (r *FacadeCommonReconciler) deleteMeshRouter(ctx context.Context, req ctrl.Request, name string) error {
	r.logger.InfoC(ctx, "[%v] Delete not used mesh router %v", req.NamespacedName, name)
	if err := r.serviceClient.Delete(ctx, req, name); err != nil {
		return err
	}

	if err := r.deploymentsClient.Delete(ctx, req, name); err != nil {
		return err
	}

	if err := r.deleteConfigMap(ctx, req, name); err != nil {
		return err
	}

	if err := r.deletePodMonitor(ctx, req, name); err != nil {
		return err
	}

	if err := r.hpaClient.Delete(ctx, req, name); err != nil {
		return err
	}

	return nil
}

func (r *FacadeCommonReconciler) applyMeshRouter(ctx context.Context, req ctrl.Request, gatewayName string, gatewayImage string, cr facade.MeshGateway) error {
	r.logger.InfoC(ctx, "[%v] Apply virtual service %s", req.NamespacedName, req.Name)
	if err := r.applyService(ctx, req, req.Name, gatewayName, cr); err != nil {
		return err
	}

	available, err := r.crPriorityService.UpdateAvailable(ctx, req, gatewayName, cr)
	if !available || err != nil {
		return err
	}

	serviceName := req.Name
	if cr.GetGatewayType() == facade.Mesh && req.Name != facade.InternalGatewayService {
		// only mesh gateway should have SERVICE_NAME_VARIABLE equal to deployment name; internal gateway is also a mesh gateway, but it is a special case
		serviceName = gatewayName
	}
	if err = r.applyDeployment(ctx, req, serviceName, gatewayName, gatewayImage, cr, true); err != nil {
		return err
	}

	if cr.GetGatewayType() == facade.Mesh && req.Name != facade.InternalGatewayService {
		// only mesh gateway should have additional service with name equal to deployment name; internal gateway is also a mesh gateway, but it is a special case
		r.logger.InfoC(ctx, "[%v] Apply gateway service %s", req.NamespacedName, gatewayName)
		if err = r.applyService(ctx, req, gatewayName, gatewayName, cr); err != nil {
			return err
		}
	}

	if err = r.applyMonitoringConfigMap(ctx, req, gatewayName, cr); err != nil {
		return err
	}
	hpa := r.hpaBuilder.Build(ctx, req, cr, gatewayName)
	if err = r.hpaClient.Create(ctx, req, hpa); err != nil {
		return err
	}

	return r.applyPodMonitor(ctx, req, gatewayName, cr)
}

func (r *FacadeCommonReconciler) applyFacadeGateway(ctx context.Context, req ctrl.Request, gatewayDeploymentName string, gatewayImage string, cr facade.MeshGateway) error {
	if err := r.applyService(ctx, req, req.Name, gatewayDeploymentName, cr); err != nil {
		return err
	}

	if err := r.applyDeployment(ctx, req, req.Name, gatewayDeploymentName, gatewayImage, cr, false); err != nil {
		return err
	}

	if err := r.applyMonitoringConfigMap(ctx, req, gatewayDeploymentName, cr); err != nil {
		return err
	}
	hpa := r.hpaBuilder.Build(ctx, req, cr, gatewayDeploymentName)
	if err := r.hpaClient.Create(ctx, req, hpa); err != nil {
		return err
	}

	return r.applyPodMonitor(ctx, req, gatewayDeploymentName, cr)
}

func (r *FacadeCommonReconciler) applyDeployment(ctx context.Context, req ctrl.Request, serviceName string, gatewayName string, publicGatewayImage string, cr facade.MeshGateway, meshRouter bool) error {
	instanceLabelValue := r.getInstanceLabelValue(gatewayName, req.Namespace)
	r.logger.InfoC(ctx, "[%v] Instance label value: %v", req.NamespacedName, instanceLabelValue)
	replicas := r.getFacadeReplicas(ctx, cr)
	r.logger.InfoC(ctx, "[%v] Calculated replicas: %v", req.NamespacedName, replicas)
	concurrency := r.getFacadeGatewayConcurrency(ctx, cr)
	r.logger.InfoC(ctx, "[%v] Calculated facade gateway concurrency: %v", req.NamespacedName, concurrency)

	var facadeMasterCR string
	if cr.GetSpec().MasterConfiguration {
		facadeMasterCR = cr.GetName()
	}

	lastAppliedCR, err := utils.JsonMarshal(utils.LastAppliedCr{
		ApiVersion: cr.GetAPIVersion(),
		Kind:       cr.GetKind(),
		Name:       cr.GetName(),
	})
	if err != nil {
		return err
	}

	deployment := templates.RouterDeployment{
		ServiceName:                serviceName,
		GatewayName:                gatewayName,
		NameSpace:                  req.Namespace,
		CrLabels:                   cr.GetLabels(),
		InstanceLabel:              instanceLabelValue,
		ArtifactDescriptionVersion: utils.ArtifactDescriptorVersion,
		ImageName:                  publicGatewayImage,
		Recourses:                  r.getFacadeResources(ctx, req, cr),
		TracingEnabled:             utils.TracingEnabled,
		TracingHost:                utils.TracingHost,
		IpStack:                    utils.IpStack,
		IpBind:                     utils.IpBind,
		MeshRouter:                 meshRouter,
		Replicas:                   replicas,
		CloudTopologies:            utils.GetCloudTopologies(ctx),
		CloudTopologyKey:           utils.CloudTopologyKey,
		XdsClusterHost:             utils.XDSClusterHost,
		XdsClusterPort:             utils.XDSClusterPort,
		TLSSecretPath:              utils.TlsSecretPath,
		TLSPasswordSecretName:      utils.TlsPasswordSecretName,
		TLSPasswordKey:             utils.TlsPasswordKey,
		MasterCR:                   facadeMasterCR,
		MasterCRName:               cr.GetName(),
		MasterCRVersion:            cr.GetAPIVersion(),
		MasterCRKind:               cr.GetKind(),
		MasterCRUID:                cr.GetUID(),
		GatewayPorts:               cr.GetSpec().GatewayPorts,
		ReadOnlyContainerEnabled:   utils.GetBoolEnvValueOrDefault(utils.ReadOnlyContainerEnabled, false),
		GwTerminationGracePeriodS:  utils.StringToIntValueOrDefault(ctx, utils.FacadeGatewayTerminationGracePeriodS, 60),
		EnvoyConcurrency:           concurrency,
		HostedBy:                   cr.GetLabels()[utils.HostedByLabel],
		LastAppliedCR:              lastAppliedCR,
	}

	return r.deploymentsClient.Apply(ctx, req, deployment.GetDeployment())
}

func (r *FacadeCommonReconciler) applyPodMonitor(ctx context.Context, req ctrl.Request, name string, cr facade.MeshGateway) error {
	if utils.MonitoringEnabled == "true" {
		podMonitorName := name + podMonitorSuffix
		if len(name)+len(podMonitorSuffix) > 63 {
			podMonitorName = name[0:63-len(podMonitorSuffix)] + podMonitorSuffix
		}
		podMonitor := &templates.FacadePodMonitor{
			Name:            podMonitorName,
			NameSpace:       req.Namespace,
			NameLabel:       podMonitorName,
			PartOfLabel:     cr.GetLabels()["app.kubernetes.io/part-of"],
			NameSelector:    name,
			MasterCR:        cr.GetName(),
			MasterCRVersion: cr.GetAPIVersion(),
			MasterCRKind:    cr.GetKind(),
			MasterCRUID:     cr.GetUID(),
		}

		return r.podMonitorClient.Create(ctx, req, podMonitor.GetPodMonitor())
	}
	return nil
}

func (r *FacadeCommonReconciler) applyMonitoringConfigMap(ctx context.Context, req ctrl.Request, gatewayName string, cr facade.MeshGateway) error {
	configMap := templates.FacadeConfigMap{
		Name:            gatewayName + monitoringConfigSuffix,
		Namespace:       req.Namespace,
		PartOfLabel:     cr.GetLabels()["app.kubernetes.io/part-of"],
		MasterCR:        cr.GetName(),
		MasterCRVersion: cr.GetAPIVersion(),
		MasterCRKind:    cr.GetKind(),
		MasterCRUID:     cr.GetUID(),
	}

	return r.configMapClient.Apply(ctx, req, configMap.GetConfigMap())
}

func (r *FacadeCommonReconciler) deletePodMonitor(ctx context.Context, req ctrl.Request, name string) error {
	r.logger.InfoC(ctx, "[%v] Delete PodMonitor %v", req.NamespacedName, name)
	podMonitorName := r.getShortName(name, podMonitorSuffix)
	return r.podMonitorClient.Delete(ctx, req, podMonitorName)
}

func (r *FacadeCommonReconciler) deleteConfigMap(ctx context.Context, req ctrl.Request, name string) error {
	configMapName := name + monitoringConfigSuffix
	return r.configMapClient.Delete(ctx, req, configMapName)
}

func (r *FacadeCommonReconciler) applyService(ctx context.Context, req ctrl.Request, name string, gatewayName string, cr facade.MeshGateway) error {
	service := templates.FacadeService{
		Name:            name,
		Namespace:       req.Namespace,
		Labels:          cr.GetLabels(),
		NameSelector:    gatewayName,
		Port:            cr.GetSpec().Port,
		GatewayPorts:    cr.GetSpec().GatewayPorts,
		MasterCR:        cr.GetName(),
		MasterCRVersion: cr.GetAPIVersion(),
		MasterCRKind:    cr.GetKind(),
		MasterCRUID:     cr.GetUID(),
	}

	return r.serviceClient.Apply(ctx, req, service.GetService())
}

func (r *FacadeCommonReconciler) getInstanceLabelValue(entityName string, suffix string) string {
	label := r.getShortName(entityName, utils.LabelsDelimiter+suffix)
	return labelRegexp.FindString(label)
}

func (r *FacadeCommonReconciler) getShortName(entityName string, suffix string) string {
	if len(entityName)+len(suffix) > 63 {
		return entityName[0:63-len(suffix)] + suffix
	}
	return entityName + suffix
}
