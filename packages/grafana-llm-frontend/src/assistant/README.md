# Tool Registry System

The Tool Registry system allows plugins to register tools that can be used by LLM agents and shared across different applications within the Grafana ecosystem.

## Overview

The system consists of:
- **Tool**: A class representing a function that can be called by an LLM
- **ToolRegistry**: A singleton registry for managing tools across plugins
- **Reactive Updates**: Observable streams for real-time updates when tools are registered/unregistered

## Core Concepts

### Tool
A tool represents a function that can be executed by an LLM agent. Each tool has:
- **Name**: Unique identifier for the tool
- **Description**: Human-readable description of what the tool does
- **Schema**: Zod schema defining the input parameters
- **Return Schema**: Zod schema defining the return type (optional)
- **Function**: The actual implementation

### Tool Registration
Tools are registered with metadata including:
- **Plugin ID**: Identifies which plugin registered the tool
- **Category**: Optional categorization for organizing tools
- **Tags**: Optional tags for searching and filtering
- **Registration Date**: When the tool was registered

## Basic Usage

### Creating a Tool

```typescript
import { z } from 'zod';
import { createTool } from './tool';

// Define parameter schema
const calculatorSchema = z.object({
  operation: z.enum(['add', 'subtract', 'multiply', 'divide']),
  a: z.number(),
  b: z.number(),
});

// Create the tool
const calculatorTool = createTool({
  name: 'calculator',
  description: 'Performs basic arithmetic operations',
  schema: calculatorSchema,
  func: async (params) => {
    const { operation, a, b } = params;
    let result: number;
    
    switch (operation) {
      case 'add': result = a + b; break;
      case 'subtract': result = a - b; break;
      case 'multiply': result = a * b; break;
      case 'divide': 
        if (b === 0) throw new Error('Cannot divide by zero');
        result = a / b; 
        break;
    }
    
    return { output: `Result: ${result}` };
  },
});
```

### Registering Tools

```typescript
import { registerTool, getToolRegistry } from './tool';

// Method 1: Using helper function
registerTool(calculatorTool, 'my-plugin-id', {
  category: 'utilities',
  tags: ['math', 'calculation'],
});

// Method 2: Using registry directly
const registry = getToolRegistry();
registry.registerTool(calculatorTool, 'my-plugin-id', {
  category: 'utilities',
  tags: ['math', 'calculation'],
});
```

### Using Tools

```typescript
const registry = getToolRegistry();

// Get a specific tool
const calculator = registry.getTool('calculator', 'my-plugin-id');
if (calculator) {
  const result = await calculator.call({
    operation: 'add',
    a: 10,
    b: 5,
  });
  console.log(result.output); // "Result: 15"
}
```

## Discovery and Querying

### Get All Tools
```typescript
const allTools = registry.getAllTools();
```

### Filter by Plugin
```typescript
const pluginTools = registry.getToolsByPlugin('my-plugin-id');
```

### Filter by Category
```typescript
const utilityTools = registry.getToolsByCategory('utilities');
```

### Filter by Tag
```typescript
const mathTools = registry.getToolsByTag('math');
```

### Search Tools
```typescript
const searchResults = registry.searchTools('calculator');
```

## Reactive Updates with Observables

The registry provides reactive streams using RxJS observables:

### Subscribe to All Events
```typescript
registry.getEvents().subscribe(event => {
  switch (event.type) {
    case 'tool_registered':
      console.log(`New tool: ${event.registration.tool.name}`);
      break;
    case 'tool_unregistered':
      console.log(`Tool removed: ${event.toolName}`);
      break;
    case 'plugin_unregistered':
      console.log(`Plugin unregistered: ${event.pluginId}`);
      break;
  }
});
```

### Subscribe to Tool Registrations Only
```typescript
registry.getToolRegistrations().subscribe(registration => {
  console.log(`New tool available: ${registration.tool.name}`);
  // Add to UI, update menus, etc.
});
```

