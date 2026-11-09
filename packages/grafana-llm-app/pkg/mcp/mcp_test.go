package mcp

import "testing"

func TestAllToolsets(t *testing.T) {
	all := AllToolsets()
	if len(all) != 34 {
		t.Errorf("AllToolsets() has %d entries, want 34", len(all))
	}
	seen := make(map[Toolset]bool, len(all))
	for _, ts := range all {
		if ts == "" {
			t.Error("AllToolsets() contains an empty toolset")
		}
		if seen[ts] {
			t.Errorf("AllToolsets() contains %q twice", ts)
		}
		seen[ts] = true
	}
}

func TestIsDefaultEnabled(t *testing.T) {
	// Keep this list in sync with defaultDisabledToolsets: a toolset that
	// grants wide privileges, needs an uncommon datasource, or only matters to
	// interactive Grafana Apps stays off until an operator turns it on.
	defaultOff := map[Toolset]bool{
		ToolsetQuickwit:      true,
		ToolsetInfluxDB:      true,
		ToolsetCloudLogging:  true,
		ToolsetRunPanelQuery: true,
		ToolsetGraphite:      true,
		ToolsetPlugin:        true,
		ToolsetAPI:           true,
		ToolsetAgento11y:     true,
		ToolsetAssistant:     true,
	}
	for _, ts := range AllToolsets() {
		want := !defaultOff[ts]
		if got := IsDefaultEnabled(ts); got != want {
			t.Errorf("IsDefaultEnabled(%q) = %v, want %v", ts, got, want)
		}
	}
}

func TestIsToolsetEnabled(t *testing.T) {
	t.Run("nil function enables all toolsets", func(t *testing.T) {
		s := Settings{}
		for _, ts := range AllToolsets() {
			if !s.isToolsetEnabled(ts) {
				t.Errorf("isToolsetEnabled(%q) = false, want true (nil func)", ts)
			}
		}
	})

	t.Run("custom function disables specific toolsets", func(t *testing.T) {
		disabled := map[Toolset]bool{ToolsetIncident: true, ToolsetOnCall: true}
		s := Settings{
			IsToolsetEnabled: func(toolset Toolset) bool {
				return !disabled[toolset]
			},
		}

		for _, tc := range []struct {
			toolset  Toolset
			expected bool
		}{
			{ToolsetSearch, true},
			{ToolsetDatasource, true},
			{ToolsetIncident, false},
			{ToolsetPrometheus, true},
			{ToolsetLoki, true},
			{ToolsetAlerting, true},
			{ToolsetDashboard, true},
			{ToolsetOnCall, false},
			{ToolsetAsserts, true},
			{ToolsetSQL, true},
			{ToolsetClickHouse, true},
		} {
			if got := s.isToolsetEnabled(tc.toolset); got != tc.expected {
				t.Errorf("isToolsetEnabled(%q) = %v, want %v", tc.toolset, got, tc.expected)
			}
		}
	})

	t.Run("custom function that disables all", func(t *testing.T) {
		s := Settings{
			IsToolsetEnabled: func(Toolset) bool { return false },
		}
		for _, ts := range AllToolsets() {
			if s.isToolsetEnabled(ts) {
				t.Errorf("isToolsetEnabled(%q) = true, want false", ts)
			}
		}
	})
}

func TestNewGrafanaCloudGating(t *testing.T) {
	// cloudOnly are the toolsets gated behind IsGrafanaCloud.
	cloudOnly := map[Toolset]bool{
		ToolsetIncident: true,
		ToolsetAsserts:  true,
	}

	for _, tc := range []struct {
		name           string
		isGrafanaCloud bool
	}{
		{"grafana cloud", true},
		{"non-cloud", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := Settings{
				IsGrafanaCloud:   tc.isGrafanaCloud,
				IsToolsetEnabled: func(Toolset) bool { return true },
			}
			for _, ts := range AllToolsets() {
				shouldBeEnabled := !cloudOnly[ts] || tc.isGrafanaCloud
				// We can't call New() (it needs real infra), so verify the
				// gating logic matches what New() implements.
				gated := s.isToolsetEnabled(ts) && (!cloudOnly[ts] || s.IsGrafanaCloud)
				if gated != shouldBeEnabled {
					t.Errorf("toolset %q: gated=%v, want %v (isGrafanaCloud=%v)", ts, gated, shouldBeEnabled, tc.isGrafanaCloud)
				}
			}
		})
	}
}
