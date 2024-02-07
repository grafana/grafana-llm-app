import React, { useState } from 'react';
import { lastValueFrom } from 'rxjs';

import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, KeyValue, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { FetchResponse, getBackendSrv, HealthCheckResult } from '@grafana/runtime';
import { Button, LoadingPlaceholder, useStyles2 } from '@grafana/ui';

import { testIds } from '../testIds';
import { ProviderConfig, ProviderSettings } from './Provider';
import { VectorConfig, VectorSettings } from './Vector';
import { ShowHealthCheckResult } from './HealthCheck';

export interface AppPluginSettings {
  provider?: ProviderSettings;
  vector?: VectorSettings;
}

export type Secrets = {
  providerKey?: string;
  vectorStoreBasicAuthPassword?: string;
  vectorEmbedderBasicAuthPassword?: string;
};

export type SecretsSet = {
  [Property in keyof Secrets]: boolean;
};

function initialSecrets(secureJsonFields: KeyValue<boolean>): SecretsSet {
  return {
    providerKey: secureJsonFields.providerKey ?? secureJsonFields.openAIKey ?? false,
    vectorEmbedderBasicAuthPassword: secureJsonFields.vectorEmbedderBasicAuthPassword ?? false,
    vectorStoreBasicAuthPassword: secureJsonFields.vectorStoreBasicAuthPassword ?? false,
  };
}

export interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta<AppPluginSettings>> {}

// migration layer to convert old to new structure
export const initialJsonData = (jsonData: KeyValue<any>): AppPluginSettings => {
  // destructure old fields and rename 'provider' to 'name'
  const {provider: name, ...oldSettings} = jsonData.openAI ?? { provider: "openai" }
    // only use values from openAI if no provider configured
  return {
    provider: jsonData.provider ?? {...oldSettings, name},
    vector: jsonData.vector,
  };
};

export const AppConfig = ({ plugin }: AppConfigProps) => {
  const s = useStyles2(getStyles);
  const { enabled, pinned, jsonData, secureJsonFields } = plugin.meta;
  const [settings, setSettings] = useState<AppPluginSettings>(initialJsonData(jsonData ?? {}));
  console.log(settings);
  const [newSecrets, setNewSecrets] = useState<Secrets>({});
  // Whether each secret is already configured in the plugin backend.
  const [configuredSecrets, setConfiguredSecrets] = useState<SecretsSet>(initialSecrets(secureJsonFields ?? {}));
  // Whether any settings have been updated.
  const [updated, setUpdated] = useState(false);

  const [isUpdating, setIsUpdating] = useState(false);
  const [healthCheck, setHealthCheck] = useState<HealthCheckResult | undefined>(undefined);

  return (
    <div data-testid={testIds.appConfig.container}>
      <ProviderConfig
        settings={settings.provider ?? {}}
        onChange={(provider) => {
          setSettings({ ...settings, provider });
          setUpdated(true);
        }}
        secrets={newSecrets}
        secretsSet={configuredSecrets}
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

      <VectorConfig
        settings={settings}
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

      {isUpdating ? (
        <LoadingPlaceholder text="Running health check..." />
      ) : (
        healthCheck && <ShowHealthCheckResult {...healthCheck} />
      )}
      <div className={s.marginTop}>
        <Button
          type="submit"
          data-testid={testIds.appConfig.submit}
          onClick={async () => {
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
            const result = await checkPluginHealth(plugin.meta.id);
            setIsUpdating(false);
            setHealthCheck(result.data);
          }}
          disabled={!updated || isUpdating}
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
