const { grafanaESModules, nodeModulesToTransform } = require('./.config/jest/utils');
const { grafanaLLMESModules } = require('@grafana/llm/jest');

// Add our additional ES modules to the list
const additionalESModules = [
  ...grafanaLLMESModules,
  ...grafanaESModules,
  'react-calendar',
  'get-user-locale',
  "marked",
  'memoize',
  'mimic-function',
  '@wojtekmaj/date-utils',
  '@react-hookz/web',
  '@ver0',
];

// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = 'UTC';

module.exports = {
  // Jest configuration provided by Grafana scaffolding
  ...require('./.config/jest.config'),
  // d3 patch releases ship .mjs only; the scaffolded transform pattern doesn't match .mjs
  transform: Object.fromEntries(
    Object.entries(require('./.config/jest.config').transform).map(([k, v]) => [
      k === '^.+\\.(t|j)sx?$' ? '^.+\\.(t|j|mj)sx?$' : k,
      v,
    ])
  ),
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
