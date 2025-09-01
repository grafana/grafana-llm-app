import resolve from "@rollup/plugin-node-resolve";
import path from "path";
import dts from "rollup-plugin-dts";
import esbuild from "rollup-plugin-esbuild";
import externals from "rollup-plugin-node-externals";

import pkg from "./package.json" with { type: "json" };

const env = process.env.NODE_ENV || "production";

const plugins = [
  externals({ deps: true, devDeps: true, packagePath: "./package.json" }),
  resolve({ browser: true }),
  esbuild(),
];

export default [
  {
    input: "src/index.ts",
    plugins: plugins,
    output: [
      {
        format: "cjs",
        sourcemap: env === "production" ? true : "inline",
        dir: path.dirname(pkg.main),
        entryFileNames: "[name].cjs"
      },
      {
        format: "esm",
        sourcemap: env === "production" ? true : "inline",
        dir: path.dirname(pkg.module),
        preserveModules: true,
        entryFileNames: "[name].mjs"
      },
    ],
    watch: {
      include: "./src/**/*",
      clearScreen: false
    },
  },
  {
    input: "src/jest.ts",
    plugins: plugins,
    output: [
      {
        format: "cjs",
        sourcemap: env === "production" ? true : "inline",
        file: "./dist/jest.cjs",
      },
      {
        format: "esm",
        sourcemap: env === "production" ? true : "inline",
        file: "./dist/esm/jest.mjs",
      },
    ],
  },
  {
    input: "src/index.ts",
    plugins: [dts()],
    output: [
      {
        file: "./dist/esm/index.d.mts",
        format: "es",
      },
      {
        file: "./dist/index.d.cts",
        format: "cjs",
      },
    ],
  },
  {
    input: "src/jest.ts",
    plugins: [dts()],
    output: [
      {
        file: "./dist/esm/jest.d.mts",
        format: "es",
      },
      {
        file: "./dist/jest.d.cts",
        format: "cjs",
      },
    ],
  },
];
