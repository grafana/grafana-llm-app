import React from 'react';

import { isLiveChannelMessageEvent, LiveChannelAddress, LiveChannelMessageEvent, LiveChannelScope } from '@grafana/data';
import { getBackendSrv, getGrafanaLiveSrv, GrafanaLiveSrv, logDebug } from '@grafana/runtime';
import { Transport } from '@modelcontextprotocol/sdk/shared/transport';
import { Client } from '@modelcontextprotocol/sdk/client/index';
import { JSONRPCMessage, JSONRPCMessageSchema, Tool as MCPTool } from '@modelcontextprotocol/sdk/types';
import { Observable, filter } from 'rxjs';
import { v4 as uuid } from 'uuid';

import { LLM_PLUGIN_ID, LLM_PLUGIN_ROUTE } from './constants';
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

    // The Grafana Live service API for publishing messages sends a message
    // to Grafana's HTTP API rather than over the live channel, for reasons
    // that are unclear (but presumably justified in the default case).
    // This is fine when there is only one Grafana instance, but when there
    // are multiple (e.g. in a HA setup), the HTTP request will be routed
    // to a random Grafana instance, while we need it to be routed to the
    // same instance that the client is connected to (since there is a
    // long-lived stream over the live channel).
    //
    // We can use the `useSocket` argument when trying to publish to the
    // live channel to force the use of the Websocket instead of the HTTP API.
    // This will work in both single-instance and HA setups. However, it's only
    // available in Grafana 11.6.0 and later. We can check for this by checking
    // if the `publish` method has a third argument, which is the `options`
    // argument.
    const hasPublishOptions = this._grafanaLiveSrv.publish?.length >= 3;
    if (hasPublishOptions) {
      // TODO: use `LivePublishOptions` from `@grafana/runtime` once
      // Grafana 11.6.0 is released. We can remove these `@ts-expect-error`
      // comments once that happens.
      //@ts-expect-error
      const options: LivePublishOptions = { useSocket: true };
      //@ts-expect-error
      return this._grafanaLiveSrv.publish(this._publishChannel, message, options);
    }

    // If that option isn't available, we can first fall back to trying to
    // drilling down into the implementation details of the Grafana Live
    // service and using the Centrifuge API directly to publish the message
    // to the same stream that the client is connected to.
    // Realistically this should work in all versions of Grafana older than
    // 9, which is much further back than this plugin even supports, so should
    // always work.
    const centrifugeSubscription =
      // @ts-expect-error
      this._grafanaLiveSrv.deps?.centrifugeSrv?.getChannel?.(
        this._publishChannel,
      )?.subscription;
    if (centrifugeSubscription) {
      return centrifugeSubscription.publish(message);
    }

    // If the centrifuge subscription is still not available for some reason,
    // fall back to the official HTTP publish method. This won't work in HA
    // setups but it's better than nothing.
    console.warn(
      "Websocket subscription not available, falling back to HTTP publish. " +
      "This may fail in HA setups. If you see this, please create an issue at " +
      "https://github.com/grafana/grafana-llm-app/issues/new."
    );
    await this._grafanaLiveSrv.publish(this._publishChannel, message);
  }

  async close(): Promise<void> {
    this._stream = undefined;
  }
}

/**
 * A result object containing a client instance and whether MCP is enabled.
 */
interface ClientResult {
  /* Whether MCP is enabled for the current Grafana instance. */
  enabled: boolean;
  /* The client instance. */
  client: Client | null;
}

// Create a map to store client instances. These will be keyed by the appName and appVersion.
// This effectively means:
// - each app will have a single client instance that is reused across the application.
// - since clients are stored outside of the MCPClientProvider component, they will be
//   cleaned up when the component unmounts.
// - this also allows users to wrap the MCPClientProvider in Suspense, which will
//   automatically suspend the component until the client is ready.
const clientMap = new Map<string, ClientResult>();

// Context holding a client instance if MCP is enabled.
const MCPClientContext = React.createContext<ClientResult | null>(null);

// Create a key for the client map.
function clientKey(appName: string, appVersion: string) {
  return `${appName}-${appVersion}`;
}

// A resource type, used with `createClientResource` to fetch the client or
// throw a promise if it's not yet ready.
type ClientResource = {
  read: () => ClientResult;
};

async function isEnabled(): Promise<boolean> {
  try {
    const settings = await getBackendSrv().get(`${LLM_PLUGIN_ROUTE}/settings`, undefined, undefined, {
      showSuccessAlert: false,
      showErrorAlert: false,
    });
    if (!settings.enabled) {
      return false;
    }
    return settings.jsonData.mcp.enabled;
  } catch (e) {
    logDebug(String(e));
    logDebug(
      'Failed to check if LLM provider is enabled. This is expected if the Grafana LLM plugin is not installed, and the above error can be ignored.'
    );
    return false;
  }
}

// Create a resource that works with Suspense.
function createClientResource(appName: string, appVersion: string): ClientResource {
  let status: 'pending' | 'success' | 'error' = 'pending';
  let result: ClientResult | null = null;
  let error: Error | null = null;

  const key = clientKey(appName, appVersion);
  const promise = (async () => {
    if (clientMap.has(key)) {
      result = clientMap.get(key)!;
      status = 'success';
      return result;
    }

    try {
      const enabled = await isEnabled();
      if (!enabled) {
        status = 'success';
        result = { client: null, enabled };
      }
      const client = new Client({
        name: appName,
        version: appVersion,
      });
      const transport = new GrafanaLiveTransport();
      await client.connect(transport);
      result = { client, enabled };
      clientMap.set(key, result);
      status = 'success';
      return result;
    } catch (e) {
      status = 'error';
      error = e as Error;
      throw e;
    }
  })();

  return {
    read() {
      if (status === 'pending') {
        if (!result?.enabled) {
          console.warn('MCP is not enabled');
          return result || { client: null, enabled: false };
        }
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
  const result = resource.read();

  // Cleanup when the component unmounts.
  React.useEffect(() => {
    return () => {
      if (result?.client) {
        result.client.close();
      }
      clientMap.delete(clientKey(appName, appVersion));
    };
  }, [result, appName, appVersion]);

  return (
    <MCPClientContext.Provider value={result}>
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
export function useMCPClient(): ClientResult {
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
