package mock

//go:generate mockgen --destination=./workqueue/stub_workqueue.go --package=mock_workqueue  --build_flags=--mod=mod k8s.io/client-go/util/workqueue RateLimitingInterface
//go:generate mockgen --destination=./client/stub_client.go --package=mock_client --build_flags=--mod=mod sigs.k8s.io/controller-runtime/pkg/client Client
//go:generate mockgen --destination=./client/stub_sub_resource_writer.go --package=mock_client --build_flags=--mod=mod sigs.k8s.io/controller-runtime/pkg/client SubResourceWriter
//go:generate mockgen -source=../../pkg/services/config_map_client.go -destination=./services/stub_config_map_client.go -package=mock_services
//go:generate mockgen -source=../../pkg/services/deployment_client.go -destination=./services/stub_deployment_client.go -package=mock_services
//go:generate mockgen -source=../../pkg/services/pod_monitor_client.go -destination=./services/stub_pod_monitor_client.go -package=mock_services
//go:generate mockgen -source=../../pkg/services/hpa_client.go -destination=./services/stub_hpa_client.go -package=mock_services
//go:generate mockgen -source=../../pkg/services/service_client.go -destination=./services/stub_service_client.go -package=mock_services
//go:generate mockgen -source=../../pkg/services/status_updater.go -destination=./services/stub_status_updater.go -package=mock_services
//go:generate mockgen -source=../../pkg/services/ready_service.go -destination=./services/stub_ready_service.go -package=mock_services
//go:generate mockgen -source=../../pkg/services/common_cr_client.go -destination=./services/stub_common_cr_client.go -package=mock_services
//go:generate mockgen -source=../../pkg/services/cr_priority_service.go -destination=./services/stub_cr_priority_service.go -package=mock_services
//go:generate mockgen -source=../../pkg/restclient/control_plane.go -destination=./restclient/stub_control_plane.go -package=mock_restclient
