import React from "react";

import { Checkbox, Field, FieldSet, InlineField, Input, Select, useStyles2, Label } from "@grafana/ui";

import { testIds } from "components/testIds";
import { AppPluginSettings, Secrets, SecretsSet, getStyles } from "./AppConfig";

import { BasicAuthConfig } from "./AuthSettings/BasicAuth";
import { SelectableValue } from "@grafana/data";

export interface VectorSettings {
  // Whether the vector service should be enabled.
  enabled?: boolean;
  // The model used by the embedder and for embeddings stored in the store.
  model?: string;
  // Embedder settings.
  embed?: EmbedderSettings;
  // Store settings.
  store?: StoreSettings;
};

export interface AuthSettings {
  url?: string;
  authType?: string;
  basicAuthUser?: string;
}

interface OpenAIEmbedderSettings extends AuthSettings {}

interface GrafanaVectorAPISettings extends AuthSettings {}


interface EmbedderSettings {
  type?: EmbedderOptions;
  openai?: OpenAIEmbedderSettings;
};

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

export function VectorConfig({ settings, secrets, secretsSet, onChange, onChangeSecrets }: {
  settings?: AppPluginSettings;
  secrets: Secrets;
  secretsSet: SecretsSet;
  onChange: (settings: VectorSettings) => void;
  onChangeSecrets: (secrets: Secrets) => void;
}) {
  const modelValues = settings?.vector?.embed?.type === "openai" ? "text-embedding-ada-002" : settings?.vector?.embed?.type === "grafana/vectorapi" ? (settings?.vector?.model ?? "BAAI/bge-small-en-v1.5"): ""
  return (
    <FieldSet label="Vector Settings">

      <Field label="Enabled" description="Enable vector database powered features.">
        <Checkbox
          name="enabled"
          data-testid={testIds.appConfig.vectorEnabled}
          value={settings?.vector?.enabled || false}
          onChange={e => onChange({ ...settings, enabled: e.currentTarget.checked })}
        />
      </Field>
  
      {settings?.vector?.enabled && (
        <>
          <Field label="Model" description="The model used by the embedder and for embeddings stored in the store" disabled={settings.vector?.embed?.type === "openai" || settings.vector?.embed?.type === undefined}>
            <Input
              width={60}
              name="model"
              data-testid={testIds.appConfig.model}
              value={modelValues}
              placeholder={""}
              onChange={e => onChange({ ...settings.vector, model: e.currentTarget.value })}
            />
          </Field>

          <EmbedderConfig
            settings={settings?.vector?.embed}
            secrets={secrets}
            secretsSet={secretsSet}
            onChange={embed => onChange({ ...settings.vector, embed}) }
            onChangeSecrets={onChangeSecrets}
          />

          <StoreConfig
            settings={settings?.vector?.store}
            secrets={secrets}
            secretsSet={secretsSet}
            onChange={store => onChange({ ...settings.vector, store })}
            onChangeSecrets={onChangeSecrets}
          />
        </>
      )}
    </FieldSet>
  )
}

type EmbedderOptions = 'openai' | 'grafana/vectorapi';


export function EmbedderConfig({ settings, secrets, secretsSet, onChange, onChangeSecrets }: {
  settings?: EmbedderSettings;
  secrets: Secrets;
  secretsSet: SecretsSet;
  onChange: (settings: EmbedderSettings) => void;
  onChangeSecrets: (secrets: Secrets) => void;
}) {
  const s = useStyles2(getStyles);

  return (
    <>
      <h4>Embedder</h4>
      <Field label="Embedder Provider" description="Select the embedder API to use">
        <>
          <Select
            options={[
              { label: "OpenAI API", value: "openai" },
              { label: "Grafana Vector API", value: "grafana/vectorapi" },
            ] as Array<SelectableValue<EmbedderOptions>>}
            placeholder="Select Embedder Provider"
            value={settings?.type}
            width={60}
            onChange={e => {
              onChange({...settings,  type: e.value});
            }}
          />
          {settings?.type === "openai" && (
            <Label> Using configured OpenAI as embedder provider </Label>
          )}
        </>
      </Field>

      {settings?.type === "grafana/vectorapi" && (
        <>
          <Field label="Auth Type">
            <Select
              options={[
                { label: "No Auth", value: "no-auth" },
                { label: "Basic Auth", value: "basic-auth" },
              ]}
              value={settings?.openai?.authType ?? "no-auth"}
              onChange={e => e.value !== undefined && onChange({ ...settings, openai: { ...settings.openai, authType: e.value} })}
              width={60}
            />
          </Field>
          <InlineField label="URL" tooltip="Address of the Grafana Vector API" labelWidth={s.inlineFieldWidth}>
            <Input
              name="url"
              value={settings?.openai?.url}
              width={s.inlineFieldInputWidth}
              data-testid={testIds.appConfig.grafanaVectorApiUrl}
              placeholder={"http://vectorapi:8889"}
              onChange={e => onChange({ ...settings, openai: { ...settings.openai, url: e.currentTarget.value }})}
            />
          </InlineField>
          {settings?.openai?.authType === "basic-auth" && (
            <BasicAuthConfig
              settings={settings.openai}
              secrets={secrets}
              secretsSet={secretsSet}
              onChange={authSettings => onChange({ ...settings, openai: authSettings })} 
              onChangeSecrets={onChangeSecrets}
              secretKey="vectorEmbedderBasicAuthPassword"
            />
          )}
        </>
      )}
    </>
  )
}

