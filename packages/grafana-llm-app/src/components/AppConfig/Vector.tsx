import React from 'react';

import { Checkbox, Field, FieldSet, InlineField, Input, Select, useStyles2, Label, SecretInput } from '@grafana/ui';

import { testIds } from 'components/testIds';
import { Secrets, SecretsSet, getStyles } from './AppConfig';

import { BasicAuthConfig } from './AuthSettings/BasicAuth';
import { SelectableValue } from '@grafana/data';

export interface VectorSettings {
  // Whether the vector service should be enabled.
  enabled?: boolean;
  // The model used by the embedder and for embeddings stored in the store.
  model?: string;
  // Embedder settings.
  embed?: EmbedderSettings;
  // Store settings.
  store?: StoreSettings;
}

export interface AuthSettings {
  url?: string;
  authType?: string;
  basicAuthUser?: string;
}

interface GrafanaVectorAPISettings extends AuthSettings {}

interface EmbedderSettings {
  type?: EmbedderOptions;
  grafanaVectorAPI?: GrafanaVectorAPISettings;
}

interface QdrantSettings {
  address?: string;
  secure?: boolean;
}

interface StoreSettings {
  type?: string;
  qdrant?: QdrantSettings;
  grafanaVectorAPI?: GrafanaVectorAPISettings;
}

export interface Props<T> {
  settings?: T;
  onChange: (settings: T) => void;
}

interface SecretProps {
  onChangeSecrets: (secrets: Secrets) => void;
  secrets: Secrets;
  secretsSet: SecretsSet;
}

const OPENAI_DEFAULT_MODEL = 'text-embedding-ada-002';
const VECTORAPI_DEFAULT_MODEL = 'BAAI/bge-small-en-v1.5';

export function VectorConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
}: Props<VectorSettings> & SecretProps) {
  // update the model value if the embedder type changes
  React.useEffect(() => {
    if (settings?.embed?.type === 'openai' && settings?.model !== OPENAI_DEFAULT_MODEL) {
      onChange({ ...settings, model: OPENAI_DEFAULT_MODEL });
    } else if (
      settings?.embed?.type === 'grafana/vectorapi' &&
      (settings?.model === undefined || settings?.model === OPENAI_DEFAULT_MODEL)
    ) {
      onChange({ ...settings, model: VECTORAPI_DEFAULT_MODEL });
    }
  }, [settings, settings?.embed?.type, onChange]);

  return (
    <FieldSet label="Vector Settings">
      <Field label="Enabled" description="Enable vector database powered features.">
        <Checkbox
          name="enabled"
          data-testid={testIds.appConfig.vectorEnabled}
          value={settings?.enabled || false}
          onChange={(e) => onChange({ ...settings, enabled: e.currentTarget.checked })}
        />
      </Field>

      {settings?.enabled && (
        <>
          <Field
            label="Model"
            description="The model used by the embedder and for embeddings stored in the store"
            disabled={settings.embed?.type === 'openai' || settings.embed?.type === undefined}
          >
            <Input
              width={60}
              name="model"
              data-testid={testIds.appConfig.model}
              value={settings?.model}
              placeholder={settings?.model}
              onChange={(e) => onChange({ ...settings, model: e.currentTarget.value })}
            />
          </Field>

          <EmbedderConfig
            settings={settings?.embed}
            secrets={secrets}
            secretsSet={secretsSet}
            onChange={(embed) => onChange({ ...settings, embed })}
            onChangeSecrets={onChangeSecrets}
          />

          <StoreConfig
            settings={settings?.store}
            secrets={secrets}
            secretsSet={secretsSet}
            onChange={(store) => onChange({ ...settings, store })}
            onChangeSecrets={onChangeSecrets}
          />
        </>
      )}
    </FieldSet>
  );
}

type EmbedderOptions = 'openai' | 'grafana/vectorapi';

export function EmbedderConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
}: Props<EmbedderSettings> & SecretProps) {
  const s = useStyles2(getStyles);

  return (
    <>
      <h4>Embedder</h4>
      <Field label="Embedder Provider" description="Select the embedder API to use">
        <>
          <Select
            options={
              [
                { label: 'OpenAI API', value: 'openai' },
                { label: 'Grafana Vector API', value: 'grafana/vectorapi' },
              ] as Array<SelectableValue<EmbedderOptions>>
            }
            placeholder="Select Embedder Provider"
            value={settings?.type}
            width={60}
            onChange={(e) => onChange({ type: e.value })}
          />
          {settings?.type === 'openai' && <Label> Using configured OpenAI as embedder provider </Label>}
        </>
      </Field>

      {settings?.type === 'grafana/vectorapi' && (
        <>
          <Field label="Auth Type">
            <Select
              options={[
                { label: 'No Auth', value: 'no-auth' },
                { label: 'Basic Auth', value: 'basic-auth' },
              ]}
              value={settings?.grafanaVectorAPI?.authType ?? 'no-auth'}
              onChange={(e) =>
                e.value !== undefined &&
                onChange({ ...settings, grafanaVectorAPI: { ...settings.grafanaVectorAPI, authType: e.value } })
              }
              width={60}
            />
          </Field>
          <InlineField label="URL" tooltip="Address of the Grafana Vector API" labelWidth={s.inlineFieldWidth}>
            <Input
              name="url"
              value={settings?.grafanaVectorAPI?.url}
              width={s.inlineFieldInputWidth}
              data-testid={testIds.appConfig.grafanaVectorApiUrl}
              placeholder={'http://vectorapi:8889'}
              onChange={(e) =>
                onChange({
                  ...settings,
                  grafanaVectorAPI: { ...settings.grafanaVectorAPI, url: e.currentTarget.value },
                })
              }
            />
          </InlineField>
          {settings?.grafanaVectorAPI?.authType === 'basic-auth' && (
            <BasicAuthConfig
              settings={settings.grafanaVectorAPI}
              secrets={secrets}
              secretsSet={secretsSet}
              onChange={(authSettings) => onChange({ ...settings, grafanaVectorAPI: authSettings })}
              onChangeSecrets={onChangeSecrets}
              secretKey="vectorEmbedderBasicAuthPassword"
            />
          )}
        </>
      )}
    </>
  );
}

