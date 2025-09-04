# Grafana LLM App Go Client

A Go client library for interacting with LLM providers through the Grafana LLM App. This client provides a unified interface to various LLM providers (OpenAI, Azure OpenAI, etc.) with authentication and configuration managed by the Grafana LLM App plugin.

## Features

- **Provider Abstraction**: Use abstract model names (`base`, `large`) instead of provider-specific model names
- **Unified Authentication**: Authentication handled through Grafana API keys
- **Health Checking**: Check if LLM providers are properly configured and available
- **Streaming Support**: Support for both regular and streaming chat completions
- **OpenAI Compatible**: Built on top of the popular `go-openai` library

## Installation

```bash
go get github.com/grafana/grafana-llm-app/llmclient
```

## Authentication

The client uses **Grafana service account tokens** for authentication, not direct LLM provider API keys. This approach provides several benefits:

1. **Centralized Configuration**: LLM provider credentials are managed in Grafana
2. **Security**: No need to distribute LLM provider API keys to client applications
3. **Access Control**: Leverage Grafana's existing authentication and authorization
4. **Provider Abstraction**: Switch between LLM providers without changing client code

### Setting up Authentication

1. **Create a Service Account Token**: Create a service account and token in your Grafana instance. See the [official Grafana documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/#add-a-token-to-a-service-account-in-grafana) for detailed instructions.
2. **Configure the LLM App**: Ensure the Grafana LLM App plugin is installed and configured with your preferred LLM provider
3. **Use the Token**: Pass the service account token to the client (not your OpenAI/Azure API key)

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/grafana/grafana-llm-app/llmclient"
    "github.com/sashabaranov/go-openai"
)

func main() {
    // Initialize client with Grafana URL and service account token
    client := llmclient.NewLLMProvider(
        "https://your-grafana-instance.com",
        "your-service-account-token",
    )

    ctx := context.Background()

    // Check if LLM provider is available
    enabled, err := client.Enabled(ctx)
    if err != nil {
        log.Fatal(err)
    }
    if !enabled {
        log.Fatal("LLM provider is not enabled or configured")
    }

    // Make a chat completion request
    req := llmclient.ChatCompletionRequest{
        ChatCompletionRequest: openai.ChatCompletionRequest{
            Messages: []openai.ChatCompletionMessage{
                {Role: "user", Content: "Hello, how are you?"},
            },
        },
        Model: llmclient.ModelBase,
    }

    resp, err := client.ChatCompletions(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(resp.Choices[0].Message.Content)
}
```

## Available Models

The client provides abstract model types that map to the best available model for each tier from your configured LLM provider:

- **`ModelBase`**: Optimized for efficiency and high-throughput tasks
- **`ModelLarge`**: Advanced model with longer context windows for complex tasks

These abstract models automatically resolve to the appropriate provider-specific models (e.g., `gpt-3.5-turbo` vs `gpt-4` for OpenAI, or equivalent models for other providers).

## API Reference

### Creating a Client

#### `NewLLMProvider(grafanaURL, serviceAccountToken string) LLMProvider`

Creates a new LLM provider client.

```go
client := llmclient.NewLLMProvider(
    "https://grafana.example.com",
    "your-service-account-token-here",
)
```

#### `NewLLMProviderWithClient(grafanaURL, serviceAccountToken string, httpClient *http.Client) LLMProvider`

Creates a client with a custom HTTP client for advanced configuration.

```go
httpClient := &http.Client{
    Timeout: time.Minute * 5,
}
client := llmclient.NewLLMProviderWithClient(
    "https://grafana.example.com",
    "your-service-account-token-here",
    httpClient,
)
```

### Health Checking

#### `Enabled(ctx context.Context) (bool, error)`

Checks if the LLM provider is properly configured and available.

```go
enabled, err := client.Enabled(ctx)
if err != nil {
    // Handle error
}
if !enabled {
    // LLM provider is not available
}
```

### Chat Completions

#### `ChatCompletions(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error)`

Makes a synchronous chat completion request.

```go
req := llmclient.ChatCompletionRequest{
    ChatCompletionRequest: openai.ChatCompletionRequest{
        Messages: []openai.ChatCompletionMessage{
            {Role: "system", Content: "You are a helpful assistant."},
            {Role: "user", Content: "Explain quantum computing"},
        },
        MaxTokens:   150,
        Temperature: 0.7,
    },
    Model: llmclient.ModelLarge, // Use the large model for complex tasks
}

resp, err := client.ChatCompletions(ctx, req)
if err != nil {
    log.Fatal(err)
}

for _, choice := range resp.Choices {
    fmt.Println(choice.Message.Content)
}
```

#### `ChatCompletionsStream(ctx context.Context, req ChatCompletionRequest) (*openai.ChatCompletionStream, error)`

Makes a streaming chat completion request.

```go
req := llmclient.ChatCompletionRequest{
    ChatCompletionRequest: openai.ChatCompletionRequest{
        Messages: []openai.ChatCompletionMessage{
            {Role: "user", Content: "Write a short story"},
        },
        Stream: true,
    },
    Model: llmclient.ModelLarge,
}

stream, err := client.ChatCompletionsStream(ctx, req)
if err != nil {
    log.Fatal(err)
}
defer stream.Close()

for {
    response, err := stream.Recv()
    if errors.Is(err, io.EOF) {
        break
    }
    if err != nil {
        log.Fatal(err)
    }

    if len(response.Choices) > 0 {
        fmt.Print(response.Choices[0].Delta.Content)
    }
}
```

## Error Handling

Errors are propagated directly from the underlying `go-openai` library. Refer to the [official documentation](https://github.com/sashabaranov/go-openai#other-examples) for more information.

## Advanced Configuration

### Using with Custom HTTP Client

For production use, you may want to configure timeouts, retry logic, or other HTTP client settings:

```go
httpClient := &http.Client{
    Timeout: time.Minute * 2,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     time.Minute,
    },
}

client := llmclient.NewLLMProviderWithClient(
    "https://grafana.example.com",
    "service-account-token",
    httpClient,
)
```

## Requirements

- Go 1.19 or later
- Access to a Grafana instance with the LLM App plugin installed and configured
- Valid Grafana service account token with appropriate permissions

## License

This library is part of the Grafana LLM App project. Please refer to the main project for license information.