function QdrantConfig({ settings, onChange }: Props<QdrantSettings>) {
  return (
    <>
      <Field label="Address" description="Address of the qdrant gRPC server">
        <Input
          width={60}
          name="url"
          data-testid={testIds.appConfig.qdrantAddress}
          value={settings?.address}
          placeholder={"localhost:6334"}
          onChange={e => onChange({ ...settings, address: e.currentTarget.value })}
        />
      </Field>
      <Field label="Secure" description="Whether to use a secure connection">
        <Checkbox
          name="secure"
          data-testid={testIds.appConfig.qdrantSecure}
          checked={settings?.secure}
          onChange={e => onChange({ ...settings, secure: e.currentTarget.checked })}
        />
      </Field>
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
          placeholder={"http://vectorapi:8889"}
          onChange={e => onChange({ ...settings, url: e.currentTarget.value })}
        />
      </InlineField>
    </>
  );
}

function StoreConfig({ settings, secrets, secretsSet, onChange, onChangeSecrets }: {
  settings?: StoreSettings;
  secrets: Secrets;
  secretsSet: SecretsSet;
  onChange: (settings: StoreSettings) => void;
  onChangeSecrets: (secrets: Secrets) => void;
}) {
  return (
    <>
      <h4>Store</h4>
      <Field label="Store Type" description="The type of store">
        <Select
          options={[
            { label: "Qdrant", value: "qdrant" },
            { label: "Grafana Vector API", value: "grafana/vectorapi" },
          ]}
          value={settings?.type}
          onChange={e => e.value !== undefined && onChange({ ...settings, type: e.value, qdrant: undefined, grafanaVectorAPI: undefined })}
          width={60}
        />
      </Field>
      {settings?.type === "qdrant" && (
        <QdrantConfig
          settings={settings.qdrant}
          onChange={qdrant => onChange({ ...settings, qdrant })}
        />
      )}
      {settings?.type === "grafana/vectorapi" && (
        <>
          <Field label="Auth Type">
            <Select
              options={[
                { label: "No Auth", value: "no-auth" },
                { label: "Basic Auth", value: "basic-auth" },
              ]}
              value={settings?.grafanaVectorAPI?.authType ?? "no-auth"}
              onChange={e => e.value !== undefined && onChange({ ...settings, grafanaVectorAPI: { ...settings.grafanaVectorAPI, authType:e.value }})}
              width={60}
            />
          </Field>
          <GrafanaVectorAPIConfig
            settings={settings.grafanaVectorAPI}
            onChange={grafanaVectorAPI => onChange({ ...settings, grafanaVectorAPI })}
          />
        </>
      )}
      {settings?.type === "grafana/vectorapi" && settings?.grafanaVectorAPI?.authType === "basic-auth" && (
        <BasicAuthConfig
          settings={settings.grafanaVectorAPI}
          secrets={secrets}
          secretsSet={secretsSet}
          onChange={authSettings => onChange({ ...settings, grafanaVectorAPI: authSettings })} 
          onChangeSecrets={onChangeSecrets}
          secretKey="vectorStoreBasicAuthPassword"
        />
      )}
    </>
  )
}
