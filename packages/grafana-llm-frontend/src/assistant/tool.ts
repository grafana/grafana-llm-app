import { z } from "zod";
import { BehaviorSubject, Observable, map, filter } from "rxjs";

// Global symbol to ensure registry uniqueness across modules
const GLOBAL_REGISTRY_SYMBOL = Symbol.for('@grafana/llm:tool-registry');

/**
 * Base interface for tool parameters
 */
export interface ToolParams {
  [key: string]: any;
}

/**
 * Interface for tool invoke options
 */
export interface ToolInvokeOptions {
  [key: string]: any;
}

/**
 * Interface for tool registration information
 */
export interface ToolRegistration<T extends ToolParams = ToolParams> {
  tool: ToolRunnable<T>;
  pluginId: string;
  category?: string;
  tags?: string[];
  registeredAt: Date;
}

/**
 * Registry event types
 */
export type ToolRegistryEvent =
  | { type: "tool_registered"; registration: ToolRegistration<any> }
  | { type: "tool_unregistered"; toolName: string; pluginId: string }
  | { type: "plugin_unregistered"; pluginId: string };

/**
 * Extended Tool interface that can be invoked by an LLM agent
 */
export interface ToolRunnable<T extends ToolParams = ToolParams> {
  invoke: (
    input: Record<string, unknown>,
    options?: ToolInvokeOptions
  ) => Promise<string>;
  name: string;
  description: string;
  zodSchema: z.ZodObject<any>;
  metadata?: {
    explainer?: () => string;
    [key: string]: any;
  };
  verboseParsingErrors?: boolean;
  responseFormat?: "content_and_artifact" | string;
}

/**
 * Base class for a Tool that can be used by an LLM agent
 */
export class Tool<T extends ToolParams = ToolParams>
  implements ToolRunnable<T>
{
  name: string;
  description: string;
  zodSchema: z.ZodObject<any>;
  private schema: z.ZodObject<any>;
  private _func: (params: T) => Promise<string>;
  metadata?: {
    explainer?: () => string;
    [key: string]: any;
  };
  verboseParsingErrors?: boolean;
  responseFormat?: "content_and_artifact" | string;

  constructor({
    name,
    description,
    schema,
    func,
    metadata,
    verboseParsingErrors,
    responseFormat,
  }: {
    name: string;
    description: string;
    schema: z.ZodObject<any>;
    func: (params: T) => Promise<string>;
    metadata?: {
      explainer?: () => string;
      [key: string]: any;
    };
    verboseParsingErrors?: boolean;
    responseFormat?: "content_and_artifact" | string;
  }) {
    this.name = name;
    this.description = description;
    this.schema = schema;
    this.zodSchema = schema;
    this._func = func;
    this.metadata = metadata;
    this.verboseParsingErrors = verboseParsingErrors;
    this.responseFormat = responseFormat;
  }

  /**
   * Execute the tool with the given parameters
   * @param input - Input parameters for the tool
   * @param options - Optional invoke options
   * @returns Promise with the tool's output
   */
  async invoke(
    input: Record<string, unknown>,
    options?: ToolInvokeOptions
  ): Promise<string> {
    try {
      // Validate input parameters
      this.schema.parse(input);

      // Execute the tool function
      const result = await this._func(input as T);

      return result;
    } catch (error) {
      if (error instanceof Error) {
        const errorMessage = this.verboseParsingErrors
          ? `Error in tool ${this.name}: ${error.message}`
          : `Error in tool ${this.name}: ${error.message}`;
        throw new Error(errorMessage);
      }
      throw error;
    }
  }

  /**
   * Legacy method for backwards compatibility
   * @deprecated Use invoke() instead
   */
  async call(params: T): Promise<string> {
    const result = await this.invoke(params as Record<string, unknown>);
    return result;
  }

  /**
   * Get the tool's schema as a JSON object
   */
  getSchema(): Record<string, any> {
    return {
      name: this.name,
      description: this.description,
      parameters: this.convertSchemaToInputSchema(this.zodSchema),
    };
  }

  /**
   * Convert zod schema to input schema format
   */
  private convertSchemaToInputSchema(
    schema: z.ZodObject<any>
  ): Record<string, unknown> {
    return schema.shape;
  }
}

/**
 * Registry for managing tools across plugins
 */
export class ToolRegistry {
  private registrations = new Map<string, ToolRegistration<any>>();
  private eventsSubject = new BehaviorSubject<ToolRegistryEvent | null>(null);
  private static instance: ToolRegistry | null = null;
  private instanceId: string;
  private createdAt: Date;

  constructor() {
    this.instanceId = Math.random().toString(36).substring(2, 15);
    this.createdAt = new Date();
  }

