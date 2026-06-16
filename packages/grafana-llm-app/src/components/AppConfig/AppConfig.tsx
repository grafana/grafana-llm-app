import React, { useEffect, useState } from 'react';
import { lastValueFrom } from 'rxjs';

import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, KeyValue, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { FetchResponse, HealthCheckResult, getBackendSrv } from '@grafana/runtime';
import { Alert, Button, LoadingPlaceholder, useStyles2 } from '@grafana/ui';

import { testIds } from '../testIds';
import { ModelSettings } from './ModelConfig';
import { ShowHealthCheckResult } from './HealthCheck';
import { LLMConfig } from './LLMConfig';
import { OpenAISettings } from './OpenAI';
import { AnthropicSettings } from './AnthropicConfig';
import { VectorConfig, VectorSettings } from './Vector';
///////////////////////

export type ProviderType = 'openai' | 'azure' | 'grafana' | 'test' | 'custom' | 'anthropic';

export interface AppPluginSettings {
  provider?: ProviderType;
  disabled?: boolean;
  openAI?: OpenAISettings;
  anthropic?: AnthropicSettings;
  vector?: VectorSettings;
  models?: ModelSettings;
  // The enableGrafanaManagedLLM flag will enable the plugin to use Grafana-managed OpenAI
  // This will only work for Grafana Cloud install plugins
  enableGrafanaManagedLLM?: boolean;
  displayVectorStoreOptions?: boolean;
  enableDevSandbox?: boolean;
}

export type Secrets = {
  openAIKey?: string;
  anthropicKey?: string;
  qdrantApiKey?: string;
  vectorStoreBasicAuthPassword?: string;
  vectorEmbedderBasicAuthPassword?: string;
};

export type SecretsSet = {
  [Property in keyof Secrets]: boolean;
};

function initialSecrets(secureJsonFields: KeyValue<boolean>): SecretsSet {
  return {
    openAIKey: secureJsonFields.openAIKey ?? false,
    anthropicKey: secureJsonFields.anthropicKey ?? false,
    vectorEmbedderBasicAuthPassword: secureJsonFields.vectorEmbedderBasicAuthPassword ?? false,
    vectorStoreBasicAuthPassword: secureJsonFields.vectorStoreBasicAuthPassword ?? false,
    qdrantApiKey: secureJsonFields.qdrantApiKey ?? false,
  };
}

export interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta<AppPluginSettings>> {}

// Helper function to get the effective provider, handling both legacy and new provider fields
export function getEffectiveProvider(settings: AppPluginSettings): ProviderType | undefined {
  // If the root provider is set, use it (new format)
  if (settings.provider) {
    return settings.provider;
  }
  // Otherwise fall back to the legacy openAI.provider field
  return settings.openAI?.provider;
}

