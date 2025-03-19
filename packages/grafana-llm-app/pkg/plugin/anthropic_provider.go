package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/sashabaranov/go-openai"
)

type anthropicProvider struct {
	settings AnthropicSettings
	models   *ModelSettings
	client   *anthropic.Client
}

func NewAnthropicProvider(settings AnthropicSettings, models *ModelSettings) (LLMProvider, error) {
	httpClientOpts := option.WithHTTPClient(&http.Client{
		Timeout: 2 * time.Minute,
	})
	client := anthropic.NewClient(
		option.WithAPIKey(settings.apiKey),
		httpClientOpts,
	)

	if settings.URL != "" {
		client = anthropic.NewClient(
			option.WithAPIKey(settings.apiKey),
			option.WithBaseURL(settings.URL),
			httpClientOpts,
		)
	}

	defaultModels := &ModelSettings{
		Default: ModelBase,
		Mapping: map[Model]string{
			ModelBase:  anthropic.ModelClaude3_5HaikuLatest,
			ModelLarge: anthropic.ModelClaude3_5SonnetLatest,
		},
	}

	return &anthropicProvider{
		settings: settings,
		models:   defaultModels,
		client:   client,
	}, nil
}

func (p *anthropicProvider) Models(ctx context.Context) (ModelResponse, error) {
	return ModelResponse{
		Data: []ModelInfo{
			{ID: ModelBase},
			{ID: ModelLarge},
		},
	}, nil
}

func convertToAnthropicToolUse(toolCall openai.ToolCall) anthropic.ToolUseBlockParam {
	var args map[string]any
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		log.DefaultLogger.Error("error unmarshalling tool call arguments, using empty map", "err", err)
		args = make(map[string]any)
	}
	return anthropic.NewToolUseBlockParam(toolCall.ID, toolCall.Function.Name, args)
}

func convertToAnthropicMessages(messages []openai.ChatCompletionMessage) ([]anthropic.MessageParam, string) {
	anthropicMessages := make([]anthropic.MessageParam, 0, len(messages))
	var systemPrompt string

	// Extract system message if present
	if len(messages) > 0 && messages[0].Role == "system" {
		systemPrompt = messages[0].Content
	}

	for _, msg := range messages {
		switch msg.Role {
		case openai.ChatMessageRoleUser:
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case openai.ChatMessageRoleAssistant:
			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]anthropic.ContentBlockParamUnion, 0, len(msg.ToolCalls))
				for _, toolCall := range msg.ToolCalls {
					toolCalls = append(toolCalls, convertToAnthropicToolUse(toolCall))
				}
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(
					toolCalls...,
				))
			}
			if len(msg.Content) > 0 {
				anthropicMessages = append(anthropicMessages, anthropic.NewAssistantMessage(
					anthropic.NewTextBlock(msg.Content),
				))
			}
		case openai.ChatMessageRoleTool:
			anthropicMessages = append(anthropicMessages, anthropic.NewUserMessage(
				anthropic.NewToolResultBlock(msg.ToolCallID, msg.Content, false)),
			)

		case openai.ChatMessageRoleSystem:
			// System messages are handled separately
			continue
		}
	}

	jdoc, _ := json.Marshal(anthropicMessages)
	log.DefaultLogger.Info("anthropic messages", "messages", string(jdoc))

	return anthropicMessages, systemPrompt
}

func convertToOpenAIResponse(resp *anthropic.Message, model string) openai.ChatCompletionResponse {
	choices := make([]openai.ChatCompletionChoice, 0, len(resp.Content))
	for _, block := range resp.Content {
		switch block.Type {
		case anthropic.ContentBlockTypeText:
			choices = append(choices, openai.ChatCompletionChoice{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role:    "assistant",
					Content: block.Text,
				},
				FinishReason: openai.FinishReasonStop,
			})

		case anthropic.ContentBlockTypeToolUse:
			choices = append(choices, openai.ChatCompletionChoice{
				Index: 0,
				Message: openai.ChatCompletionMessage{
					Role: "assistant",
					ToolCalls: []openai.ToolCall{
						{
							Type: openai.ToolTypeFunction,
							ID:   block.ID,
							Function: openai.FunctionCall{
								Name:      block.Name,
								Arguments: string(block.Input),
							},
						},
					},
				},
				FinishReason: openai.FinishReasonToolCalls,
			})
		}
	}
	return openai.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   model,
		Choices: choices,
		Usage: openai.Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
}

