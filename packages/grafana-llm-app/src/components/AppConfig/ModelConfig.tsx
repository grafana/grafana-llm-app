import React from 'react';

import { Field, FieldSet, Input, Label, Select } from '@grafana/ui';
import { openai } from '@grafana/llm';

import { OpenAIProvider } from './OpenAI';

export type ModelMapping = Partial<Record<openai.Model, string>>;

export interface ModelSettings {
  default?: openai.Model;
  mapping: ModelMapping;
}

export interface ModelMappingConfig {
  model: openai.Model;
  name: string;
  label: string;
  description: string;
}
const DEFAULT_MODEL = openai.Model.BASE;
const DEFAULT_MODEL_NAMES: ModelMappingConfig[] = [
  {
    model: openai.Model.BASE,
    name: 'gpt-3.5-turbo',
    label: 'Base',
    description: 'A fast and cost-effective model for efficient, high-throughput tasks.',
  },
  {
    model: openai.Model.LARGE,
    name: 'gpt-4-turbo',
    label: 'Large',
    description:
      'A larger, higher cost model for more advanced tasks with longer context windows.',
  },
];

const initModelSettings = (settings: ModelSettings): ModelSettings => ({
  default: settings.default ?? DEFAULT_MODEL,
  // If the settings are empty, set the default models
  // If the settings are not empty, filter out any models that are not in the default list
  mapping: settings.mapping
    ? Object.fromEntries(
      Object.entries(settings.mapping)
        .filter(([m, _]) => DEFAULT_MODEL_NAMES.find((d) => d.model === m))
    ) as ModelMapping
    : Object.fromEntries(DEFAULT_MODEL_NAMES.map((entry) => ([entry.model, entry.name]))) as ModelMapping,
});

export function ModelConfig({
  provider,
  settings,
  onChange,
}: {
  provider: OpenAIProvider;
  settings: ModelSettings;
  onChange: (settings: ModelSettings) => void;
}) {
  settings = initModelSettings(settings);

  return (
    <FieldSet>
      <Field
        label="Default Model"
        description="The default model is used when no model is specified in the chat request."
      >
        <Select
          options={DEFAULT_MODEL_NAMES.map((entry) => ({ label: entry.label, value: entry.model }))}
          width={60}
          value={settings.default ?? DEFAULT_MODEL}
          onChange={(e) => onChange({ ...settings, default: e.value ?? openai.Model.BASE })}
        />
      </Field>

      {/*
        Only show custom model mappings for non-Azure providers.
        When using the Azure provider users can just customise the deployments
        instead.
      */ }
      {provider !== "azure" && (
        <>
          <Label description="Set custom models used for LLM features.">Custom overrides</Label>
          {DEFAULT_MODEL_NAMES.map((entry, i) => {
            const modelSetting = settings.mapping[entry.model];
            // If the model is not in the settings, add it with the default name
            if (!modelSetting) {
              onChange({ ...settings, mapping: { ...settings.mapping, [entry.model]: entry.name } });
            }

            return (
              <Field key={i} label={`${entry.label}`} description={entry.description}>
                <Input
                  width={60}
                  type="text"
                  name="model"
                  value={modelSetting ?? entry.name}
                  onChange={(e) => {
                    const newModelName = e.currentTarget.value;
                    const newMapping = {
                      ...settings.mapping,
                      [entry.model]: newModelName,
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