function QdrantConfig({
  settings,
  secrets,
  secretsSet,
  onChange,
  onChangeSecrets,
}: Props<QdrantSettings> & SecretProps) {
  const s = useStyles2(getStyles);
  return (
    <>
      <Field label="Address" description="Address of the qdrant gRPC server">
        <Input
          width={s.inlineFieldInputWidth}
          name="url"
          data-testid={testIds.appConfig.qdrantAddress}
          value={settings?.address}
          placeholder={'localhost:6334'}
          onChange={(e) => onChange({ ...settings, address: e.currentTarget.value })}
        />
      </Field>
      <Field label="Secure" description="Whether to use a secure connection">
        <Checkbox
          name="secure"
          data-testid={testIds.appConfig.qdrantSecure}
          checked={settings?.secure}
          onChange={(e) => onChange({ ...settings, secure: e.currentTarget.checked })}
        />
      </Field>
      {settings?.secure && (
        <Field label="API key" description="API key for the qdrant gRPC server">
          <SecretInput
            name="qdrantApiKey"
            width={s.inlineFieldInputWidth}
            data-testid={testIds.appConfig.qdrantApiKey}
            onChange={(e) => onChangeSecrets({ ...secrets, qdrantApiKey: e.currentTarget.value })}
            onReset={() => onChangeSecrets({ ...secrets, qdrantApiKey: '' })}
            isConfigured={secretsSet.qdrantApiKey ?? false}
            value={secrets.qdrantApiKey}
          />
        </Field>
      )}
    </>
  );
}

function GrafanaVectorAPIConfig({ settings, onChange }: Props<GrafanaVectorAPISettings>) {
  const s = useStyles2(getStyles);
  return (
    <>
      <InlineField label="URL" tooltip="Address of the Grafana Vector API" labelWidth={s.inlineFieldWidth}>
        <Input
          name="url"
          value={settings?.url}
          width={s.inlineFieldInputWidth}
          data-testid={testIds.appConfig.grafanaVectorApiUrl}
          placeholder={'http://vectorapi:8889'}
          onChange={(e) => onChange({ ...settings, url: e.currentTarget.value })}
        />
      </InlineField>
    </>
  );
}

function StoreConfig({ settings, secrets, secretsSet, onChange, onChangeSecrets }: Props<StoreSettings> & SecretProps) {
  return (
    <>
      <h4>Store</h4>
      <Field label="Store Type" description="The type of store">
        <Select
          options={[
            { label: 'Qdrant', value: 'qdrant' },
            { label: 'Grafana Vector API', value: 'grafana/vectorapi' },
          ]}
          value={settings?.type}
          onChange={(e) =>
            e.value !== undefined &&
            onChange({ ...settings, type: e.value, qdrant: undefined, grafanaVectorAPI: undefined })
          }
          width={60}
        />
      </Field>
      {settings?.type === 'qdrant' && (
        <QdrantConfig
          settings={settings.qdrant}
          secrets={secrets}
          secretsSet={secretsSet}
          onChange={(qdrant) => onChange({ ...settings, qdrant })}
          onChangeSecrets={onChangeSecrets}
        />
      )}
      {settings?.type === 'grafana/vectorapi' && (
        <>
          <Field label="Auth Type">
            <Select
              options={[
                { label: 'No Auth', value: 'no-auth' },
                { label: 'Basic Auth', value: 'basic-auth' },
              ]}
              value={settings?.grafanaVectorAPI?.authType ?? 'no-auth'}
              onChange={(e) =>
                e.value !== undefined &&
                onChange({ ...settings, grafanaVectorAPI: { ...settings.grafanaVectorAPI, authType: e.value } })
              }
              width={60}
            />
          </Field>
          <GrafanaVectorAPIConfig
            settings={settings.grafanaVectorAPI}
            onChange={(grafanaVectorAPI) => onChange({ ...settings, grafanaVectorAPI })}
          />
        </>
      )}
      {settings?.type === 'grafana/vectorapi' && settings?.grafanaVectorAPI?.authType === 'basic-auth' && (
        <BasicAuthConfig
          settings={settings.grafanaVectorAPI}
          secrets={secrets}
          secretsSet={secretsSet}
          onChange={(authSettings) => onChange({ ...settings, grafanaVectorAPI: authSettings })}
          onChangeSecrets={onChangeSecrets}
          secretKey="vectorStoreBasicAuthPassword"
        />
      )}
    </>
  );
}
