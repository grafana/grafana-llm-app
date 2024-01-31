import { css } from '@emotion/css';
import React, { useState } from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { Alert, Button, Card, Field, FieldSet, Icon, Modal, useStyles2 } from '@grafana/ui';

import { AppPluginSettings, Secrets, SecretsSet } from './AppConfig';
import { OpenAIConfig, OpenAIProvider } from './OpenAI';
import { OpenAILogo } from './OpenAILogo';

// LLMOptions are the 3 possible UI options for LLMs (grafana-provided cloud-only).
export type LLMOptions = 'grafana-provided' | 'openai' | 'disabled';

// This maps the current settings to decide what UI selection (LLMOptions) to show
function getLLMOptionFromSettings(settings: AppPluginSettings): LLMOptions {
  if (settings.openAI?.provider === 'azure' || settings.openAI?.provider === 'openai') {
    return 'openai';
  } else if (settings.openAI?.provider === 'grafana' && settings.llmGateway?.optInStatus) {
    return 'grafana-provided';
  } else {
    return 'disabled';
  }
}

export function LLMConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
}: {
  settings: AppPluginSettings;
  onChange: (settings: AppPluginSettings) => void;
  secrets: Secrets;
  secretsSet: SecretsSet;
  onChangeSecrets: (secrets: Secrets) => void;
}) {
  const s = useStyles2(getStyles);
  const llmGatewayEnabled = settings.llmGateway?.url !== undefined; // if URL specified, llm-gateway available

  // llmOption is the currently chosen LLM option in the UI
  const [llmOption, setLLMOption] = useState<LLMOptions>(getLLMOptionFromSettings(settings));
  // previousOpenAIProvider caches the value of the openAI provider, as it is overwritten by the grafana option
  const [previousOpenAIProvider, setPreviousOpenAIProvider] = useState<OpenAIProvider>();
  // optIn indicates if the user has opted in to Grafana-managed OpenAI
  const [optIn, setOptIn] = useState<boolean>(settings.llmGateway?.optInStatus || false);

  // 2 modals: opt-in and opt-out
  const [optInModalIsOpen, setOptInModalIsOpen] = useState<boolean>(false);
  const [optOutModalIsOpen, setOptOutModalIsOpen] = useState<boolean>(false);
  const showOptInModal = () => {
    setOptInModalIsOpen(true);
  };
  const dismissOptInModal = () => {
    setOptInModalIsOpen(false);
    // TODO: Reset scroll position of the T&Cs.
    setLLMOption('disabled');
  };
  const showOptOutModal = () => {
    setOptOutModalIsOpen(true);
  };
  const dismissOptOutModal = () => {
    setOptOutModalIsOpen(false);
  };

  const doOptIn = () => {
    setOptIn(true);
    setOptInModalIsOpen(false);

    onChange({
      ...settings,
      openAI: { provider: 'grafana' },
      llmGateway: { ...settings.llmGateway, optInStatus: true },
    });
  };

  const doOptOut = () => {
    setOptIn(false);
    dismissOptOutModal();

    onChange({
      ...settings,
      openAI: { provider: undefined },
      llmGateway: { ...settings.llmGateway, optInStatus: false },
    });
    setLLMOption('disabled');
  };

  // Handlers for when different LLM options are chosen in the UI
  const selectLLMDisabled = () => {
    // Cache if OpenAI or Azure is used, so can restore
    if (previousOpenAIProvider !== undefined) {
      setPreviousOpenAIProvider(settings.openAI?.provider);
    }

    onChange({ ...settings, openAI: { provider: undefined } });
    setLLMOption('disabled');
  };

  const selectGrafanaManaged = () => {
    // Cache if OpenAI or Azure is used, so can restore
    if (previousOpenAIProvider !== undefined) {
      setPreviousOpenAIProvider(settings.openAI?.provider);
    }
    if (settings.llmGateway?.optInStatus) {
      // as already opted-in, can immediately use this setting. Otherwise requires Opt-In to use.
      onChange({ ...settings, openAI: { provider: 'grafana' } });
    }

    setLLMOption('grafana-provided');
  };

  const selectOpenAI = () => {
    // Restore the provider
    const newSettings = { ...settings, openAI: { provider: previousOpenAIProvider } };
    onChange(newSettings);

    onChange({ ...settings, openAI: { provider: 'openai' } });
    setLLMOption('openai');
  };

  return (
    <>
      <Modal
        title="Enable OpenAI access via Grafana"
        isOpen={optInModalIsOpen}
        onDismiss={dismissOptInModal}
        onClickBackdrop={dismissOptInModal}
      >
        <Alert title="To enable OpenAI via Grafana, please note the following:" severity="info">
          <ul>
            <li>Some data from your Grafana instance will be sent to OpenAI.</li>
            <li>Grafana imposes usage limits for this service.</li>
          </ul>
        </Alert>

        <p>To proceed please agree to the following terms & conditions:</p>
        <div className={s.divWithScrollbar}>
          <h4>FIXME! Terms &amp; Conditions for the Grafana-managed OpenAI</h4>

          <h5>1. Acceptance of Terms</h5>
          <p>
            By using the Grafana-managed OpenAI proxy service (the &quot;Service&quot;), you agree to be bound by these
            Terms & Conditions (&quot;Terms&quot;). If you do not agree with these Terms, do not use the Service. The
            Service acts as a bridge, forwarding your requests to OpenAI and returning responses. It&apos;s designed to
            enhance your Grafana platform experience by leveraging OpenAI&apos;s capabilities.
          </p>

          <h5>2. Privacy & Data Exposure</h5>
          <p>
            Understand that by using the Service, certain data from your Grafana instance, including but not limited to,
            metrics, logs, and dashboard configurations, may be exposed to OpenAI to fulfill your requests. Although we
            strive to ensure the confidentiality and security of your data, by agreeing to these Terms, you grant us
            permission to share this data with OpenAI as necessary. We encourage you to review both our privacy policy
            and that of OpenAI to understand how your data is handled.
          </p>

          <h5>3. Usage Restrictions</h5>
          <p>
            The Service is provided for your personal and internal business use only. You are prohibited from using the
            Service for any illegal or unauthorized purpose. Additionally, you must not attempt to gain unauthorized
            access to the Service, other accounts, computer systems, or networks connected to the Service through
            hacking, password mining, or any other means.
          </p>

          <h5>4. Service Modifications and Availability</h5>
          <p>
            We reserve the right at any time and from time to time to modify or discontinue, temporarily or permanently,
            the Service (or any part thereof) with or without notice. You agree that we shall not be liable to you or to
            any third party for any modification, suspension, or discontinuance of the Service. We do not guarantee the
            availability of the Service and it may be subject to downtimes and periodic maintenance.
          </p>

          <h5>5. Limitation of Liability</h5>
          <p>
            To the fullest extent permitted by law, in no event will we, our affiliates, officers, employees, agents,
            suppliers, or licensors be liable for any indirect, incidental, special, consequential, or exemplary
            damages, including but not limited to, damages for loss of profits, goodwill, use, data, or other intangible
            losses (even if we have been advised of the possibility of such damages), resulting from the use or the
            inability to use the Service.
          </p>

          <h5>6. Changes to Terms</h5>
          <p>
            We reserve the right, at our sole discretion, to change, modify, add, or remove portions of these Terms at
            any time. It is your responsibility to check these Terms periodically for changes. Your continued use of the
            Service following the posting of changes will mean that you accept and agree to the changes.
          </p>
        </div>
        <Modal.ButtonRow>
          <Button variant="secondary" fill="outline" onClick={dismissOptInModal}>
            Cancel
          </Button>
          <Button onClick={doOptIn}>I Agree</Button>
        </Modal.ButtonRow>
      </Modal>

      <Modal
        title="Disable OpenAI access via Grafana"
        isOpen={optOutModalIsOpen}
        onDismiss={dismissOptOutModal}
        onClickBackdrop={dismissOptOutModal}
      >
        This will disable all Grafana&rsquo;s LLM features. Are you sure you want to continue?
        <Modal.ButtonRow>
          <Button variant="secondary" fill="outline" onClick={dismissOptOutModal}>
            Cancel
          </Button>
          <Button onClick={doOptOut}>Disable</Button>
        </Modal.ButtonRow>
      </Modal>

      <FieldSet label="OpenAI Settings" className={s.sidePadding}>
        {llmGatewayEnabled && (
          <Card
            isSelected={llmOption === 'grafana-provided'}
            onClick={selectGrafanaManaged}
            className={s.cardWithoutBottomMargin}
          >
            <Card.Heading>Use OpenAI provided by Grafana</Card.Heading>
            <Card.Description>
              Enable LLM features in Grafana by using a connection to OpenAI that is provided by Grafana
            </Card.Description>
            <Card.Figure>
              <Icon name="grafana" size="lg" />
            </Card.Figure>
          </Card>
        )}
        {llmGatewayEnabled && llmOption === 'grafana-provided' && (
          <div className={s.optionDetails}>
            <Field>
              {optIn ? (
                <>
                  <p>
                    You <b>have</b> enabled the Grafana-managed OpenAI.
                  </p>
                  <p>
                    This means some data from your Grafana instance is being sent to OpenAI. Note that usage limits will
                    apply.
                  </p>
                  <p>
                    If you would like to disable this, click here:&nbsp;
                    <Button onClick={showOptOutModal} variant="destructive" size="sm">
                      Disable OpenAI access via Grafana
                    </Button>
                  </p>
                </>
              ) : (
                <>
                  <p>If you wish to use Grafana&rsquo;s LLM-based features, you must supply your own OpenAI key.</p>
                  <p>Alternatively you can use the Grafana-managed LLM by opting-in by clicking here:</p>
                  <Button onClick={showOptInModal} size="lg">
                    Enable OpenAI access via Grafana
                  </Button>
                </>
              )}
            </Field>
          </div>
        )}

        <Card isSelected={llmOption === 'openai'} onClick={selectOpenAI} className={s.cardWithoutBottomMargin}>
          <Card.Heading>Use your own OpenAI account</Card.Heading>
          <Card.Description>Enable LLM features in Grafana using your own OpenAI details</Card.Description>
          <Card.Figure>
            <OpenAILogo width={20} height={20} />
          </Card.Figure>
        </Card>

        {llmOption === 'openai' && (
          <div className={s.optionDetails}>
            <OpenAIConfig
              settings={settings.openAI ?? {}}
              onChange={(openAI) => onChange({ ...settings, openAI })}
              secrets={secrets}
              secretsSet={secretsSet}
              onChangeSecrets={onChangeSecrets}
            />
          </div>
        )}

        <Card isSelected={llmOption === 'disabled'} onClick={selectLLMDisabled} className={s.cardWithoutBottomMargin}>
          <Card.Heading>Disable all LLM features in Grafana</Card.Heading>
          <Card.Description>&nbsp;</Card.Description>
          <Card.Figure>
            <Icon name="times" size="lg" />
          </Card.Figure>
        </Card>
      </FieldSet>
    </>
  );
}

export const getStyles = (theme: GrafanaTheme2) => ({
  sidePadding: css`
    margin-left: ${theme.spacing(1)};
    margin-right: ${theme.spacing(1)};
    width: 1000px;
  `,
  divWithScrollbar: css`
    overflow-y: auto;
    max-height: 450px;
    margin-left: ${theme.spacing(1)};
    margin-right: ${theme.spacing(1)};
    padding: ${theme.spacing(1)};
    border: 1px solid #383951;
  `,
  optionDetails: css`
    margin-left: ${theme.spacing(1)};
    margin-right: ${theme.spacing(1)};
    padding: ${theme.spacing(1)};
    border: 1px solid #383951;
  `,
  cardWithoutBottomMargin: css`
    margin-bottom: 0;
    margin-top: ${theme.spacing(1)};
    height: 80px;
  `,
  inlineFieldWidth: 15,
  inlineFieldInputWidth: 40,
});
