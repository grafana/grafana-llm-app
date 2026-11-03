package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	mcpgrafana "github.com/grafana/mcp-grafana/v2"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// pluginManifest is the subset of src/plugin.json this test reads.
type pluginManifest struct {
	IAM struct {
		Permissions []struct {
			Action string `json:"action"`
			Scope  string `json:"scope"`
		} `json:"permissions"`
	} `json:"iam"`
}

// pluginAccessScopes are the apps whose tools are exposed through the built-in
// MCP server. A tool that calls one of these apps declares the generic
// "plugins.app:access" action, so the scope is what actually grants access.
var pluginAccessScopes = []string{
	"grafana-incident-app",
	"grafana-irm-app",
	"grafana-assistant-app",
	"grafana-agento11y-app",
}

// TestPluginJSONGrantsRequiredPermissions keeps src/plugin.json's iam.permissions
// in step with the actions the bundled mcp-grafana tools declare. Every tool in
// mcp-grafana carries a "grafana.com/requiredPermissions" meta entry, and the
// plugin's service account only gets what plugin.json asks for, so a missing
// entry means a tool that always fails with 403.
func TestPluginJSONGrantsRequiredPermissions(t *testing.T) {
	ctx := t.Context()

	srv, err := New(Settings{
		IsGrafanaCloud:   true,
		IsToolsetEnabled: func(Toolset) bool { return true },
	}, "test")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(srv.Close)

	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()
	if _, err := srv.Server.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("connect server: %v", err)
	}
	client, err := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "iam-test", Version: "test"}, nil).
		Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}

	list, err := client.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(list.Tools) == 0 {
		t.Fatal("server registered no tools")
	}

	var (
		required          []string
		needsPluginAccess bool
		missingMeta       []string
	)
	for _, tool := range list.Tools {
		perms, declared := mcpgrafana.ToolRequiredPermissions(tool)
		if !declared {
			missingMeta = append(missingMeta, tool.Name)
			continue
		}
		for _, action := range perms {
			// Actions of the form "group/resource:verb" are API server RBAC
			// actions. Grafana only enforces them for on-behalf-of tokens, and
			// the plugin calls Grafana with its own service account, so the
			// legacy action is the one that has to be granted.
			if strings.Contains(action, "/") {
				continue
			}
			if action == "plugins.app:access" {
				needsPluginAccess = true
				continue
			}
			required = append(required, action)
		}
	}
	sort.Strings(missingMeta)
	if len(missingMeta) > 0 {
		t.Fatalf("tools without required permissions metadata: %v", missingMeta)
	}

	manifestPath := filepath.Join("..", "..", "src", "plugin.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read %s: %v", manifestPath, err)
	}
	var manifest pluginManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse %s: %v", manifestPath, err)
	}

	granted := make(map[string]map[string]bool, len(manifest.IAM.Permissions))
	for _, perm := range manifest.IAM.Permissions {
		if perm.Action == "" {
			t.Errorf("%s: iam.permissions entry with empty action", manifestPath)
			continue
		}
		if granted[perm.Action] == nil {
			granted[perm.Action] = map[string]bool{}
		}
		granted[perm.Action][perm.Scope] = true
	}

	sort.Strings(required)
	required = slices.Compact(required)

	var missing []string
	for _, action := range required {
		if _, ok := granted[action]; !ok {
			missing = append(missing, action)
		}
	}
	if len(missing) > 0 {
		t.Errorf("%s does not grant actions required by the bundled MCP tools: %v", manifestPath, missing)
	}

	// Annotation writes are scoped, not just action-gated: Grafana evaluates
	// annotations:create with the "annotations:type:organization" scope for
	// organization annotations (pkg/api/annotations.go canCreateAnnotation), so
	// an unscoped entry alone returns 403.
	for _, action := range []string{
		"annotations:create",
		"annotations:read",
		"annotations:write",
		"annotations:delete",
	} {
		if !granted[action]["annotations:type:organization"] {
			t.Errorf("%s does not grant %s for scope annotations:type:organization", manifestPath, action)
		}
	}

	if needsPluginAccess {
		for _, pluginID := range pluginAccessScopes {
			if !granted["plugins.app:access"]["plugins:"+pluginID] {
				t.Errorf("%s does not grant plugins.app:access for scope plugins:%s", manifestPath, pluginID)
			}
		}
		// The manifest scopes are written as "plugins:<id>".
		for scope := range granted["plugins.app:access"] {
			if !strings.HasPrefix(scope, "plugins:") {
				t.Errorf("%s: plugins.app:access scope %q should look like plugins:<id>", manifestPath, scope)
			}
		}
	}
}
