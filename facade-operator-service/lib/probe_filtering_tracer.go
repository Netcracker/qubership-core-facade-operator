package lib

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/logging"
	"go.opentelemetry.io/otel"
	zipkintr "go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

var probeFilterTracerLog = logging.GetLogger("probe-filtering-tracer")

type probeFilteringZipkinTracer struct {
	enabled     bool
	host        string
	samplerRate int
	serviceName string
	namespace   string
}

func newProbeFilteringZipkinTracer() *probeFilteringZipkinTracer {
	enabled, err := strconv.ParseBool(configloader.GetOrDefaultString("tracing.enabled", "false"))
	if err != nil {
		panic(err)
	}
	rate, err := strconv.Atoi(configloader.GetOrDefaultString("tracing.sampler.ratelimiting", "10"))
	if err != nil {
		panic(err)
	}
	return &probeFilteringZipkinTracer{
		enabled:     enabled,
		host:        configloader.GetOrDefaultString("tracing.host", ""),
		samplerRate: rate,
		serviceName: configloader.GetOrDefaultString("microservice.name", ""),
		namespace:   configloader.GetOrDefaultString("microservice.namespace", "unknown"),
	}
}

func (t *probeFilteringZipkinTracer) ServerName() string {
	return t.serviceName
}

func (t *probeFilteringZipkinTracer) RegisterTracerProvider() (bool, error) {
	if !t.enabled {
		probeFilterTracerLog.Debugf("zipkin tracer is disabled")
		return false, nil
	}
	if t.serviceName == "" {
		return false, errors.New("you must specify microservice.name configuration parameter")
	}
	if t.host == "" {
		return false, errors.New("tracing host is empty")
	}
	if t.samplerRate <= 0 {
		return false, errors.New("tracing sampler rate limiting parameter must be more than 0")
	}

	exporter, err := zipkintr.New(fmt.Sprintf("http://%s:9411/api/v2/spans", t.host))
	if err != nil {
		return false, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)),
		sdktrace.WithResource(resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceNameKey.String(t.serviceName+"-"+t.namespace),
			semconv.ServiceNamespaceKey.String(t.namespace))),
		sdktrace.WithSampler(newProbeFilteringSampler(float64(t.samplerRate))),
	)
	otel.SetTracerProvider(tp)
	probeFilterTracerLog.Debug("zipkin tracer with probe filtering was registered as global tracer provider")
	return true, nil
}
