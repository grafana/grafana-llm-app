import { PluginType } from '@grafana/data';
import { render, screen } from '@testing-library/react';
import { testIds } from 'components/testIds';
import React from 'react';
import { AppConfig, AppConfigProps } from './AppConfig';

describe('Components/AppConfig', () => {
  let props: AppConfigProps;

  beforeEach(() => {
    jest.resetAllMocks();

    props = {
      plugin: {
        meta: {
          id: 'sample-app',
          name: 'Sample App',
          type: PluginType.app,
          enabled: true,
          jsonData: {
            displayVectorStoreOptions: true,
            vector: {
              enabled: true,
              store: {
                type: 'qdrant',
              },
            },
          },
        },
      },
      query: {},
    } as unknown as AppConfigProps;
  });

  test('renders the "API Settings" fieldset with API key, API url inputs and button', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.queryByRole('group', { name: /openai settings/i })).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.provider)).toBeInTheDocument();
    // expect(screen.queryByTestId(testIds.appConfig.openAIKey)).toBeInTheDocument();
    // expect(screen.queryByTestId(testIds.appConfig.openAIOrganizationID)).toBeInTheDocument();
    // expect(screen.queryByTestId(testIds.appConfig.openAIUrl)).toBeInTheDocument();
    // expect(screen.queryByTestId(testIds.appConfig.model)).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: /vector settings/i })).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.qdrantSecure)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.qdrantAddress)).toBeInTheDocument();
    // Don't expect to see the Grafana vector API field when type is qdrant
    expect(screen.queryByTestId(testIds.appConfig.grafanaVectorApiUrl)).toBeNull();
    expect(screen.queryByRole('button', { name: /save & test/i })).toBeInTheDocument();
  });
});
