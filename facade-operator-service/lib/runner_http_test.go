package lib

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/netcracker/qubership-core-lib-go/v3/configloader"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpServerResponses(t *testing.T) {
	app := newProbeTestApp(t)
	tests := []struct {
		name            string
		path            string
		wantStatus      int
		wantContentType string
		wantBody        string
	}{
		{name: "health probe", path: "/health", wantStatus: http.StatusOK, wantContentType: "application/json", wantBody: `"ok"`},
		{name: "unknown path", path: "/missing", wantStatus: http.StatusNotFound, wantContentType: "text/plain; charset=utf-8", wantBody: "Not Found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := app.Test(httptest.NewRequest(http.MethodGet, tt.path, nil), fiber.TestConfig{Timeout: 0})
			require.NoError(t, err, "GET %s", tt.path)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err, "read body of GET %s", tt.path)

			assert.Equal(t, tt.wantStatus, resp.StatusCode, "status")
			assert.Equal(t, tt.wantContentType, resp.Header.Get("Content-Type"), "Content-Type")
			assert.Equal(t, tt.wantBody, string(body), "body")
		})
	}
}

func newProbeTestApp(t *testing.T) *fiber.App {
	t.Helper()
	registerFiberSecurityForTests()
	usePrometheusRegistry(t)
	t.Setenv("MICROSERVICE_NAME", "facade-operator")
	t.Setenv("MICROSERVICE_NAMESPACE", "test")
	configloader.Init(configloader.EnvPropertySource())
	app, err := newFiberServerApp(fiber.Config{}, "0")
	require.NoError(t, err)
	app.Get("/health", healthProbe)
	return app
}

func usePrometheusRegistry(t *testing.T) {
	t.Helper()
	registry := prometheus.NewRegistry()
	previousRegisterer, previousGatherer := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	prometheus.DefaultRegisterer, prometheus.DefaultGatherer = registry, registry
	t.Cleanup(func() {
		prometheus.DefaultRegisterer, prometheus.DefaultGatherer = previousRegisterer, previousGatherer
	})
}
