import { css } from '@emotion/css';
import React, { useState } from 'react';

import { GrafanaTheme2 } from '@grafana/data';
import { Alert, Button, Card, Field, FieldSet, Icon, Modal, useStyles2 } from '@grafana/ui';

import { AppPluginSettings, Secrets, SecretsSet } from './AppConfig';
import { OpenAIConfig, OpenAIProvider } from './OpenAI';

// LLMOptions are the 3 possible UI options for LLMs.
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

    onChange({ ...settings, openAI: { provider: 'grafana' }, llmGateway: { optInStatus: true } });
  };

  const doOptOut = () => {
    setOptIn(false);
    dismissOptOutModal();

    onChange({ ...settings, openAI: { provider: undefined }, llmGateway: { optInStatus: false } });
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
          <h4>Terms for using Grafana-managed OpenAI</h4>
          <p>
            Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et
            dolore magna aliqua. Facilisi etiam dignissim diam quis. Eget lorem dolor sed viverra ipsum nunc. Netus et
            malesuada fames ac turpis egestas. Integer malesuada nunc vel risus commodo. Mattis aliquam faucibus purus
            in.
          </p>
          <p>
            Gravida dictum fusce ut placerat orci nulla pellentesque dignissim enim. Pharetra et ultrices neque ornare
            aenean euismod elementum nisi quis. Eget mi proin sed libero enim sed faucibus turpis. Egestas dui id ornare
            arcu. Sed faucibus turpis in eu mi bibendum. Vestibulum mattis ullamcorper velit sed ullamcorper morbi
            tincidunt ornare. In iaculis nunc sed augue lacus viverra vitae congue eu. Quisque id diam vel quam
            elementum pulvinar etiam non. Augue neque gravida in fermentum et sollicitudin ac. Pretium viverra
            suspendisse potenti nullam ac tortor vitae purus faucibus.
          </p>
          <p>
            Auctor neque vitae tempus quam pellentesque nec nam aliquam sem. Id diam vel quam elementum. Congue quisque
            egestas diam in arcu cursus. Fringilla ut morbi tincidunt augue interdum velit euismod in pellentesque.
            Potenti nullam ac tortor vitae purus faucibus. Nunc consequat interdum varius sit amet mattis vulputate enim
            nulla. Mauris commodo quis imperdiet massa tincidunt nunc pulvinar sapien et. Nam aliquam sem et tortor
            consequat id porta nibh. Pharetra convallis posuere morbi leo urna molestie at elementum. Gravida cum sociis
            natoque penatibus et. Et netus et malesuada fames ac turpis egestas. Turpis egestas sed tempus urna et. Enim
            blandit volutpat maecenas volutpat blandit aliquam. Donec ac odio tempor orci dapibus.
          </p>
          <p>
            Orci dapibus ultrices in iaculis nunc sed augue. Facilisis gravida neque convallis a cras semper auctor.
            Odio tempor orci dapibus ultrices in. Id nibh tortor id aliquet lectus proin nibh nisl condimentum. Sit amet
            massa vitae tortor condimentum lacinia quis vel. Ac orci phasellus egestas tellus rutrum tellus. Lacus
            vestibulum sed arcu non odio euismod lacinia. Aliquet eget sit amet tellus cras. Tortor pretium viverra
            suspendisse potenti nullam. Risus at ultrices mi tempus. Risus at ultrices mi tempus imperdiet. Mattis enim
            ut tellus elementum sagittis vitae. Nunc sed velit dignissim sodales ut eu sem. Enim nunc faucibus a
            pellentesque sit.
          </p>
          <p>
            Tincidunt arcu non sodales neque sodales ut etiam sit. Ut faucibus pulvinar elementum integer enim neque
            volutpat ac. Facilisis sed odio morbi quis commodo odio aenean sed adipiscing. Dignissim enim sit amet
            venenatis urna cursus. Lacus luctus accumsan tortor posuere ac ut. Habitant morbi tristique senectus et.
            Turpis cursus in hac habitasse platea. Commodo odio aenean sed adipiscing diam donec adipiscing tristique
            risus. Turpis tincidunt id aliquet risus feugiat in ante metus dictum. Accumsan tortor posuere ac ut
            consequat semper viverra. Et molestie ac feugiat sed lectus vestibulum mattis ullamcorper velit. Sed nisi
            lacus sed viverra.
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
        {llmOption === 'grafana-provided' && (
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
            <Icon name="sitemap" size="lg" />
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
