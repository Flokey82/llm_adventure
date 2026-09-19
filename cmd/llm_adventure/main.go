package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/Flokey82/llm_adventure/adventure"
	animusLLM "github.com/Flokey82/animus/pkg/llm"
	"github.com/sashabaranov/go-openai"
)

const (
	ModeTUILLM      = "tui-llm"
	ModeInteractive = "interactive"
)

func main() {
	var baseURL, model, toolModel, roomPrompt, dmConfigPath, loadSavePath string
	useAnimusDM := flag.Bool("dm", true, "Use autonomous Animus Dungeon Master agent")
	dmRandom := flag.Bool("dm-random", false, "Randomize Dungeon Master persona and narrative style")
	flag.StringVar(&dmConfigPath, "dm-config", "", "Path to custom Dungeon Master JSON config to load")
	flag.StringVar(&loadSavePath, "load", "", "Path to saved world state JSON file to restore on start")
	flag.StringVar(&baseURL, "base-url", "http://192.168.86.208:8000/api/v1", "Base URL for the OpenAI API")
	flag.StringVar(&model, "model", "Gemma-4-26B-A4B-it-MTP-GGUF", "LLM model to use for narration and dialogue")
	flag.StringVar(&toolModel, "tool-model", "granite-4.0-h-tiny-GGUF", "Fast LLM model to use for reflex tool calling")
	flag.StringVar(&roomPrompt, "room-prompt", "You are a dark fantasy writer. Generate a static room description based on the provided tags. Keep it concise and atmospheric.", "System prompt for room generation")
	flag.Parse()

	// Configure for Lemonade Server
	config := openai.DefaultConfig("sk-no-key-required")
	config.BaseURL = baseURL // Your Lemonade Server
	client := openai.NewClientWithConfig(config)

	// Allow an optional deterministic seed via env var ONEIROS_SEED or CLI arg later.
	game := adventure.NewGame()

	// Inject the dynamic room generator used when exploring without a door.
	game.AI_GenerateRoom = func(fromRoom *adventure.Room, direction string) *adventure.Room {
		systemPrompt := `You are a dark fantasy writer designing a procedural text adventure game.
Return a JSON object describing a new room. Structure:
{
  "name": "a short snake_case id like 'dark_cave'",
  "room_type": "a thematic room type like 'torture_chamber', 'overgrown_greenhouse', etc.",
  "base_prompt": "A comma separated list of atmospheric details",
  "items": ["list", "of", "items"], // optional
  "furniture": ["list", "of", "notable", "features", "or", "furniture"], // optional
  "secrets": ["a hidden detail", "a secret cache"] // optional, hidden info discoverable by searching
}`
		userMsg := fmt.Sprintf("The player is in '%s' and moved '%s' into the unknown. Generate what they find.", fromRoom.ID, direction)

		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userMsg},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		})
		if err != nil || len(resp.Choices) == 0 {
			return nil
		}

		var roomSpec struct {
			Name       string   `json:"name"`
			RoomType   string   `json:"room_type"`
			BasePrompt string   `json:"base_prompt"`
			Items      []string `json:"items"`
			Furniture  []string `json:"furniture"`
			Secrets    []string `json:"secrets"`
		}
		if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &roomSpec); err != nil {
			return nil
		}

		// Ensure unique ID
		newID := roomSpec.Name
		for i := 1; ; i++ {
			if _, exists := game.Rooms[newID]; !exists {
				break
			}
			newID = fmt.Sprintf("%s_%d", roomSpec.Name, i)
		}

		// Calculate rough coordinates offset
		newX, newY := fromRoom.X, fromRoom.Y
		switch direction {
		case "north":
			newY--
		case "south":
			newY++
		case "east":
			newX++
		case "west":
			newX--
		}

		newRoom := &adventure.Room{
			ID:         newID,
			RoomType:   roomSpec.RoomType,
			BasePrompt: roomSpec.BasePrompt,
			Items:      roomSpec.Items,
			Furniture:  roomSpec.Furniture,
			Secrets:    roomSpec.Secrets,
			Doors:      make(map[string]*adventure.Door),
			X:          newX,
			Y:          newY,
		}

		// Create connecting door
		oppDir := "south"
		switch direction {
		case "south":
			oppDir = "north"
		case "west":
			oppDir = "east"
		case "east":
			oppDir = "west"
		}

		door := &adventure.Door{
			Open: true,
			A:    fromRoom.ID, ADir: direction,
			B:    newRoom.ID, BDir: oppDir,
		}
		fromRoom.Doors[direction] = door
		newRoom.Doors[oppDir] = door

		return newRoom
	}

	// Inject the AI describer so rooms can be generated on first visit.
	game.AI_GenerateDescription = func(prompt string) string {
		fmt.Printf("\n[System] Generating new room description via LLM...\n")
		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: roomPrompt},
				{Role: openai.ChatMessageRoleUser, Content: prompt},
			},
		})
		if err != nil || len(resp.Choices) == 0 {
			return "A room obscured by swirling mists (Generation Error)."
		}
		return resp.Choices[0].Message.Content
	}

	// Inject a structured character chat handler used by Game.TalkTo().
	// It spins up a focused request asking for a JSON response containing
	// the dialogue and a disposition change.
	game.AI_CharacterChat = func(npc *adventure.NPC, userMessage string) string {
		systemPrompt := fmt.Sprintf(`%s
You must explicitly respond in valid JSON format only, matching this structure:
{
  "message": "The dialogue response",
  "disposition_change": 0 // integer from -20 to 20 representing how this interaction changes your mood
}`, npc.Persona)

		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
				{Role: openai.ChatMessageRoleUser, Content: userMessage},
			},
			ResponseFormat: &openai.ChatCompletionResponseFormat{
				Type: openai.ChatCompletionResponseFormatTypeJSONObject,
			},
		})
		if err != nil || len(resp.Choices) == 0 {
			return "(no response)"
		}

		replyContent := resp.Choices[0].Message.Content
		var chatResp struct {
			Message           string `json:"message"`
			DispositionChange int    `json:"disposition_change"`
		}

		if err := json.Unmarshal([]byte(replyContent), &chatResp); err != nil {
			// Fallback: if the model failed to return strict JSON, just return the content.
			return replyContent
		}

		// Apply the disposition change (clamped between 0 and 100)
		npc.Disposition += chatResp.DispositionChange
		if npc.Disposition < 0 {
			npc.Disposition = 0
		} else if npc.Disposition > 100 {
			npc.Disposition = 100
		}

		return chatResp.Message
	}

	// Determine the mode based on positional arguments, default to ModeTUILLM
	mode := ModeTUILLM
	if flag.NArg() > 0 {
		mode = flag.Arg(0)
	}

	switch mode {
	case ModeTUILLM:
		if *useAnimusDM {
			var dmCfg adventure.DMConfig
			if dmConfigPath != "" {
				loadedCfg, err := adventure.LoadDMConfig(dmConfigPath)
				if err != nil {
					fmt.Printf("Warning: failed to load DM config from %s: %v\n", dmConfigPath, err)
					if *dmRandom {
						dmCfg = adventure.GenerateRandomDMConfig(time.Now().UnixNano())
					} else {
						dmCfg = adventure.DefaultDMConfig()
					}
				} else {
					dmCfg = *loadedCfg
				}
			} else if *dmRandom {
				dmCfg = adventure.GenerateRandomDMConfig(time.Now().UnixNano())
			} else {
				dmCfg = adventure.DefaultDMConfig()
			}

			dm := adventure.NewDungeonMaster(game, animusLLM.Config{
				BaseURL:   baseURL,
				Model:     model,
				ToolModel: toolModel,
			}, dmCfg)

			if loadSavePath != "" {
				if err := adventure.LoadWorld(loadSavePath, game, dm); err != nil {
					fmt.Printf("Warning: failed to load world save from %s: %v\n", loadSavePath, err)
				} else {
					fmt.Printf("Restored world save from %s\n", loadSavePath)
				}
			}

			if err := game.RunTUIWithDM(dm); err != nil {
				fmt.Printf("tui-dm error: %v\n", err)
			}
		} else {
			if err := game.RunTUIWithLLM(client, model); err != nil {
				fmt.Printf("tui-llm error: %v\n", err)
			}
		}
	case ModeInteractive:
		if err := game.RunInteractive(client, model); err != nil {
			fmt.Printf("interactive mode error: %v\n", err)
		}
	default:
		fmt.Printf("Unknown mode: %s\n", mode)
	}
}