  /**
   * Get the singleton instance of the tool registry
   * Uses both local singleton and global symbol-based registry for cross-module compatibility
   */
  static getInstance(): ToolRegistry {
    // First, try to get from global symbol registry
    const global = globalThis as any;
    if (global[GLOBAL_REGISTRY_SYMBOL]) {
      // Update local instance reference if needed
      if (!ToolRegistry.instance) {
        ToolRegistry.instance = global[GLOBAL_REGISTRY_SYMBOL];
      }
      return global[GLOBAL_REGISTRY_SYMBOL];
    }

    // If no global instance exists, create one
    if (!ToolRegistry.instance) {
      ToolRegistry.instance = new ToolRegistry();
      // Store in global symbol registry for cross-module access
      global[GLOBAL_REGISTRY_SYMBOL] = ToolRegistry.instance;
    }

    return ToolRegistry.instance;
  }

  /**
   * Get debug information about this registry instance
   */
  getDebugInfo(): {
    instanceId: string;
    createdAt: Date;
    toolCount: number;
    pluginCount: number;
    isGlobalInstance: boolean;
    registeredPlugins: string[];
  } {
    const pluginIds = new Set<string>();
    for (const registration of this.registrations.values()) {
      pluginIds.add(registration.pluginId);
    }

    return {
      instanceId: this.instanceId,
      createdAt: this.createdAt,
      toolCount: this.registrations.size,
      pluginCount: pluginIds.size,
      isGlobalInstance: (globalThis as any)[GLOBAL_REGISTRY_SYMBOL] === this,
      registeredPlugins: Array.from(pluginIds),
    };
  }

  /**
   * Verify that this is the same registry instance across different imports
   * Returns true if all instances point to the same registry
   */
  static verifyRegistryConsistency(): {
    isConsistent: boolean;
    localInstance: string | null;
    globalInstance: string | null;
    message: string;
  } {
    const localId = ToolRegistry.instance?.instanceId || null;
    const globalId = (globalThis as any)[GLOBAL_REGISTRY_SYMBOL]?.instanceId || null;

    const isConsistent = localId === globalId && localId !== null;

    return {
      isConsistent,
      localInstance: localId,
      globalInstance: globalId,
      message: isConsistent 
        ? 'Registry instances are consistent across imports'
        : `Registry instances are inconsistent. Local: ${localId}, Global: ${globalId}`,
    };
  }

  /**
   * Register a tool with the registry
   * @param tool - The tool to register
   * @param pluginId - ID of the plugin registering the tool
   * @param options - Additional registration options
   */
  registerTool<T extends ToolParams>(
    tool: ToolRunnable<T>,
    pluginId: string,
    options?: {
      category?: string;
      tags?: string[];
    }
  ): void {
    const key = this.getToolKey(tool.name, pluginId);

    if (this.registrations.has(key)) {
      throw new Error(
        `Tool ${tool.name} is already registered by plugin ${pluginId}`
      );
    }

    const registration: ToolRegistration<T> = {
      tool,
      pluginId,
      category: options?.category,
      tags: options?.tags,
      registeredAt: new Date(),
    };

    this.registrations.set(key, registration);
    this.eventsSubject.next({ type: "tool_registered", registration });
  }

  /**
   * Unregister a specific tool
   * @param toolName - Name of the tool to unregister
   * @param pluginId - ID of the plugin that registered the tool
   */
  unregisterTool(toolName: string, pluginId: string): boolean {
    const key = this.getToolKey(toolName, pluginId);
    const existed = this.registrations.delete(key);

    if (existed) {
      this.eventsSubject.next({
        type: "tool_unregistered",
        toolName,
        pluginId,
      });
    }

    return existed;
  }

  /**
   * Unregister all tools from a specific plugin
   * @param pluginId - ID of the plugin to unregister tools from
   */
  unregisterPlugin(pluginId: string): number {
    let count = 0;
    const keysToDelete: string[] = [];

    for (const [key, registration] of this.registrations) {
      if (registration.pluginId === pluginId) {
        keysToDelete.push(key);
        count++;
      }
    }

    keysToDelete.forEach((key) => this.registrations.delete(key));

    if (count > 0) {
      this.eventsSubject.next({ type: "plugin_unregistered", pluginId });
    }

    return count;
  }

  /**
   * Get a specific tool by name and plugin ID
   * @param toolName - Name of the tool
   * @param pluginId - ID of the plugin that registered the tool
   */
  getTool<T extends ToolParams = ToolParams>(
    toolName: string,
    pluginId: string
  ): ToolRunnable<T> | null {
    const key = this.getToolKey(toolName, pluginId);
    const registration = this.registrations.get(key);
    return registration ? (registration.tool as ToolRunnable<T>) : null;
  }

  /**
   * Get all registered tools
   */
  getAllTools(): ToolRegistration<any>[] {
    return Array.from(this.registrations.values());
  }