### Subscribe to Tool Unregistrations Only
```typescript
registry.getToolUnregistrations().subscribe(({ toolName, pluginId }) => {
  console.log(`Tool ${toolName} no longer available`);
  // Remove from UI, update menus, etc.
});
```

## Plugin Lifecycle Management

### Plugin Registration (in plugin initialization)
```typescript
export function onPluginStart() {
  const pluginId = 'my-awesome-plugin';
  
  // Register multiple tools
  registerTool(calculatorTool, pluginId, { category: 'utilities' });
  registerTool(weatherTool, pluginId, { category: 'external-api' });
}
```

### Plugin Cleanup (in plugin destruction)
```typescript
export function onPluginStop() {
  const pluginId = 'my-awesome-plugin';
  const registry = getToolRegistry();
  
  // Option 1: Unregister specific tools
  unregisterTool('calculator', pluginId);
  unregisterTool('weather', pluginId);
  
  // Option 2: Unregister all tools from plugin
  registry.unregisterPlugin(pluginId);
}
```

## Cross-App Communication

Since the registry is a singleton, tools registered by one plugin can be used by any other part of the application:

### In a Plugin (Tool Provider)
```typescript
// Plugin A registers a tool
registerTool(myTool, 'plugin-a');
```

### In Another App (Tool Consumer)
```typescript
// App B uses the tool
const registry = getToolRegistry();
const tool = registry.getTool('myTool', 'plugin-a');
if (tool) {
  const result = await tool.call(params);
}
```

### Real-time Updates Across Apps
```typescript
// App B reacts to new tools from any plugin
registry.getToolRegistrations().subscribe(registration => {
  updateUIWithNewTool(registration);
});
```

## Cross-Import Registry Functionality

The Tool Registry is designed to work seamlessly across different package imports, ensuring that tools registered in one package (e.g., `grafana/grafana`) are accessible from another package (e.g., `grafana/dash-app`).

### How It Works

The registry uses a dual-layer singleton pattern:
1. **Local Singleton**: Standard singleton pattern within the module
2. **Global Symbol Registry**: Uses `Symbol.for()` to create a global reference that persists across module boundaries

### Usage Across Packages

#### In `grafana/grafana` (Tool Registration)
```typescript
import { registerTool, createTool, debugToolRegistry } from '@grafana/llm';

// Create and register a tool
const grafanaTool = createTool({
  name: 'grafana-datasource-query',
  description: 'Query Grafana datasources',
  schema: z.object({
    datasourceId: z.string(),
    query: z.string(),
  }),
  func: async (params) => {
    // Implementation
    return `Queried datasource ${params.datasourceId}: ${params.query}`;
  },
});

registerTool(grafanaTool, 'grafana-core');

// Debug to verify registration
debugToolRegistry();
```

#### In `grafana/dash-app` (Tool Consumption)
```typescript
import { getToolRegistry, debugToolRegistry } from '@grafana/llm';

// Get the same registry instance
const registry = getToolRegistry();

// Access tools registered by grafana/grafana
const grafanaTool = registry.getTool('grafana-datasource-query', 'grafana-core');
if (grafanaTool) {
  const result = await grafanaTool.invoke({
    datasourceId: 'prometheus-1',
    query: 'up',
  });
  console.log(result);
}

// Debug to verify same registry instance
debugToolRegistry();
```

### Debugging Cross-Import Issues

The registry provides debugging utilities to verify it's working correctly across imports:

#### Debug Registry State
```typescript
import { debugToolRegistry, ToolRegistry } from '@grafana/llm';

// Log comprehensive debug information
debugToolRegistry();

// Manual verification
const consistency = ToolRegistry.verifyRegistryConsistency();
console.log('Registry consistency:', consistency);

if (!consistency.isConsistent) {
  console.warn('⚠️ Registry instances are inconsistent across imports!');
  console.log('Local instance ID:', consistency.localInstance);
  console.log('Global instance ID:', consistency.globalInstance);
}
```

