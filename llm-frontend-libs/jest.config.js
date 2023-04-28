const standard = require('@grafana/toolkit/src/config/jest.plugin.config');
const config = standard.jestConfig();

// This process will use the same config that `yarn test` is using
module.exports = {
  ...config,
  watchPathIgnorePatterns: ['<rootDir>/node_modules/'],
  setupFilesAfterEnv: ['@testing-library/jest-dom/extend-expect'],
};
