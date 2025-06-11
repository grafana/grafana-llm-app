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
    {
      role: 'system',
      content:
        'You are a helpful assistant with deep knowledge of the Grafana, Prometheus and general observability ecosystem.',
    },
    { role: 'user', content: message },
  ];

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

  setReply(response.choices[0].message.content!);
  setStarted(false);
  setFinished(true);
}

// Helper function to handle streaming chat completions
async function handleStreamingChat(
  message: string,
  tools: Tool[],
  client: any,
  toolCalls: Map<string, RenderedToolCall>,
  setToolCalls: (calls: Map<string, RenderedToolCall>) => void,
  setReply: (reply: string) => void,
  setStarted: (started: boolean) => void,
  setFinished: (finished: boolean) => void
) {
  const messages: llm.Message[] = [
    {
      role: 'system',
      content:
        'You are a helpful assistant with deep knowledge of the Grafana, Prometheus and general observability ecosystem.',
    },
    { role: 'user', content: message },
  ];

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
      setStarted(false);
      setFinished(true);
    })
  );

  // Subscribe to content messages immediately
  contentMessages.subscribe(setReply);

  let toolCallMessages = await lastValueFrom(toolCallsStream.pipe(llm.accumulateToolCalls()));

  while (toolCallMessages.tool_calls.length > 0) {
    messages.push(toolCallMessages);

    const tcs = toolCallMessages.tool_calls.filter((tc) => tc.type === 'function');
    await Promise.all(tcs.map((fc) => handleToolCall(fc, client, toolCalls, setToolCalls, messages)));

    // `messages` now contains all tool call request and responses so far.
    // Send it back to the LLM to get its response given those tool calls.
    stream = llm.streamChatCompletions({
      model: llm.Model.LARGE,
      messages,
      tools: mcp.convertToolsToOpenAI(tools),
    });

    [toolCallsStream, otherMessages] = partition(
      stream,
      (chunk: llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>) => llm.isToolCallsMessage(chunk.choices[0].delta)
    );

    // Include a pretend 'first message' in the reply, in case the model chose to send anything before its tool calls.
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

    contentMessages.subscribe(setReply);

    toolCallMessages = await lastValueFrom(toolCallsStream.pipe(llm.accumulateToolCalls()));
  }
}

function AvailableTools({ tools }: { tools: Tool[] }) {
  return (
    <Stack direction="column">
      <h4>Available MCP Tools</h4>
      <ul>
        {tools.map((tool, i) => (
          <li key={i}>{tool.annotations?.title ?? tool.name}</li>
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
          <li
            key={i}
            style={{
              marginBottom: '16px',
              padding: '12px',
              backgroundColor: 'var(--background-color-secondary)',
              borderRadius: '4px',
            }}
          >
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '8px' }}>
              <span style={{ fontWeight: 500 }}>{toolCall.name}</span>
              <code
                style={{ backgroundColor: 'var(--background-color-primary)', padding: '2px 6px', borderRadius: '4px' }}
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
                  marginTop: '4px',
                  fontSize: '0.9em',
                }}
              >
                <Icon name="exclamation-triangle" size="sm" style={{ marginRight: '4px' }} />
                {toolCall.error}
              </div>
            )}
            {!toolCall.error && toolCall.response && (
              <CollapsableSection
                label={<span style={{ fontSize: '0.7em', fontWeight: 500 }}>Response</span>}
                isOpen={false}
              >
                <pre
                  style={{
                    backgroundColor: 'var(--background-color-primary)',
                    padding: '8px',
                    borderRadius: '4px',
                    marginTop: '8px',
                    overflow: 'auto',
                    maxHeight: '300px',
                    fontSize: '0.9em',
                  }}
                >
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
  const { client } = mcp.useMCPClient();
  // The current input value.
  const [input, setInput] = useState('');
  // The final message to send to the LLM, updated when the button is clicked.
  const [message, setMessage] = useState('');
  // The latest reply from the LLM.
  const [reply, setReply] = useState('');

  const [toolCalls, setToolCalls] = useState<Map<string, RenderedToolCall>>(new Map());

  const [useStream, setUseStream] = useState(false);

  const [started, setStarted] = useState(false);
  const [finished, setFinished] = useState(true);

  const { loading, error, value } = useAsync(async () => {
    const enabled = await llm.enabled();
    if (!enabled) {
      return { enabled, tools: [] };
    }

    const { tools } = (await client?.listTools()) ?? { tools: [] };
    if (message === '') {
      return { enabled, tools };
    }

    setStarted(true);
    setFinished(false);

    try {
      if (!useStream) {
        await handleNonStreamingChat(
          message,
          tools,
          client,
          toolCalls,
          setToolCalls,
          setReply,
          setStarted,
          setFinished
        );
      } else {
        await handleStreamingChat(message, tools, client, toolCalls, setToolCalls, setReply, setStarted, setFinished);
      }
    } catch (e) {
      console.error('Error in chat completion:', e);
      setFinished(true);
      setStarted(false);
    }

    return { enabled: true, tools };
  }, [message]);

  if (error) {
    return <div>Error: {error.message}</div>;
  }

  return (
    <div>
      {value?.enabled ? (
        <Stack direction="column">
          <TextArea value={input} onChange={(e) => setInput(e.currentTarget.value)} placeholder="Enter a message" />
          <br />
          <Stack direction="row" justifyContent="space-evenly">
            <Button
              type="submit"
              onClick={() => {
                setMessage(input);
                setUseStream(true);
              }}
            >
              Submit Stream
            </Button>
            <Button
              type="submit"
              onClick={() => {
                setMessage(input);
                setUseStream(false);
              }}
            >
              Submit Request
            </Button>
          </Stack>
          <br />
          {!useStream && (
            <div>
              {loading && <Spinner />}
              <p style={{ whiteSpace: 'pre-wrap' }}>{reply}</p>
            </div>
          )}
          {useStream && <div>{reply}</div>}
          <Stack direction="row" justifyContent="space-evenly">
            <div>{started ? 'Response is started' : 'Response is not started'}</div>
            <div>{finished ? 'Response is finished' : 'Response is not finished'}</div>
          </Stack>
          <br />
          <br />
          <Stack direction="row" justifyContent="space-evenly">
            <AvailableTools tools={value.tools!} />
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
  };

  return (
    <FieldSet label="Development Sandbox">
      <Button onClick={() => setModalIsOpen(true)}>Open development sandbox</Button>
      <Modal title="Development Sandbox" isOpen={modalIsOpen} onDismiss={closeModal}>
        <Suspense fallback={<LoadingPlaceholder text="Loading MCP..." />}>
          <mcp.MCPClientProvider appName="Grafana App With Model Context Protocol" appVersion="1.0.0">
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
