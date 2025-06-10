const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');
const { grafanaLLMESModules } = require('@grafana/llm/jest');

// Add our additional ES modules to the list
const additionalESModules = [...grafanaLLMESModules, ...grafanaESModules];

// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = 'UTC';

module.exports = {
  // Jest configuration provided by Grafana scaffolding
  ...require('./.config/jest.config'),
  // Override the transformIgnorePatterns to include our additional modules
  transformIgnorePatterns: [nodeModulesToTransform(additionalESModules)],
};
