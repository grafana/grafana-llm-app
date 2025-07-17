import React, { Suspense, useState } from 'react';
import {
  Button,
  FieldSet,
  Icon,
  LoadingPlaceholder,
  Modal,
  Spinner,
  Stack,
  TextArea,
  CollapsableSection,
} from '@grafana/ui';
import { useAsync } from 'react-use';
import { finalize, lastValueFrom, partition, startWith } from 'rxjs';
import { llm, mcp } from '@grafana/llm';
import { CallToolResultSchema, Tool } from '@modelcontextprotocol/sdk/types';

interface RenderedToolCall {
  name: string;
  arguments: string;
  running: boolean;
  error?: string;
  response?: any;
}

// Helper function to handle tool calls
async function handleToolCall(
  fc: { function: { name: string; arguments: string }; id: string },
  client: any,
  toolCalls: Map<string, RenderedToolCall>,
  setToolCalls: (calls: Map<string, RenderedToolCall>) => void,
  messages: llm.Message[]
) {
  const { function: f, id } = fc;
  console.log('f', f);

  setToolCalls(new Map(toolCalls.set(id, { name: f.name, arguments: f.arguments, running: true })));

  const args = JSON.parse(f.arguments);

  try {
    const response = await client.callTool({ name: f.name, arguments: args });
    const toolResult = CallToolResultSchema.parse(response);
    const textContent = toolResult.content
      .filter((c) => c.type === 'text')
      .map((c) => c.text)
      .join('');
    messages.push({ role: 'tool', tool_call_id: id, content: textContent });
    setToolCalls(new Map(toolCalls.set(id, { name: f.name, arguments: f.arguments, running: false, response })));
  } catch (e: any) {
    const error = e.message ?? e.toString();
    messages.push({ role: 'tool', tool_call_id: id, content: error });
    setToolCalls(new Map(toolCalls.set(id, { name: f.name, arguments: f.arguments, running: false, error })));
  }
}



function AvailableTools({ tools }: { tools: Tool[] }) {
  return (
    <div>
      <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
        {tools.map((tool, i) => (
          <li key={i} style={{ 
            padding: '4px 0', 
            fontSize: '13px',
            borderBottom: '1px solid var(--border-color)',
            marginBottom: '4px'
          }}>
            {tool.annotations?.title ?? tool.name}
          </li>
        ))}
      </ul>
    </div>
  );
}

