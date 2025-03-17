import React from 'react';

import { isLiveChannelMessageEvent, LiveChannelAddress, LiveChannelMessageEvent, LiveChannelScope } from '@grafana/data';
import { getGrafanaLiveSrv, GrafanaLiveSrv } from '@grafana/runtime';
import { Transport } from '@modelcontextprotocol/sdk/shared/transport';
import { Client } from '@modelcontextprotocol/sdk/client/index';
import { JSONRPCMessage, JSONRPCMessageSchema, Tool as MCPTool } from '@modelcontextprotocol/sdk/types';
import { Observable, filter } from 'rxjs';
import { v4 as uuid } from 'uuid';

import { LLM_PLUGIN_ID } from './constants';
import { Tool as OpenAITool } from './openai';

const MCP_GRAFANA_PATH = 'mcp/grafana'

/**
 * An MCP transport which uses the Grafana LLM plugin's built-in MCP server,
 * over Grafana Live.
 *
 * Use this with a client from `@modelcontextprotocol/sdk`.
 *
 * @experimental
 */
export class GrafanaLiveTransport implements Transport {
  _grafanaLiveSrv: GrafanaLiveSrv = getGrafanaLiveSrv()

  /**
   * The Grafana Live channel used by this transport.
   */
  _subscribeChannel: LiveChannelAddress;

  /**
   * The Grafana Live channel used by this transport.
   */
  _publishChannel: LiveChannelAddress;

  /**
   * The Grafana Live stream over which MCP messages are received.
   */
  _stream?: Observable<LiveChannelMessageEvent<unknown>>;

  // Methods defined as part of the Transport interface.
  // These will be attached by the client.
  onclose?: (() => void) | undefined;
  onerror?: ((error: Error) => void) | undefined;
  onmessage?: ((message: JSONRPCMessage) => void) | undefined;

  constructor(path?: string) {
    if (path === undefined) {
      // Construct a unique path for this transport.
      const pathId = uuid();
      path = `${MCP_GRAFANA_PATH}/${pathId}`;
    }
    this._subscribeChannel = {
      scope: LiveChannelScope.Plugin,
      namespace: LLM_PLUGIN_ID,
      path: `${path}/subscribe`,
    };
    this._publishChannel = {
      scope: LiveChannelScope.Plugin,
      namespace: LLM_PLUGIN_ID,
      path: `${path}/publish`,
    };
  }

  async start(): Promise<void> {
    if (this._stream !== undefined) {
      throw new Error(
        "GrafanaLiveTransport already started! If using Client class, note that connect() calls start() automatically."
      );
    }

    const stream = this._grafanaLiveSrv.getStream(this._subscribeChannel)
      .pipe(filter((event) => isLiveChannelMessageEvent(event)));
    this._stream = stream;
    stream.subscribe((event) => {
      let message: JSONRPCMessage;
      try {
        message = JSONRPCMessageSchema.parse(event.message);
      } catch (error) {
        this.onerror?.(error as Error)
        return;
      }
      this.onmessage?.(message);
    });
  }

  async send(message: JSONRPCMessage): Promise<void> {
    if (this._stream === undefined) {
      throw new Error("not connected");
    }
    // @ts-ignore
    return this._grafanaLiveSrv.publish(this._publishChannel, message, { useSocket: true })
      .then(() => undefined)
      .catch(error => {
        this.onerror?.(error as Error)
      });
  }

  async close(): Promise<void> {
    this._stream = undefined;
  }
}

// Create a map to store client instances. These will be keyed by the appName and appVersion.
// This effectively means:
// - each app will have a single client instance that is reused across the application.
// - since clients are stored outside of the MCPClientProvider component, they will be
//   cleaned up when the component unmounts.
// - this also allows users to wrap the MCPClientProvider in Suspense, which will
//   automatically suspend the component until the client is ready.
const clientMap = new Map<string, Client>();

// Context holding a client instance.
const MCPClientContext = React.createContext<Client | null>(null);

// Create a key for the client map.
function clientKey(appName: string, appVersion: string) {
  return `${appName}-${appVersion}`;
}

