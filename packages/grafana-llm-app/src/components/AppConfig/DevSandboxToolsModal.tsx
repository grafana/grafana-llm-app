import React from 'react';
import { Button, Modal } from '@grafana/ui';
import { Tool } from '@modelcontextprotocol/sdk/types';

interface DevSandboxToolsModalProps {
  isOpen: boolean;
  onClose: () => void;
  tools: Tool[];
}

function AvailableToolsList({ tools }: { tools: Tool[] }) {
  return (
    <div>
      <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
        {tools.map((tool, i) => (
          <li key={i} style={{ 
            padding: '8px 0', 
            fontSize: '14px',
            borderBottom: '1px solid var(--border-color)',
            marginBottom: '8px'
          }}>
            <div style={{ fontWeight: 500, marginBottom: '4px' }}>
              {tool.annotations?.title ?? tool.name}
            </div>
            {tool.description && (
              <div style={{ 
                fontSize: '12px', 
                color: 'var(--text-color-secondary)',
                lineHeight: '1.4'
              }}>
                {tool.description}
              </div>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}

export function DevSandboxToolsModal({ isOpen, onClose, tools }: DevSandboxToolsModalProps) {
  return (
    <Modal title={`Available MCP Tools (${tools.length})`} isOpen={isOpen} onDismiss={onClose}>
      <div style={{ 
        maxHeight: '500px', 
        overflowY: 'auto',
        padding: '8px 0'
      }}>
        {tools.length === 0 ? (
          <div style={{ 
            color: 'var(--text-color-secondary)', 
            fontStyle: 'italic',
            textAlign: 'center',
            padding: '24px'
          }}>
            No MCP tools available
          </div>
        ) : (
          <AvailableToolsList tools={tools} />
        )}
      </div>
      <div style={{ marginTop: '16px', display: 'flex', justifyContent: 'flex-end' }}>
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
} 
