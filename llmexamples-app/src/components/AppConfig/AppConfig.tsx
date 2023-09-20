import React, { ChangeEvent, useState } from 'react';
import { lastValueFrom } from 'rxjs';
import { css } from '@emotion/css';
import { AppPluginMeta, GrafanaTheme2, PluginConfigPageProps, PluginMeta } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { Button, Field, FieldSet, SecretInput, useStyles2 } from '@grafana/ui';
import { testIds } from '../testIds';

type State = {
  // Tells us if the Grafana API key secret is set.
  isGrafanaAPIKeySet: boolean;
  // A Grafana API key.
  grafanaAPIKey: string;
};

interface AppPluginSettings { }

export interface AppConfigProps extends PluginConfigPageProps<AppPluginMeta<AppPluginSettings>> { }

export const AppConfig = ({ plugin }: AppConfigProps) => {
  const s = useStyles2(getStyles);
  const { enabled, pinned, secureJsonFields } = plugin.meta;
  const [state, setState] = useState<State>({
    grafanaAPIKey: '',
    isGrafanaAPIKeySet: Boolean(secureJsonFields?.grafanaApiKey),
  });

  const onResetApiKey = () =>
    setState({
      ...state,
      grafanaAPIKey: '',
      isGrafanaAPIKeySet: false,
    });

  const onChange = (event: ChangeEvent<HTMLInputElement>) => {
    setState({
      ...state,
      [event.target.name]: event.target.value.trim(),
    });
  };

  return (
    <div data-testid={testIds.appConfig.container}>
      <FieldSet label="Settings">
        <Field label="Grafana API Key" description="A Grafana API Key">
          <SecretInput
            width={60}
            data-testid={testIds.appConfig.apiKey}
            name="grafanaAPIKey"
            value={state.grafanaAPIKey}
            isConfigured={state.isGrafanaAPIKeySet}
            onChange={onChange}
            onReset={onResetApiKey}
          />
        </Field>

        <div className={s.marginTop}>
          <Button
            type="submit"
            data-testid={testIds.appConfig.submit}
            onClick={() =>
              updatePluginAndReload(plugin.meta.id, {
                enabled,
                pinned,
                // This cannot be queried later by the frontend.
                // We don't want to override it in case it was set previously and left untouched now.
                secureJsonData: state.isGrafanaAPIKeySet
                  ? undefined
                  : {
                    grafanaApiKey: state.grafanaAPIKey,
                  },
              })
            }
            disabled={Boolean(
              (!state.isGrafanaAPIKeySet && !state.grafanaAPIKey)
            )}
          >
            Save API settings
          </Button>
        </div>
      </FieldSet>
    </div>
  );
};

const getStyles = (theme: GrafanaTheme2) => ({
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
