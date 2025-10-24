import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { Button, Field, Input, TextArea, Switch, Select } from '@grafana/ui';

interface ToolParameterFormProps {
  schema: any; // JSON Schema object
  onParametersChange: (parameters: Record<string, any>) => void;
  onSubmit: () => void;
  isLoading?: boolean;
}

interface FormField {
  name: string;
  type: string;
  required: boolean;
  description?: string;
  example?: any;
  enum?: string[];
  default?: any;
}

/**
 * Parses a JSON schema and extracts form field definitions
 */
function parseSchema(schema: any): FormField[] {
  if (!schema?.properties) {
    return [];
  }

  const required = schema.required || [];
  return Object.entries(schema.properties).map(([name, prop]: [string, any]) => ({
    name,
    type: prop.type || 'string',
    required: required.includes(name),
    description: prop.description,
    example: prop.example,
    enum: prop.enum,
    default: prop.default,
  }));
}

const FormFieldComponent = React.memo(
  ({ field, value, onChange }: { field: FormField; value: any; onChange: (value: any) => void }) => {
    const handleChange = (newValue: any) => {
      onChange(newValue);
    };

    const fieldProps = {
      label: field.name,
      description: field.description,
      required: field.required,
    };

    // Handle enum/select fields
    if (field.enum && field.enum.length > 0) {
      const options = field.enum.map((option) => ({ label: option, value: option }));
      return (
        <Field {...fieldProps}>
          <Select
            options={options}
            value={value}
            onChange={(option) => handleChange(option?.value)}
            placeholder={field.example ? `e.g., ${field.example}` : `Select ${field.name}...`}
            isClearable={!field.required}
          />
        </Field>
      );
    }

    // Handle different field types
    switch (field.type) {
      case 'boolean':
        return (
          <Field {...fieldProps}>
            <Switch value={Boolean(value)} onChange={(e) => handleChange(e.currentTarget.checked)} />
          </Field>
        );

      case 'number':
      case 'integer':
        return (
          <Field {...fieldProps}>
            <Input
              type="number"
              value={value || ''}
              onChange={(e) => {
                const numValue =
                  field.type === 'integer' ? parseInt(e.currentTarget.value, 10) : parseFloat(e.currentTarget.value);
                handleChange(isNaN(numValue) ? undefined : numValue);
              }}
              placeholder={field.example ? `e.g., ${field.example}` : undefined}
            />
          </Field>
        );

      case 'array':
        return (
          <Field {...fieldProps} description={`${field.description || ''} (Enter one item per line)`}>
            <TextArea
              value={Array.isArray(value) ? value.join('\n') : ''}
              onChange={(e) => {
                const lines = e.currentTarget.value.split('\n').filter((line) => line.trim());
                handleChange(lines.length > 0 ? lines : undefined);
              }}
              placeholder={
                field.example
                  ? `e.g.,\n${Array.isArray(field.example) ? field.example.join('\n') : field.example}`
                  : 'Enter items, one per line'
              }
              rows={3}
            />
          </Field>
        );

      case 'object':
        return (
          <Field {...fieldProps} description={`${field.description || ''} (Enter valid JSON)`}>
            <TextArea
              value={typeof value === 'object' ? JSON.stringify(value, null, 2) : value || ''}
              onChange={(e) => {
                try {
                  const parsed = JSON.parse(e.currentTarget.value);
                  handleChange(parsed);
                } catch {
                  // Keep the raw value if it's not valid JSON yet
                  handleChange(e.currentTarget.value);
                }
              }}
              placeholder={
                field.example ? `e.g.,\n${JSON.stringify(field.example, null, 2)}` : '{\n  "key": "value"\n}'
              }
              rows={4}
            />
          </Field>
        );

      case 'string':
      default:
        // For long descriptions or if it looks like it might be multi-line, use TextArea
        const isLongField =
          (field.description && field.description.length > 100) ||
          field.name.toLowerCase().includes('description') ||
          field.name.toLowerCase().includes('query') ||
          field.name.toLowerCase().includes('message');

        if (isLongField) {
          return (
            <Field {...fieldProps}>
              <TextArea
                value={value || ''}
                onChange={(e) => handleChange(e.currentTarget.value || undefined)}
                placeholder={field.example ? `e.g., ${field.example}` : undefined}
                rows={3}
              />
            </Field>
          );
        }

        return (
          <Field {...fieldProps}>
            <Input
              value={value || ''}
              onChange={(e) => handleChange(e.currentTarget.value || undefined)}
              placeholder={field.example ? `e.g., ${field.example}` : undefined}
            />
          </Field>
        );
    }
  }
);

FormFieldComponent.displayName = 'FormFieldComponent';

/**
 * A dynamic form component that generates input fields based on a JSON schema.
 * Provides a user-friendly interface for entering tool parameters.
 */
export function ToolParameterForm({ schema, onParametersChange, onSubmit, isLoading }: ToolParameterFormProps) {
  // Memoize fields to prevent recreation on every render
  const fields = useMemo(() => parseSchema(schema), [schema]);

  // Initialize form data with defaults
  const [formData, setFormData] = useState<Record<string, any>>(() => {
    const initialData: Record<string, any> = {};
    fields.forEach((field) => {
      if (field.default !== undefined) {
        initialData[field.name] = field.default;
      }
    });
    return initialData;
  });

  // Update parent when form data changes - use a callback to avoid dependency issues
  useEffect(() => {
    // Only include fields that have values
    const cleanedData = Object.entries(formData).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== '' && value !== null) {
          acc[key] = value;
        }
        return acc;
      },
      {} as Record<string, any>
    );

    // Use a timeout to debounce updates to parent
    const timeoutId = setTimeout(() => {
      onParametersChange(cleanedData);
    }, 100);

    return () => clearTimeout(timeoutId);
  }, [formData, onParametersChange]);

  const handleFieldChange = useCallback((fieldName: string, value: any) => {
    setFormData((prev) => ({
      ...prev,
      [fieldName]: value,
    }));
  }, []);

  const isFormValid = () => {
    return fields.every((field) => {
      if (!field.required) {
        return true;
      }
      const value = formData[field.name];
      return value !== undefined && value !== '' && value !== null;
    });
  };

  if (fields.length === 0) {
    return (
      <div style={{ padding: '16px', textAlign: 'center' }}>
        <p style={{ color: 'var(--text-color-secondary)', marginBottom: '16px' }}>This tool requires no parameters.</p>
        <Button variant="primary" onClick={onSubmit} disabled={isLoading}>
          {isLoading ? 'Running...' : 'Run Tool'}
        </Button>
      </div>
    );
  }

  return (
    <div style={{ padding: '16px' }}>
      <div style={{ marginBottom: '20px' }}>
        {fields.map((field) => (
          <div key={field.name} style={{ marginBottom: '16px' }}>
            <FormFieldComponent
              field={field}
              value={formData[field.name]}
              onChange={(value) => handleFieldChange(field.name, value)}
            />
          </div>
        ))}
      </div>

      <Button variant="primary" onClick={onSubmit} disabled={isLoading || !isFormValid()} style={{ width: '100%' }}>
        {isLoading ? 'Running...' : 'Run Tool'}
      </Button>

      {!isFormValid() && (
        <div
          style={{
            marginTop: '8px',
            fontSize: '12px',
            color: 'var(--error-color)',
            textAlign: 'center',
          }}
        >
          Please fill in all required fields
        </div>
      )}
    </div>
  );
}
