import React, { useState } from 'react';
import { lastValueFrom } from 'rxjs';

import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, KeyValue, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { FetchResponse, getBackendSrv, HealthCheckResult } from '@grafana/runtime';
import { Button, LoadingPlaceholder, useStyles2 } from '@grafana/ui';

import { testIds } from '../testIds';
import { OpenAIConfig, OpenAISettings } from './OpenAI';
import { VectorConfig, VectorSettings } from './Vector';
import { ShowHealthCheckResult } from './HealthCheck';

export interface AppPluginSettings {
  openAI?: OpenAISettings;
  vector?: VectorSettings;
};

export type Secrets = {
  openAIKey?: string;
}

export type SecretsSet = {
  [Property in keyof Secrets]: boolean;
}

function initialSecrets(secureJsonFields: KeyValue<boolean>): SecretsSet {
  return {
    openAIKey: secureJsonFields.openAIKey ?? false,
  };
}

export interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta<AppPluginSettings>> { }

export const AppConfig = ({ plugin }: AppConfigProps) => {
  const s = useStyles2(getStyles);
  const { enabled, pinned, jsonData, secureJsonFields } = plugin.meta;
  const [settings, setSettings] = useState<AppPluginSettings>(jsonData ?? {});
  const [newSecrets, setNewSecrets] = useState<Secrets>({});
  // Whether each secret is already configured in the plugin backend.
  const [configuredSecrets, setConfiguredSecrets] = useState<SecretsSet>(initialSecrets(secureJsonFields ?? {}));
  // Whether any settings have been updated.
  const [updated, setUpdated] = useState(false);

  const [isUpdating, setIsUpdating] = useState(false);
  const [healthCheck, setHealthCheck] = useState<HealthCheckResult | undefined>(undefined);

  return (
    <div data-testid={testIds.appConfig.container}>

      <OpenAIConfig
        settings={settings.openAI ?? {}}
        onChange={openAI => {
          setSettings({ ...settings, openAI })
          setUpdated(true);
        }}
        secrets={newSecrets}
        secretsSet={configuredSecrets}
        onChangeSecrets={secrets => {
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
        onChange={(vector) => {
          setSettings({ ...settings, vector });
          setUpdated(true);
        }}
      />

      {isUpdating ? <LoadingPlaceholder text="Running health check..." /> : healthCheck && (
        <ShowHealthCheckResult {...healthCheck} />
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
  return lastValueFrom(response) as Promise<FetchResponse<HealthCheckResult>>
}
