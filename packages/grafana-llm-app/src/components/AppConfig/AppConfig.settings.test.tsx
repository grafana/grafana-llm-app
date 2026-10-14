import { getBackendSrv } from '@grafana/runtime';
import { of } from 'rxjs';
import { updateAndSavePluginSettings } from './AppConfig';

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: jest.fn(),
}));

jest.mock('@grafana/runtime/unstable', () => ({
  ...jest.requireActual('@grafana/runtime/unstable'),
  updateAppPluginSettings: jest.fn(),
}));

const runtimeSettings = jest.requireMock<{ updateAppPluginSettings: jest.Mock | undefined }>(
  '@grafana/runtime/unstable'
);
const nativeUpdate = runtimeSettings.updateAppPluginSettings as jest.Mock;

const pluginId = 'grafana-llm-app';
const settings = {
  enabled: true,
  jsonData: { provider: 'openai' },
  secureJsonData: { openAIKey: 'test-key' },
};

describe('plugin settings API compatibility', () => {
  afterEach(() => {
    runtimeSettings.updateAppPluginSettings = nativeUpdate;
  });
  beforeEach(() => {
    jest.resetAllMocks();
    runtimeSettings.updateAppPluginSettings = nativeUpdate;
  });

  test('uses the runtime settings API when the legacy route is unavailable', async () => {
    const fetch = jest.fn(() => {
      throw new Error('Legacy plugin settings route is unavailable');
    });
    (getBackendSrv as jest.Mock).mockReturnValue({ fetch });
    nativeUpdate.mockResolvedValue({ id: pluginId, ...settings });

    await expect(updateAndSavePluginSettings(pluginId, false, settings)).resolves.toBeUndefined();
    expect(nativeUpdate).toHaveBeenCalledWith(pluginId, settings);
    expect(fetch).not.toHaveBeenCalled();
  });

  test('falls back to the legacy API on older Grafana versions', async () => {
    runtimeSettings.updateAppPluginSettings = undefined;
    const post = jest.fn().mockResolvedValue({});
    const get = jest.fn().mockResolvedValue({ id: pluginId, ...settings });
    (getBackendSrv as jest.Mock).mockReturnValue({ post, get });

    await expect(updateAndSavePluginSettings(pluginId, false, settings)).resolves.toBeUndefined();
    expect(post).toHaveBeenCalledWith(`/api/plugins/${pluginId}/settings`, settings, { validatePath: true });
    expect(get).toHaveBeenCalledWith(`/api/plugins/${pluginId}/settings`, undefined, undefined, {
      showErrorAlert: false,
      validatePath: true,
    });
    expect(nativeUpdate).not.toHaveBeenCalled();
  });

  test('propagates native API failures', async () => {
    const error = new Error('Saving settings failed');
    nativeUpdate.mockRejectedValue(error);

    await expect(updateAndSavePluginSettings(pluginId, false, settings)).rejects.toBe(error);
  });

  test('persists Grafana.com settings before updating Grafana', async () => {
    const fetch = jest.fn().mockReturnValue(of({ ok: true, data: {} }));
    (getBackendSrv as jest.Mock).mockReturnValue({ fetch });
    nativeUpdate.mockResolvedValue({ id: pluginId, ...settings });

    await updateAndSavePluginSettings(pluginId, true, settings);

    expect(fetch).toHaveBeenCalledWith({
      url: '/api/plugins/grafana-llm-app/resources/save-plugin-settings',
      method: 'POST',
      data: { jsonData: settings.jsonData, secureJsonData: settings.secureJsonData },
    });
    expect(nativeUpdate).toHaveBeenCalledWith(pluginId, settings);
    expect(fetch.mock.invocationCallOrder[0]).toBeLessThan(nativeUpdate.mock.invocationCallOrder[0]);
  });

  test('does not update Grafana if persisting to Grafana.com fails', async () => {
    const error = { message: 'Provisioning update failed' };
    const fetch = jest.fn().mockReturnValue(of({ ok: false, data: error }));
    (getBackendSrv as jest.Mock).mockReturnValue({ fetch });

    await expect(updateAndSavePluginSettings(pluginId, true, settings)).rejects.toBe(error);
    expect(nativeUpdate).not.toHaveBeenCalled();
  });
});
