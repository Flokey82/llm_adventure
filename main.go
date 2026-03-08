package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Flokey82/llm_adventure/adventure"
	"github.com/sashabaranov/go-openai"
)

const (
	ModeTUILLM      = "tui-llm"
	ModeInteractive = "interactive"
)

func main() {
	// Ollama for testing with local models
	baseURL := "http://localhost:11434/v1"
	model := "granite4"

	// Configure for Lemonade Server
	config := openai.DefaultConfig("sk-no-key-required")
	config.BaseURL = baseURL // Your Lemonade Server
	client := openai.NewClientWithConfig(config)

	// Allow an optional deterministic seed via env var ONEIROS_SEED or CLI arg later.
	game := adventure.NewGame()

	// Inject the AI describer so rooms can be generated on first visit.
	game.AI_GenerateDescription = func(prompt string) string {
		fmt.Printf("\n[System] Generating new room description via LLM...\n")
		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: "You are a dark fantasy writer. Generate a static room description based on the provided tags. Keep it concise and atmospheric."},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		})
		if err != nil || len(resp.Choices) == 0 {
			return "A room obscured by swirling mists (Generation Error)."
		}

		// Inject a basic character chat handler used by Game.TalkTo(). This spins up
		// a focused request where the system prompt is the NPC's persona.
		game.AI_CharacterChat = func(systemPrompt, userMessage string) string {
			resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
				Model: model,
				Messages: []openai.ChatCompletionMessage{
					{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
					{Role: openai.ChatMessageRoleUser, Content: userMessage},
				},
			})
			if err != nil || len(resp.Choices) == 0 {
				return "(no response)"
			}
			return resp.Choices[0].Message.Content
		}
		return resp.Choices[0].Message.Content
	}

	// Determine the mode based on the first CLI argument, default to ModeTUILLM
	mode := ModeTUILLM
	if len(os.Args) > 1 {
		mode = os.Args[1]
	}

	switch mode {
	case ModeTUILLM:
		if err := game.RunTUIWithLLM(client, model); err != nil {
			fmt.Printf("tui-llm error: %v\n", err)
		}
	case ModeInteractive:
		if err := game.RunInteractive(client, model); err != nil {
			fmt.Printf("interactive mode error: %v\n", err)
		}
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
	}
}
