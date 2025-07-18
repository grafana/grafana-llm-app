import React, { Suspense, useState } from 'react';
import { Button, FieldSet, LoadingPlaceholder, Modal } from '@grafana/ui';
import { useAsync } from 'react-use';
import { llm, mcp } from '@grafana/llm';
import { DevSandboxChat } from './DevSandboxChat';
import { DevSandboxToolsModal } from './DevSandboxToolsModal';
import { DevSandboxToolCallsModal, RenderedToolCall } from './DevSandboxToolCallsModal';

function DevSandboxContent() {
  const { client } = mcp.useMCPClient();

  // Main state
  const [useStream, setUseStream] = useState(true);
  const [toolCalls, setToolCalls] = useState<Map<string, RenderedToolCall>>(new Map());

  // Modal states
  const [showToolsModal, setShowToolsModal] = useState(false);
  const [showToolCallsModal, setShowToolCallsModal] = useState(false);

  // Get available tools for the modals
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
      <DevSandboxChat useStream={useStream} toolCalls={toolCalls} setToolCalls={setToolCalls} />

      {/* Bottom controls row */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '16px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          {/* Streaming toggle */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
            <label htmlFor="stream-toggle" style={{ fontSize: '14px' }}>
              Streaming:
            </label>
            <input
              id="stream-toggle"
              type="checkbox"
              checked={useStream}
              onChange={(e) => setUseStream(e.target.checked)}
            />
          </div>

          {/* Tool modal buttons */}
          <Button variant="secondary" size="sm" onClick={() => setShowToolsModal(true)}>
            Tool Inspector ({toolsData?.tools?.length || 0})
          </Button>
          <Button variant="secondary" size="sm" onClick={() => setShowToolCallsModal(true)}>
            Tool Calls ({toolCalls.size})
          </Button>
        </div>
      </div>

      {/* Tool Modals */}
      <DevSandboxToolsModal
        isOpen={showToolsModal}
        onClose={() => setShowToolsModal(false)}
        tools={toolsData?.tools || []}
      />

      <DevSandboxToolCallsModal
        isOpen={showToolCallsModal}
        onClose={() => setShowToolCallsModal(false)}
        toolCalls={toolCalls}
      />
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