export const AppConfig = ({ plugin }: AppConfigProps) => {
  const s = useStyles2(getStyles);
  const { enabled, pinned, jsonData, secureJsonFields } = plugin.meta;
  const [settings, setSettings] = useState<AppPluginSettings>(jsonData ?? {});
  const [newSecrets, setNewSecrets] = useState<Secrets>({});
  // Whether each secret is already configured in the plugin backend.
  const [configuredSecrets, setConfiguredSecrets] = useState<SecretsSet>(initialSecrets(secureJsonFields ?? {}));
  // Whether any settings have been updated.
  const [updated, setUpdated] = useState(false);

  const [managedLLMOptIn, setManagedLLMOptIn] = useState<boolean>(false);

  const [isUpdating, setIsUpdating] = useState(false);
  const [healthCheck, setHealthCheck] = useState<HealthCheckResult | undefined>(undefined);

  const validateInputs = (): string | undefined => {
    // Check if Grafana-provided OpenAI enabled, that it has been opted-in
    if (settings?.provider === 'grafana' && !managedLLMOptIn) {
      return 'You must click the "I Accept" checkbox to use OpenAI provided by Grafana';
    }
    return;
  };
  const errorState = validateInputs();

  useEffect(() => {
    const fetchData = async () => {
      const optIn = await getLLMOptInState();
      setManagedLLMOptIn(optIn);
    };

    if (settings.enableGrafanaManagedLLM === true) {
      fetchData();
    }
  }, [settings.enableGrafanaManagedLLM]);

  // Helper function to mark settings as updated and clear health check
  const markAsUpdated = () => {
    setUpdated(true);
    setHealthCheck(undefined);
  };

  const doSave = async () => {
    if (errorState !== undefined) {
      return;
    }
    setIsUpdating(true);
    setHealthCheck(undefined);

    const originalSettings = { ...plugin.meta.jsonData };

    let key: keyof SecretsSet;
    const secureJsonData: Secrets = {};
    for (key in configuredSecrets) {
      // Only include secrets that are not already configured on the backend,
      // otherwise we'll overwrite them.
      if (!configuredSecrets[key]) {
        secureJsonData[key] = newSecrets[key];
      }
    }

    // Migrate the provider to the new format before saving
    const settingsToSave = {
      ...settings,
      provider: getEffectiveProvider(settings),
    };

    try {
      await updateAndSavePluginSettings(plugin.meta.id, settings.enableGrafanaManagedLLM, {
        enabled,
        pinned,
        jsonData: settingsToSave,
        secureJsonData,
      });

      // Note: health-check uses the state saved in the plugin settings.
      let healthCheckResult: HealthCheckResult | undefined = undefined;
      const effectiveProvider = getEffectiveProvider(settings);
      if (effectiveProvider !== undefined) {
        const result = await checkPluginHealth(plugin.meta.id);
        healthCheckResult = result.data;

        if (healthCheckResult.status?.toLowerCase() !== 'ok') {
          await updateAndSavePluginSettings(plugin.meta.id, settings.enableGrafanaManagedLLM, {
            enabled,
            pinned,
            jsonData: originalSettings,
            secureJsonData: {},
          });
          // Revert UI to original settings to match backend state
          setSettings(originalSettings);
          setHealthCheck(healthCheckResult);
          setIsUpdating(false);
          return;
        }
      }
      setHealthCheck(healthCheckResult);

      // If moving away from Grafana-managed LLM, opt-out of the feature automatically
      // This logic should only be triggered if the Grafana-managed LLM feature is enabled (Grafana Cloud Only)
      if (settings.enableGrafanaManagedLLM === true) {
        if (managedLLMOptIn && effectiveProvider !== 'grafana') {
          await saveLLMOptInState(false);
        } else {
          await saveLLMOptInState(managedLLMOptIn);
        }
      }

      // Update local state to immediately reflect saved settings in the UI
      setSettings(settingsToSave);

      setIsUpdating(false);
      setUpdated(false);
    } catch (e) {
      // Rollback to original settings on any error
      try {
        await updateAndSavePluginSettings(plugin.meta.id, settings.enableGrafanaManagedLLM, {
          enabled,
          pinned,
          jsonData: originalSettings,
          secureJsonData: {},
        });
        // Revert UI to original settings to match backend state
        setSettings(originalSettings);
      } catch (rollbackError) {
        console.error('Failed to rollback settings:', rollbackError);
      }
      setIsUpdating(false);
      throw e;
    }
  };

  return (
    <div data-testid={testIds.appConfig.container}>
      <LLMConfig
        settings={settings}
        onChange={(newSettings: AppPluginSettings) => {
          setSettings(newSettings);
          markAsUpdated();
        }}
        secrets={newSecrets}
        secretsSet={configuredSecrets}
        optIn={managedLLMOptIn}
        setOptIn={(optIn) => {
          setManagedLLMOptIn(optIn);
          markAsUpdated();
        }}
        onChangeSecrets={(secrets: Secrets) => {
          // Update the new secrets.
          setNewSecrets(secrets);
          // Mark each secret as not configured. This will cause it to be included
          // in the request body when the user clicks "Save settings".
          for (const key of Object.keys(secrets)) {
            setConfiguredSecrets({ ...configuredSecrets, [key]: false });
          }
          markAsUpdated();
        }}
      />
      {settings.displayVectorStoreOptions === true && (
        <VectorConfig
          settings={settings.vector}
          secrets={newSecrets}
          secretsSet={configuredSecrets}
          onChange={(vector) => {
            setSettings({ ...settings, vector });
            markAsUpdated();
          }}
          onChangeSecrets={(secrets) => {
            // Update the new secrets.
            setNewSecrets(secrets);
            // Mark each secret as not configured. This will cause it to be included
            // in the request body when the user clicks "Save settings".
            for (const key of Object.keys(secrets)) {
              setConfiguredSecrets({ ...configuredSecrets, [key]: false });
            }
            markAsUpdated();
          }}
        />
      )}

      {errorState !== undefined && <Alert title={errorState} severity="error" />}
      {isUpdating ? (
        <LoadingPlaceholder text="Running health check..." />
      ) : (
        healthCheck && <ShowHealthCheckResult {...healthCheck} />
      )}
      <div className={s.marginTop}>
        <Button
          type="submit"
          data-testid={testIds.appConfig.submit}
          onClick={doSave}
          disabled={!updated || isUpdating || errorState !== undefined}
        >
          Save &amp; test
        </Button>
      </div>
    </div>
  );
};

