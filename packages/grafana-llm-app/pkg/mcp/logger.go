package mcp

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
)

// Logger is a minimal implementation of mcp.Logger that logs to the Grafana backend.
// It bridges MCP logging to Grafana's logging system by implementing the mcp.Logger interface.
type Logger struct{}

// Infof logs an informational message using Grafana's logger.
func (l *Logger) Infof(format string, v ...any) {
	log.DefaultLogger.Info(fmt.Sprintf(format, v...))
}

// Errorf logs an error message using Grafana's logger.
func (l *Logger) Errorf(format string, v ...any) {
	log.DefaultLogger.Error(fmt.Sprintf(format, v...))
}