function ToolCalls({ toolCalls }: { toolCalls: Map<string, RenderedToolCall> }) {
  return (
    <div>
      {toolCalls.size === 0 ? (
        <div style={{ 
          color: 'var(--text-color-secondary)', 
          fontStyle: 'italic'
        }}>
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
                border: '1px solid var(--border-color)'
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
                <span style={{ fontWeight: 500 }}>{toolCall.name}</span>
                <code
                  style={{ 
                    backgroundColor: 'var(--background-color-secondary)', 
                    padding: '4px 8px', 
                    borderRadius: '4px',
                    fontSize: '12px'
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

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

const BasicChatTest = ({
  showAvailableTools,
  setShowAvailableTools,
  showToolCalls,
  setShowToolCalls,
  useStream,
  setUseStream,
}: {
  showAvailableTools: boolean;
  setShowAvailableTools: (show: boolean) => void;
  showToolCalls: boolean;
  setShowToolCalls: (show: boolean) => void;
  useStream: boolean;
  setUseStream: (useStream: boolean) => void;
}) => {
  const { client } = mcp.useMCPClient();
  // Chat state
  const [chatHistory, setChatHistory] = useState<ChatMessage[]>([]);
  const [currentInput, setCurrentInput] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  
  // Tool state
  const [toolCalls, setToolCalls] = useState<Map<string, RenderedToolCall>>(new Map());
  
  // Ref for auto-scrolling
  const chatContainerRef = React.useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages are added
  React.useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [chatHistory]);

  // Get available tools
  const { loading: toolsLoading, error: toolsError, value: toolsData } = useAsync(async () => {
    const enabled = await llm.enabled();
    if (!enabled) {
      return { enabled: false, tools: [] };
    }
    const { tools } = (await client?.listTools()) ?? { tools: [] };
    return { enabled: true, tools };
  }, [client]);

  const sendMessage = async () => {
    if (!currentInput.trim() || isGenerating || !toolsData?.enabled) {
      return;
    }

    const userMessage: ChatMessage = {
      role: 'user',
      content: currentInput.trim(),
      timestamp: new Date(),
    };

    // Add user message to history
    setChatHistory(prev => [...prev, userMessage]);
    setCurrentInput('');
    setIsGenerating(true);
    setToolCalls(new Map());

    // Create assistant message placeholder
    const assistantMessage: ChatMessage = {
      role: 'assistant',
      content: '',
      timestamp: new Date(),
    };

    setChatHistory(prev => [...prev, assistantMessage]);

    const messages: llm.Message[] = [
      {
        role: 'system',
        content:
          'You are a helpful assistant with deep knowledge of the Grafana, Prometheus and general observability ecosystem.',
      },
      ...chatHistory.map(msg => ({ role: msg.role, content: msg.content })),
      { role: 'user', content: userMessage.content },
    ];

    try {
      if (useStream) {
        await handleStreamingChatWithHistory(messages, toolsData.tools, assistantMessage);
      } else {
        await handleNonStreamingChatWithHistory(messages, toolsData.tools, assistantMessage);
      }
    } catch (error) {
      console.error('Error in chat completion:', error);
      // Update the assistant message with error
      setChatHistory(prev => prev.map((msg, idx) => 
        idx === prev.length - 1 && msg.role === 'assistant' 
          ? { ...msg, content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}` }
          : msg
      ));
    } finally {
      setIsGenerating(false);
    }
  };

  const handleStreamingChatWithHistory = async (
    messages: llm.Message[],
    tools: any[],
    assistantMessage: ChatMessage
  ) => {
    let stream = llm.streamChatCompletions({
      model: llm.Model.LARGE,
      messages,
      tools: mcp.convertToolsToOpenAI(tools),
    });

    let [toolCallsStream, otherMessages] = partition(
      stream,
      (chunk: llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>) => llm.isToolCallsMessage(chunk.choices[0].delta)
    );

    let contentMessages = otherMessages.pipe(
      llm.accumulateContent(),
      finalize(() => {
        console.log('stream finalized');
      })
    );

    // Subscribe to content updates
    contentMessages.subscribe(content => {
      setChatHistory(prev => prev.map((msg, idx) => 
        idx === prev.length - 1 && msg.role === 'assistant' 
          ? { ...msg, content }
          : msg
      ));
    });

    let toolCallMessages = await lastValueFrom(toolCallsStream.pipe(llm.accumulateToolCalls()));

    while (toolCallMessages.tool_calls.length > 0) {
      messages.push(toolCallMessages);

      const tcs = toolCallMessages.tool_calls.filter((tc) => tc.type === 'function');
      await Promise.all(tcs.map((fc) => handleToolCall(fc, client, toolCalls, setToolCalls, messages)));

      stream = llm.streamChatCompletions({
        model: llm.Model.LARGE,
        messages,
        tools: mcp.convertToolsToOpenAI(tools),
      });

      [toolCallsStream, otherMessages] = partition(
        stream,
        (chunk: llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>) => llm.isToolCallsMessage(chunk.choices[0].delta)
      );

      const firstMessage: Partial<llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>> = {
        choices: [{ delta: { role: 'assistant', content: '' } }],
      };

      contentMessages = otherMessages.pipe(
        //@ts-expect-error
        startWith(firstMessage),
        llm.accumulateContent(),
        finalize(() => {
          console.log('stream finalized');
        })
      );

      contentMessages.subscribe(content => {
        setChatHistory(prev => prev.map((msg, idx) => 
          idx === prev.length - 1 && msg.role === 'assistant' 
            ? { ...msg, content }
            : msg
        ));
      });

      toolCallMessages = await lastValueFrom(toolCallsStream.pipe(llm.accumulateToolCalls()));
    }
  };

  const handleNonStreamingChatWithHistory = async (
    messages: llm.Message[],
    tools: any[],
    assistantMessage: ChatMessage
  ) => {
    let response = await llm.chatCompletions({
      model: llm.Model.BASE,
      messages,
      tools: mcp.convertToolsToOpenAI(tools),
    });

    let functionCalls = response.choices[0].message.tool_calls?.filter((tc) => tc.type === 'function') ?? [];

    while (functionCalls.length > 0) {
      messages.push(response.choices[0].message);
      await Promise.all(functionCalls.map((fc) => handleToolCall(fc, client, toolCalls, setToolCalls, messages)));

      response = await llm.chatCompletions({
        model: llm.Model.LARGE,
        messages,
        tools: mcp.convertToolsToOpenAI(tools),
      });
      functionCalls = response.choices[0].message.tool_calls?.filter((tc) => tc.type === 'function') ?? [];
    }

    // Update the assistant message in history
    setChatHistory(prev => prev.map((msg, idx) => 
      idx === prev.length - 1 && msg.role === 'assistant' 
        ? { ...msg, content: response.choices[0].message.content || '' }
        : msg
    ));
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  };

  if (toolsError) {
    return <div>Error: {toolsError.message}</div>;
  }

  if (!toolsData?.enabled) {
    return <div>LLM plugin not enabled.</div>;
  }

  return (
    <div>
      <Stack direction="column" gap={3}>
        {/* Chat history */}
        <div 
          ref={chatContainerRef}
          style={{ 
            height: '400px', 
            overflowY: 'auto', 
            border: '1px solid var(--border-color)', 
            borderRadius: '8px',
            padding: '16px',
            backgroundColor: 'var(--background-color-secondary)',
          }}
        >
          {chatHistory.length === 0 ? (
            <div style={{ color: 'var(--text-color-secondary)', fontStyle: 'italic' }}>
              Start a conversation by typing a message below...
            </div>
          ) : (
            <Stack direction="column" gap={1}>
              {chatHistory.map((message, index) => (
                <div
                  key={index}
                  style={{
                    display: 'flex',
                    flexDirection: 'row',
                    marginBottom: '12px',
                    width: '100%'
                  }}
                >
                  <div
                    style={{
                      maxWidth: '90%',
                      padding: '10px 14px',
                      borderRadius: '12px',
                      backgroundColor: message.role === 'user' 
                        ? '#007acc' 
                        : 'var(--background-color-primary)',
                      color: message.role === 'user' 
                        ? 'white' 
                        : 'var(--text-color-primary)',
                      whiteSpace: 'pre-wrap',
                      wordBreak: 'break-word',
                      boxShadow: '0 1px 2px rgba(0, 0, 0, 0.1)',
                      border: message.role === 'assistant' ? '1px solid var(--border-color)' : 'none',
                      fontSize: '14px',
                      lineHeight: '1.4'
                    }}
                  >
                    {message.content || (message.role === 'assistant' && isGenerating && index === chatHistory.length - 1 ? 
                      <span style={{ opacity: 0.7 }}>...</span> : '')}
                  </div>
                </div>
              ))}
            </Stack>
          )}
        </div>

        {/* Input area */}
        <Stack direction="row" gap={2}>
          <TextArea
            value={currentInput}
            onChange={(e) => setCurrentInput(e.currentTarget.value)}
            onKeyDown={handleKeyPress}
            placeholder="Type your message... (Enter to send, Shift+Enter for new line)"
            disabled={isGenerating}
            style={{ flex: 1 }}
            rows={3}
          />
          <Button
            onClick={sendMessage}
            disabled={!currentInput.trim() || isGenerating || toolsLoading}
            variant="primary"
          >
            {isGenerating ? <Spinner size="sm" /> : 'Send'}
          </Button>
        </Stack>

        {/* Collapsible Available Tools */}
        {showAvailableTools && (
          <div style={{ 
            border: '1px solid var(--border-color)', 
            borderRadius: '8px',
            backgroundColor: 'var(--background-color-secondary)',
            maxHeight: '200px',
            overflowY: 'auto'
          }}>
            <div style={{ 
              padding: '12px', 
              borderBottom: '1px solid var(--border-color)',
              backgroundColor: 'var(--background-color-primary)'
            }}>
              <h4 style={{ margin: 0, fontSize: '14px', fontWeight: 600 }}>Available MCP Tools</h4>
            </div>
            <div style={{ padding: '12px' }}>
              <AvailableTools tools={toolsData?.tools || []} />
            </div>
          </div>
        )}

        {/* Collapsible Tool Calls */}
        {showToolCalls && (
          <div style={{ 
            border: '1px solid var(--border-color)', 
            borderRadius: '8px',
            backgroundColor: 'var(--background-color-secondary)',
            maxHeight: '300px',
            overflowY: 'auto'
          }}>
            <div style={{ 
              padding: '12px', 
              borderBottom: '1px solid var(--border-color)',
              backgroundColor: 'var(--background-color-primary)'
            }}>
              <h4 style={{ margin: 0, fontSize: '14px', fontWeight: 600 }}>Tool Calls</h4>
            </div>
            <div style={{ padding: '12px' }}>
              <ToolCalls toolCalls={toolCalls} />
            </div>
          </div>
        )}
      </Stack>
    </div>
  );
};

export const DevSandbox = () => {
  const [modalIsOpen, setModalIsOpen] = useState(false);
  
  // Move toggle states up to this level
  const [showAvailableTools, setShowAvailableTools] = useState(false);
  const [showToolCalls, setShowToolCalls] = useState(false);
  const [useStream, setUseStream] = useState(true);
  
  const closeModal = () => {
    setModalIsOpen(false);
  };

  return (
    <FieldSet label="Development Sandbox">
      <Button onClick={() => setModalIsOpen(true)}>Open development sandbox</Button>
      <Modal 
        title="Development Sandbox" 
        isOpen={modalIsOpen} 
        onDismiss={closeModal}
      >
        <Suspense fallback={<LoadingPlaceholder text="Loading MCP..." />}>
          <mcp.MCPClientProvider appName="Grafana App With Model Context Protocol" appVersion="1.0.0">
            <BasicChatTest 
              showAvailableTools={showAvailableTools}
              setShowAvailableTools={setShowAvailableTools}
              showToolCalls={showToolCalls}
              setShowToolCalls={setShowToolCalls}
              useStream={useStream}
              setUseStream={setUseStream}
            />
          </mcp.MCPClientProvider>
        </Suspense>
        
        {/* Bottom controls row */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '16px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            {/* Streaming toggle */}
            <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <label htmlFor="stream-toggle" style={{ fontSize: '14px' }}>Streaming:</label>
              <input
                id="stream-toggle"
                type="checkbox"
                checked={useStream}
                onChange={(e) => setUseStream(e.target.checked)}
              />
            </div>
            
            {/* Tool toggle buttons */}
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setShowAvailableTools(!showAvailableTools)}
              icon={showAvailableTools ? 'angle-up' : 'angle-down'}
            >
              Available Tools
            </Button>
            <Button
              variant="secondary"
              size="sm"
              onClick={() => setShowToolCalls(!showToolCalls)}
              icon={showToolCalls ? 'angle-up' : 'angle-down'}
            >
              Tool Calls
            </Button>
          </div>
          
          <Button variant="primary" onClick={closeModal}>
            Close
          </Button>
        </div>
      </Modal>
    </FieldSet>
  );
};
