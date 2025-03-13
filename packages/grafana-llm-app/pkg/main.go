package main

import (
	"os"

	"github.com/grafana/grafana-llm-app/pkg/plugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// version is the version of the plugin.
// It is set at build time by the Mage target, which passes
// the version to the Go linker using `-X main.version=x.y.z`.
var version = "development"

func init() {
	plugin.PluginVersion = version
}

func main() {
	log.DefaultLogger.Info("Starting plugin process", "version", version)
	// Start listening to requests sent from Grafana. This call is blocking so
	// it won't finish until Grafana shuts down the process or the plugin choose
	// to exit by itself using os.Exit. Manage automatically manages life cycle
	// of app instances. It accepts app instance factory as first
	// argument. This factory will be automatically called on incoming request
	// from Grafana to create different instances of `App` (per plugin
	// ID).
	if err := app.Manage("grafana-llm-app", plugin.NewApp, app.ManageOpts{}); err != nil {
		log.DefaultLogger.Error(err.Error())
		os.Exit(1)
	}
}
