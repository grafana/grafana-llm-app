import { LiveChannelAddress, LiveChannelScope } from "@grafana/data";
import { logWarning } from "@grafana/runtime";

import { SemVer } from "semver";

export const LLM_PLUGIN_ID = "grafana-llm-app";
export const LLM_PLUGIN_ROUTE = `/api/plugins/${LLM_PLUGIN_ID}`;

// Grafana 12.4 renamed `LiveChannelAddress.namespace` to `stream` and now
// builds the channel id from `stream`, ignoring `namespace`. Older versions do
// the reverse. We must set both so a single plugin build subscribes to the
// correct channel regardless of which Grafana version is running; omitting
// `stream` on 12.4+ yields a `plugin/undefined/...` channel that never routes.
// The intersection adds `stream` to the type without losing type checking on
// the other fields, even when building against a pre-12.4 `@grafana/data`.
type PluginLiveChannelAddress = LiveChannelAddress & { stream: string };

export function pluginLiveChannel(
  path: string,
  data?: unknown,
): PluginLiveChannelAddress {
  return {
    scope: LiveChannelScope.Plugin,
    namespace: LLM_PLUGIN_ID,
    stream: LLM_PLUGIN_ID,
    path,
    data,
  };
}

// The LLM app was at version 0.2.0 before we added the health check.
// If the health check fails, or the details don't exist on the response,
// we should assume it's this older version.
export let LLM_PLUGIN_VERSION = new SemVer("0.2.0");

export function setLLMPluginVersion(version: string) {
  try {
    LLM_PLUGIN_VERSION = new SemVer(version);
  } catch (e) {
    logWarning(
      "Failed to parse version of grafana-llm-app; assuming old version is present.",
    );
  }
}
