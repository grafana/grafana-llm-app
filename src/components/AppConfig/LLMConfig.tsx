import { css } from '@emotion/css';
import React, { useState } from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { Button, Card, Checkbox, FieldSet, Icon, useStyles2 } from '@grafana/ui';

import { AppPluginSettings, Secrets, SecretsSet } from './AppConfig';
import { OpenAIConfig, OpenAIProvider } from './OpenAI';
import { OpenAILogo } from './OpenAILogo';

// LLMOptions are the 3 possible UI options for LLMs (grafana-provided cloud-only).
export type LLMOptions = 'grafana-provided' | 'openai' | 'disabled';

// This maps the current settings to decide what UI selection (LLMOptions) to show
function getLLMOptionFromSettings(settings: AppPluginSettings): LLMOptions {
  if (
    settings.openAI?.provider === 'azure' ||
    settings.openAI?.provider === 'openai' ||
    settings.openAI?.provider === 'pulze'
  ) {
    return 'openai';
  } else if (settings.openAI?.provider === 'grafana') {
    return 'grafana-provided';
  } else {
    return 'disabled';
  }
}

export function LLMConfig({
  settings,
  secrets,
  secretsSet,
  optIn,
  setOptIn,
  onChange,
  onChangeSecrets,
}: {
  settings: AppPluginSettings;
  onChange: (settings: AppPluginSettings) => void;
  secrets: Secrets;
  secretsSet: SecretsSet;
  optIn: boolean;
  setOptIn: (optIn: boolean) => void;
  onChangeSecrets: (secrets: Secrets) => void;
}) {
  const s = useStyles2(getStyles);
  // should only be relevant for Grafana Cloud
  const allowGrafanaManagedLLM = settings.enableGrafanaManagedLLM === true;

  // llmOption is the currently chosen LLM option in the UI
  const [llmOption, setLLMOption] = useState<LLMOptions>(getLLMOptionFromSettings(settings));
  // previousOpenAIProvider caches the value of the openAI provider, as it is overwritten by the grafana option
  const [previousOpenAIProvider, setPreviousOpenAIProvider] = useState<OpenAIProvider>();

  const optInChange = () => {
    setOptIn(!optIn);
  };

  // Handlers for when different LLM options are chosen in the UI
  const selectLLMDisabled = () => {
    if (llmOption !== 'disabled') {
      // Cache if OpenAI or Azure provider is used, so can restore
      if (previousOpenAIProvider === undefined) {
        setPreviousOpenAIProvider(settings.openAI?.provider);
      }

      onChange({ ...settings, openAI: { provider: undefined } });
      setLLMOption('disabled');
    }
  };

  const selectGrafanaManaged = (e: React.MouseEvent<HTMLElement, MouseEvent>) => {
    if (llmOption !== 'grafana-provided') {
      // Cache if OpenAI or Azure provider is used, so can restore
      if (previousOpenAIProvider === undefined) {
        setPreviousOpenAIProvider(settings.openAI?.provider);
      }

      onChange({ ...settings, openAI: { provider: 'grafana' } });
      setLLMOption('grafana-provided');
    }
  };

  const selectOpenAI = () => {
    if (llmOption !== 'openai') {
      // Restore the provider (OpenAI or Azure) & clear the cache
      onChange({ ...settings, openAI: { provider: previousOpenAIProvider } });
      setPreviousOpenAIProvider(undefined);

      setLLMOption('openai');
    }
  };

  return (
    <FieldSet label="OpenAI Settings" className={s.sidePadding}>
      {allowGrafanaManagedLLM && (
        <Card
          isSelected={llmOption === 'grafana-provided'}
          onClick={selectGrafanaManaged}
          className={s.cardWithoutBottomMargin}
        >
          <Card.Heading>Use OpenAI provided by Grafana</Card.Heading>
          <Card.Description>
            <p>Enable LLM features in Grafana by using a connection to OpenAI that is provided by Grafana</p>
            {llmOption === 'grafana-provided' && (
              <>
                <div className={s.openaiTermsBox}>
                  <ul>
                    <li>Grafana uses OpenAI’s API platform to provide LLM functionality.</li>
                    <li>
                      OpenAI does not train models on inputs or outputs of their API platform. OpenAI does retain data
                      for a short time to provide the services and monitor for abuse. All data is encrypted in transit
                      and at rest.
                    </li>
                    <li>
                      Visit the OpenAI trust portal for more details:{' '}
                      <Button
                        size="sm"
                        variant="secondary"
                        onClick={(e) => window.open('https://trust.openai.com/', '_blank')}
                      >
                        https://trust.openai.com/
                      </Button>
                    </li>
                    <li>
                      AI features are clearly marked in Grafana, and each feature sends minimal data to OpenAI, and only
                      on user request (for example, when someone clicks the button to request an Incident auto-summary).
                    </li>
                    <li>
                      By enabling this integration, I accept that Grafana shares limited data to the OpenAI API as
                      needed to provide LLM-powered features.
                    </li>
                  </ul>
                </div>
                <Checkbox
                  value={optIn}
                  onClick={optInChange}
                  label="I accept limited data sharing with OpenAI as described above"
                />
              </>
            )}
          </Card.Description>
          <Card.Figure>
            <Icon name="grafana" size="lg" />
          </Card.Figure>
        </Card>
      )}

      <Card isSelected={llmOption === 'openai'} onClick={selectOpenAI} className={s.cardWithoutBottomMargin}>
        <Card.Heading>Use your own OpenAI account</Card.Heading>
        <Card.Description>
          <p>Enable LLM features in Grafana using your own OpenAI account</p>
          {llmOption === 'openai' && (
            <OpenAIConfig
              settings={settings.openAI ?? {}}
              onChange={(openAI) => onChange({ ...settings, openAI })}
              secrets={secrets}
              secretsSet={secretsSet}
              onChangeSecrets={onChangeSecrets}
            />
          )}
        </Card.Description>
        <Card.Figure>
          <OpenAILogo width={20} height={20} />
        </Card.Figure>
      </Card>

      <Card isSelected={llmOption === 'disabled'} onClick={selectLLMDisabled} className={s.cardWithoutBottomMargin}>
        <Card.Heading>Disable all LLM features in Grafana</Card.Heading>
        <Card.Description>&nbsp;</Card.Description>
        <Card.Figure>
          <Icon name="times" size="lg" />
        </Card.Figure>
      </Card>
    </FieldSet>
  );
}

export const getStyles = (theme: GrafanaTheme2) => ({
  sidePadding: css`
    margin-left: ${theme.spacing(1)};
    margin-right: ${theme.spacing(1)};
  `,
  nestedList: css`
    margin-left: ${theme.spacing(3)};
  `,
  openaiTermsBox: css`
    overflow-y: auto;
    z-index: 2;
    margin-right: ${theme.spacing(3)};
    padding: ${theme.spacing(1)} ${theme.spacing(1)} ${theme.spacing(1)} ${theme.spacing(3)};
    border: 1px solid ${theme.colors.border.medium};
    background: ${theme.colors.background.primary};
    color: ${theme.colors.text.primary};
  `,
  cardWithoutBottomMargin: css`
    margin-bottom: 0;
    margin-top: ${theme.spacing(1)};
  `,
  inlineFieldWidth: 15,
  inlineFieldInputWidth: 40,
});
