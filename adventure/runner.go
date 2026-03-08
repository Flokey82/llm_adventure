package adventure

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// RunInteractive runs an interactive loop using the provided OpenAI client and model.
// It manages messages, tool dispatch, and user input. The function returns when the
// user types `quit` or `exit`, or if an unrecoverable client error occurs.
func (g *Game) RunInteractive(client *openai.Client, model string) error {
	ctx := context.Background()
	reader := bufio.NewReader(os.Stdin)

	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: `You are the narrator of a text adventure game. Use the 'look' tool immediately when the game starts or room changes. Rely on tool outputs for state. Do not invent items, exits, or doors. Describe scene atmospherically. Call tools rather than saying you did something.`,
		},
		{Role: openai.ChatMessageRoleUser, Content: "Start the game. Look around."},
	}

	for {
		// Simple pruning to avoid unbounded context growth: keep system prompt and
		// last N messages (here N=10).
		if len(messages) > 12 {
			sys := messages[0]
			recent := messages[len(messages)-10:]
			messages = []openai.ChatCompletionMessage{sys}
			messages = append(messages, recent...)
		}

		req := openai.ChatCompletionRequest{Model: model, Messages: messages, Tools: Tools()}
		resp, err := client.CreateChatCompletion(ctx, req)
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			continue
		}

		msg := resp.Choices[0].Message
		messages = append(messages, msg)

		if len(msg.ToolCalls) > 0 {
			toolMsgs, logs := g.ExecuteToolCallsFromMessage(msg)
			messages = append(messages, toolMsgs...)
			for _, l := range logs {
				fmt.Println("[tool]", l)
			}

			// Advance world state (ticks) after tool execution and print events
			events := g.Tick()
			for _, e := range events {
				fmt.Println("[tick]", e)
			}

			// Re-query the model for narration after tool execution
			resp2, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{Model: model, Messages: messages})
			if err != nil {
				return err
			}
			if len(resp2.Choices) > 0 {
				fmt.Println(resp2.Choices[0].Message.Content)
				messages = append(messages, resp2.Choices[0].Message)
			}
			continue
		}

		// Narration output
		fmt.Printf("\nNarrator: %s\n", msg.Content)

		// read user input
		fmt.Print("\n> ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)

		if userInput == "quit" || userInput == "exit" {
			return nil
		}

		// Try a quick local command match to bypass the LLM for clear commands
		handled, out, ambiguous := g.ExecuteQuickCommand(userInput)
		if handled {
			if len(ambiguous) > 0 {
				// Ask user to disambiguate
				fmt.Println(out)
				for i, opt := range ambiguous {
					fmt.Printf("%d) %s\n", i+1, opt)
				}
				// prompt selection
				for {
					fmt.Print("Choose number: ")
					choiceRaw, _ := reader.ReadString('\n')
					choiceRaw = strings.TrimSpace(choiceRaw)
					if choiceRaw == "" {
						fmt.Println("Cancelled.")
						break
					}
					// try parse as number
					idx := -1
					fmt.Sscanf(choiceRaw, "%d", &idx)
					if idx >= 1 && idx <= len(ambiguous) {
						sel := ambiguous[idx-1]
						res := g.TakeItem(sel)
						// record user + tool output
						messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userInput})
						messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleTool, Content: res})
						fmt.Println(res)
						break
					}
					// also allow matching by name
					for _, opt := range ambiguous {
						if strings.EqualFold(opt, choiceRaw) {
							res := g.TakeItem(opt)
							messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userInput})
							messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleTool, Content: res})
							fmt.Println(res)
							goto handledDone
						}
					}
					fmt.Println("Invalid choice, try again or press Enter to cancel.")
				}
			handledDone:
				continue
			}
			// not ambiguous: record the user message and the tool output so the
			// model sees the state, then request narration from the model but
			// do NOT provide tool definitions (we handled the action locally).
			messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userInput})
			messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleTool, Content: out})
			fmt.Println(out)

			// Advance world state for this local action and print tick events
			events := g.Tick()
			for _, e := range events {
				fmt.Println("[tick]", e)
			}

			// ask for narration without tools
			respN, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{Model: model, Messages: messages})
			if err != nil {
				return err
			}
			if len(respN.Choices) > 0 {
				fmt.Println(respN.Choices[0].Message.Content)
				messages = append(messages, respN.Choices[0].Message)
			}
			continue
		}

		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userInput})
	}
}
