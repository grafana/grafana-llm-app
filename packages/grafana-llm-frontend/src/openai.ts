/**
 * @deprecated This module is deprecated and will be removed in a future version.
 * Please use the vendor-neutral `llm.ts` module instead.
 *
 * All exports from this file are re-exported from `llm.ts` for backward compatibility.
 *
 * BREAKING CHANGE in v0.13.0: The health check response format has changed from
 * { details: { openAI: { configured: true, ok: true } } }
 * to
 * { details: { llmProvider: { configured: true, ok: true } } }
 *
 * This module now handles both formats for backward compatibility, but will be removed in a future version.
 */

import { getBackendSrv } from "@grafana/runtime";
import { LLM_PLUGIN_ROUTE } from "./constants";

// Re-export everything from llm.ts except enabled
export * from "./llm";

// Override enabled function to handle both old and new formats
export const enabled = async (): Promise<boolean> => {
  try {
    const settings = await getBackendSrv().get(`${LLM_PLUGIN_ROUTE}/settings`);
    if (!settings.enabled) {
      return false;
    }

    const health = await getBackendSrv().get(`${LLM_PLUGIN_ROUTE}/health`);
    const details = health.details;

    // Handle both new and old formats
    if (details.llmProvider) {
      return details.llmProvider.configured && details.llmProvider.ok;
    }
    if (details.openAI) {
      return details.openAI.configured && details.openAI.ok;
    }
    return false;
  } catch (e) {
    return false;
  }
};
