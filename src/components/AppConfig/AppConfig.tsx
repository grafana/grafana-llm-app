import React, { useState } from 'react';
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
export interface LLMGatewaySettings {
  // Opt-in to LLMGateway?
  isOptIn?: boolean;
  // URL for LLMGateway
  url?: string;
}

export interface AppPluginSettings {
  openAI?: OpenAISettings;
  vector?: VectorSettings;
  // The enableGrafanaManagedLLM flag will enable the plugin to use Grafana-managed OpenAI
  // This will only work for Grafana Cloud install plugins
  enableGrafanaManagedLLM?: boolean;
  // Config used for Grafana-managed LLM
  llmGateway?: LLMGatewaySettings;
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
  console.log(settings);
  const [newSecrets, setNewSecrets] = useState<Secrets>({});
  // Whether each secret is already configured in the plugin backend.
  const [configuredSecrets, setConfiguredSecrets] = useState<SecretsSet>(initialSecrets(secureJsonFields ?? {}));
  // Whether any settings have been updated.
  const [updated, setUpdated] = useState(false);
  const [optInUpdated, setOptInUpdated] = useState(false);

  const [isUpdating, setIsUpdating] = useState(false);
  const [healthCheck, setHealthCheck] = useState<HealthCheckResult | undefined>(undefined);

  const validateInputs = (): string | undefined => {
    // Check if Grafana-provided OpenAI enabled, that it has been opted-in
    if (settings?.openAI?.provider === 'grafana' && !settings?.llmGateway?.isOptIn) {
      return "You must click the 'Enable OpenAI access via Grafana' button to use OpenAI provided by Grafana";
    }
    return;
  };
  const errorState = validateInputs();

  const doSave = async () => {
    if (errorState !== undefined) {
      return;
    }
    // Push LLM opt-in state, will also check if the user is allowed to opt-in
    if (settings.enableGrafanaManagedLLM && optInUpdated) {
      const optInResult = await saveLLMOptInState(settings.llmGateway?.isOptIn as boolean);
      setOptInUpdated(false);
      if (!optInResult) {
        setIsUpdating(false);
        setUpdated(false);
        return;
      }
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
    await updatePlugin(plugin.meta.id, {
      enabled,
      pinned,
      jsonData: settings,
      secureJsonData,
    });
    // If disabling LLM features, no health check needed
    if (settings.openAI?.provider !== undefined) {
      const result = await checkPluginHealth(plugin.meta.id);
      setHealthCheck(result.data);
    }

    setIsUpdating(false);
    setUpdated(false);
  };

  return (
    <div data-testid={testIds.appConfig.container}>
      <LLMConfig
        settings={settings}
        onChange={(newSettings: AppPluginSettings) => {
          if (newSettings.llmGateway?.isOptIn !== settings.llmGateway?.isOptIn) {
            setOptInUpdated(true);
          }
          setSettings(newSettings);
          setUpdated(true);
        }}
        secrets={newSecrets}
        secretsSet={configuredSecrets}
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

export const updatePlugin = (pluginId: string, data: Partial<PluginMeta>) => {
  const response = getBackendSrv().fetch({
    url: `/api/plugins/${pluginId}/settings`,
    method: 'POST',
    data,
  });

  return lastValueFrom(response);
};

const checkPluginHealth = (pluginId: string): Promise<FetchResponse<HealthCheckResult>> => {
  const response = getBackendSrv().fetch({
    url: `/api/plugins/${pluginId}/health`,
  });
  return lastValueFrom(response) as Promise<FetchResponse<HealthCheckResult>>;
};

export async function saveLLMOptInState(optIn: boolean): Promise<boolean> {
  return lastValueFrom(
    getBackendSrv().fetch({
      url: `api/plugins/grafana-llm-app/resources/save-llm-state`,
      method: 'POST',
      data: { optIn },
    })
  )
    .then((response: FetchResponse) => {
      if (!response.ok) {
        console.error(`Error using Grafana-managed LLM: ${response.status} ${response.data.message}`);
        return false;
      }
      return true;
    })
    .catch((error) => {
      console.error(`Error using Grafana-managed LLM: ${error.status} ${error.data.message}`);
      return false;
    });
}
