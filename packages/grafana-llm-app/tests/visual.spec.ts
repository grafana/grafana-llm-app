import { test, expect } from '@grafana/plugin-e2e';

test.describe('LLM App Visual Tests', () => {
  test('should render main page with beautiful layout', async ({ page }) => {
    // Set consistent viewport size
    await page.setViewportSize({ width: 1280, height: 720 });

    await page.goto('/a/grafana-llm-app');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // Take a screenshot of just the main content area
    const mainContent = page.locator('.plugin-page-content');
    if ((await mainContent.count()) > 0) {
      await expect(mainContent).toHaveScreenshot('main-content.png', {
        animations: 'disabled',
      });
    }

    console.log('✅ Visual test completed - screenshots captured');
  });

  test('should render sections properly on different viewport sizes', async ({ page }) => {
    // Test desktop view
    await page.setViewportSize({ width: 1200, height: 800 });
    await page.goto('/a/grafana-llm-app');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    await expect(page).toHaveScreenshot('desktop-view.png', {
      animations: 'disabled',
    });

    // Test tablet view
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.waitForTimeout(1000);

    await expect(page).toHaveScreenshot('tablet-view.png', {
      animations: 'disabled',
    });

    // Test mobile view
    await page.setViewportSize({ width: 375, height: 667 });
    await page.waitForTimeout(1000);

    await expect(page).toHaveScreenshot('mobile-view.png', {
      animations: 'disabled',
    });

    console.log('✅ Responsive design tests completed');
  });

  test('should show loading states properly', async ({ page }) => {
    await page.goto('/a/grafana-llm-app');

    // Try to capture any loading states that might appear
    // This is useful for testing the Suspense fallbacks and loading indicators

    // Wait for initial load but not too long to catch loading states
    await page.waitForTimeout(500);

    // Look for any loading indicators
    const loadingElements = await page.locator('text=Loading, text=Connecting, [data-testid*="loading"]').count();

    if (loadingElements > 0) {
      await expect(page).toHaveScreenshot('loading-state.png', {
        animations: 'disabled',
      });
      console.log('✅ Captured loading state screenshot');
    }

    // Wait for full load
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Take final loaded state
    await expect(page).toHaveScreenshot('loaded-state.png', {
      animations: 'disabled',
    });

    console.log('✅ Loading state test completed');
  });

  test('should handle dark and light themes', async ({ page }) => {
    await page.goto('/a/grafana-llm-app');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(2000);

    // Try to detect current theme and take appropriate screenshots
    const isDarkTheme = await page.evaluate(() => {
      const body = document.body;
      const computedStyle = window.getComputedStyle(body);
      const bgColor = computedStyle.backgroundColor;
      // Simple heuristic: if background is dark, we're in dark theme
      return bgColor.includes('rgb(') && bgColor.split(',')[0].split('(')[1] < 128;
    });

    const themeType = isDarkTheme ? 'dark' : 'light';

    await expect(page).toHaveScreenshot(`${themeType}-theme.png`, {
      animations: 'disabled',
    });

    console.log(`✅ Captured ${themeType} theme screenshot`);
  });
});

test.describe('Component Visual Tests', () => {
  test('should render individual sections correctly', async ({ page }) => {
    await page.goto('/a/grafana-llm-app');
    await page.waitForLoadState('networkidle');
    await page.waitForTimeout(3000);

    // Test Models section if visible
    const modelsContainer = page.getByTestId('models-container');
    if ((await modelsContainer.count()) > 0) {
      await expect(modelsContainer).toHaveScreenshot('models-section.png', {
        animations: 'disabled',
      });
      console.log('✅ Models section screenshot captured');
    }

    // Test MCP Tools section if visible
    const mcpContainer = page.getByTestId('mcp-tools-container');
    if ((await mcpContainer.count()) > 0) {
      await expect(mcpContainer).toHaveScreenshot('mcp-tools-section.png', {
        animations: 'disabled',
      });
      console.log('✅ MCP Tools section screenshot captured');
    }

    // Test different MCP states if they exist
    const mcpStates = [
      { selector: '[data-testid="mcp-tools-disabled"]', name: 'mcp-disabled' },
      { selector: '[data-testid="mcp-tools-loading"]', name: 'mcp-loading' },
      { selector: '[data-testid="mcp-tools-error"]', name: 'mcp-error' },
      { selector: '[data-testid="mcp-tools-empty"]', name: 'mcp-empty' },
      { selector: '[data-testid="mcp-tools-list"]', name: 'mcp-tools-list' },
    ];

    for (const state of mcpStates) {
      const element = page.locator(state.selector);
      if ((await element.count()) > 0 && (await element.isVisible())) {
        await expect(element).toHaveScreenshot(`${state.name}.png`, {
          animations: 'disabled',
        });
        console.log(`✅ ${state.name} state screenshot captured`);
      }
    }
  });
});
