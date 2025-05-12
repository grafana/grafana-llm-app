package plugin

import (
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