func convertToAnthropicTool(tool openai.Tool) anthropic.ToolParam {
	parameters := tool.Function.Parameters
	if parameters == nil {
		parameters = map[string]any{
			"type":       "object",
			"properties": nil,
		}
	}
	return anthropic.ToolParam{
		Name:        anthropic.F(tool.Function.Name),
		Description: anthropic.F(tool.Function.Description),
		InputSchema: anthropic.F(parameters),
	}
}

func (p *anthropicProvider) ChatCompletion(ctx context.Context, req ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	messages, systemPrompt := convertToAnthropicMessages(req.Messages)
	log.DefaultLogger.Info("model", "model", req.Model)
	model := req.Model.toAnthropic(p.models)

	// Anthropic requires a max tokens value
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}

	// Convert tools to `anthropic.ToolUnionUnionParam`, not `anthropic.ToolParam`
	// despite what the Anthropic README says.
	// See https://github.com/anthropics/anthropic-sdk-go/issues/138.
	tools := make([]anthropic.ToolUnionUnionParam, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, convertToAnthropicTool(tool))
	}

	// Create Anthropic request
	params := anthropic.MessageNewParams{
		Model:     anthropic.F(model),
		MaxTokens: anthropic.F(int64(req.MaxTokens)),
		Messages:  anthropic.F(messages),
		Tools:     anthropic.F(tools),
	}

	if systemPrompt != "" {
		params.System = anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(systemPrompt),
		})
	}

	// Make request
	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		log.DefaultLogger.Error("error creating anthropic chat completion", "err", err)
		return openai.ChatCompletionResponse{}, err
	}

	return convertToOpenAIResponse(resp, model), nil
}

