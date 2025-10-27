const { grafanaLLMESModules } = require("@grafana/llm/jest");

// Helper function to transform specific packages in node_modules
const nodeModulesToTransform = (moduleNames) =>
  `node_modules\/(?!(${moduleNames.join("|")})\/)`;

// Array of known nested grafana package dependencies that only bundle an ESM version
const grafanaESModules = [
  ".pnpm", // Support using pnpm symlinked packages
  "d3",
  "d3-color",
  "d3-force",
  "d3-interpolate",
  "d3-scale-chromatic",
  "ol",
  "react-colorful",
  "uuid",
  "delaunator",
  "internmap",
  "robust-predicates",
];

// Add our additional ES modules to the list
const additionalESModules = [...grafanaLLMESModules, ...grafanaESModules];

// force timezone to UTC to allow tests to work regardless of local timezone
// generally used by snapshots, but can affect specific tests
process.env.TZ = "UTC";

module.exports = {
  testEnvironment: "jest-environment-jsdom",
  testMatch: ["<rootDir>/src/**/*.{spec,test,jest}.{js,jsx,ts,tsx}"],
  transform: {
    "^.+\\.(t|j)sx?$": [
      "@swc/jest",
      {
        sourceMaps: "inline",
        jsc: {
          parser: {
            syntax: "typescript",
            tsx: true,
            decorators: false,
            dynamicImport: true,
          },
          target: "es2022",
        },
      },
    ],
  },
  // Jest will throw `Cannot use import statement outside module` if it tries to load an
  // ES module without it being transformed first.
  transformIgnorePatterns: [nodeModulesToTransform(additionalESModules)],
};
