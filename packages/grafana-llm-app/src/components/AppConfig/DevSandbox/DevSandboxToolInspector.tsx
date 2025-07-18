import React, { useState } from 'react';
import { Button, Spinner, Icon, TextArea } from '@grafana/ui';
import { Tool } from '@modelcontextprotocol/sdk/types';
import { mcp } from '@grafana/llm';

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
  const [callResult, setCallResult] = useState<ToolCallResult | null>(null);

  const handleTestTool = async () => {
    if (!client) {
      return;
    }

    setCallResult({ loading: true });

    try {
      // Parse the parameters input
      let parameters = {};
      if (parametersInput.trim()) {
        try {
          parameters = JSON.parse(parametersInput);
        } catch (e) {
          setCallResult({
            loading: false,
            success: false,
            error: `Invalid JSON parameters: ${e instanceof Error ? e.message : 'Unknown error'}`
          });
          return;
        }
      }

      // Call the tool
      const response = await client.callTool({
        name: tool.name,
        arguments: parameters
      });

      setCallResult({
        loading: false,
        success: true,
        response
      });
    } catch (error) {
      setCallResult({
        loading: false,
        success: false,
        error: error instanceof Error ? error.message : 'Unknown error'
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

    Object.keys(properties).forEach(key => {
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
    <div style={{
      border: '1px solid var(--border-color)',
      borderRadius: '8px',
      marginBottom: '12px',
      backgroundColor: 'var(--background-color-primary)'
    }}>
      {/* Tool Header */}
      <div 
        style={{
          padding: '12px 16px',
          borderBottom: expanded ? '1px solid var(--border-color)' : 'none',
          cursor: 'pointer',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center'
        }}
        onClick={() => setExpanded(!expanded)}
      >
        <div>
          <div style={{ fontWeight: 500, fontSize: '15px', marginBottom: '4px' }}>
            {tool.annotations?.title ?? tool.name}
          </div>
          {tool.description && (
            <div style={{ 
              fontSize: '13px', 
              color: 'var(--text-color-secondary)',
              lineHeight: '1.4'
            }}>
              {tool.description}
            </div>
          )}
        </div>
        <Icon name={expanded ? 'angle-up' : 'angle-down'} />
      </div>

      {/* Expanded Content */}
      {expanded && (
        <div style={{ padding: '16px' }}>
          {/* Tool Schema */}
          <div style={{ marginBottom: '20px' }}>
            <h4 style={{ margin: '0 0 8px 0', fontSize: '14px' }}>Input Schema</h4>
            <pre style={{
              backgroundColor: 'var(--background-color-secondary)',
              padding: '12px',
              borderRadius: '4px',
              fontSize: '12px',
              lineHeight: '1.4',
              overflow: 'auto',
              maxHeight: '200px',
              border: '1px solid var(--border-color)'
            }}>
              {formatSchema(tool.inputSchema)}
            </pre>
          </div>

          {/* Parameter Input */}
          <div style={{ marginBottom: '16px' }}>
            <div style={{ 
              display: 'flex', 
              justifyContent: 'space-between', 
              alignItems: 'center',
              marginBottom: '8px'
            }}>
              <h4 style={{ margin: 0, fontSize: '14px' }}>Test Parameters</h4>
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
          </div>

          {/* Test Button */}
          <div style={{ marginBottom: '16px' }}>
            <Button
              variant="primary"
              size="sm"
              onClick={handleTestTool}
              disabled={callResult?.loading || !client}
            >
              {callResult?.loading ? <Spinner size="sm" /> : 'Test Tool'}
            </Button>
          </div>

          {/* Call Result */}
          {callResult && !callResult.loading && (
            <div style={{ marginTop: '16px' }}>
              <h4 style={{ margin: '0 0 8px 0', fontSize: '14px' }}>
                Result {callResult.success ? 
                  <Icon name="check" style={{ color: 'var(--success-color)', marginLeft: '8px' }} /> :
                  <Icon name="exclamation-triangle" style={{ color: 'var(--error-color)', marginLeft: '8px' }} />
                }
              </h4>
              
              {callResult.error ? (
                <div style={{
                  backgroundColor: 'var(--error-background)',
                  color: 'var(--error-text-color)',
                  padding: '12px',
                  borderRadius: '4px',
                  fontSize: '13px'
                }}>
                  {callResult.error}
                </div>
              ) : (
                <pre style={{
                  backgroundColor: 'var(--background-color-secondary)',
                  padding: '12px',
                  borderRadius: '4px',
                  fontSize: '12px',
                  lineHeight: '1.4',
                  overflow: 'auto',
                  maxHeight: '300px',
                  border: '1px solid var(--border-color)'
                }}>
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

  const filteredTools = tools.filter(tool => {
    const searchLower = searchFilter.toLowerCase();
    const name = (tool.annotations?.title ?? tool.name).toLowerCase();
    const description = (tool.description || '').toLowerCase();
    return name.includes(searchLower) || description.includes(searchLower);
  });

  if (tools.length === 0) {
    return (
      <div style={{ 
        color: 'var(--text-color-secondary)', 
        fontStyle: 'italic',
        textAlign: 'center',
        padding: '24px'
      }}>
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
            fontSize: '14px'
          }}
        />
      </div>

      {/* Tools List */}
      <div>
        {filteredTools.length === 0 ? (
          <div style={{ 
            color: 'var(--text-color-secondary)', 
            fontStyle: 'italic',
            textAlign: 'center',
            padding: '24px'
          }}>
            No tools match your search
          </div>
        ) : (
          filteredTools.map((tool, i) => (
            <ToolInspector key={i} tool={tool} />
          ))
        )}
      </div>
    </div>
  );
} 
