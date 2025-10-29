const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');
const { grafanaLLMESModules } = require('@grafana/llm/jest');

// Add our additional ES modules to the list
const additionalESModules = [
  ...grafanaLLMESModules,
  ...grafanaESModules,
  'react-calendar',
  'get-user-locale',
  'memoize',
  'mimic-function',
  '@wojtekmaj/date-utils',
];

// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = 'UTC';

module.exports = {
  // Jest configuration provided by Grafana scaffolding
  ...require('./.config/jest.config'),
  // Override the transformIgnorePatterns to include our additional modules
  transformIgnorePatterns: [nodeModulesToTransform(additionalESModules)],
  // Map @grafana/llm to source files to avoid React duplication issues in tests
  moduleNameMapper: {
    ...require('./.config/jest.config').moduleNameMapper,
    '^@grafana/llm$': '<rootDir>/../grafana-llm-frontend/src/index.ts',
    '^@grafana/llm/jest$': '<rootDir>/../grafana-llm-frontend/src/jest.ts',
    // Force React to resolve to a single instance
    '^react$': require.resolve('react'),
    '^react-dom$': require.resolve('react-dom'),
  },
};
