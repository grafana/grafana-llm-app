package plugin

import (
	"context"
	"net/http"
	"strings"

	llm "github.com/grafana/grafana-llm-app/llmclient"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

const (
	GrafanaAPIKey = "grafanaApiKey"
)

// Make sure App implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. Plugin should not implement all these interfaces - only those which are
// required for a particular task.
var (
	_ backend.CallResourceHandler   = (*App)(nil)
	_ instancemgmt.InstanceDisposer = (*App)(nil)
	_ backend.CheckHealthHandler    = (*App)(nil)
)

type AppSettings struct {
}

// App is an example app backend plugin which can respond to data queries.
type App struct {
	backend.CallResourceHandler

	openai llm.OpenAI
}

// NewApp creates a new example *App instance.
func NewApp(ctx context.Context, appSettings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Debug("Creating new app instance")
	var app App

	// Use a httpadapter (provided by the SDK) for resource calls. This allows us
	// to use a *http.ServeMux for resource calls, so we can map multiple routes
	// to CallResource without having to implement extra logic.
	mux := http.NewServeMux()
	app.registerRoutes(mux)
	app.CallResourceHandler = httpadapter.New(mux)

	cfg := backend.GrafanaConfigFromContext(ctx)
	grafanaAppURL := strings.TrimRight(cfg.Get("GF_APP_URL"), "/")
	if grafanaAppURL == "" {
		// For debugging purposes only
		grafanaAppURL = "http://localhost:3000"
	}
	// Note: this requires a Grafana API key to be stored in the plugin settings.
	// We can also add a helper function to use the [OAuthTokenRetriever][oauth] to fetch us
	// a service account token but this is currently behind a feature toggle.
	// oauth: https://pkg.go.dev/github.com/grafana/grafana-plugin-sdk-go@v0.176.0/experimental/oauthtokenretriever
	app.openai = llm.NewOpenAI(grafanaAppURL, appSettings.DecryptedSecureJSONData[GrafanaAPIKey])

	return &app, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created.
func (a *App) Dispose() {}

// CheckHealth handles health checks sent from Grafana to the plugin.
func (a *App) CheckHealth(_ context.Context, _ *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	log.DefaultLogger.Info("check health")
	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "ok",
	}, nil
}
