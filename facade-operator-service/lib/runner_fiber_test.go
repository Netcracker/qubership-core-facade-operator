package lib

import (
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/netcracker/qubership-core-lib-go-fiber-server-utils/v2/security"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/netcracker/qubership-core-lib-go/v3/serviceloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var registerFiberSecurityOnce sync.Once

func registerFiberSecurityForTests() {
	registerFiberSecurityOnce.Do(func() {
		serviceloader.Register(1, &security.DummyFiberServerSecurityMiddleware{})
	})
}

func TestNewFiberServerAppRegistersTracerWhenEnabled(t *testing.T) {
	registerFiberSecurityForTests()
	prev := otel.GetTracerProvider()
	t.Cleanup(func() { otel.SetTracerProvider(prev) })

	t.Setenv("TRACING_ENABLED", "true")
	t.Setenv("TRACING_HOST", "127.0.0.1")
	t.Setenv("TRACING_SAMPLER_RATELIMITING", "10")
	t.Setenv("MICROSERVICE_NAME", "facade-operator")
	t.Setenv("MICROSERVICE_NAMESPACE", "test")
	configloader.Init(configloader.EnvPropertySource())

	app, err := newFiberServerApp(fiber.Config{}, "0")
	require.NoError(t, err)
	require.NotNil(t, app)

	_, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider)
	assert.True(t, ok, "expected SDK tracer provider when tracing is enabled")

	resp, err := app.Test(httptest.NewRequest(fiber.MethodGet, "/prometheus", nil), -1)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}
