import React, { useState } from 'react';
import { Button, Spinner, Icon, TextArea, CollapsableSection, Tab, TabsBar } from '@grafana/ui';
import { Tool } from '@modelcontextprotocol/sdk/types';
import { mcp } from '@grafana/llm';
import { ToolParameterForm } from './ToolParameterForm';

// Tool category mapping
const TOOL_CATEGORIES: Record<string, string> = {
  // Admin
  list_teams: 'Admin',
  list_users_by_org: 'Admin',
  // Search
  search_dashboards: 'Search',
  // Dashboard
  get_dashboard_by_uid: 'Dashboard',
  update_dashboard: 'Dashboard',
  get_dashboard_panel_queries: 'Dashboard',
  // Datasources
  list_datasources: 'Datasources',
  get_datasource_by_uid: 'Datasources',
  get_datasource_by_name: 'Datasources',
  // Prometheus
  query_prometheus: 'Prometheus',
  list_prometheus_metric_metadata: 'Prometheus',
  list_prometheus_metric_names: 'Prometheus',
  list_prometheus_label_names: 'Prometheus',
  list_prometheus_label_values: 'Prometheus',
  // Incident
  list_incidents: 'Incident',
  create_incident: 'Incident',
  add_activity_to_incident: 'Incident',
  get_incident: 'Incident',
  // Loki
  query_loki_logs: 'Loki',
  list_loki_label_names: 'Loki',
  list_loki_label_values: 'Loki',
  query_loki_stats: 'Loki',
  // Alerting
  list_alert_rules: 'Alerting',
  get_alert_rule_by_uid: 'Alerting',
  list_contact_points: 'Alerting',
  // OnCall
  list_oncall_schedules: 'OnCall',
  get_oncall_shift: 'OnCall',
  get_current_oncall_users: 'OnCall',
  list_oncall_teams: 'OnCall',
  list_oncall_users: 'OnCall',
  // Sift
  get_sift_investigation: 'Sift',
  get_sift_analysis: 'Sift',
  list_sift_investigations: 'Sift',
  find_error_pattern_logs: 'Sift',
  find_slow_requests: 'Sift',
  // Pyroscope
  list_pyroscope_label_names: 'Pyroscope',
  list_pyroscope_label_values: 'Pyroscope',
  list_pyroscope_profile_types: 'Pyroscope',
  fetch_pyroscope_profile: 'Pyroscope',
  // Asserts
  get_assertions: 'Asserts',
};

interface ToolInspectorProps {
  tool: Tool;
}

interface ToolCallResult {
  loading: boolean;
  success?: boolean;
  response?: any;
  error?: string;
}

