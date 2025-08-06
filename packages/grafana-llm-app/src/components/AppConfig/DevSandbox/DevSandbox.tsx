import React, { Suspense, useState, useEffect } from 'react';
import { Button, Checkbox, FieldSet, LoadingPlaceholder, Modal, Tab, TabsBar, Field, Input, Alert } from '@grafana/ui';
import { useAsync } from 'react-use';
import { llm, mcp } from '@grafana/llm';
import { Client } from '@modelcontextprotocol/sdk/client/index';
import { StreamableHTTPClientTransport } from '@modelcontextprotocol/sdk/client/streamableHttp';
import { DevSandboxChat } from './DevSandboxChat';
import { RenderedToolCall } from './types';
import { DevSandboxToolInspector } from './DevSandboxToolInspector';
import { ToolCallsList } from './ToolCallsList';

type SandboxTab = 'chat' | 'tools' | 'tool_calls' | 'settings';

interface DevSandboxContentProps {
  onSettingsChange?: (settings: { useCustomServer: boolean; customUrl: string; authToken: string }) => void;
  currentSettings?: { useCustomServer: boolean; customUrl: string; authToken: string };
}

function DevSandboxContent({ onSettingsChange, currentSettings }: DevSandboxContentProps) {
  const { client } = mcp.useMCPClient();
  const [activeTab, setActiveTab] = useState<SandboxTab>('chat');

  // Main state
  const [useStream, setUseStream] = useState(true);
  const [toolCalls, setToolCalls] = useState<Map<string, RenderedToolCall>>(new Map());

  // Get available tools
  const {
    loading: _,
    error: toolsError,
    value: toolsData,
  } = useAsync(async () => {
    const enabled = await llm.enabled();
    if (!enabled) {
      return { enabled: false, tools: [] };
    }
    const { tools } = (await client?.listTools()) ?? { tools: [] };
    return { enabled: true, tools };
  }, [client]);

  if (toolsError) {
    return <div>Error: {toolsError.message}</div>;
  }

  if (!toolsData?.enabled) {
    return <div>LLM plugin not enabled.</div>;
  }

  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
        <TabsBar>
          <Tab label="Chat" active={activeTab === 'chat'} onChangeTab={() => setActiveTab('chat')} />
          <Tab
            label="Tool Inspector"
            active={activeTab === 'tools'}
            onChangeTab={() => setActiveTab('tools')}
            counter={toolsData?.tools?.length || 0}
          />
          <Tab
            label="Tool Calls"
            active={activeTab === 'tool_calls'}
            onChangeTab={() => setActiveTab('tool_calls')}
            counter={toolCalls.size}
          />
          <Tab label="Settings" active={activeTab === 'settings'} onChangeTab={() => setActiveTab('settings')} />
        </TabsBar>
        {activeTab === 'chat' && (
          <Checkbox label="Streaming" value={useStream} onChange={(e) => setUseStream(e.currentTarget.checked)} />
        )}
      </div>

      <div style={{ paddingTop: 10, minHeight: 400, display: 'flex', flexDirection: 'column' }}>
        {activeTab === 'chat' && (
          <DevSandboxChat useStream={useStream} toolCalls={toolCalls} setToolCalls={setToolCalls} />
        )}
        {activeTab === 'tools' && <DevSandboxToolInspector tools={toolsData?.tools || []} />}
        {activeTab === 'tool_calls' && <ToolCallsList toolCalls={toolCalls} />}
        {activeTab === 'settings' && <SettingsTab currentSettings={currentSettings} onSettingsChange={onSettingsChange} />}
      </div>
    </>
  );
}

// Settings tab component
interface SettingsTabProps {
  currentSettings?: { useCustomServer: boolean; customUrl: string; authToken: string };
  onSettingsChange?: (settings: { useCustomServer: boolean; customUrl: string; authToken: string }) => void;
}

function SettingsTab({ currentSettings, onSettingsChange }: SettingsTabProps) {
  const [useCustomServer, setUseCustomServer] = useState(currentSettings?.useCustomServer || false);
  const [customUrl, setCustomUrl] = useState(currentSettings?.customUrl || '');
  const [authToken, setAuthToken] = useState(currentSettings?.authToken || '');

  useEffect(() => {
    if (onSettingsChange) {
      onSettingsChange({ useCustomServer, customUrl, authToken });
    }
  }, [useCustomServer, customUrl, authToken, onSettingsChange]);

  return (
    <div style={{ maxWidth: '600px' }}>
      <h3 style={{ marginBottom: '16px' }}>MCP Server Configuration</h3>
      
      <div style={{ marginBottom: '16px' }}>
        <label style={{ display: 'flex', alignItems: 'center', cursor: 'pointer' }}>
          <Checkbox
            value={useCustomServer}
            onChange={(e) => setUseCustomServer(e.currentTarget.checked)}
          />
          <span style={{ marginLeft: '8px' }}>
            <strong>Use Custom MCP Server</strong> - Connect to a remote MCP server instead of the local one
          </span>
        </label>
      </div>
      
      {useCustomServer && (
        <>
          <Field 
            label="MCP Server URL" 
            description="Full URL to the MCP streamable HTTP endpoint"
            required
          >
            <Input
              value={customUrl}
              onChange={(e) => setCustomUrl(e.currentTarget.value)}
              placeholder="https://your-grafana.grafana.net/api/plugins/grafana-llm-app/resources/mcp/grafana"
            />
          </Field>
          
          <Field 
            label="Authentication Token" 
            description="Bearer token for authentication (e.g., Service Account token: glsa_xxx, or API key)"
            required
          >
            <Input
              type="password"
              value={authToken}
              onChange={(e) => setAuthToken(e.currentTarget.value)}
              placeholder="glsa_xxxxxxxxxxxxx"
            />
          </Field>
          
          <Alert title="Example Values" severity="info">
            <div style={{ fontSize: '12px' }}>
              <strong>URL Format:</strong> https://[your-stack].grafana.net/api/plugins/grafana-llm-app/resources/mcp/grafana<br/>
              <strong>Token:</strong> Create a service account token in your Grafana Cloud instance with appropriate permissions
            </div>
          </Alert>
          
          <Alert title="Security Note" severity="info">
            The authentication token will only be used within this dev sandbox session and won&apos;t be stored.
          </Alert>
        </>
      )}
    </div>
  );
}

