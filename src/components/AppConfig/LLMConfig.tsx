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
  if (settings.openAI?.provider === 'azure' || settings.openAI?.provider === 'openai') {
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
      {' '}
      {allowGrafanaManagedLLM && (
        <div onClick={selectGrafanaManaged}>
          <Card
            isSelected={llmOption === 'grafana-provided'}
            // onClick={selectGrafanaManaged} // prevents events passing to children, use parent div instead!
            className={s.cardWithoutBottomMargin}
          >
            <Card.Heading>Use OpenAI provided by Grafana</Card.Heading>
            <Card.Description>
              <p>Enable LLM features in Grafana by using a connection to OpenAI that is provided by Grafana</p>
              {llmOption === 'grafana-provided' && (
                <>
                  <div className={s.openaiTermsBox}>
                    <p>
                      To enable OpenAI via Grafana Labs, please note that some data from your Grafana instance will be
                      sent to OpenAI when you use the LLM-based features. Grafana Labs imposes usage limits for this
                      service.
                    </p>
                    <p>
                      Additionally, the following terms (&quot;AI Terms&quot;) are hereby added to and become part of
                      your licensing agreement with Grafana Labs (the &quot;Agreement&quot;) as additional terms.
                      Capitalized terms not defined in these AI Terms have the meanings given in the Agreement. These
                      terms apply to your specific use of the OpenAI via Grafana Labs feature(s), and are separate,
                      necessary terms regarding your use of this feature and therefore are not &apos;click-wrap&apos;,
                      &apos;shrink-wrap&apos;, different or additional terms, or the like, to the extent your licensing
                      agreement with Grafana Labs purports to supersede any such terms.
                    </p>
                    <ul>
                      <li>Grafana Labs uses OpenAI&apos;s API platform to provide the LLM features.</li>
                      <li>
                        OpenAI does not train aggregated models on inputs or outputs of the API platform as used in
                        connection with Grafana Labs Product(s).
                      </li>
                      <li>
                        OpenAI does retain data for a short time in order to provide the services and monitor for abuse.
                        All data sent to OpenAI is encrypted in transit and at rest.
                      </li>
                      <li>
                        All features utilizing OpenAI are clearly marked in the Grafana Labs Product(s), and each
                        feature sends minimal data to OpenAI&mdash;and only at the request of a user (for example, when
                        a user clicks the button to request an Incident auto-summary).
                      </li>
                      <li>
                        Grafana Labs will add new features regularly that utilize features connecting to OpenAI&apos;s
                        APIs, which may include, but are not limited to:
                        <ul>
                          <li>Explaining Flamegraphs & offer suggestions to fix issues</li>
                          <li>Incident auto-summary</li>
                          <li>
                            Suggesting names & descriptions for panels & dashboards, and summarize differences when
                            saving changes
                          </li>
                          <li>Explaining error log lines in Sift</li>
                          <li>Generating KQL queries in the Azure Data Explorer plugin</li>
                        </ul>
                      </li>
                      <li>
                        Visit the OpenAI trust portal for more detail about OpenAI:{' '}
                        <Button
                          size="sm"
                          variant="secondary"
                          onClick={(e) => {
                            window.open('https://trust.openai.com/', '_blank');
                            e.stopPropagation();
                          }}
                        >
                          https://trust.openai.com/
                        </Button>
                      </li>
                      <li>
                        If you enable this feature, OpenAI will be a subprocessor of Grafana Labs for the purpose of any
                        data processing agreement you may have in place with Grafana Labs.
                      </li>
                      <li>
                        Disclaimer. Outputs are generated through machine learning processes and are not tested,
                        verified, endorsed or guaranteed to be accurate, complete or current by Grafana Labs. You should
                        independently review and verify all outputs as to appropriateness for any or all of your use
                        cases or applications. The warranties, disclaimers, and limitations of liability in the
                        Agreement apply to the AI Features.
                      </li>
                    </ul>
                  </div>
                  <Checkbox value={optIn} onClick={optInChange} label="I accept the AI Terms above" />
                </>
              )}
            </Card.Description>
            <Card.Figure>
              <Icon name="grafana" size="lg" />
            </Card.Figure>
          </Card>
        </div>
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
  openaiTermsBox: css({
    'overflow-y': 'auto',
    height: '250px',
    'margin-right': theme.spacing(3),
    'margin-bottom': theme.spacing(1),
    padding: `${theme.spacing(1)} ${theme.spacing(2)} ${theme.spacing(1)} ${theme.spacing(2)}`,
    border: `1px solid ${theme.colors.border.medium}`,
    background: theme.colors.background.primary,
    color: theme.colors.text.primary,

    ' ul': {
      // space important, matches all children of type 'ul'
      'padding-left': theme.spacing(2),
    },
    '> ul > li:not(:last-child)': {
      // slight vertical padding between main bullet points
      'margin-bottom': theme.spacing(0.5),
    },
  }),
  cardWithoutBottomMargin: css`
    margin-bottom: 0;
    margin-top: ${theme.spacing(1)};
  `,
});
