import React, { Suspense, useState } from 'react';
import { Button, Checkbox, FieldSet, LoadingPlaceholder, Modal, Tab, TabsBar } from '@grafana/ui';
import { useAsync } from 'react-use';
import { llm, mcp } from '@grafana/llm';
import { DevSandboxChat } from './DevSandboxChat';
import { RenderedToolCall } from './types';
import { DevSandboxToolInspector } from './DevSandboxToolInspector';

type SandboxTab = 'chat' | 'tools';

function DevSandboxContent() {
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
      </div>
    </>
  );
}

export const DevSandbox = () => {
  const [modalIsOpen, setModalIsOpen] = useState(false);

  const closeModal = () => {
    setModalIsOpen(false);
  };

  return (
    <FieldSet label="Development Sandbox">
      <Button onClick={() => setModalIsOpen(true)}>Open development sandbox</Button>
      <Modal title="Development Sandbox" isOpen={modalIsOpen} onDismiss={closeModal}>
        <Suspense fallback={<LoadingPlaceholder text="Loading MCP..." />}>
          <mcp.MCPClientProvider appName="Grafana App With Model Context Protocol" appVersion="1.0.0">
            <DevSandboxContent />
          </mcp.MCPClientProvider>
        </Suspense>

        <div style={{ marginTop: '16px', display: 'flex', justifyContent: 'flex-end' }}>
          <Button variant="primary" onClick={closeModal}>
            Close
          </Button>
        </div>
      </Modal>
    </FieldSet>
  );
};
