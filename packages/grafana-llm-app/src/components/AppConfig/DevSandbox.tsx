import React, { Suspense, useCallback, useState } from "react";
import { Button, FieldSet, Icon, LoadingPlaceholder, Modal, Spinner, Stack, TextArea, CollapsableSection } from "@grafana/ui";
import { llm, mcp } from "@grafana/llm";
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

  try{
  const response = await client.callTool({ name: f.name, arguments: args });
  const toolResult = CallToolResultSchema.parse(response);
    const textContent = toolResult.content.filter(c => c.type === 'text').map(c => c.text).join('');
    messages.push({ role: 'tool', tool_call_id: id, content: textContent });
    setToolCalls(new Map(toolCalls.set(id, { name: f.name, arguments: f.arguments, running: false, response })));
  } catch (e: any) {
    const error = e.message ?? e.toString();
    messages.push({ role: 'tool', tool_call_id: id, content: error });
    setToolCalls(new Map(toolCalls.set(id, { name: f.name, arguments: f.arguments, running: false, error })));
  }
}

// Helper function to handle non-streaming chat completions
async function handleNonStreamingChat(
  message: string,
  tools: Tool[],
  client: any,
  toolCalls: Map<string, RenderedToolCall>,
  setToolCalls: (calls: Map<string, RenderedToolCall>) => void,
  setReply: (reply: string) => void,
  setStarted: (started: boolean) => void,
  setFinished: (finished: boolean) => void
) {
  setToolCalls(new Map());
  const messages: llm.Message[] = [
    { role: 'system', content: 'You are a helpful assistant with deep knowledge of the Grafana, Prometheus and general observability ecosystem.' },
    { role: 'user', content: message },
  ];

  let response = await llm.chatCompletions({
    model: llm.Model.BASE,
    messages,
    tools: mcp.convertToolsToOpenAI(tools),
  });

  let functionCalls = response.choices[0].message.tool_calls?.filter(tc => tc.type === 'function') ?? [];
  
  while (functionCalls.length > 0) {
    messages.push(response.choices[0].message);
    await Promise.all(functionCalls.map(fc => handleToolCall(fc, client, toolCalls, setToolCalls, messages)));
    
    response = await llm.chatCompletions({
      model: llm.Model.LARGE,
      messages,
      tools: mcp.convertToolsToOpenAI(tools),
    });
    functionCalls = response.choices[0].message.tool_calls?.filter(tc => tc.type === 'function') ?? [];
  }

  setReply(response.choices[0].message.content!);
  setStarted(false);
  setFinished(true);
}

function AvailableTools({ tools }: { tools: Tool[] }) {
  return (
    <Stack direction="column">
      <h4>Available MCP Tools</h4>
      <ul>
        {tools.map((tool, i) => (
          <li key={i}>{tool.name}</li>
        ))}
      </ul>
    </Stack>
  );
}