// A resource type, used with `createClientResource` to fetch the client or
// throw a promise if it's not yet ready.
type ClientResource = {
  read: () => Client;
};

// Create a resource that works with Suspense.
function createClientResource(appName: string, appVersion: string): ClientResource {
  let status: 'pending' | 'success' | 'error' = 'pending';
  let result: Client | null = null;
  let error: Error | null = null;

  const key = clientKey(appName, appVersion);
  const promise = (async () => {
    if (clientMap.has(key)) {
      result = clientMap.get(key)!;
      status = 'success';
      return result;
    }

    try {
      const client = new Client({
        name: appName,
        version: appVersion,
      });
      const transport = new GrafanaLiveTransport();
      await client.connect(transport);
      clientMap.set(key, client);
      status = 'success';
      result = client;
      return client;
    } catch (e) {
      status = 'error';
      error = e as Error;
      throw e;
    }
  })();

  return {
    read() {
      if (status === 'pending') {
        throw promise;
      } else if (status === 'error') {
        throw error;
      } else if (status === 'success' && result) {
        return result;
      }
      throw new Error('Unexpected resource state');
    },
  };
}

interface MCPClientProviderProps {
  appName: string;
  appVersion: string;
  children: React.ReactNode;
}

/**
 * MCPClientProvider is a React context provider that creates an MCP client
 * and manages its lifecycle.
 *
 * It should be used to wrap the entire application in a single provider.
 * This ensures that the client is created once and reused across the application.
 *
 * It also supports Suspense, which will suspend the component until the client
 * is ready. This allows you to use the client in components that are not yet
 * ready, such as those that are loading data.
 *
 * Example usage:
 * ```tsx
 * <Suspense fallback={<LoadingPlaceholder />}>
 *   <ErrorBoundary>
 *     {({ error }) => {
 *       if (error) {
 *         return <div>Something went wrong: {error.message}</div>;
 *       }
 *       return (
 *         <MCPClientProvider appName="MyApp" appVersion="1.0.0">
 *           <YourComponent />
 *         </MCPClientProvider>
 *       );
 *     }}
 *   </ErrorBoundary>
 * </Suspense>
 * ```
 *
 * @experimental
 */
export function MCPClientProvider({
  appName,
  appVersion,
  children,
}: MCPClientProviderProps) {
  const resource = createClientResource(appName, appVersion);

  // This will either return the client or throw a promise/error.
  // If it throws a promise, Suspense will suspend the component until it resolves.
  // If it throws an error, it should be caught by an ErrorBoundary.
  const client = resource.read();

  // Cleanup when the component unmounts.
  React.useEffect(() => {
    return () => {
      if (client) {
        client.close();
      }
      clientMap.delete(clientKey(appName, appVersion));
    };
  }, [client, appName, appVersion]);

  return (
    <MCPClientContext.Provider value={client}>
      {children}
    </MCPClientContext.Provider>
  );
}

/**
 * Convenience hook to use an MCP client from a component.
 *
 * This hook should be used within an `MCPClientProvider`.
 *
 * @experimental
 */
export function useMCPClient(): Client {
  const client = React.useContext(MCPClientContext);
  if (client === null) {
    throw new Error('useMCPClient must be used within an MCPClientProvider');
  }
  return client;
}

/**
 * Re-export of the Client class from the MCP SDK.
 *
 * @experimental
 */
export { Client };

/**
 * Convert an array of MCP tools to an array of OpenAI tools.
 *
 * This is useful when you want to use the MCP client with the LLM plugin's
 * `chatCompletions` or `streamChatCompletions` functions.
 *
 * @experimental
 */
export function convertToolsToOpenAI(tools: MCPTool[]): OpenAITool[] {
  return tools.map(convertToolToOpenAI);
}

function convertToolToOpenAI(tool: MCPTool): OpenAITool {
  return {
    type: 'function',
    function: {
      name: tool.name,
      description: tool.description,
      parameters: tool.inputSchema.properties !== undefined ? tool.inputSchema : undefined,
    },
  };
}