// Custom MCP client wrapper
function CustomMCPWrapper({ children, customUrl, authToken }: { children: React.ReactNode; customUrl?: string; authToken?: string }) {
  const [customClient, setCustomClient] = useState<Client | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  
  // Store original useMCPClient function reference
  const [originalUseMCPClient] = useState(() => mcp.useMCPClient);

  useEffect(() => {
    if (!customUrl || !authToken) {
      setCustomClient(null);
      return;
    }

    let clientInstance: Client | null = null;
    const originalFetch = window.fetch;

    // Override fetch to add authorization header
    window.fetch = async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
      
      // Only add auth header to our custom MCP URL
      if (url && url.startsWith(customUrl)) {
        init = init || {};
        init.headers = {
          ...init.headers,
          'Authorization': `Bearer ${authToken}`,
        };
      }
      
      return originalFetch(input, init);
    };

    const createCustomClient = async () => {
      setLoading(true);
      setError(null);
      try {
        clientInstance = new Client({
          name: "DevSandbox Custom Client",
          version: "1.0.0",
        });

        // Create transport without headers (they'll be added by our fetch override)
        const transport = new StreamableHTTPClientTransport(
          new URL(customUrl),
          {
            reconnectionOptions: {
              maxRetries: 5,
              initialReconnectionDelay: 1000,
              maxReconnectionDelay: 5000,
              reconnectionDelayGrowFactor: 1.5,
            },
          }
        );

        await clientInstance.connect(transport);
        setCustomClient(clientInstance);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to connect to custom MCP server');
        setCustomClient(null);
      } finally {
        setLoading(false);
      }
    };

    createCustomClient();

    return () => {
      // Restore original fetch
      window.fetch = originalFetch;
      
      if (clientInstance) {
        clientInstance.close();
      }
    };
  }, [customUrl, authToken]);

  // Override and restore the mcp.useMCPClient hook
  useEffect(() => {
    if (customClient) {
      mcp.useMCPClient = () => ({ client: customClient, enabled: !!customClient });
    }
    
    return () => {
      mcp.useMCPClient = originalUseMCPClient;
    };
  }, [customClient, originalUseMCPClient]);

  if (error) {
    return <Alert title="Custom MCP Connection Error" severity="error">{error}</Alert>;
  }

  if (loading) {
    return <LoadingPlaceholder text="Connecting to custom MCP server..." />;
  }

  return <>{children}</>;
}

export const DevSandbox = () => {
  const [modalIsOpen, setModalIsOpen] = useState(false);
  const [settings, setSettings] = useState({
    useCustomServer: false,
    customUrl: '',
    authToken: '',
  });

  const closeModal = () => {
    setModalIsOpen(false);
  };

  const handleSettingsChange = (newSettings: typeof settings) => {
    setSettings(newSettings);
  };

  return (
    <FieldSet label="Development Sandbox">
      <Button onClick={() => setModalIsOpen(true)}>Open development sandbox</Button>
      <Modal title="Development Sandbox" isOpen={modalIsOpen} onDismiss={closeModal}>
        {settings.useCustomServer && settings.customUrl && settings.authToken ? (
          <CustomMCPWrapper customUrl={settings.customUrl} authToken={settings.authToken}>
            <DevSandboxContent onSettingsChange={handleSettingsChange} currentSettings={settings} />
          </CustomMCPWrapper>
        ) : (
          <Suspense fallback={<LoadingPlaceholder text="Loading MCP..." />}>
            <mcp.MCPClientProvider appName="Grafana App With Model Context Protocol" appVersion="1.0.0">
              <DevSandboxContent onSettingsChange={handleSettingsChange} currentSettings={settings} />
            </mcp.MCPClientProvider>
          </Suspense>
        )}

        <div style={{ marginTop: '16px', display: 'flex', justifyContent: 'flex-end' }}>
          <Button variant="primary" onClick={closeModal}>
            Close
          </Button>
        </div>
      </Modal>
    </FieldSet>
  );
};
