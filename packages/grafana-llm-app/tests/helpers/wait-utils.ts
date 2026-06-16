import { Page, expect } from '@playwright/test';

/**
 * Utility functions for waiting in tests with more reliable element-based waiting
 * instead of arbitrary timeouts.
 */

export interface WaitOptions {
  timeout?: number;
  retries?: number;
  retryDelay?: number;
}

/**
 * Wait for the plugin to be loaded and responding
 */
export async function waitForPluginReady(page: Page, timeout = 10000): Promise<void> {
  await page.waitForFunction(
    async () => {
      try {
        const response = await fetch('/api/plugins/grafana-llm-app/settings');
        return response.ok;
      } catch (error) {
        return false;
      }
    },
    { timeout }
  );
}

/**
 * Wait for MCP tools/list request specifically.
 */
export async function waitForMCPToolsListRequest(page: Page): Promise<void> {
  await page.waitForRequest(
    (request) => {
      return (
        request.url().includes('/api/plugins/grafana-llm-app/resources/mcp/grafana') &&
        request.method() === 'POST' &&
        request.postData() === `{"method":"tools/list","jsonrpc":"2.0","id":1}`
      );
    },
    { timeout: 10000 }
  );
}

export async function waitForMCPToolsList(page: Page): Promise<void> {
  await page.waitForSelector('[data-testid="mcp-tools-list"]', { state: 'visible', timeout: 10000 });
}
