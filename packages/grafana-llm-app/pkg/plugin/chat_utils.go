package plugin

import (
	"strings"

	"github.com/sashabaranov/go-openai"
)

// ForceUserMessage ensures that there is at least one user message in the chat completion request
// by converting the last message to a user message if no user messages are found.
func ForceUserMessage(req *openai.ChatCompletionRequest) {
	if len(req.Messages) == 0 {
		return
	}

	hasUserMessage := false
	for _, message := range req.Messages {
		if message.Role == "user" {
			hasUserMessage = true
			break
		}
	}

	if !hasUserMessage {
		req.Messages[len(req.Messages)-1].Role = "user"
	}
}

// NormalizeTokenLimits ensures that the request has valid token limits for the target model.
// For models requiring MaxCompletionTokens (e.g. o1, o3, gpt-5), MaxTokens is migrated to
// MaxCompletionTokens if not already set, and MaxTokens is cleared to avoid API rejection.
func NormalizeTokenLimits(req *openai.ChatCompletionRequest) {
	if RequiresMaxCompletionTokens(req.Model) {
		if req.MaxTokens > 0 && req.MaxCompletionTokens == 0 {
			req.MaxCompletionTokens = req.MaxTokens
		}
		req.MaxTokens = 0
	}
}

// RequiresMaxCompletionTokens checks if a model name is known to require MaxCompletionTokens
// instead of MaxTokens (e.g. OpenAI o1/o3 reasoning models, gpt-5, etc.).
func RequiresMaxCompletionTokens(model string) bool {
	lower := strings.ToLower(model)
	return strings.HasPrefix(lower, "o1") ||
		strings.HasPrefix(lower, "o3") ||
		strings.HasPrefix(lower, "o4") ||
		strings.HasPrefix(lower, "gpt-5")
}

// IsMaxTokensUnsupportedError detects if an API error specifically indicates that
// MaxTokens is unsupported and MaxCompletionTokens must be used instead.
// Matches both OpenAI and Azure Foundry/OpenAI error formats without false positives.
func IsMaxTokensUnsupportedError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	hasMaxTokens := strings.Contains(lower, "max_tokens") || strings.Contains(lower, "maxtokens")
	hasMaxCompletionTokens := strings.Contains(lower, "max_completion_tokens") || strings.Contains(lower, "maxcompletiontokens")
	return hasMaxTokens && hasMaxCompletionTokens
}
