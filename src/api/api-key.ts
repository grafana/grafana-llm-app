import { config } from '@grafana/runtime';

import { adminApiPost } from './core-api';
import { createApiKey } from './grafana-api';

const apiKeyName = 'grafana-llm-temporary-key';

// Create a temporary Admin API key
//
// This function returns a string containing the new key, which will expire in 2 minutes.
async function createTemporaryApiKey(): Promise<string | undefined> {
  const key = await createApiKey(apiKeyName, 'Admin', 120);

  return key?.key;
}

export async function saveLLMOptInState(optIn: boolean, optInChangedBy: string): Promise<void> {
  const key = await createTemporaryApiKey();

  // Request that the plugin backend saves the LLM state to GCom
  await adminApiPost('/save-llm-state', {
    data: {
      grafanaUrl: config.appUrl,
      apiKey: key,
      optIn,
      optInChangedBy,
    },
  });
}
