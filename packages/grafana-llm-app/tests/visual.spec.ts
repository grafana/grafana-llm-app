import { test, expect } from '@grafana/plugin-e2e';
import { waitForMCPToolsList, waitForMCPToolsListRequest } from './helpers/wait-utils';

test.describe('LLM App Visual Tests', () => {
  test('should render main page with beautiful layout', async ({ page }) => {
    // Set consistent viewport size
    await page.setViewportSize({ width: 1280, height: 720 });

    await page.goto('/a/grafana-llm-app');

    // Wait for the page to load and MCP to be ready
    await waitForMCPToolsListRequest(page);

    await waitForMCPToolsList(page);

    // Take a screenshot of just the main content area
    await page.waitForSelector('[data-testid="main-page-container"]', {
      state: 'visible',
    });
    const mainContainer = page.getByTestId('main-page-container');
    if ((await mainContainer.count()) > 0) {
      await expect(mainContainer).toHaveScreenshot('main-container.png');
      console.log('✅ Main page container captured');
    }
    await expect(page).toHaveScreenshot('full-page.png', {
      animations: 'disabled',
    });

    console.log('✅ Visual test completed - screenshots captured');
  });
});
