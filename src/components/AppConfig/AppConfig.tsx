import React, { useState } from 'react';
import { lastValueFrom } from 'rxjs';

import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, KeyValue, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { Button, useStyles2 } from '@grafana/ui';

import { testIds } from '../testIds';
import { OpenAIConfig, OpenAISettings } from './OpenAI';
import { VectorConfig, VectorSettings } from './Vector';

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

      <div className={s.marginTop}>
        <Button
          type="submit"
          data-testid={testIds.appConfig.submit}
          onClick={() => {
            let key: keyof SecretsSet;
            const secureJsonData: Secrets = {};
            for (key in configuredSecrets) {
              // Only include secrets that are not already configured on the backend,
              // otherwise we'll overwrite them.
              if (!configuredSecrets[key]) {
                secureJsonData[key] = newSecrets[key];
              }
            }
            updatePluginAndReload(plugin.meta.id, {
              enabled,
              pinned,
              jsonData: settings,
              secureJsonData,
            })
          }}
          disabled={!updated}
        >
          Save settings
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

const updatePluginAndReload = async (pluginId: string, data: Partial<PluginMeta<AppPluginSettings>>) => {
  try {
    await updatePlugin(pluginId, data);

    // Reloading the page as the changes made here wouldn't be propagated to the actual plugin otherwise.
    // This is not ideal, however unfortunately currently there is no supported way for updating the plugin state.
    window.location.reload();
  } catch (e) {
    console.error('Error while updating the plugin', e);
  }
};

export const updatePlugin = async (pluginId: string, data: Partial<PluginMeta>) => {
  const response = getBackendSrv().fetch({
    url: `/api/plugins/${pluginId}/settings`,
    method: 'POST',
    data,
  });

  return lastValueFrom(response);
};