  /**
   * Get tools by plugin ID
   * @param pluginId - ID of the plugin
   */
  getToolsByPlugin(pluginId: string): ToolRegistration<any>[] {
    return Array.from(this.registrations.values()).filter(
      (registration) => registration.pluginId === pluginId
    );
  }

  /**
   * Get tools by category
   * @param category - Category to filter by
   */
  getToolsByCategory(category: string): ToolRegistration<any>[] {
    return Array.from(this.registrations.values()).filter(
      (registration) => registration.category === category
    );
  }

  /**
   * Get tools by tag
   * @param tag - Tag to filter by
   */
  getToolsByTag(tag: string): ToolRegistration<any>[] {
    return Array.from(this.registrations.values()).filter((registration) =>
      registration.tags?.includes(tag)
    );
  }

  /**
   * Search tools by name (partial match)
   * @param searchTerm - Term to search for in tool names
   */
  searchTools(searchTerm: string): ToolRegistration<any>[] {
    const lowerSearchTerm = searchTerm.toLowerCase();
    return Array.from(this.registrations.values()).filter(
      (registration) =>
        registration.tool.name.toLowerCase().includes(lowerSearchTerm) ||
        registration.tool.description.toLowerCase().includes(lowerSearchTerm)
    );
  }

  /**
   * Get an observable of all registry events
   */
  getEvents(): Observable<ToolRegistryEvent> {
    return this.eventsSubject
      .asObservable()
      .pipe(filter((event): event is ToolRegistryEvent => event !== null));
  }

  /**
   * Get an observable of tool registrations
   */
  getToolRegistrations(): Observable<ToolRegistration<any>> {
    return this.getEvents().pipe(
      filter((event) => event.type === "tool_registered"),
      map(
        (event) =>
          (
            event as {
              type: "tool_registered";
              registration: ToolRegistration<any>;
            }
          ).registration
      )
    );
  }

  /**
   * Get an observable of tool unregistrations
   */
  getToolUnregistrations(): Observable<{ toolName: string; pluginId: string }> {
    return this.getEvents().pipe(
      filter((event) => event.type === "tool_unregistered"),
      map((event) => ({
        toolName: (event as any).toolName,
        pluginId: (event as any).pluginId,
      }))
    );
  }

  /**
   * Get current count of registered tools
   */
  getToolCount(): number {
    return this.registrations.size;
  }

  /**
   * Get count of tools by plugin
   */
  getToolCountByPlugin(): Map<string, number> {
    const counts = new Map<string, number>();
    for (const registration of this.registrations.values()) {
      const current = counts.get(registration.pluginId) || 0;
      counts.set(registration.pluginId, current + 1);
    }
    return counts;
  }

  /**
   * Clear all registered tools (useful for testing)
   */
  clear(): void {
    this.registrations.clear();
  }

  private getToolKey(toolName: string, pluginId: string): string {
    return `${pluginId}:${toolName}`;
  }
}

/**
 * Create a new tool with the given configuration
 */
export function createTool<T extends ToolParams>({
  name,
  description,
  schema,
  func,
  metadata,
  verboseParsingErrors,
  responseFormat,
}: {
  name: string;
  description: string;
  schema: z.ZodObject<any>;
  func: (params: T) => Promise<string>;
  metadata?: {
    explainer?: () => string;
    [key: string]: any;
  };
  verboseParsingErrors?: boolean;
  responseFormat?: "content_and_artifact" | string;
}): ToolRunnable<T> {
  return new Tool<T>({
    name,
    description,
    schema,
    func,
    metadata,
    verboseParsingErrors,
    responseFormat,
  });
}

/**
 * Get the global tool registry instance
 */
export function getToolRegistry(): ToolRegistry {
  return ToolRegistry.getInstance();
}

/**
 * Helper function to register a tool
 */
export function registerTool<T extends ToolParams>(
  tool: ToolRunnable<T>,
  pluginId: string,
  options?: {
    category?: string;
    tags?: string[];
  }
): void {
  getToolRegistry().registerTool(tool, pluginId, options);
}

/**
 * Helper function to unregister a tool
 */
export function unregisterTool(toolName: string, pluginId: string): boolean {
  return getToolRegistry().unregisterTool(toolName, pluginId);
}

/**
 * Debug helper to verify registry consistency across imports
 */
export function debugToolRegistry(): void {
  const registry = getToolRegistry();
  const debugInfo = registry.getDebugInfo();
  const consistency = ToolRegistry.verifyRegistryConsistency();
  
  console.log('🔧 Tool Registry Debug Info:', {
    ...debugInfo,
    consistency,
  });
}

export const zod = z;
export type { z };

// setTimeout(() => {
//   examplePluginRegistration();
// }, 15000);