function ToolCalls({ toolCalls }: { toolCalls: Map<string, RenderedToolCall> }) {
  return (
    <div>
      <h4>Tool Calls</h4>
      {toolCalls.size === 0 && <div>No tool calls yet</div>}
      <ul style={{ listStyle: 'none', padding: 0 }}>
        {Array.from(toolCalls.values()).map((toolCall, i) => (
          <li key={i} style={{ marginBottom: '16px', padding: '12px', backgroundColor: 'var(--background-color-secondary)', borderRadius: '4px' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
              <span style={{ fontWeight: 500 }}>{toolCall.name}</span>
              <code style={{ backgroundColor: 'var(--background-color-primary)', padding: '2px 6px', borderRadius: '4px' }}>
                {toolCall.arguments}
              </code>
              {toolCall.running ? (
                <Spinner size="sm" />
              ) : (
                <Icon name="check" size="sm" style={{ color: 'var(--success-color)' }} />
              )}
            </div>
            {toolCall.error && (
              <div style={{ 
                backgroundColor: 'var(--error-background)', 
                color: 'var(--error-text-color)',
                padding: '8px',
                borderRadius: '4px',
                marginTop: '4px',
                fontSize: '0.9em'
              }}>
                <Icon name="exclamation-triangle" size="sm" style={{ marginRight: '4px' }} />
                {toolCall.error}
              </div>
            )}
            {!toolCall.error && toolCall.response && (
              <CollapsableSection 
                label={<span style={{ fontSize: '0.7em', fontWeight: 500 }}>Response</span>} 
                isOpen={false}
              >
                <pre style={{ 
                  backgroundColor: 'var(--background-color-primary)', 
                  padding: '8px',
                  borderRadius: '4px',
                  marginTop: '8px',
                  overflow: 'auto',
                  maxHeight: '300px',
                  fontSize: '0.9em'
                }}>
                  {JSON.stringify(toolCall.response, null, 2)}
                </pre>
              </CollapsableSection>
            )}
          </li>
        ))}
      </ul>
    </div>
  );
}

const BasicChatTest = () => {
  const client = mcp.useMCPClient();
  // The current input value.
  const [input, setInput] = useState('');
  // The latest reply from the LLM.
  const [reply, setReply] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [tools, setTools] = useState<Tool[]>([]);

  const [toolCalls, setToolCalls] = useState<Map<string, RenderedToolCall>>(new Map());

  const [started, setStarted] = useState(false);
  const [finished, setFinished] = useState(true);

  // Initialize the hook for streaming mode, but we'll only use it when useStream is true
  const {
    setMessages,
    reply: streamReply,
    streamStatus,
    error,
    toolCalls: streamToolCalls,
    isEnabled
  } = llm.useLLMStreamWithTools(
    client,
    llm.Model.LARGE,
    1,
    (title) => console.error(title),
    tools,
    "You are a helpful assistant with deep knowledge of the Grafana, Prometheus and general observability ecosystem."
  );

  // When the component mounts, check if LLM is enabled and fetch available tools
  React.useEffect(() => {
    const initializeTools = async () => {
      try {
        if (!isEnabled) {
          setIsLoading(false);
          return;
        }

        const { tools: availableTools } = await client.listTools();
        setTools(availableTools);
        setIsLoading(false);
      } catch (error) {
        console.error('Error initializing tools:', error);
      }
      setIsLoading(false);
    };

    if (isEnabled !== undefined) {
      initializeTools();
    }
  }, [client, isEnabled]);

    // Handle form submission
  const handleStreamingSubmit = useCallback(() => {
    if (!input.trim()) {
        return;
    }
        
    setMessages([{ role: 'user', content: input }]);
    setInput('');
  }, [input, setMessages, setInput]);


  // Update UI state based on stream status
  React.useEffect(() => {
      console.log("Stream status effect")
      // Update started/finished based on streamStatus
      if (streamStatus === llm.StreamStatus.GENERATING) {
        setStarted(true);
        setFinished(false);
      } else if (streamStatus === llm.StreamStatus.COMPLETED || streamStatus === llm.StreamStatus.IDLE) {
        setStarted(false);
        setFinished(true);
      }
  }, [streamStatus, setStarted, setFinished]);

  // Show the reply from the hook when streaming
  React.useEffect(() => {
    console.log("Reply effect")
    if (streamReply) {
      setReply(streamReply);
    }
  }, [streamReply, setReply]);

  // Show tool calls from the hook when streaming
  React.useEffect(() => {
    console.log("Tool calls effect")
    if (streamToolCalls.size > 0) {
      // Convert from the hook's tool call format to our RenderedToolCall format
      const convertedToolCalls = new Map<string, RenderedToolCall>();
      streamToolCalls.forEach((call, id) => {
        convertedToolCalls.set(id, {
          name: call.name,
          arguments: call.arguments,
          running: call.running,
          error: call.error,
          response: call.response
        });
      });
      setToolCalls(convertedToolCalls); 
    }
  }, [streamToolCalls, setToolCalls]);



  if (error) {
    return <div>Error: {error.message}</div>;
  }

  return (
    <div>
      {isEnabled ? (
        <Stack direction="column">
          <TextArea
            value={input}
            onChange={(e) => setInput(e.currentTarget.value)}
            placeholder="Enter a message"
          />
          <br />
          <Stack direction="row" justifyContent="space-evenly">
            <Button type="submit" onClick={() => { handleStreamingSubmit(); }}>Submit Stream</Button>
            <Button type="submit" onClick={() => { handleNonStreamingChat(input, tools, client, toolCalls, setToolCalls, setReply, setStarted, setFinished); }}>Submit Request</Button>
          </Stack>
          <br />
          {isLoading && <Spinner />}
          {!isLoading && (
            <div>
              <p style={{ whiteSpace: 'pre-wrap' }}>{reply}</p>
            </div>
          )}
          <Stack direction="row" justifyContent="space-evenly">
            <div>{started ? "Response is started" : "Response is not started"}</div>
            <div>{finished ? "Response is finished" : "Response is not finished"}</div>
          </Stack>
          <br />
          <br />
          <Stack direction="row" justifyContent="space-evenly">
            <AvailableTools tools={tools} />
            <ToolCalls toolCalls={toolCalls} />
          </Stack>
        </Stack>
      ) : (
        <div>LLM plugin not enabled.</div>
      )}
    </div>
  );
};

export const DevSandbox = () => {
  const [modalIsOpen, setModalIsOpen] = useState(false);
  const closeModal = () => {
    setModalIsOpen(false);
  }

  return (
    <FieldSet label="Development Sandbox">
      <Button onClick={() => setModalIsOpen(true)}>Open development sandbox</Button>
      <Modal title="Development Sandbox" isOpen={modalIsOpen} onDismiss={closeModal}>
        <Suspense fallback={<LoadingPlaceholder text="Loading MCP..." />}>
          <mcp.MCPClientProvider
            appName="Grafana App With Model Context Protocol"
            appVersion="1.0.0"
          >
            <BasicChatTest />
          </mcp.MCPClientProvider>
        </Suspense>
        <Button variant="primary" onClick={closeModal}>
          Close
        </Button>
      </Modal>
    </FieldSet>
  );
};
