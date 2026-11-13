import type { Configuration } from 'webpack';
import grafanaConfig from './.config/webpack/webpack.config';

export default async function config(env: Parameters<typeof grafanaConfig>[0]): Promise<Configuration> {
  const baseConfig = await grafanaConfig(env);
  const externals = baseConfig.externals ?? [];

  return {
    ...baseConfig,
    // The compatibility shim must detect APIs on the host, not a bundled runtime copy.
    externals: [...(Array.isArray(externals) ? externals : [externals]), '@grafana/runtime/unstable'],
  };
}
