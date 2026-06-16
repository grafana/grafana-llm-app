import { enabled } from "./openai";
import { LLM_PLUGIN_ROUTE } from "./constants";
import { getBackendSrv } from "@grafana/runtime";

jest.mock("@grafana/runtime", () => ({
  getBackendSrv: jest.fn(),
}));

describe("enabled", () => {
  it("should return false if not configured", async () => {
    (getBackendSrv as jest.Mock).mockImplementation(() => ({
      get: jest.fn().mockReturnValue(Promise.resolve({ enabled: false })),
    }));

    // Call the enabled function
    const result = await enabled();
    expect(result).toBe(false);
  });

  it("should return true if configured with new llmProvider format", async () => {
    (getBackendSrv as jest.Mock).mockImplementation(() => ({
      get: jest.fn().mockImplementation((url: string) => {
        if (url === `${LLM_PLUGIN_ROUTE}/settings`) {
          return Promise.resolve({ enabled: true });
        } else if (url === `${LLM_PLUGIN_ROUTE}/health`) {
          return Promise.resolve({
            details: { llmProvider: { configured: true, ok: true } },
          });
        }
        throw new Error("unexpected url");
      }),
    }));

    const result = await enabled();
    expect(result).toBe(true);
  });

  it("should return true if configured with legacy openAI format", async () => {
    (getBackendSrv as jest.Mock).mockImplementation(() => ({
      get: jest.fn().mockImplementation((url: string) => {
        if (url === `${LLM_PLUGIN_ROUTE}/settings`) {
          return Promise.resolve({ enabled: true });
        } else if (url === `${LLM_PLUGIN_ROUTE}/health`) {
          return Promise.resolve({
            details: { openAI: { configured: true, ok: true } },
          });
        }
        throw new Error("unexpected url");
      }),
    }));

    const result = await enabled();
    expect(result).toBe(true);
  });

  it("should return false if neither format is present", async () => {
    (getBackendSrv as jest.Mock).mockImplementation(() => ({
      get: jest.fn().mockImplementation((url: string) => {
        if (url === `${LLM_PLUGIN_ROUTE}/settings`) {
          return Promise.resolve({ enabled: true });
        } else if (url === `${LLM_PLUGIN_ROUTE}/health`) {
          return Promise.resolve({ details: {} });
        }
        throw new Error("unexpected url");
      }),
    }));

    const result = await enabled();
    expect(result).toBe(false);
  });
});
