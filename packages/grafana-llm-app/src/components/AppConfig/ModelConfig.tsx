import React from 'react';

import { Badge, Button, Divider, Field, FieldSet, Input, Label, Select } from '@grafana/ui';
import { openai } from '@grafana/llm';

import { ProviderType } from './AppConfig';

export type ModelMapping = Partial<Record<openai.Model, string>>;

export interface ModelSettings {
  default?: openai.Model;
  mapping: ModelMapping;
}

export interface ModelMappingConfig {
  id: openai.Model;
  name: string;
  label: string;
  description: string;
}
const DEFAULT_MODEL_ID = openai.Model.BASE;
function defaultModelMappingConfig(provider: ProviderType): ModelMappingConfig[] {
  return provider === 'anthropic'
    ? [
        {
          id: openai.Model.BASE,
          name: 'claude-4-sonnet-20250514',
          label: 'Base',
          description: 'An effective, balanced model, suitable for most tasks.',
        },
        {
          id: openai.Model.LARGE,
          name: 'claude-4-sonnet-20250514',
          label: 'Large',
          description: 'An effective, balanced model, suitable for most tasks.',
        },
      ]
    : [
        {
          id: openai.Model.BASE,
          name: 'gpt-4.1-mini',
          label: 'Base',
          description: 'A fast and cost-effective model for efficient, high-throughput tasks.',
        },
        {
          id: openai.Model.LARGE,
          name: 'gpt-4.1',
          label: 'Large',
          description: 'A larger, higher cost model for more advanced tasks with longer context windows.',
        },
      ];
}

const initModelSettings = (
  provider: ProviderType,
  settings: ModelSettings,
  defaultModelMapping: ModelMappingConfig[]
): ModelSettings => {
  return {
    default: settings.default ?? DEFAULT_MODEL_ID,
    // If the settings are empty, set the default models
    // If the settings are not empty, filter out any models that are not in the default list
    mapping: settings.mapping
      ? (Object.fromEntries(
          Object.entries(settings.mapping).filter(([m, _]) => defaultModelMapping.find((d) => d.id === m))
        ) as ModelMapping)
      : (Object.fromEntries(defaultModelMapping.map((entry) => [entry.id, entry.name])) as ModelMapping),
  };
};

export function ModelConfig({
  provider,
  settings,
  onChange,
}: {
  provider: ProviderType;
  settings: ModelSettings;
  onChange: (settings: ModelSettings) => void;
}) {
  const defaultModelMapping = defaultModelMappingConfig(provider);
  settings = initModelSettings(provider, settings, defaultModelMapping);
  const setDefault = (model: openai.Model) => onChange({ ...settings, default: model });

  return (
    <FieldSet>
      {/*
        Only show custom model mappings for non-Azure providers.
        When using the Azure provider users can just customise the deployments
        instead.
      */}
      {provider === 'azure' && (
        <Field
          label="Default Model"
          description="The default model is used when no model is specified in the chat request."
        >
          <Select
            options={defaultModelMapping.map((entry) => ({ label: entry.label, value: entry.id }))}
            width={60}
            value={settings.default ?? DEFAULT_MODEL_ID}
            onChange={(e) => setDefault(e.value ?? (DEFAULT_MODEL_ID as openai.Model))}
          />
        </Field>
      )}
      {provider !== 'azure' && (
        <>
          <Label description="Map custom models used for LLM features. The default model is used when no model is specified in the chat request.">
            Model mappings
          </Label>
          {defaultModelMapping.map((entry, i) => {
            const modelSetting = settings.mapping[entry.id];
            const isDefault = settings.default === entry.id;
            const FieldLabel = (
              <>
                <Label>
                  {entry.label}
                  <Divider direction="vertical" spacing={1} />
                  {isDefault ? (
                    <div style={{ margin: '1px auto' }}>
                      <Badge text="Default" color="blue" />
                    </div>
                  ) : (
                    <Button variant="secondary" size="sm" onClick={() => setDefault(entry.id)}>
                      Set as default
                    </Button>
                  )}
                </Label>
              </>
            );

            return (
              <Field key={i} label={FieldLabel} description={entry.description}>
                <Input
                  width={60}
                  type="text"
                  name="model"
                  value={modelSetting}
                  placeholder={entry.name}
                  onChange={(e) => {
                    const newModelName = e.currentTarget.value;
                    const newMapping = {
                      ...settings.mapping,
                      [entry.id]: newModelName || undefined,
                    };
                    onChange({ ...settings, mapping: newMapping });
                  }}
                />
              </Field>
            );
          })}
        </>
      )}
    </FieldSet>
  );
}
