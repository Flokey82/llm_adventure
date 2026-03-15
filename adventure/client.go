package adventure

import "context"

import "github.com/sashabaranov/go-openai"

// LLMClient abstracts the requirements for communicating with an LLM.
// This interface allows for easier testing by injecting mock clients.
type LLMClient interface {
	CreateChatCompletion(ctx context.Context, request openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
}
