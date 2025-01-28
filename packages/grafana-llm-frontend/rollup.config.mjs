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
      },
      {
        format: "esm",
        sourcemap: env === "production" ? true : "inline",
        dir: path.dirname(pkg.module),
        preserveModules: true,
      },
    ],
    watch: {
      include: "./src/**/*",
      clearScreen: false
    },
  },
  {
    input: "src/index.ts",
    plugins: [dts()],
    output: {
      file: "./dist/index.d.ts",
      format: "es",
    },
  },
];
