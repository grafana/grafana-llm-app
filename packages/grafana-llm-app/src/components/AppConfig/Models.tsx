import React from 'react';

import { Field, FieldSet, Input, Label, Select } from '@grafana/ui';
import { openai } from '@grafana/llm';

export interface ModelSettings {
  default?: openai.Model;
  models: ModelMapping[];
}

export interface ModelMapping {
  model: openai.Model;
  name: string;
}

export interface ModelMappingConfig extends ModelMapping {
  label: string;
  description: string;
}
const DEFAULT_MODEL_NAMES: ModelMappingConfig[] = [
  {
    model: openai.Model.SMALL,
    name: 'gpt-3.5-turbo',
    label: 'Base',
    description: 'A fast and cost-effective model for efficient, high-throughput tasks.',
  },
  {
    model: openai.Model.MEDIUM,
    name: 'gpt-4-turbo',
    label: 'Medium',
    description:
      'A more advanced model with broader general knowledge and more advanced reasoning capabilities at longer context windows.',
  },
  {
    model: openai.Model.LARGE,
    name: 'gpt-4',
    label: 'Pro',
    description:
      'A large and high-cost model, for more complex analysis, longer tasks with multiple steps, and higher-order math and coding tasks.',
  },
];

export function ModelConfig({
  settings,
  onChange,
}: {
  settings: ModelSettings;
  onChange: (settings: ModelSettings) => void;
}) {
  return (
    <FieldSet>
      <Field
        label="Default Model"
        description="The default model is used when no model is specified in the chat request."
      >
        <Select
          options={DEFAULT_MODEL_NAMES.map((entry) => ({ label: entry.label, value: entry.model }))}
          width={60}
          value={settings.default ?? openai.Model.SMALL}
          onChange={(e) => onChange({ ...settings, default: e.value ?? openai.Model.SMALL })}
        />
      </Field>

      <Label description="Set a custom model used for the LLM features.">Custom overrides</Label>
      {DEFAULT_MODEL_NAMES.map((entry, i) => {
        const modelSetting = settings.models?.find((m) => m.model === entry.model);
        // If the model is not in the settings, add it with the default name
        if (!modelSetting) {
          onChange({ ...settings, models: [...(settings.models ?? []), { model: entry.model, name: entry.name }] });
        }

        return (
          <Field key={i} label={`${entry.label}`} description={entry.description}>
            <Input
              width={60}
              type="text"
              name="model"
              value={modelSetting?.name ?? entry.name}
              onChange={(e) => {
                const newModelName = e.currentTarget.value;
                const newSettings = settings.models.map((m) =>
                  m.model === entry.model ? { ...m, name: newModelName } : m
                );
                onChange({ ...settings, models: newSettings });
              }}
            />
          </Field>
        );
      })}
    </FieldSet>
  );
}