func (p *anthropicProvider) ChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (<-chan ChatCompletionStreamResponse, error) {
	messages, systemPrompt := convertToAnthropicMessages(req.Messages)

	// Anthropic requires a max tokens value
	if req.MaxTokens == 0 {
		req.MaxTokens = 1000
	}

	// Convert tools to `anthropic.ToolUnionUnionParam`, not `anthropic.ToolParam`
	// despite what the Anthropic README says.
	// See https://github.com/anthropics/anthropic-sdk-go/issues/138.
	tools := make([]anthropic.ToolUnionUnionParam, 0, len(req.Tools))
	for _, tool := range req.Tools {
		tools = append(tools, convertToAnthropicTool(tool))
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.F(req.Model.toAnthropic(p.models)),
		MaxTokens: anthropic.F(int64(req.MaxTokens)),
		Messages:  anthropic.F(messages),
		Tools:     anthropic.F(tools),
	}

	if systemPrompt != "" {
		params.System = anthropic.F([]anthropic.TextBlockParam{
			anthropic.NewTextBlock(systemPrompt),
		})
	}

	stream := p.client.Messages.NewStreaming(ctx, params)
	c := make(chan ChatCompletionStreamResponse)

	go func() {
		defer close(c)

		message := anthropic.Message{}
		for stream.Next() {
			event := stream.Current()
			if err := message.Accumulate(event); err != nil {
				log.DefaultLogger.Error("error accumulating message", "err", err)
				c <- ChatCompletionStreamResponse{Error: err}
				return
			}

			// See https://docs.anthropic.com/en/api/messages-streaming#raw-http-stream-response
			// for docs on how this flow works.
			switch event := event.AsUnion().(type) {
			case anthropic.MessageStartEvent:
			// We don't need to emit anything here.

			case anthropic.ContentBlockStartEvent:
				// This could be text or a tool call.
				switch block := event.ContentBlock.AsUnion().(type) {
				case anthropic.TextBlock:
				// For text there's nothing useful in the initial start block.

				case anthropic.ThinkingBlock:
				case anthropic.RedactedThinkingBlock:
				// Not sure how to handle these?

				case anthropic.ToolUseBlock:
					// Emit a delta indicating the start of a tool call.
					// This will contain the tool call ID and the tool name, but not the arguments;
					// those will come in later deltas.
					c <- ChatCompletionStreamResponse{
						ChatCompletionStreamResponse: openai.ChatCompletionStreamResponse{
							ID:     message.ID,
							Object: "chat.completion.chunk",
							Choices: []openai.ChatCompletionStreamChoice{
								{
									Index: 0,
									Delta: openai.ChatCompletionStreamChoiceDelta{
										Content: block.Name,
										ToolCalls: []openai.ToolCall{
											{
												Type: openai.ToolTypeFunction,
												ID:   block.ID,
											},
										},
									},
								},
							},
						},
					}

				}

			case anthropic.ContentBlockDeltaEvent:
				// This is the main delta event. For text it contains the text delta. For tool calls
				// it contains the arguments to the tool call as partial JSON.
				switch delta := event.Delta.AsUnion().(type) {
				case anthropic.TextDelta:
					c <- ChatCompletionStreamResponse{
						ChatCompletionStreamResponse: openai.ChatCompletionStreamResponse{
							ID:      message.ID,
							Object:  "chat.completion.chunk",
							Created: time.Now().Unix(),
							Model:   string(req.Model),
							Choices: []openai.ChatCompletionStreamChoice{
								{
									Index: 0,
									Delta: openai.ChatCompletionStreamChoiceDelta{
										Content: delta.Text,
									},
								},
							},
						},
					}
				case anthropic.InputJSONDelta:
					// Emit the partial JSON of the arguments.
					// This should match the output here:
					// https://platform.openai.com/docs/guides/function-calling?api-mode=chat&strict-mode=enabled#streaming.
					c <- ChatCompletionStreamResponse{
						ChatCompletionStreamResponse: openai.ChatCompletionStreamResponse{
							ID:     message.ID,
							Object: "chat.completion.chunk",
							Choices: []openai.ChatCompletionStreamChoice{
								{
									Index: 0,
									Delta: openai.ChatCompletionStreamChoiceDelta{
										Content: string(delta.PartialJSON),
										ToolCalls: []openai.ToolCall{
											{
												Function: openai.FunctionCall{
													Arguments: string(delta.PartialJSON),
												},
											},
										},
									},
								},
							},
						},
					}

				// TODO: do we need to handle these?
				case anthropic.CitationsDelta:
				case anthropic.ThinkingDelta:
				case anthropic.SignatureDelta:

				}
			case anthropic.ContentBlockStopEvent:
			// End of the current block. This doesn't contain anything useful.

			case anthropic.MessageDeltaEvent:
				// This contains the finish reason.
				var finishReason openai.FinishReason
				switch event.Delta.StopReason {
				// https://docs.anthropic.com/en/api/messages#response-stop-reason
				case anthropic.MessageDeltaEventDeltaStopReasonToolUse:
					finishReason = openai.FinishReasonToolCalls
				case anthropic.MessageDeltaEventDeltaStopReasonEndTurn:
					finishReason = openai.FinishReasonStop
				case anthropic.MessageDeltaEventDeltaStopReasonMaxTokens:
					finishReason = openai.FinishReasonLength
				case anthropic.MessageDeltaEventDeltaStopReasonStopSequence:
					// I don't think this has a parallel in OpenAI.
					finishReason = openai.FinishReasonStop
				}
				c <- ChatCompletionStreamResponse{
					ChatCompletionStreamResponse: openai.ChatCompletionStreamResponse{
						ID:     message.ID,
						Object: "chat.completion.chunk",
						Choices: []openai.ChatCompletionStreamChoice{
							{
								Index:        0,
								Delta:        openai.ChatCompletionStreamChoiceDelta{},
								FinishReason: finishReason,
							},
						},
					},
				}
			case anthropic.MessageStopEvent:
				// This is empty.
			}
		}

		if err := stream.Err(); err != nil {
			if err != io.EOF {
				log.DefaultLogger.Error("anthropic stream error", "err", err)
				c <- ChatCompletionStreamResponse{Error: err}
			}
		}
	}()

	return c, nil
}

func (p *anthropicProvider) ListAssistants(ctx context.Context, limit *int, order *string, after *string, before *string) (openai.AssistantsList, error) {
	return openai.AssistantsList{}, fmt.Errorf("anthropic does not support assistants")
}
