import React from 'react';
import { render, screen } from '@testing-library/react';
import { PluginType } from '@grafana/data';
import { AppConfig, AppConfigProps, AppPluginSettings, initialJsonData } from './AppConfig';
import { testIds } from 'components/testIds';

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

  test('OpenAI provider: renders the "Provider Settings" fieldset with API key, API url inputs and button', () => {
    const plugin = { meta: { ...props.plugin.meta, enabled: false } };

    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.queryByRole('group', { name: /provider settings/i })).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.openAIKey)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.openAIOrganizationID)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.openAIUrl)).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: /vector settings/i })).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.model)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.qdrantSecure)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.qdrantAddress)).toBeInTheDocument();
    // Don't expect to see the Grafana vector API field when type is qdrant
    expect(screen.queryByTestId(testIds.appConfig.grafanaVectorApiUrl)).toBeNull();
    expect(screen.queryByRole('button', { name: /save & test/i })).toBeInTheDocument();
  });

  test('Pulze provider: renders the "API Settings" fieldset with API key, API url inputs and button', () => {
    const jsonData = props.plugin.meta.jsonData!;
    jsonData.provider = {
      name: 'pulze',
    };
    const plugin = { meta: { ...props.plugin.meta, enabled: false } };

    console.log('123', JSON.stringify(plugin, null, '  '));
    // @ts-ignore - We don't need to provide `addConfigPage()` and `setChannelSupport()` for these tests
    render(<AppConfig plugin={plugin} query={props.query} />);

    expect(screen.queryByRole('group', { name: /provider settings/i })).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.openAIKey)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.openAIUrl)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.pulzeModel)).toBeInTheDocument();
    expect(screen.queryByRole('group', { name: /vector settings/i })).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.model)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.qdrantSecure)).toBeInTheDocument();
    expect(screen.queryByTestId(testIds.appConfig.qdrantAddress)).toBeInTheDocument();
    // Don't expect to see the Grafana vector API field when type is qdrant
    expect(screen.queryByTestId(testIds.appConfig.grafanaVectorApiUrl)).toBeNull();
    expect(screen.queryByRole('button', { name: /save & test/i })).toBeInTheDocument();
  });
});

describe('Test initialJsonData', () => {
  const cases: [any, AppPluginSettings][] = [
    // old settings -> converts to new settings
    [
      {
        openAI: {
          provider: 'openai',
        },
        vector: {
          enabled: true,
          store: {
            type: 'qdrant',
          },
        },
      },
      {
        provider: { name: 'openai' },
        vector: { enabled: true, store: { type: 'qdrant' } },
      },
    ],
    // new settings -> nothing changes
    [
      {
        provider: {
          name: 'openai',
        },
        vector: {
          enabled: true,
          store: {
            type: 'qdrant',
          },
        },
      },
      {
        provider: { name: 'openai' },
        vector: { enabled: true, store: { type: 'qdrant' } },
      },
    ],
    // old + new settings -> new settings win
    [
      {
        openai: {
          provider: 'openai',
        },
        provider: {
          name: 'pulze',
        },
        vector: {
          enabled: true,
          store: {
            type: 'qdrant',
          },
        },
      },
      {
        provider: { name: 'pulze' },
        vector: { enabled: true, store: { type: 'qdrant' } },
      },
    ],
  ];

  test.each(cases)('initialJsonData(%s) should return (%s)', (settings, expected) => {
    expect(initialJsonData(settings)).toEqual(expected);
  });
});
