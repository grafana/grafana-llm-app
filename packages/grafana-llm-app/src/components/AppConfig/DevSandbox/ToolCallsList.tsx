import React from 'react';
import { Spinner, Icon, CollapsableSection } from '@grafana/ui';
import { RenderedToolCall } from './types';

export function ToolCallsList({ toolCalls }: { toolCalls: Map<string, RenderedToolCall> }) {
  return (
    <div>
      {toolCalls.size === 0 ? (
        <div
          style={{
            color: 'var(--text-color-secondary)',
            fontStyle: 'italic',
            textAlign: 'center',
            padding: '24px',
          }}
        >
          No tool calls yet
        </div>
      ) : (
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {Array.from(toolCalls.values()).map((toolCall, i) => (
            <li
              key={i}
              style={{
                marginBottom: '16px',
                padding: '12px',
                backgroundColor: 'var(--background-color-primary)',
                borderRadius: '8px',
                border: '1px solid var(--border-color)',
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                <span style={{ fontWeight: 500 }}>{toolCall.name}</span>
                <code
                  style={{
                    backgroundColor: 'var(--background-color-secondary)',
                    padding: '4px 8px',
                    borderRadius: '4px',
                    fontSize: '12px',
                  }}
                >
                  {toolCall.arguments}
                </code>
                {toolCall.running ? (
                  <Spinner size="sm" />
                ) : (
                  <Icon name="check" size="sm" style={{ color: 'var(--success-color)' }} />
                )}
              </div>
              {toolCall.error && (
                <div
                  style={{
                    backgroundColor: 'var(--error-background)',
                    color: 'var(--error-text-color)',
                    padding: '8px',
                    borderRadius: '4px',
                    marginTop: '8px',
                    fontSize: '14px',
                  }}
                >
                  <Icon name="exclamation-triangle" size="sm" style={{ marginRight: '4px' }} />
                  {toolCall.error}
                </div>
              )}
              {!toolCall.error && toolCall.response && (
                <CollapsableSection
                  label={<span style={{ fontSize: '14px', fontWeight: 500 }}>Response</span>}
                  isOpen={false}
                >
                  <pre
                    style={{
                      backgroundColor: 'var(--background-color-secondary)',
                      padding: '8px',
                      borderRadius: '4px',
                      marginTop: '8px',
                      overflow: 'auto',
                      maxHeight: '200px',
                      fontSize: '12px',
                    }}
                  >
                    {JSON.stringify(toolCall.response, null, 2)}
                  </pre>
                </CollapsableSection>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
