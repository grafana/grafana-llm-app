import {
  enabled,
  extractContent,
  ChatCompletionsResponse,
  ChatCompletionsChunk,
} from "./llm";
import { LLM_PLUGIN_ROUTE } from "./constants";
import { getBackendSrv } from "@grafana/runtime";
import { of } from "rxjs";

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

  it("should return true if configured", async () => {
    (getBackendSrv as jest.Mock).mockImplementation(() => ({
      get: jest.fn().mockImplementation((url: string) => {
        if (url === `${LLM_PLUGIN_ROUTE}/settings`) {
          return Promise.resolve({ enabled: true });
        } else if (url === `${LLM_PLUGIN_ROUTE}/health`) {
          return Promise.resolve({
            details: { llmProvider: { configured: true, ok: true } },
          });
        }
        // raise an error if we get here
        throw new Error("unexpected url");
      }),
    }));

    // Call the enabled function
    const result = await enabled();
    expect(result).toBe(true);
  });
});

describe("extractContent", () => {
  it("should handle empty choices array without throwing", (done) => {
    const emptyResponse: ChatCompletionsResponse<ChatCompletionsChunk> = {
      id: "test-id",
      object: "chat.completion.chunk",
      created: Date.now(),
      model: "test-model",
      choices: [], // Empty choices array
      usage: {
        prompt_tokens: 0,
        completion_tokens: 0,
        total_tokens: 0,
      },
    };

    const source$ = of(emptyResponse);
    const result$ = source$.pipe(extractContent());

    const results: string[] = [];
    result$.subscribe({
      next: (value) => results.push(value),
      error: (err) => {
        // Should not error
        done(err);
      },
      complete: () => {
        // Should complete successfully with no emitted values
        expect(results).toEqual([]);
        done();
      },
    });
  });

  it("should extract content from valid content messages", (done) => {
    const validResponse: ChatCompletionsResponse<ChatCompletionsChunk> = {
      id: "test-id",
      object: "chat.completion.chunk",
      created: Date.now(),
      model: "test-model",
      choices: [
        {
          delta: {
            content: "Hello, world!",
            role: "assistant" as const,
          },
        },
      ],
      usage: {
        prompt_tokens: 10,
        completion_tokens: 5,
        total_tokens: 15,
      },
    };

    const source$ = of(validResponse);
    const result$ = source$.pipe(extractContent());

    const results: string[] = [];
    result$.subscribe({
      next: (value) => results.push(value),
      error: (err) => {
        done(err);
      },
      complete: () => {
        expect(results).toEqual(["Hello, world!"]);
        done();
      },
    });
  });
});
