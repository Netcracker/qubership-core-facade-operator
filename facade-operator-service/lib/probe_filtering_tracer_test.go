package lib

import (
	"testing"

	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func initProbeFilterConfig(t *testing.T) {
	t.Helper()
	configloader.Init(configloader.EnvPropertySource())
}

func TestNewProbeFilteringZipkinTracerReadsConfig(t *testing.T) {
	t.Setenv("TRACING_ENABLED", "true")
	t.Setenv("TRACING_HOST", "diag-agent")
	t.Setenv("TRACING_SAMPLER_RATELIMITING", "5")
	t.Setenv("MICROSERVICE_NAME", "facade-operator")
	t.Setenv("MICROSERVICE_NAMESPACE", "cloud-core")
	initProbeFilterConfig(t)

	tracer := newProbeFilteringZipkinTracer()
	assert.True(t, tracer.enabled)
	assert.Equal(t, "diag-agent", tracer.host)
	assert.Equal(t, 5, tracer.samplerRate)
	assert.Equal(t, "facade-operator", tracer.serviceName)
	assert.Equal(t, "cloud-core", tracer.namespace)
	assert.Equal(t, "facade-operator", tracer.ServerName())
}

func TestNewProbeFilteringZipkinTracerDefaults(t *testing.T) {
	t.Setenv("TRACING_ENABLED", "")
	t.Setenv("TRACING_HOST", "")
	t.Setenv("TRACING_SAMPLER_RATELIMITING", "")
	t.Setenv("MICROSERVICE_NAME", "")
	t.Setenv("MICROSERVICE_NAMESPACE", "")
	initProbeFilterConfig(t)

	tracer := newProbeFilteringZipkinTracer()
	assert.False(t, tracer.enabled)
	assert.Equal(t, "", tracer.host)
	assert.Equal(t, 10, tracer.samplerRate)
	assert.Equal(t, "", tracer.serviceName)
	assert.Equal(t, "unknown", tracer.namespace)
}

func TestNewProbeFilteringZipkinTracerPanicsOnInvalidEnabled(t *testing.T) {
	t.Setenv("TRACING_ENABLED", "not-a-bool")
	t.Setenv("TRACING_SAMPLER_RATELIMITING", "10")
	initProbeFilterConfig(t)

	assert.Panics(t, func() { newProbeFilteringZipkinTracer() })
}

func TestNewProbeFilteringZipkinTracerPanicsOnInvalidSamplerRate(t *testing.T) {
	t.Setenv("TRACING_ENABLED", "false")
	t.Setenv("TRACING_SAMPLER_RATELIMITING", "abc")
	initProbeFilterConfig(t)

	assert.Panics(t, func() { newProbeFilteringZipkinTracer() })
}

func TestRegisterTracerProviderDisabled(t *testing.T) {
	tracer := &probeFilteringZipkinTracer{enabled: false}
	ok, err := tracer.RegisterTracerProvider()
	assert.NoError(t, err)
	assert.False(t, ok)
}

func TestRegisterTracerProviderValidationErrors(t *testing.T) {
	ok, err := (&probeFilteringZipkinTracer{enabled: true, host: "h", samplerRate: 10}).RegisterTracerProvider()
	assert.False(t, ok)
	assert.ErrorContains(t, err, "microservice.name")

	ok, err = (&probeFilteringZipkinTracer{enabled: true, serviceName: "svc", samplerRate: 10}).RegisterTracerProvider()
	assert.False(t, ok)
	assert.ErrorContains(t, err, "tracing host is empty")

	ok, err = (&probeFilteringZipkinTracer{enabled: true, serviceName: "svc", host: "h", samplerRate: 0}).RegisterTracerProvider()
	assert.False(t, ok)
	assert.ErrorContains(t, err, "rate limiting")
}

func TestRegisterTracerProviderEnabled(t *testing.T) {
	prev := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	tracer := &probeFilteringZipkinTracer{
		enabled:     true,
		host:        "127.0.0.1",
		samplerRate: 10,
		serviceName: "facade-operator",
		namespace:   "test-ns",
	}
	ok, err := tracer.RegisterTracerProvider()
	require.NoError(t, err)
	assert.True(t, ok)

	tp, isSDK := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	require.True(t, isSDK)
	require.NotNil(t, tp)
}
