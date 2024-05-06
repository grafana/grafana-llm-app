import React from 'react';

import { Field, FieldSet, Input, Select } from '@grafana/ui';
import { openai } from '@grafana/llm';

export interface ModelSettings {
  default?: openai.Model;
  models: ModelMapping[];
}

export interface ModelMapping {
  model: openai.Model;
  name: string;
}

export interface ModelMappingConfig extends ModelMapping{
  description: string;
}
const DEFAULT_CHAT_MODEL_NAMES: ModelMappingConfig[] = [
  { model: openai.Model.Small, name: 'gpt-3.5-turbo', description: 'The smallest model, but still very powerful.' },
  { model: openai.Model.Medium, name: 'gpt-4-turbo', description: 'A medium-sized model.' },
  { model: openai.Model.Large, name: 'gpt-4', description: 'A large model.' },
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
      {DEFAULT_CHAT_MODEL_NAMES.map((entry, i) => {
        const modelSetting = settings.models?.find((m) => m.model === entry.model);
        // If the model is not in the settings, add it with the default name
        if (!modelSetting) {
          onChange({ ...settings, models: [...(settings.models ?? []), { model: entry.model, name: entry.name }] });
        }

        return ( <Field key={i} label={`Model-${entry.model}`} description={entry.description}>
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
      )
    })}
      
    
      <Field label="Default Model" description="The default model to use for chat.">
        <Select
          options={[
            { label: 'Small', value: openai.Model.Small },
            { label: 'Medium', value: openai.Model.Medium },
            { label: 'Large', value: openai.Model.Large },
          ]}
          width={60}
          value={settings.default ?? openai.Model.Small}
          onChange={(e) => onChange({ ...settings, default: e.value ?? openai.Model.Small})}
        />
      </Field>
    </FieldSet>
  );
}
