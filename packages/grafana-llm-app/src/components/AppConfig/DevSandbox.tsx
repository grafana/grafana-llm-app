import React, { Suspense, useState } from "react";
import { Button, FieldSet, Icon, Input, LoadingPlaceholder, Modal, Spinner } from "@grafana/ui";
import { useAsync } from "react-use";
import { finalize } from "rxjs";
import { mcp, openai } from "@grafana/llm";
import { CallToolResultSchema } from '@modelcontextprotocol/sdk/types';

interface RenderedToolCall {
  name: string;
  arguments: string;
  running: boolean;
  error?: string;
}

function ToolCalls({ toolCalls }: { toolCalls: Map<string, RenderedToolCall> }) {
  return (
    <div>
      <h3>Tool Calls</h3>
      {toolCalls.size === 0 && <div>No tool calls yet</div>}
      <ul>
        {Array.from(toolCalls.values()).map((toolCall, i) => (
          <li key={i}>
            <div>
              {toolCall.name}
              {' '}
              (<code>{toolCall.arguments}</code>)
              {' '}
              <Icon name={toolCall.running ? 'spinner' : 'check'} size='sm' />
              {toolCall.error && <code>{toolCall.error}</code>}
            </div>
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
  // The final message to send to the LLM, updated when the button is clicked.
  const [message, setMessage] = useState('');
  // The latest reply from the LLM.
  const [reply, setReply] = useState('');

  const [toolCalls, setToolCalls] = useState<Map<string, RenderedToolCall>>(new Map());

  const [useStream, setUseStream] = useState(false);

  const [started, setStarted] = useState(false);
  const [finished, setFinished] = useState(true);

  const { loading, error, value } = useAsync(async () => {
    // Check if the LLM plugin is enabled and configured.
    // If not, we won't be able to make requests, so return early.
    console.log("Logging works");
    const openAIHealthDetails = await openai.enabled();
    console.log("openAIHealthDetails: ", openAIHealthDetails);
    const enabled = openAIHealthDetails;
    console.log("enabled: ", enabled);
    if (!enabled) {
      return { enabled };
    }
    if (message === '') {
      return { enabled };
    }

    const { tools } = await client.listTools();

    setStarted(true);
    setFinished(false);
    if (!useStream) {
      setToolCalls(new Map());
      const messages: openai.Message[] = [
        { role: 'system', content: 'You are a cynical assistant.' },
        { role: 'user', content: message },
      ];
      // Make a single request to the LLM.
      let response = await openai.chatCompletions({
        model: openai.Model.BASE,
        messages,
        tools: mcp.convertToolsToOpenAI(tools),
      });

      // Handle any function calls, looping until there are no more.
      let functionCalls = response.choices[0].message.tool_calls?.filter(tc => tc.type === 'function') ?? [];
      while (functionCalls.length > 0) {
        // We need to include the 'tool_call' request in future responses.
        messages.push(response.choices[0].message);

        // Submit all tool requests.
        await Promise.all(functionCalls.map(async (fc) => {
          // Update the tool call state for rendering.
          setToolCalls(new Map(toolCalls.set(fc.id, { name: fc.function.name, arguments: fc.function.arguments, running: true })));
          const { function: f, id } = fc;
          try {
            // OpenAI sends arguments as a JSON string, so we need to parse it.
            const args = JSON.parse(f.arguments);
            const response = await client.callTool({ name: f.name, arguments: args });
            const toolResult = CallToolResultSchema.parse(response);
            // Just handle text results for now.
            const textContent = toolResult.content.filter(c => c.type === 'text').map(c => c.text).join('');
            // Add the result to the message, with the correct role and id.
            messages.push({ role: 'tool', tool_call_id: id, content: textContent });
            // Update the tool call state for rendering.
            setToolCalls(new Map(toolCalls.set(id, { name: f.name, arguments: f.arguments, running: false })));
          } catch (e: any) {
            const error = e.message ?? e.toString();
            messages.push({ role: 'tool', tool_call_id: id, content: error });
            // Update the tool call state for rendering.
            setToolCalls(new Map(toolCalls.set(id, { name: f.name, arguments: f.arguments, running: false, error })));
          }
        }));
        // `messages` now contains all tool call request and responses so far.
        // Send it back to the LLM to get its response given those tool calls.
        response = await openai.chatCompletions({
          model: openai.Model.LARGE,
          messages,
          tools: mcp.convertToolsToOpenAI(tools),
        });
        functionCalls = response.choices[0].message.tool_calls?.filter(tc => tc.type === 'function') ?? [];
      }
      // No more function calls, so we can just use the final response.
      setReply(response.choices[0].message.content!);
      setStarted(false);
      setFinished(true);
      return { enabled, response };
    } else {
      // Stream the completions. Each element is the next stream chunk.
      const stream = openai.streamChatCompletions({
        model: openai.Model.BASE,
        messages: [
          { role: 'system', content: 'You are a cynical assistant.' },
          { role: 'user', content: message },
        ],
      }).pipe(
        // Accumulate the stream content into a stream of strings, where each
        // element contains the accumulated message so far.
        openai.accumulateContent(),
        // The stream is just a regular Observable, so we can use standard rxjs
        // functionality to update state, e.g. recording when the stream
        // has completed.
        // The operator decision tree on the rxjs website is a useful resource:
        // https://rxjs.dev/operator-decision-tree.
        finalize(() => {
          setStarted(false);
          setFinished(true);
        })
      );
      // Subscribe to the stream and update the state for each returned value.
      return {
        enabled,
        stream: stream.subscribe(setReply),
      };
    }
  }, [message]);

  if (error) {
    // TODO: handle errors.
    return <div>error</div>;
  }

  return (
    <div>
      {value?.enabled ? (
        <>
          <Input
            value={input}
            onChange={(e) => setInput(e.currentTarget.value)}
            placeholder="Enter a message"
          />
          <br />
          <Button type="submit" onClick={() => { setMessage(input); setUseStream(true); }}>Submit Stream</Button>
          <Button type="submit" onClick={() => { setMessage(input); setUseStream(false); }}>Submit Request</Button>
          <br />
          {!useStream && <div>{loading ? <Spinner /> : reply}</div>}
          {useStream && <div>{reply}</div>}
          <div>{started ? "Response is started" : "Response is not started"}</div>
          <div>{finished ? "Response is finished" : "Response is not finished"}</div>
          <br />
          <br />
          <ToolCalls toolCalls={toolCalls} />
        </>
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
