/**
 * Jest configuration utilities for @grafana/llm
 *
 * This module exports the additional ES modules that need to be transformed
 * when using @grafana/llm in Jest tests.
 */

/**
 * List of ES modules used by @grafana/llm that need to be transformed by Jest.
 * Add these to your Jest transformIgnorePatterns configuration.
 */
export const grafanaLLMESModules = [
  "@modelcontextprotocol/sdk",
  "pkce-challenge",
  "marked",
];

/**
 * Helper function to create transformIgnorePatterns for Jest.
 * Use this with Grafana's nodeModulesToTransform utility.
 *
 * @example
 * ```javascript
 * // jest.config.js
 * const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');
 * const { grafanaLLMESModules } = require('@grafana/llm/jest');
 *
 * module.exports = {
 *   ...require('./.config/jest.config'),
 *   transformIgnorePatterns: [nodeModulesToTransform([...grafanaESModules, ...grafanaLLMESModules])],
 * };
 * ```
 */
