import React, { useEffect, useState } from 'react';
import { lastValueFrom } from 'rxjs';

import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, KeyValue, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { FetchResponse, HealthCheckResult, getBackendSrv } from '@grafana/runtime';
import { Alert, Button, LoadingPlaceholder, useStyles2 } from '@grafana/ui';

import { testIds } from '../testIds';
import { ShowHealthCheckResult } from './HealthCheck';
import { LLMConfig } from './LLMConfig';
import { OpenAISettings } from './OpenAI';
import { VectorConfig, VectorSettings } from './Vector';

///////////////////////

export interface AppPluginSettings {
  openAI?: OpenAISettings;
  vector?: VectorSettings;
  // The enableGrafanaManagedLLM flag will enable the plugin to use Grafana-managed OpenAI
  // This will only work for Grafana Cloud install plugins
  enableGrafanaManagedLLM?: boolean;
}

export type Secrets = {
  openAIKey?: string;
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
    vectorEmbedderBasicAuthPassword: secureJsonFields.vectorEmbedderBasicAuthPassword ?? false,
    vectorStoreBasicAuthPassword: secureJsonFields.vectorStoreBasicAuthPassword ?? false,
    qdrantApiKey: secureJsonFields.qdrantApiKey ?? false,
  };
}

export interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta<AppPluginSettings>> {}

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
    if (settings?.openAI?.provider === 'grafana' && !managedLLMOptIn) {
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

  useEffect(() => {
    // clear health check status if any setting changed
    if (updated) {
      setHealthCheck(undefined);
    }
  }, [updated]);

  const doSave = async () => {
    if (errorState !== undefined) {
      return;
    }
    setIsUpdating(true);
    setHealthCheck(undefined);
    let key: keyof SecretsSet;
    const secureJsonData: Secrets = {};
    for (key in configuredSecrets) {
      // Only include secrets that are not already configured on the backend,
      // otherwise we'll overwrite them.
      if (!configuredSecrets[key]) {
        secureJsonData[key] = newSecrets[key];
      }
    }
    try {
      await updateAndSavePluginSettings(plugin.meta.id, settings.enableGrafanaManagedLLM, {
        enabled,
        pinned,
        jsonData: settings,
        secureJsonData,
      });
    } catch (e) {
      setIsUpdating(false);
      throw e;
    }

    // Note: health-check uses the state saved in the plugin settings.
    let healthCheckResult: HealthCheckResult | undefined = undefined;
    if (settings.openAI?.provider !== undefined) {
      const result = await checkPluginHealth(plugin.meta.id);
      healthCheckResult = result.data;
    }
    setHealthCheck(healthCheckResult);

    // If moving away from Grafana-managed LLM, opt-out of the feature automatically
    if (managedLLMOptIn && settings.openAI?.provider !== 'grafana') {
      await saveLLMOptInState(false);
    } else {
      await saveLLMOptInState(managedLLMOptIn);
    }

    // Update the frontend settings explicitly, it is otherwise not updated until page reload
    plugin.meta.jsonData = settings;

    setIsUpdating(false);
    setUpdated(false);
  };

  return (
    <div data-testid={testIds.appConfig.container}>
      <LLMConfig
        settings={settings}
        onChange={(newSettings: AppPluginSettings) => {
          setSettings(newSettings);
          setUpdated(true);
        }}
        secrets={newSecrets}
        secretsSet={configuredSecrets}
        optIn={managedLLMOptIn}
        setOptIn={(optIn) => {
          setManagedLLMOptIn(optIn);
          setUpdated(true);
        }}
        onChangeSecrets={(secrets: Secrets) => {
          // Update the new secrets.
          setNewSecrets(secrets);
          // Mark each secret as not configured. This will cause it to be included
          // in the request body when the user clicks "Save settings".
          for (const key of Object.keys(secrets)) {
            setConfiguredSecrets({ ...configuredSecrets, [key]: false });
          }
          setUpdated(true);
        }}
      />

      <VectorConfig
        settings={settings.vector}
        secrets={newSecrets}
        secretsSet={configuredSecrets}
        onChange={(vector) => {
          setSettings({ ...settings, vector });
          setUpdated(true);
        }}
        onChangeSecrets={(secrets) => {
          // Update the new secrets.
          setNewSecrets(secrets);
          // Mark each secret as not configured. This will cause it to be included
          // in the request body when the user clicks "Save settings".
          for (const key of Object.keys(secrets)) {
            setConfiguredSecrets({ ...configuredSecrets, [key]: false });
          }
          setUpdated(true);
        }}
      />

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
