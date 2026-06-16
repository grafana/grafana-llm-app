// Mock problematic imports before importing the module
jest.mock("@modelcontextprotocol/sdk/client/index", () => ({
  Client: jest.fn(),
}));

jest.mock("@modelcontextprotocol/sdk/client/streamableHttp", () => ({
  StreamableHTTPClientTransport: jest.fn(),
}));

jest.mock("@modelcontextprotocol/sdk/types", () => ({
  JSONRPCMessageSchema: {
    parse: jest.fn(),
  },
}));

jest.mock("@grafana/runtime", () => ({
  getBackendSrv: jest.fn(),
  logDebug: jest.fn(),
  getGrafanaLiveSrv: jest.fn(),
  config: {
    appUrl: "http://localhost:3000/",
  },
  isLiveChannelMessageEvent: jest.fn(),
  LiveChannelScope: {
    Plugin: "plugin",
  },
}));

jest.mock("rxjs", () => ({
  Observable: jest.fn(),
  filter: jest.fn(),
}));

jest.mock("uuid", () => ({
  v4: jest.fn(() => "test-uuid"),
}));

import { enabled } from "./mcp";
import { LLM_PLUGIN_ROUTE } from "./constants";
import { getBackendSrv, logDebug } from "@grafana/runtime";

describe("mcp enabled function", () => {
  let mockGet: jest.Mock;

  beforeEach(() => {
    jest.clearAllMocks();
    mockGet = jest.fn();
    (getBackendSrv as jest.Mock).mockImplementation(() => ({
      get: mockGet,
    }));
  });

  it("should return false if plugin is not enabled", async () => {
    mockGet.mockResolvedValue({
      enabled: false,
      jsonData: {},
    });

    const result = await enabled();

    expect(result).toBe(false);
    expect(mockGet).toHaveBeenCalledWith(
      `${LLM_PLUGIN_ROUTE}/settings`,
      undefined,
      undefined,
      {
        showSuccessAlert: false,
        showErrorAlert: false,
      },
    );
  });

  it("should return false if plugin is enabled but MCP is disabled (legacy enabled property)", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {
          enabled: false,
        },
      },
    });

    const result = await enabled();

    expect(result).toBe(false);
  });

  it("should return true if plugin is enabled and MCP is enabled (legacy enabled property)", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {
          enabled: true,
        },
      },
    });

    const result = await enabled();

    expect(result).toBe(true);
  });

  it("should handle truthy values for legacy enabled property", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {
          enabled: 1,
        },
      },
    });

    const result = await enabled();

    expect(result).toBe(true);
  });

  it("should handle falsy values for legacy enabled property", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {
          enabled: 0,
        },
      },
    });

    const result = await enabled();

    expect(result).toBe(false);
  });

  it("should return false if plugin is enabled but MCP is disabled (new disabled property)", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {
          disabled: true,
        },
      },
    });

    const result = await enabled();

    expect(result).toBe(false);
  });

  it("should return true if plugin is enabled and MCP is not disabled (new disabled property)", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {
          disabled: false,
        },
      },
    });

    const result = await enabled();

    expect(result).toBe(true);
  });

  it("should return true if plugin is enabled and MCP disabled property is undefined", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {},
      },
    });

    const result = await enabled();

    expect(result).toBe(true);
  });

  it("should return true if plugin is enabled and no MCP configuration exists", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {},
    });

    const result = await enabled();

    expect(result).toBe(true);
  });

  it("should prioritize legacy enabled property over disabled property", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {
          enabled: true,
          disabled: true,
        },
      },
    });

    const result = await enabled();

    expect(result).toBe(true);
  });

  it("should prioritize legacy enabled=false over disabled=false", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: {
          enabled: false,
          disabled: false,
        },
      },
    });

    const result = await enabled();

    expect(result).toBe(false);
  });

  it("should return false and log debug messages when API call fails", async () => {
    const error = new Error("API call failed");
    mockGet.mockRejectedValue(error);

    const result = await enabled();

    expect(result).toBe(false);
    expect(logDebug).toHaveBeenCalledWith("Error: API call failed");
    expect(logDebug).toHaveBeenCalledWith(
      "Failed to check if LLM provider is enabled. This is expected if the Grafana LLM plugin is not installed, and the above error can be ignored.",
    );
  });

  it("should handle non-Error exceptions", async () => {
    const error = "String error";
    mockGet.mockRejectedValue(error);

    const result = await enabled();

    expect(result).toBe(false);
    expect(logDebug).toHaveBeenCalledWith("String error");
  });

  it("should handle null/undefined exceptions", async () => {
    mockGet.mockRejectedValue(null);

    const result = await enabled();

    expect(result).toBe(false);
    expect(logDebug).toHaveBeenCalledWith("null");
  });

  it("should handle empty jsonData", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: null,
    });

    const result = await enabled();

    expect(result).toBe(false);
  });

  it("should handle missing jsonData property", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
    });

    const result = await enabled();

    expect(result).toBe(false);
  });

  it("should handle null mcp configuration", async () => {
    mockGet.mockResolvedValue({
      enabled: true,
      jsonData: {
        mcp: null,
      },
    });

    const result = await enabled();

    expect(result).toBe(true);
  });
});
