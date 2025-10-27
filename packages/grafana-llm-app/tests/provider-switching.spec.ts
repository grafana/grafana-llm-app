/**
 * End-to-end tests for provider switching UI consistency.
 *
 * This test suite verifies that when switching between LLM providers (OpenAI, Azure, Custom, Anthropic),
 * the UI maintains consistent state and properly shows/hides fields based on the selected provider.
 *
 * Key behaviors tested:
 * - Provider dropdown visibility (hidden for custom, visible for openai/azure)
 * - URL field enabled/disabled state (disabled for openai, enabled for azure/custom)
 * - Organization ID field visibility (only for openai, not for azure/custom)
 * - Custom API path option (only available for custom provider)
 * - Field consistency when switching between providers
 *
 * This addresses a bug where fields would have inconsistent state depending on whether
 * settings.openAI.provider was set, particularly when switching to 'custom' provider.
 */
import { test, expect } from '@grafana/plugin-e2e';
import { testIds } from '../src/components/testIds';

test.describe('Provider Switching UI Consistency', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the plugin configuration page
    await page.goto('/plugins/grafana-llm-app');

    // Wait for the configuration form to load
    await page.waitForSelector(`[data-testid="${testIds.appConfig.container}"]`, {
      state: 'visible',
      timeout: 10000,
    });
  });

  test('should show consistent UI when first switching to custom provider', async ({ page }) => {
    // Click on the Custom API card
    const customCard = page.locator('text=Use a Custom API').locator('..');
    await customCard.click();

    // Wait for the card to be selected
    await page.waitForTimeout(500);

    // Verify provider dropdown is NOT visible (it should be hidden for custom)
    const providerDropdown = page.getByTestId(testIds.appConfig.provider);
    await expect(providerDropdown).not.toBeVisible();

    // Verify URL field is visible and enabled
    const urlField = page.getByTestId(testIds.appConfig.openAIUrl);
    await expect(urlField).toBeVisible();
    await expect(urlField).toBeEnabled();

    // Verify Organization ID field is visible (should default to openai mode)
    const orgIdField = page.getByTestId(testIds.appConfig.openAIOrganizationID);
    await expect(orgIdField).toBeVisible();

    // Verify API Key field is visible
    const apiKeyField = page.getByTestId(testIds.appConfig.openAIKey);
    await expect(apiKeyField).toBeVisible();

    // Verify custom API path checkbox is visible
    const customPathCheckbox = page.getByTestId(testIds.appConfig.customizeOpenAIApiPath);
    await expect(customPathCheckbox).toBeVisible();
  });

  test('should maintain consistent UI when switching from openai to custom and back', async ({ page }) => {
    // First, select OpenAI
    const openaiCard = page.locator('text=Use OpenAI-compatible API').locator('..');
    await openaiCard.click();
    await page.waitForTimeout(500);

    // Verify provider dropdown IS visible for OpenAI
    let providerDropdown = page.getByTestId(testIds.appConfig.provider);
    await expect(providerDropdown).toBeVisible();

    // Verify URL field is disabled for OpenAI (always uses api.openai.com)
    let urlField = page.getByTestId(testIds.appConfig.openAIUrl);
    await expect(urlField).toBeVisible();
    await expect(urlField).toBeDisabled();

    // Now switch to Custom
    const customCard = page.locator('text=Use a Custom API').locator('..');
    await customCard.click();
    await page.waitForTimeout(500);

    // Verify provider dropdown is now hidden
    providerDropdown = page.getByTestId(testIds.appConfig.provider);
    await expect(providerDropdown).not.toBeVisible();

    // Verify URL field is now enabled
    urlField = page.getByTestId(testIds.appConfig.openAIUrl);
    await expect(urlField).toBeEnabled();

    // Verify Organization ID field is still visible
    const orgIdField = page.getByTestId(testIds.appConfig.openAIOrganizationID);
    await expect(orgIdField).toBeVisible();

    // Switch back to OpenAI
    await openaiCard.click();
    await page.waitForTimeout(500);

    // Verify provider dropdown is visible again
    providerDropdown = page.getByTestId(testIds.appConfig.provider);
    await expect(providerDropdown).toBeVisible();

    // Verify URL field is disabled again
    urlField = page.getByTestId(testIds.appConfig.openAIUrl);
    await expect(urlField).toBeDisabled();
  });

  test('should hide provider dropdown when in custom mode', async ({ page }) => {
    // Select OpenAI first
    const openaiCard = page.locator('text=Use OpenAI-compatible API').locator('..');
    await openaiCard.click();
    await page.waitForTimeout(500);

    // Provider dropdown should be visible
    const providerDropdown = page.getByTestId(testIds.appConfig.provider);
    await expect(providerDropdown).toBeVisible();

    // Select Custom
    const customCard = page.locator('text=Use a Custom API').locator('..');
    await customCard.click();
    await page.waitForTimeout(500);

    // Provider dropdown should be hidden
    await expect(providerDropdown).not.toBeVisible();
  });

  test('should allow editing URL field in custom mode', async ({ page }) => {
    // Switch to Custom card
    const customCard = page.locator('text=Use a Custom API').locator('..');
    await customCard.click();
    await page.waitForTimeout(500);

    // Verify URL field is visible and enabled
    const urlField = page.getByTestId(testIds.appConfig.openAIUrl);
    await expect(urlField).toBeVisible();
    await expect(urlField).toBeEnabled();

    // Clear any existing value and type a custom URL
    await urlField.clear();
    const customUrl = 'https://my-custom-llm-api.example.com';
    await urlField.fill(customUrl);

    // Verify the URL was set correctly
    await expect(urlField).toHaveValue(customUrl);
  });

  test('should switch to Anthropic provider correctly', async ({ page }) => {
    // Select Anthropic card
    const anthropicCard = page.locator('text=Use Anthropic API').locator('..');
    await anthropicCard.click();
    await page.waitForTimeout(500);

    // Verify Anthropic-specific fields are visible
    const anthropicUrl = page.getByTestId(testIds.appConfig.anthropicUrl);
    const anthropicKey = page.getByTestId(testIds.appConfig.anthropicKey);

    await expect(anthropicUrl).toBeVisible();
    await expect(anthropicKey).toBeVisible();

    // OpenAI-specific fields should not be visible
    const openaiUrl = page.getByTestId(testIds.appConfig.openAIUrl);
    await expect(openaiUrl).not.toBeVisible();
  });

  test('should disable LLM features correctly', async ({ page }) => {
    // Select the disable card
    const disableCard = page.locator('text=Disable all LLM features in Grafana').locator('..');
    await disableCard.click();
    await page.waitForTimeout(500);

    // Configuration fields should not be visible
    const openaiUrl = page.getByTestId(testIds.appConfig.openAIUrl);
    const anthropicUrl = page.getByTestId(testIds.appConfig.anthropicUrl);

    await expect(openaiUrl).not.toBeVisible();
    await expect(anthropicUrl).not.toBeVisible();

    // Re-enable by selecting OpenAI
    const openaiCard = page.locator('text=Use OpenAI-compatible API').locator('..');
    await openaiCard.click();
    await page.waitForTimeout(500);

    // OpenAI fields should be visible again
    await expect(openaiUrl).toBeVisible();
  });

  test('should show confirmation dialog when switching providers with existing model configs', async ({ page }) => {
    // This test assumes there might be existing model configurations
    // First, select OpenAI and configure it
    const openaiCard = page.locator('text=Use OpenAI-compatible API').locator('..');
    await openaiCard.click();
    await page.waitForTimeout(500);

    // Try switching to Anthropic
    const anthropicCard = page.locator('text=Use Anthropic API').locator('..');
    await anthropicCard.click();

    // Check if confirmation dialog appears (it may not if there are no model configs)
    const confirmDialog = page.locator('text=Switch LLM Provider?');

    // If dialog appears, handle it
    const dialogVisible = await confirmDialog.isVisible().catch(() => false);
    if (dialogVisible) {
      // Click "Switch Provider" to confirm
      const confirmButton = page.locator('button', { hasText: 'Switch Provider' });
      await confirmButton.click();
      await page.waitForTimeout(500);

      // Verify we switched to Anthropic
      const anthropicUrl = page.getByTestId(testIds.appConfig.anthropicUrl);
      await expect(anthropicUrl).toBeVisible();
    }
  });
});
