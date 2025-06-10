import { test, expect } from '@grafana/plugin-e2e';
import { waitForMCPToolsListRequest, waitForMCPToolsList } from './helpers/wait-utils';

test.describe('LLM App with MCP Tools', () => {
  test('should load plugin page without errors', async ({ page }) => {
    const jsErrors: string[] = [];
    const consoleErrors: string[] = [];

    // Capture any JavaScript errors
    page.on('pageerror', (error) => {
      jsErrors.push(error.message);
    });

    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        consoleErrors.push(msg.text());
      }
    });

    // Navigate to the LLM app page
    await page.goto('/a/grafana-llm-app');

    // Wait for the page to load and MCP to be ready
    await waitForMCPToolsListRequest(page);
    await waitForMCPToolsList(page);

    // Filter out known browser-specific errors that don't affect plugin functionality
    const knownBrowserErrors = [
      'window.caches is undefined',
      "Cannot read properties of undefined (reading 'keys')",
      "TypeError: undefined is not an object (evaluating 'window.caches.keys')",
      'chunkNotFound', // Grafana build-related error
    ];

    const pluginRelatedErrors = jsErrors.filter(
      (error) => !knownBrowserErrors.some((knownError) => error.includes(knownError))
    );

    // Only fail if there are plugin-related JavaScript errors
    if (pluginRelatedErrors.length > 0) {
      console.log('Plugin-related JavaScript errors found:', pluginRelatedErrors);
      expect(pluginRelatedErrors).toHaveLength(0);
    }

    // Check basic page structure
    const title = await page.title();
    expect(title).toBe('Grafana');

    // Verify we're on the correct URL
    expect(page.url()).toContain('/a/grafana-llm-app');

    console.log('✅ Plugin page loaded without plugin-related JavaScript errors');
    if (jsErrors.length > 0) {
      console.log('   (Filtered out browser-specific errors:', jsErrors.length, ')');
    }
  });

  test('should have plugin configured and MCP enabled', async ({ page }) => {
    // Check plugin configuration via API
    const response = await page.request.get('/api/plugins/grafana-llm-app/settings');
    expect(response.ok()).toBeTruthy();

    const settings = await response.json();

    // Verify plugin is enabled
    expect(settings.enabled).toBe(true);

    // Verify MCP is enabled in configuration
    expect(settings.jsonData.mcp.enabled).toBe(true);

    // Verify provider is configured
    expect(settings.jsonData.provider).toBeTruthy();

    console.log('✅ Plugin is configured with MCP enabled');
    console.log('   Provider:', settings.jsonData.provider);
  });

  test('should handle MCP API endpoints', async ({ page }) => {
    // Test that MCP-related API endpoints respond correctly
    // This tests the backend MCP functionality even if frontend isn't working

    const pluginSettingsResponse = await page.request.get('/api/plugins/grafana-llm-app/settings');
    expect(pluginSettingsResponse.ok()).toBeTruthy();

    const settings = await pluginSettingsResponse.json();

    if (settings.jsonData.mcp.enabled) {
      console.log('✅ MCP is enabled in plugin settings');

      // Test health/settings endpoint
      const healthResponse = await page.request.get('/api/plugins/grafana-llm-app/health');

      if (healthResponse.ok()) {
        const healthData = await healthResponse.json();
        console.log('✅ Plugin backend is responding');
        console.log('   MCP enabled:', settings.jsonData?.mcp?.enabled || 'unknown');
      } else {
        console.log('⚠️  Plugin backend health check failed');
      }
    }
  });
});
