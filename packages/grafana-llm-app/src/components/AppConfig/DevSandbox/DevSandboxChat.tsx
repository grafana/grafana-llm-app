import React, { useState, useRef, useEffect } from 'react';
import { Button, Spinner, Stack, TextArea } from '@grafana/ui';
import { useAsync } from 'react-use';
import { finalize, lastValueFrom, partition, startWith } from 'rxjs';
import { llm, mcp } from '@grafana/llm';
import { CallToolResultSchema } from '@modelcontextprotocol/sdk/types';
import { RenderedToolCall } from './types';

interface ChatMessage {
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
}

interface DevSandboxChatProps {
  useStream: boolean;
  toolCalls: Map<string, RenderedToolCall>;
  setToolCalls: (calls: Map<string, RenderedToolCall>) => void;
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

export function DevSandboxChat({ useStream, toolCalls, setToolCalls }: DevSandboxChatProps) {
  const { client } = mcp.useMCPClient();

  // Chat state
  const [chatHistory, setChatHistory] = useState<ChatMessage[]>([]);
  const [currentInput, setCurrentInput] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);

  // Ref for auto-scrolling
  const chatContainerRef = useRef<HTMLDivElement>(null);

  // Auto-scroll to bottom when new messages are added
  useEffect(() => {
    if (chatContainerRef.current) {
      chatContainerRef.current.scrollTop = chatContainerRef.current.scrollHeight;
    }
  }, [chatHistory]);

  // Get available tools
  const {
    loading: toolsLoading,
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

  const handleStreamingChatWithHistory = async (messages: llm.Message[], tools: any[]) => {
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
    contentMessages.subscribe((content) => {
      setChatHistory((prev) =>
        prev.map((msg, idx) => (idx === prev.length - 1 && msg.role === 'assistant' ? { ...msg, content } : msg))
      );
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

      contentMessages.subscribe((content) => {
        setChatHistory((prev) =>
          prev.map((msg, idx) => (idx === prev.length - 1 && msg.role === 'assistant' ? { ...msg, content } : msg))
        );
      });

      toolCallMessages = await lastValueFrom(toolCallsStream.pipe(llm.accumulateToolCalls()));
    }
  };

  const handleNonStreamingChatWithHistory = async (messages: llm.Message[], tools: any[]) => {
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
    setChatHistory((prev) =>
      prev.map((msg, idx) =>
        idx === prev.length - 1 && msg.role === 'assistant'
          ? { ...msg, content: response.choices[0].message.content || '' }
          : msg
      )
    );
  };

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
    setChatHistory((prev) => [...prev, userMessage]);
    setCurrentInput('');
    setIsGenerating(true);
    setToolCalls(new Map());

    // Create assistant message placeholder
    const assistantMessage: ChatMessage = {
      role: 'assistant',
      content: '',
      timestamp: new Date(),
    };

    setChatHistory((prev) => [...prev, assistantMessage]);

    const messages: llm.Message[] = [
      {
        role: 'system',
        content:
          'You are a helpful assistant with deep knowledge of the Grafana, Prometheus and general observability ecosystem.',
      },
      ...chatHistory.map((msg) => ({ role: msg.role, content: msg.content })),
      { role: 'user', content: userMessage.content },
    ];

    try {
      if (useStream) {
        await handleStreamingChatWithHistory(messages, toolsData.tools);
      } else {
        await handleNonStreamingChatWithHistory(messages, toolsData.tools);
      }
    } catch (error) {
      console.error('Error in chat completion:', error);
      // Update the assistant message with error
      setChatHistory((prev) =>
        prev.map((msg, idx) =>
          idx === prev.length - 1 && msg.role === 'assistant'
            ? { ...msg, content: `Error: ${error instanceof Error ? error.message : 'Unknown error'}` }
            : msg
        )
      );
    } finally {
      setIsGenerating(false);
    }
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
                  width: '100%',
                }}
              >
                <div
                  style={{
                    maxWidth: '90%',
                    padding: '10px 14px',
                    borderRadius: '12px',
                    backgroundColor: message.role === 'user' ? '#007acc' : 'var(--background-color-primary)',
                    color: message.role === 'user' ? 'white' : 'var(--text-color-primary)',
                    whiteSpace: 'pre-wrap',
                    wordBreak: 'break-word',
                    boxShadow: '0 1px 2px rgba(0, 0, 0, 0.1)',
                    border: message.role === 'assistant' ? '1px solid var(--border-color)' : 'none',
                    fontSize: '14px',
                    lineHeight: '1.4',
                  }}
                >
                  {message.content ||
                    (message.role === 'assistant' && isGenerating && index === chatHistory.length - 1 ? (
                      <span style={{ opacity: 0.7 }}>...</span>
                    ) : (
                      ''
                    ))}
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
        <Button onClick={sendMessage} disabled={!currentInput.trim() || isGenerating || toolsLoading} variant="primary">
          {isGenerating ? <Spinner size="sm" /> : 'Send'}
        </Button>
      </Stack>
    </Stack>
  );
}
