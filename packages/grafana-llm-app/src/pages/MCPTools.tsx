import React from 'react';
import { useAsync } from 'react-use';
import { mcp } from '@grafana/llm';
import { ErrorBoundary } from '@grafana/ui';
import { testIds } from '../components/testIds';

export function MCPToolsList() {
  const { client, enabled } = mcp.useMCPClient();

  const { error, loading, value } = useAsync(async () => {
    if (!enabled || !client) {
      return null;
    }
    const result = await client.listTools();
    return result.tools;
  }, [client, enabled]);

  if (!enabled) {
    return (
      <div data-testid={testIds.mcpTools.container}>
        <p data-testid={testIds.mcpTools.disabled}>MCP is not enabled in this Grafana instance.</p>
      </div>
    );
  }

  if (loading) {
    return (
      <div data-testid={testIds.mcpTools.container}>
        <span data-testid={testIds.mcpTools.loading}>Loading MCP tools...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div data-testid={testIds.mcpTools.container}>
        <span data-testid={testIds.mcpTools.error}>Error loading MCP tools: {error.message}</span>
      </div>
    );
  }

  if (!value || value.length === 0) {
    return (
      <div data-testid={testIds.mcpTools.container}>
        <p data-testid={testIds.mcpTools.empty}>No MCP tools available.</p>
      </div>
    );
  }

  return (
    <div data-testid={testIds.mcpTools.container} style={{ maxHeight: '400px', overflowY: 'auto' }}>
      <h2>Available MCP Tools</h2>
      <ul data-testid={testIds.mcpTools.list}>
        {value.map((tool: any) => (
          <li key={tool.name} data-testid={testIds.mcpTools.toolItem}>
            <strong data-testid={testIds.mcpTools.toolName}>{tool.name}</strong>
            {tool.description && <span data-testid={testIds.mcpTools.toolDescription}> - {tool.description}</span>}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function MCPToolsWithProvider() {
  return (
    <React.Suspense fallback={<div data-testid={testIds.mcpTools.container}>Connecting to MCP server...</div>}>
      <ErrorBoundary>
        {({ error }) => {
          if (error) {
            return (
              <div data-testid={testIds.mcpTools.container}>
                <span data-testid={testIds.mcpTools.error}>Error loading MCP tools: {error.message}</span>
              </div>
            );
          }
          return (
            <mcp.MCPClientProvider appName="grafana-llm-app" appVersion="0.21.1">
              <MCPToolsList />
            </mcp.MCPClientProvider>
          );
        }}
      </ErrorBoundary>
    </React.Suspense>
  );
}
