import React, { Suspense, useState } from "react";
import { Button, FieldSet, Icon, LoadingPlaceholder, Modal, Spinner, Stack, TextArea } from "@grafana/ui";
import { useAsync } from "react-use";
import { finalize, lastValueFrom, map, partition, startWith, toArray } from "rxjs";
import { llm, mcp, openai } from "@grafana/llm";
import { CallToolResultSchema, Tool } from '@modelcontextprotocol/sdk/types';

interface RenderedToolCall {
  name: string;
  arguments: string;
  running: boolean;
  error?: string;
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
      return { enabled, tools: [] };
    }
    const { tools } = await client.listTools();

    if (message === '') {
      return { enabled, tools };
    }

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
      return { enabled, tools };
    } else {
      // Stream the completions. Each element is the next stream chunk.
      const messages: llm.Message[] = [
        { role: 'system', content: 'You are a cynical assistant.' },
        { role: 'user', content: message },
      ];
      let stream = llm.streamChatCompletions({
        model: openai.Model.LARGE,
        messages,
        tools: mcp.convertToolsToOpenAI(tools),
      });
      let [toolCallsStream, otherMessages] = partition(
        stream,
        (chunk: llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>) => llm.isToolCallsMessage(chunk.choices[0].delta),
      );
      let contentMessages = otherMessages.pipe(
        // Accumulate the stream content into a stream of strings, where each
        // element contains the accumulated message so far.
        llm.accumulateContent(),
        // The stream is just a regular Observable, so we can use standard rxjs
        // functionality to update state, e.g. recording when the stream
        // has completed.
        // The operator decision tree on the rxjs website is a useful resource:
        // https://rxjs.dev/operator-decision-tree.
        finalize(() => {
          console.log('stream finalized');
          setStarted(false);
          setFinished(true);
        })
      );
      // Get all the tool call messages as an array.
      let toolCallMessages = await lastValueFrom(toolCallsStream.pipe(
        map(
          (response: llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>) => (response.choices[0].delta as llm.ToolCallsMessage)
        ),
        toArray(),
      ));
      // Handle any function calls, looping until there are no more.
      while (toolCallMessages.length > 0) {
        // The way tool use works for streaming chat completions is pretty nuts. We'll get lots of
        // chunks; the first will include the tool name, id and index, then some others will
        // gradually populate the 'arguments' JSON string, so we need to loop over them all and
        // reconstruct the full tool call for each index.
        const recoveredToolCallMessage: llm.ToolCallsMessage = {
          role: 'assistant',
          tool_calls: [],
        };
        for (const msg of toolCallMessages) {
          for (const tc of msg.tool_calls) {
            if (tc.index! >= recoveredToolCallMessage.tool_calls.length) {
              // We have a new tool call, so let's create one with a sensible empty 'arguments' string.
              recoveredToolCallMessage.tool_calls.push({ ...tc, function: { ...tc.function, arguments: tc.function.arguments ?? '' } });
            } else {
              // This refers to an existing tool call, so continue reconstructing the arguments.
              recoveredToolCallMessage.tool_calls[tc.index!].function.arguments += tc.function.arguments;
            }
          }
        }
        messages.push(recoveredToolCallMessage)
        const tcs = recoveredToolCallMessage.tool_calls.filter(tc => tc.type === 'function');

        // Submit all tool requests.
        await Promise.all(tcs.map(async (fc) => {
          // Update the tool call state for rendering.
          const { function: f, id } = fc;
          setToolCalls(new Map(toolCalls.set(id, { name: f.name, arguments: f.arguments, running: true })));
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
        // Stream the completions. Each element is the next stream chunk.
        stream = llm.streamChatCompletions({
          model: 'gpt-4o',
          messages,
          tools: mcp.convertToolsToOpenAI(tools),
        });
        [toolCallsStream, otherMessages] = partition(
          stream,
          (chunk: llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>) => llm.isToolCallsMessage(chunk.choices[0].delta),
        );
        // Include a pretend 'first message' in the reply, in case the model chose to send anything before its tool calls.
        const firstMessage: Partial<llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>> = {
          choices: [{ delta: { role: 'assistant', content: reply } }],
        };
        contentMessages = otherMessages.pipe(
          //@ts-expect-error
          startWith(firstMessage),
          // Accumulate the stream content into a stream of strings, where each
          // element contains the accumulated message so far.
          llm.accumulateContent(),
          // The stream is just a regular Observable, so we can use standard rxjs
          // functionality to update state, e.g. recording when the stream
          // has completed.
          // The operator decision tree on the rxjs website is a useful resource:
          // https://rxjs.dev/operator-decision-tree.
          finalize(() => {
            console.log('stream finalized');
          })
        );
        // Subscribe to the stream and update the state for each returned value.
        contentMessages.subscribe((val) => {
          setReply(val);
        });
        toolCallMessages = await lastValueFrom(toolCallsStream.pipe(
          map(
            (response: llm.ChatCompletionsResponse<llm.ChatCompletionsChunk>) => (response.choices[0].delta as llm.ToolCallsMessage)
          ),
          toArray(),
        ));
      }
    }
    return { enabled: true, tools };
  }, [message]);

  if (error) {
    // TODO: handle errors.
    return <div>error</div>;
  }

  return (
    <div>
      {value?.enabled ? (
        <Stack direction="column">
          <TextArea
            value={input}
            onChange={(e) => setInput(e.currentTarget.value)}
            placeholder="Enter a message"
          />
          <br />
          <Stack direction="row" justifyContent="space-evenly">
            <Button type="submit" onClick={() => { setMessage(input); setUseStream(true); }}>Submit Stream</Button>
            <Button type="submit" onClick={() => { setMessage(input); setUseStream(false); }}>Submit Request</Button>
          </Stack>
          <br />
          {!useStream && <div>{loading ? <Spinner /> : reply}</div>}
          {useStream && <div>{reply}</div>}
          <Stack direction="row" justifyContent="space-evenly">
            <div>{started ? "Response is started" : "Response is not started"}</div>
            <div>{finished ? "Response is finished" : "Response is not finished"}</div>
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
