import { enabled } from './openai';
import { LLM_PLUGIN_ROUTE } from './constants';
import { getBackendSrv } from '@grafana/runtime';

jest.mock('@grafana/runtime', () => ({
  getBackendSrv: jest.fn(),
}));

describe('enabled', () => {
  it('should return false if not configured', async () => {
    (getBackendSrv as jest.Mock).mockImplementation(() => ({
      get: jest.fn().mockReturnValue(Promise.resolve({ enabled: false })),
    }));

    // Call the enabled function
    const result = await enabled();
    expect(result).toBe(false);
  });

  it('should return true if configured', async () => {
    (getBackendSrv as jest.Mock).mockImplementation(() => ({
      get: jest.fn().mockImplementation((url: string) => {
        if (url === `${LLM_PLUGIN_ROUTE}/settings`) {
          return Promise.resolve({ enabled: true });
        } else if (url === `${LLM_PLUGIN_ROUTE}/health`) {
          return Promise.resolve({ details: { openAI: { configured: true, ok: true } } });
        }
        // raise an error if we get here
        throw new Error('unexpected url');
      }),
    }));

    // Call the enabled function
    const result = await enabled();
    expect(result).toBe(true);
  });
});
