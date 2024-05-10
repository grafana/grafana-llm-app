import React from 'react';

import { Badge, Button, Divider, Field, FieldSet, Input, Label } from '@grafana/ui';
import { openai } from '@grafana/llm';

import { OpenAIProvider } from './OpenAI';

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
const MODEL_MAPPING_CONFIG: ModelMappingConfig[] = [
  {
    id: openai.Model.BASE,
    name: 'gpt-3.5-turbo',
    label: 'Base',
    description: 'A fast and cost-effective model for efficient, high-throughput tasks.',
  },
  {
    id: openai.Model.LARGE,
    name: 'gpt-4-turbo',
    label: 'Large',
    description: 'A larger, higher cost model for more advanced tasks with longer context windows.',
  },
];

const initModelSettings = (settings: ModelSettings): ModelSettings => ({
  default: settings.default ?? DEFAULT_MODEL_ID,
  // If the settings are empty, set the default models
  // If the settings are not empty, filter out any models that are not in the default list
  mapping: settings.mapping
    ? (Object.fromEntries(
        Object.entries(settings.mapping).filter(([m, _]) => MODEL_MAPPING_CONFIG.find((d) => d.id === m))
      ) as ModelMapping)
    : (Object.fromEntries(MODEL_MAPPING_CONFIG.map((entry) => [entry.id, entry.name])) as ModelMapping),
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
  const setDefault = (model: openai.Model) => onChange({ ...settings, default: model });

  return (
    <FieldSet>
      {/*
        Only show custom model mappings for non-Azure providers.
        When using the Azure provider users can just customise the deployments
        instead.
      */}
      {provider !== 'azure' && (
        <>
          <Label description="Map custom models used for LLM features. The default model is used when no model is specified in the chat request.">
            Model mappings
          </Label>
          {MODEL_MAPPING_CONFIG.map((entry, i) => {
            const modelSetting = settings.mapping[entry.id];
            // If the model is not in the settings, add it with the default name
            if (!modelSetting) {
              onChange({ ...settings, mapping: { ...settings.mapping, [entry.id]: entry.name } });
            }
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
              <>
                <Field key={i} label={FieldLabel} description={entry.description}>
                  <Input
                    width={60}
                    type="text"
                    name="model"
                    value={modelSetting ?? entry.name}
                    onChange={(e) => {
                      const newModelName = e.currentTarget.value;
                      const newMapping = {
                        ...settings.mapping,
                        [entry.id]: newModelName,
                      };
                      onChange({ ...settings, mapping: newMapping });
                    }}
                  />
                </Field>
              </>
            );
          })}
        </>
      )}
    </FieldSet>
  );
}