export const getStyles = (theme: GrafanaTheme2) => ({
  colorWeak: css`
    color: ${theme.colors.text.secondary};
  `,
  marginTop: css`
    margin-top: ${theme.spacing(3)};
  `,
  inlineFieldWidth: 15,
  inlineFieldInputWidth: 40,
});

export const updateGrafanaPluginSettings = (pluginId: string, data: Partial<PluginMeta>) => {
  const response = getBackendSrv().fetch({
    url: `/api/plugins/${pluginId}/settings`,
    method: 'POST',
    data,
  });

  return lastValueFrom(response);
};

export const updateGcomProvisionedPluginSettings = (data: Partial<PluginMeta>) => {
  const response = getBackendSrv().fetch({
    url: `/api/plugins/grafana-llm-app/resources/save-plugin-settings`,
    method: 'POST',
    data,
  });

  return lastValueFrom(response);
};

export const updateAndSavePluginSettings = async (
  pluginId: string,
  persistToGcom = false,
  data: Partial<PluginMeta>
) => {
  const gcomPluginData = {
    jsonData: data.jsonData,
    secureJsonData: data.secureJsonData,
  };

  if (persistToGcom === true) {
    await updateGcomProvisionedPluginSettings(gcomPluginData).then((response: FetchResponse) => {
      if (!response.ok) {
        throw response.data;
      }
    });
  }
  await updateGrafanaPluginSettings(pluginId, data).then((response: FetchResponse) => {
    if (!response.ok) {
      throw response.data;
    }
  });
};

const checkPluginHealth = (pluginId: string): Promise<FetchResponse<HealthCheckResult>> => {
  const response = getBackendSrv().fetch({
    url: `/api/plugins/${pluginId}/health`,
  });
  return lastValueFrom(response) as Promise<FetchResponse<HealthCheckResult>>;
};

export async function saveLLMOptInState(optIn: boolean): Promise<void> {
  return lastValueFrom(
    getBackendSrv().fetch({
      url: `api/plugins/grafana-llm-app/resources/grafana-llm-state`,
      method: 'POST',
      data: { allowed: optIn },
    })
  ).then((response: FetchResponse) => {
    if (!response.ok) {
      throw response.data;
    }
  });
}

export async function getLLMOptInState(): Promise<boolean> {
  return lastValueFrom(
    getBackendSrv().fetch({
      url: `api/plugins/grafana-llm-app/resources/grafana-llm-state`,
      method: 'GET',
    })
  ).then((response: FetchResponse) => {
    if (!response.ok || response.data?.status !== 'success') {
      throw response.data;
    }
    return response.data.data?.allowed ?? false;
  });
}
