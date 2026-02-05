package lib

import (
	"context"
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	v1cert "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	"github.com/gofiber/fiber/v2"
	facadeV1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1"
	facadeV1Alpha "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/facade/v1alpha"
	monitoringV1 "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/api/monitoring/v1"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/controllers"
	localLog "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/log"
	customerrors "github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/errors"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/indexes"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/restclient"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/services"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/templates"
	"github.com/netcracker/qubership-core-facade-operator/facade-operator-service/v2/pkg/utils"
	errs "github.com/netcracker/qubership-core-lib-go-error-handling/v3/errors"
	fiberserver "github.com/netcracker/qubership-core-lib-go-fiber-server-utils/v2"
	"github.com/netcracker/qubership-core-lib-go-fiber-server-utils/v2/server"
	"github.com/netcracker/qubership-core-lib-go-rest-utils/v2/consul-propertysource"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	openshiftv1 "github.com/openshift/api/route/v1"
	hpav2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = logging.GetLogger("setup")
	ctx      = context.WithValue(context.Background(), "requestId", "")
)

func RunService() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(facadeV1Alpha.AddToScheme(scheme))
	utilruntime.Must(facadeV1.AddToScheme(scheme))
	utilruntime.Must(monitoringV1.AddToScheme(scheme))
	utilruntime.Must(v1cert.AddToScheme(scheme))
	utilruntime.Must(openshiftv1.Install(scheme))
	utilruntime.Must(hpav2.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme
	consulPS := consul.NewLoggingPropertySource()
	propertySources := configloader.BasePropertySources(configloader.YamlPropertySourceParams{ConfigFilePath: "application.yaml"})
	configloader.InitWithSourcesArray(append(propertySources, consulPS))
	consul.StartWatchingForPropertiesWithRetry(context.Background(), consulPS, func(event interface{}, err error) {})

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8082", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{
		Encoder: localLog.NewZapEncoder(),
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	namespace := os.Getenv("CLOUD_NAMESPACE")
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		NewCache: func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
			opts.DefaultNamespaces = map[string]cache.Config{
				namespace: {},
			}
			return cache.New(config, opts)
		},
		Scheme: scheme,
		WebhookServer: webhook.NewServer(webhook.Options{
			Port: 9443,
		}),
		Metrics:          metricsserver.Options{BindAddress: metricsAddr},
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "a0366e12",
	})
	if err != nil {
		setupLog.Error(errs.ToLogFormat(errs.NewError(customerrors.UnknownErrorCode, "Unable to start manager", err)))
		os.Exit(1)
	}

	setupReconcilers(mgr, namespace)
	if err = indexes.IndexFields(context.Background(), mgr.GetCache()); err != nil {
		setupLog.Error(errs.ToLogFormat(errs.NewError(customerrors.UnknownErrorCode, "Unable to index k8s cache", err)))
		os.Exit(1)
	}

	startServer(mgr)
}

func startServer(mgr manager.Manager) {
	setupLog.Info("Start server...")

	fiberConfig := fiber.Config{
		Network:     fiber.NetworkTCP,
		IdleTimeout: 30 * time.Second,
	}
	pprofPort := configloader.GetOrDefaultString("pprof.port", "6060")
	app, err := fiberserver.New(fiberConfig).
		WithPprof(pprofPort).
		WithPrometheus("/prometheus").
		WithApiVersion().
		Process()
	if err != nil {
		setupLog.Error("Error while create app because: " + err.Error())
		return
	}
	app.Get("/health", healthProbe)
	app.Get("/ready", healthProbe)
	go server.StartServer(app, "http.server.bind")

	//+kubebuilder:scaffold:builder
	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(errs.ToLogFormat(errs.NewError(customerrors.UnknownErrorCode, "Can not run manager", err)))
		os.Exit(1)
	}
}

func healthProbe(c *fiber.Ctx) error {
	return c.Status(http.StatusOK).JSON("ok")
}

func setupReconcilers(mgr manager.Manager, namespace string) {
	maxConcurrentReconciles, err := strconv.Atoi(os.Getenv("MAX_CONCURRENT_RECONCILES"))
	if err != nil {
		setupLog.Error(errs.ToLogFormat(errs.NewError(customerrors.InitParamsValidationError, "Can not parse MAX_CONCURRENT_RECONCILES value. Value should be integer", err)))
		os.Exit(1)
	}
	client := mgr.GetClient()
	ingressBuilder := templates.NewIngressTemplateBuilder(
		utils.GetBoolEnvValueOrDefault("X509_AUTHENTICATION_ENABLED", false),
		utils.GetBoolEnvValueOrDefault("COMPOSITE_PLATFORM", false),
		os.Getenv("BASELINE_PROJ"))

	commonCRClient := services.NewCommonCRClient(client)
	serviceClient := services.NewServiceClient(client)
	deploymentClient := services.NewDeploymentClientImpl(client)
	configMapClient := services.NewConfigMapClient(client)
	podMonitorClient := services.NewPodMonitorClient(client)
	hpaClient := services.NewHPAClient(client)
	ingressClient := services.NewIngressClientAggregator(client, ingressBuilder, commonCRClient)
	controlPlaneClient := restclient.NewControlPlaneClient()
	statusUpdater := services.NewStatusUpdater(client)
	readyService := services.NewReadyService(deploymentClient, statusUpdater)
	crPriorityService := services.NewCRPriorityService(deploymentClient, commonCRClient)
	commonFacadeReconciler := controllers.NewFacadeCommonReconciler(
		client,
		serviceClient,
		deploymentClient,
		configMapClient,
		podMonitorClient,
		hpaClient,
		ingressClient,
		controlPlaneClient,
		ingressBuilder,
		statusUpdater,
		readyService,
		commonCRClient,
		crPriorityService)
	facadeServiceReconciler := controllers.NewFacadeServiceReconciler(commonFacadeReconciler)
	meshGatewayReconciler := controllers.NewGatewayReconciler(commonFacadeReconciler)
	if err = facadeServiceReconciler.SetupFacadeServiceManager(mgr, maxConcurrentReconciles, client, deploymentClient, commonCRClient); err != nil {
		setupLog.Error(errs.ToLogFormat(errs.NewError(customerrors.UnknownErrorCode, "Unable to create FacadeService controller", err)))
		os.Exit(1)
	}
	if err = meshGatewayReconciler.SetupMeshGatewayManager(mgr, maxConcurrentReconciles, client, deploymentClient, commonCRClient); err != nil {
		setupLog.Error(errs.ToLogFormat(errs.NewError(customerrors.UnknownErrorCode, "Unable to create MeshGateway controller", err)))
		os.Exit(1)
	}
	configMapReconciler := controllers.NewConfigMapReconciller(client, configMapClient)
	if err = configMapReconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(errs.ToLogFormat(errs.NewError(customerrors.UnknownErrorCode, "Unable to create FacadeService controller", err)))
		os.Exit(1)
	}
}