function ToolInspector({ tool }: ToolInspectorProps) {
  const { client } = mcp.useMCPClient();
  const [expanded, setExpanded] = useState(false);
  const [parametersInput, setParametersInput] = useState('{}');
  const [formParameters, setFormParameters] = useState<any>({});
  const [inputMode, setInputMode] = useState<'form' | 'json'>('form');
  const [callResult, setCallResult] = useState<ToolCallResult | null>(null);

  const handleTestTool = async () => {
    if (!client) {
      return;
    }

    setCallResult({ loading: true });

    try {
      // Use parameters from form or JSON based on current mode
      let parameters = {};

      if (inputMode === 'form') {
        parameters = formParameters;
      } else {
        if (parametersInput.trim()) {
          try {
            parameters = JSON.parse(parametersInput);
          } catch (e) {
            setCallResult({
              loading: false,
              success: false,
              error: `Invalid JSON parameters: ${e instanceof Error ? e.message : 'Unknown error'}`,
            });
            return;
          }
        }
      }

      // Call the tool
      const response = await client.callTool({
        name: tool.name,
        arguments: parameters,
      });

      setCallResult({
        loading: false,
        success: true,
        response,
      });
    } catch (error) {
      setCallResult({
        loading: false,
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error',
      });
    }
  };

  const formatSchema = (schema: any) => {
    if (!schema) {
      return 'No schema available';
    }

    try {
      return JSON.stringify(schema, null, 2);
    } catch {
      return String(schema);
    }
  };

  const generateExampleParameters = () => {
    if (!tool.inputSchema?.properties) {
      return '{}';
    }

    const example: any = {};
    const properties = tool.inputSchema.properties;

    Object.keys(properties).forEach((key) => {
      const prop = properties[key] as any;
      if (prop && typeof prop === 'object') {
        if (prop.type === 'string') {
          example[key] = prop.example || 'example_value';
        } else if (prop.type === 'number') {
          example[key] = prop.example || 42;
        } else if (prop.type === 'boolean') {
          example[key] = prop.example || true;
        } else if (prop.type === 'array') {
          example[key] = prop.example || [];
        } else if (prop.type === 'object') {
          example[key] = prop.example || {};
        } else {
          example[key] = prop.example || null;
        }
      }
    });

    return JSON.stringify(example, null, 2);
  };

  const fillExampleParameters = () => {
    setParametersInput(generateExampleParameters());
  };

  return (
    <div
      style={{
        border: '1px solid var(--border-color)',
        borderRadius: '8px',
        marginBottom: '12px',
        backgroundColor: 'var(--background-color-primary)',
      }}
    >
      {/* Tool Header */}
      <div
        style={{
          padding: '12px 16px',
          borderBottom: expanded ? '1px solid var(--border-color)' : 'none',
          cursor: 'pointer',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
        }}
        onClick={() => setExpanded(!expanded)}
      >
        <div style={{ flex: 1 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '4px' }}>
            <code
              style={{
                fontWeight: 500,
                fontSize: '15px',
                backgroundColor: 'var(--background-color-secondary)',
                padding: '2px 6px',
                borderRadius: '3px',
                border: '1px solid var(--border-color)',
              }}
            >
              {tool.name}
            </code>
            {tool.annotations?.title && tool.annotations.title !== tool.name && (
              <span style={{ fontSize: '14px', color: 'var(--text-color-secondary)' }}>({tool.annotations.title})</span>
            )}
          </div>
          {tool.description && (
            <div
              style={{
                fontSize: '13px',
                color: 'var(--text-color-secondary)',
                lineHeight: '1.4',
                marginBottom: '4px',
              }}
            >
              {tool.description}
            </div>
          )}
          {/* Additional metadata */}
          <div style={{ display: 'flex', gap: '12px', fontSize: '12px', color: 'var(--text-color-secondary)' }}>
            {(() => {
              const properties = tool.inputSchema?.properties;
              const paramCount =
                properties && typeof properties === 'object' && properties !== null
                  ? Object.keys(properties).length
                  : 0;
              return paramCount > 0 ? <span>Parameters: {paramCount}</span> : null;
            })()}
            {(() => {
              const required = tool.inputSchema?.required;
              const requiredCount = Array.isArray(required) ? required.length : 0;
              return requiredCount > 0 ? <span>Required: {requiredCount}</span> : null;
            })()}
          </div>
        </div>
        <Icon name={expanded ? 'angle-up' : 'angle-down'} />
      </div>

      {/* Expanded Content */}
      {expanded && (
        <div style={{ padding: '16px' }}>
          {/* Tool Schema */}
          <div style={{ marginBottom: '20px' }}>
            <h4 style={{ margin: '0 0 8px 0', fontSize: '14px' }}>Input Schema</h4>
            <pre
              style={{
                backgroundColor: 'var(--background-color-secondary)',
                padding: '12px',
                borderRadius: '4px',
                fontSize: '12px',
                lineHeight: '1.4',
                overflow: 'auto',
                maxHeight: '200px',
                border: '1px solid var(--border-color)',
              }}
            >
              {formatSchema(tool.inputSchema)}
            </pre>
          </div>

          {/* Parameter Input */}
          <div style={{ marginBottom: '16px' }}>
            <div style={{ marginBottom: '12px' }}>
              <h4 style={{ margin: '0 0 8px 0', fontSize: '14px' }}>Test Parameters</h4>
              <TabsBar>
                <Tab label="Form" active={inputMode === 'form'} onChangeTab={() => setInputMode('form')} />
                <Tab label="JSON" active={inputMode === 'json'} onChangeTab={() => setInputMode('json')} />
              </TabsBar>
            </div>

            {inputMode === 'form' ? (
              <ToolParameterForm
                schema={tool.inputSchema}
                onParametersChange={setFormParameters}
                onSubmit={handleTestTool}
                isLoading={callResult?.loading}
              />
            ) : (
              <div>
                <div
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    marginBottom: '8px',
                  }}
                >
                  <span style={{ fontSize: '12px', color: 'var(--text-color-secondary)' }}>JSON Mode</span>
                  <Button
                    variant="secondary"
                    size="sm"
                    onClick={fillExampleParameters}
                    disabled={!tool.inputSchema?.properties}
                  >
                    Fill Example
                  </Button>
                </div>
                <TextArea
                  value={parametersInput}
                  onChange={(e) => setParametersInput(e.currentTarget.value)}
                  placeholder="Enter JSON parameters for testing..."
                  rows={4}
                  style={{ fontFamily: 'monospace', fontSize: '12px' }}
                />
                <div style={{ marginTop: '12px' }}>
                  <Button
                    variant="primary"
                    size="sm"
                    onClick={handleTestTool}
                    disabled={callResult?.loading || !client}
                  >
                    {callResult?.loading ? <Spinner size="sm" /> : 'Test Tool'}
                  </Button>
                </div>
              </div>
            )}
          </div>

          {/* Call Result */}
          {callResult && !callResult.loading && (
            <div style={{ marginTop: '16px' }}>
              <h4 style={{ margin: '0 0 8px 0', fontSize: '14px' }}>
                Result{' '}
                {callResult.success ? (
                  <Icon name="check" style={{ color: 'var(--success-color)', marginLeft: '8px' }} />
                ) : (
                  <Icon name="exclamation-triangle" style={{ color: 'var(--error-color)', marginLeft: '8px' }} />
                )}
              </h4>

              {callResult.error ? (
                <div
                  style={{
                    backgroundColor: 'var(--error-background)',
                    color: 'var(--error-text-color)',
                    padding: '12px',
                    borderRadius: '4px',
                    fontSize: '13px',
                  }}
                >
                  {callResult.error}
                </div>
              ) : (
                <pre
                  style={{
                    backgroundColor: 'var(--background-color-secondary)',
                    padding: '12px',
                    borderRadius: '4px',
                    fontSize: '12px',
                    lineHeight: '1.4',
                    overflow: 'auto',
                    maxHeight: '300px',
                    border: '1px solid var(--border-color)',
                  }}
                >
                  {JSON.stringify(callResult.response, null, 2)}
                </pre>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

interface DevSandboxToolInspectorProps {
  tools: Tool[];
}

export function DevSandboxToolInspector({ tools }: DevSandboxToolInspectorProps) {
  const [searchFilter, setSearchFilter] = useState('');

  const filteredTools = tools.filter((tool) => {
    const searchLower = searchFilter.toLowerCase();
    const name = (tool.annotations?.title ?? tool.name).toLowerCase();
    const description = (tool.description || '').toLowerCase();
    return name.includes(searchLower) || description.includes(searchLower);
  });

  // Group tools by category
  const groupedTools = filteredTools.reduce(
    (groups, tool) => {
      const category = TOOL_CATEGORIES[tool.name] || 'Ungrouped';
      if (!groups[category]) {
        groups[category] = [];
      }
      groups[category].push(tool);
      return groups;
    },
    {} as Record<string, Tool[]>
  );

  // Sort categories with Ungrouped at the end
  const sortedCategories = Object.keys(groupedTools).sort((a, b) => {
    if (a === 'Ungrouped') {
      return 1;
    }
    if (b === 'Ungrouped') {
      return -1;
    }
    return a.localeCompare(b);
  });

  if (tools.length === 0) {
    return (
      <div
        style={{
          color: 'var(--text-color-secondary)',
          fontStyle: 'italic',
          textAlign: 'center',
          padding: '24px',
        }}
      >
        No MCP tools available
      </div>
    );
  }

  return (
    <div>
      {/* Search Filter */}
      <div style={{ marginBottom: '16px' }}>
        <input
          type="text"
          placeholder="Search tools..."
          value={searchFilter}
          onChange={(e) => setSearchFilter(e.target.value)}
          style={{
            width: '100%',
            padding: '8px 12px',
            border: '1px solid var(--border-color)',
            borderRadius: '4px',
            backgroundColor: 'var(--background-color-primary)',
            fontSize: '14px',
          }}
        />
      </div>

      {/* Tools List */}
      <div>
        {filteredTools.length === 0 ? (
          <div
            style={{
              color: 'var(--text-color-secondary)',
              fontStyle: 'italic',
              textAlign: 'center',
              padding: '24px',
            }}
          >
            No tools match your search
          </div>
        ) : (
          sortedCategories.map((category) => (
            <CollapsableSection key={category} label={`${category} (${groupedTools[category].length})`} isOpen={true}>
              <div style={{ marginTop: '8px' }}>
                {groupedTools[category].map((tool, i) => (
                  <ToolInspector key={i} tool={tool} />
                ))}
              </div>
            </CollapsableSection>
          ))
        )}
      </div>
    </div>
  );
}
