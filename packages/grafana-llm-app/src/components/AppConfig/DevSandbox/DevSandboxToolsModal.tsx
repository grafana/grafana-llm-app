import React from 'react';
import { Button, Modal } from '@grafana/ui';
import { Tool } from '@modelcontextprotocol/sdk/types';
import { DevSandboxToolInspector } from './DevSandboxToolInspector';

interface DevSandboxToolsModalProps {
  isOpen: boolean;
  onClose: () => void;
  tools: Tool[];
}

export function DevSandboxToolsModal({ isOpen, onClose, tools }: DevSandboxToolsModalProps) {
  return (
    <Modal title={`MCP Tool Inspector (${tools.length} tools)`} isOpen={isOpen} onDismiss={onClose}>
      <div style={{ 
        maxHeight: '600px', 
        overflowY: 'auto',
        padding: '8px 0'
      }}>
        <DevSandboxToolInspector tools={tools} />
      </div>
      <div style={{ marginTop: '16px', display: 'flex', justifyContent: 'flex-end' }}>
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
} 