#### Get Debug Information
```typescript
const registry = getToolRegistry();
const debugInfo = registry.getDebugInfo();

console.log('Registry Debug Info:', {
  instanceId: debugInfo.instanceId,
  toolCount: debugInfo.toolCount,
  pluginCount: debugInfo.pluginCount,
  isGlobalInstance: debugInfo.isGlobalInstance,
  registeredPlugins: debugInfo.registeredPlugins,
});
```

### Best Practices for Cross-Import Usage

1. **Import Consistency**: Always import from the same package (`@grafana/llm`)
2. **Early Registration**: Register tools early in your application lifecycle
3. **Debug on Startup**: Use `debugToolRegistry()` during development to verify consistency
4. **Package Version Alignment**: Ensure all packages use the same version of `@grafana/llm`

### Troubleshooting

#### Problem: Tools registered in one package aren't visible in another
**Solutions:**
1. Verify both packages import from the same `@grafana/llm` version
2. Check for multiple versions in `node_modules` (run `npm ls @grafana/llm`)
3. Use `debugToolRegistry()` in both packages to compare instance IDs
4. Ensure tools are registered before attempting to access them

#### Problem: Registry consistency check fails
**Solutions:**
1. Check if packages are bundled separately (may create separate instances)
2. Verify import paths are identical
3. Look for symlink issues in development
4. Clear `node_modules` and reinstall dependencies

#### Example Debugging Session
```typescript
// In package 1 (registration)
import { registerTool, debugToolRegistry } from '@grafana/llm';

registerTool(myTool, 'plugin-1');
console.log('After registration in package 1:');
debugToolRegistry();

// In package 2 (consumption)
import { getToolRegistry, debugToolRegistry } from '@grafana/llm';

console.log('In package 2:');
debugToolRegistry();

const registry = getToolRegistry();
const tools = registry.getAllTools();
console.log('Available tools:', tools.map(t => t.tool.name));
```

## Error Handling

### Tool Execution Errors
```typescript
try {
  const result = await tool.call(params);
} catch (error) {
  if (error.message.includes('validation')) {
    // Handle parameter validation errors
  } else {
    // Handle tool execution errors
  }
}
```

### Registration Errors
```typescript
try {
  registerTool(tool, pluginId);
} catch (error) {
  // Tool with same name already registered by this plugin
}
```

## Best Practices

1. **Unique Tool Names**: Use descriptive, unique names for tools within your plugin
2. **Comprehensive Schemas**: Define thorough Zod schemas for validation
3. **Error Handling**: Always handle errors gracefully in tool functions
4. **Plugin Cleanup**: Always unregister tools when your plugin is disabled/unloaded
5. **Categories and Tags**: Use consistent categorization for better discoverability
6. **Documentation**: Provide clear descriptions for your tools
7. **Cross-Import Testing**: Test tool registration and consumption across different packages
8. **Version Consistency**: Maintain consistent `@grafana/llm` versions across all packages

## Advanced Usage

### Custom Return Schemas
```typescript
const returnSchema = z.object({
  output: z.string(),
  metadata: z.object({
    executionTime: z.number(),
    cached: z.boolean(),
  }),
});

const advancedTool = createTool({
  name: 'advanced-tool',
  description: 'Tool with custom return schema',
  schema: inputSchema,
  returnSchema,
  func: async (params) => {
    const startTime = Date.now();
    // ... tool logic
    return {
      output: 'result',
      metadata: {
        executionTime: Date.now() - startTime,
        cached: false,
      },
    };
  },
});
```

### Monitoring and Statistics
```typescript
// Get registry statistics
console.log('Total tools:', registry.getToolCount());

const countsByPlugin = registry.getToolCountByPlugin();
countsByPlugin.forEach((count, pluginId) => {
  console.log(`${pluginId}: ${count} tools`);
});
```

## Integration Examples

See `tool-examples.ts` for comprehensive examples of:
- Creating and registering tools
- Consuming tools from other apps
- Using reactive streams
- Plugin lifecycle management
- Error handling patterns 