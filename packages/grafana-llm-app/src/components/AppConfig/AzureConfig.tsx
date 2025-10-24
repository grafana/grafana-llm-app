import React from 'react';

import { IconButton, InlineField, InlineFieldRow, Input, Select } from '@grafana/ui';

export type AzureModelDeployments = Array<[string, string]>;

export function AzureModelDeploymentConfig({
  modelMapping,
  modelNames,
  onChange,
}: {
  modelMapping: AzureModelDeployments;
  modelNames: string[];
  onChange: (modelMapping: AzureModelDeployments) => void;
}) {
  return (
    <>
      <IconButton
        name="plus"
        aria-label="Add model mapping"
        onClick={(e) => {
          e.preventDefault();
          onChange([...modelMapping, ['', '']]);
        }}
      />
      {modelMapping.map(([model, deployment], i) => (
        <ModelMappingField
          key={i}
          model={model}
          deployment={deployment}
          modelNames={modelNames}
          onChange={(model, deployment) => {
            onChange([...modelMapping.slice(0, i), [model, deployment], ...modelMapping.slice(i + 1)]);
          }}
          onRemove={() => onChange([...modelMapping.slice(0, i), ...modelMapping.slice(i + 1)])}
        />
      ))}
    </>
  );
}

function ModelMappingField({
  model,
  deployment,
  modelNames,
  onChange,
  onRemove,
}: {
  model: string;
  deployment: string;
  modelNames: string[];
  onChange: (model: string, deployment: string) => void;
  onRemove: () => void;
}): React.ReactElement {
  return (
    <InlineFieldRow>
      <InlineField label="Model">
        <Select
          placeholder="model label"
          options={modelNames.filter((n) => n !== deployment && n !== '').map((value) => ({ label: value, value }))}
          value={model}
          onChange={(event) => event.value !== undefined && onChange(event.value, deployment)}
        />
      </InlineField>
      <InlineField label="Deployment">
        <Input
          width={40}
          name="AzureDeployment"
          placeholder="deployment name"
          value={deployment}
          onChange={(event) => event.currentTarget.value !== undefined && onChange(model, event.currentTarget.value)}
        />
      </InlineField>
      <IconButton
        name="trash-alt"
        aria-label="Remove model mapping"
        onClick={(e) => {
          e.preventDefault();
          onRemove();
        }}
      />
    </InlineFieldRow>
  );
}
