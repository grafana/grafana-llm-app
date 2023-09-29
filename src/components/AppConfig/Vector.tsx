import React from "react";

import { Field, FieldSet, Input, Select, Switch, useStyles2 } from "@grafana/ui";

import { testIds } from "components/testIds";
import { getStyles } from "./AppConfig";

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

interface OpenAIEmbedderSettings {
  url?: string;
};

interface EmbedderSettings {
  type?: string;
  openai?: OpenAIEmbedderSettings;
};

interface QdrantSettings {
  address?: string;
  secure?: boolean;
}

interface GrafanaVectorAPISettings {
  url?: string;
}

interface StoreSettings {
  type?: string;
  qdrant?: QdrantSettings;
  grafanaVectorAPI?: GrafanaVectorAPISettings;
}

interface Props<T> {
  settings?: T;
  onChange: (settings: T) => void;
}

export function VectorConfig({ settings, onChange }: Props<VectorSettings>) {
  return (
    <FieldSet label="Vector Settings">

      <Field label="Enabled" description="Enable vector database powered features.">
        <Switch
          name="enabled"
          data-testid={testIds.appConfig.vectorEnabled}
          defaultChecked={settings?.enabled}
          checked={settings?.enabled}
          onChange={e => onChange({ ...settings, enabled: e.currentTarget.checked })}
        />
      </Field>

      <Field label="Model" description="The model used by the embedder and for embeddings stored in the store">
        <Input
          width={60}
          name="model"
          data-testid={testIds.appConfig.model}
          value={settings?.model}
          placeholder={""}
          onChange={e => onChange({ ...settings, model: e.currentTarget.value })}
        />
      </Field>

      <EmbedderConfig
        settings={settings?.embed}
        onChange={embed => onChange({ ...settings, embed })}
      />

      <StoreConfig
        settings={settings?.store}
        onChange={store => onChange({ ...settings, store })}
      />
    </FieldSet>
  )
}

function OpenAIEmbedderConfig({ settings, onChange }: Props<OpenAIEmbedderSettings>) {
  const s = useStyles2(getStyles);
  return (
    <Field label="OpenAI API URL" description="" className={s.marginTop}>
      <Input
        width={60}
        name="url"
        data-testid={testIds.appConfig.openAIUrl}
        value={settings?.url}
        placeholder={"https://api.openai.com"}
        onChange={e => onChange({ ...settings, url: e.currentTarget.value })}
      />
    </Field>
  );
}

function EmbedderConfig({ settings, onChange }: Props<EmbedderSettings>) {
  return (
    <>
      <h4>Embedder</h4>
      <Field label="Embedder Type" description="The type of embedder">
        <Select
          options={[
            { label: "OpenAI", value: "openai" },
          ]}
          value={settings?.type}
          onChange={e => e.value !== undefined && onChange({ ...settings, type: e.value })}
          width={60}
        />
      </Field>
      {settings?.type === "openai" && (
        <OpenAIEmbedderConfig
          settings={settings.openai}
          onChange={openai => onChange({ ...settings, openai })}
        />
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
        <Switch
          name="secure"
          data-testid={testIds.appConfig.qdrantSecure}
          defaultChecked={settings?.secure}
          checked={settings?.secure}
          onChange={e => onChange({ ...settings, secure: e.currentTarget.checked })}
        />
      </Field>
    </>
  );
}

function GrafanaVectorAPIConfig({ settings, onChange }: Props<GrafanaVectorAPISettings>) {
  return (
    <Field label="URL" description="URL of the Grafana Vector API">
      <Input
        width={60}
        name="url"
        data-testid={testIds.appConfig.grafanaVectorApiUrl}
        value={settings?.url}
        placeholder={"https://vectorapi.grafana.com"}
        onChange={e => onChange({ ...settings, url: e.currentTarget.value })}
      />
    </Field>
  );
}

function StoreConfig({ settings, onChange }: Props<StoreSettings>) {
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
          onChange={e => e.value !== undefined && onChange({ ...settings, type: e.value })}
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
        <GrafanaVectorAPIConfig
          settings={settings.grafanaVectorAPI}
          onChange={grafanaVectorAPI => onChange({ ...settings, grafanaVectorAPI })}
        />
      )}

    </>
  )
}
