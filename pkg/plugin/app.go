package plugin

import (
	"context"
	"net/http"
	"sync"

	"github.com/grafana/grafana-llm-app/pkg/plugin/vector"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/resource/httpadapter"
)

// Make sure App implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. Plugin should not implement all these interfaces - only those which are
// required for a particular task.
var (
	_ backend.CallResourceHandler   = (*App)(nil)
	_ instancemgmt.InstanceDisposer = (*App)(nil)
	_ backend.CheckHealthHandler    = (*App)(nil)
	_ backend.StreamHandler         = (*App)(nil)
)

// App is an example app backend plugin which can respond to data queries.
type App struct {
	backend.CallResourceHandler

	vectorService vector.Service

	healthCheckClient healthCheckClient
	healthCheckMutex  sync.Mutex
	healthOpenAI      *openAIHealthDetails
	healthVector      *vectorHealthDetails
	settings          *Settings
}

// NewApp creates a new example *App instance.
func NewApp(ctx context.Context, appSettings backend.AppInstanceSettings) (instancemgmt.Instance, error) {
	log.DefaultLogger.Debug("Creating new app instance")
	var app App
	var err error

	log.DefaultLogger.Debug("Loading settings")
	app.settings, err = loadSettings(appSettings)
	if err != nil {
		log.DefaultLogger.Error("Error loading settings", "err", err)
		return nil, err
	}

	// Use a httpadapter (provided by the SDK) for resource calls. This allows us
	// to use a *http.ServeMux for resource calls, so we can map multiple routes
	// to CallResource without having to implement extra logic.
	mux := http.NewServeMux()
	app.registerRoutes(mux, *app.settings)
	app.CallResourceHandler = httpadapter.New(mux)

	if app.settings.Vector.Enabled {
		log.DefaultLogger.Debug("Creating vector service")
		app.vectorService, err = vector.NewService(
			app.settings.Vector,
			appSettings.DecryptedSecureJSONData,
		)
		if err != nil {
			log.DefaultLogger.Error("Error creating vector service", "err", err)
			return nil, err
		}
	}

	app.healthCheckClient = &http.Client{}
	app.healthCheckMutex = sync.Mutex{}

	return &app, nil
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created.
func (a *App) Dispose() {
	if a.vectorService != nil {
		a.vectorService.Cancel()
	}
}
