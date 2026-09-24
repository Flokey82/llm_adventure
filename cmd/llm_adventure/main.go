package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Flokey82/llm_adventure/adventure"
	"github.com/sashabaranov/go-openai"
)

const (
	ModeTUILLM      = "tui-llm"
	ModeInteractive = "interactive"
)

func main() {
	var baseURL, model, roomPrompt, scenarioFlag, loadSavePath string
	var selectScenario bool
	flag.StringVar(&baseURL, "base-url", "http://192.168.86.208:8000/api/v1", "Base URL for the OpenAI API")
	flag.StringVar(&model, "model", "granite-4.0-h-tiny-GGUF", "LLM model to use")
	flag.StringVar(&roomPrompt, "room-prompt", "", "Custom prompt for room generation (defaults to scenario atmosphere)")
	flag.StringVar(&scenarioFlag, "scenario", "", "Scenario to play: victorian, starship, cyberpunk, sunken, or any custom description")
	flag.BoolVar(&selectScenario, "select", true, "Interactively choose or describe a scenario on startup")
	flag.StringVar(&loadSavePath, "load", "", "Path to saved game JSON file to load on start")
	flag.Parse()

	// Configure for Lemonade Server
	config := openai.DefaultConfig("sk-no-key-required")
	config.BaseURL = baseURL // Your Lemonade Server
	client := openai.NewClientWithConfig(config)

	// Determine Scenario
	presets := adventure.PresetScenarios()
	var selectedScenario adventure.Scenario

	if selectScenario {
		fmt.Println("==================================================")
		fmt.Println("        Choose an Adventure Setting               ")
		fmt.Println("==================================================")
		fmt.Println("1) Victorian Manor   - Gothic mystery, decaying halls, dust & mist")
		fmt.Println("2) Derelict Starship - Deep-space isolation, emergency lights, xenomorphs")
		fmt.Println("3) Cyberpunk Sprawl  - Neon megacity alleyways, netrunners & chop shops")
		fmt.Println("4) Sunken Galleon    - Sunken pirate shipwreck on a glowing coral reef")
		fmt.Println("5) Custom Setting    - Describe your own world for the LLM to generate")
		fmt.Print("\nSelect scenario [1-5]: ")

		var choice string
		fmt.Scanln(&choice)
		choice = strings.TrimSpace(choice)

		switch choice {
		case "1":
			selectedScenario = presets["victorian"]
		case "2":
			selectedScenario = presets["starship"]
		case "3":
			selectedScenario = presets["cyberpunk"]
		case "4":
			selectedScenario = presets["sunken"]
		case "5":
			fmt.Print("Describe your setting (e.g. 'Antarctic lab during a fungal blizzard'): ")
			scanner := bufio.NewScanner(os.Stdin)
			if scanner.Scan() {
				desc := strings.TrimSpace(scanner.Text())
				if desc != "" {
					fmt.Println("Generating custom world with LLM...")
					genSc, err := adventure.GenerateScenario(context.Background(), client, model, desc)
					if err != nil {
						fmt.Printf("Generation failed (%v), falling back to Victorian Manor.\n", err)
						selectedScenario = presets["victorian"]
					} else {
						selectedScenario = *genSc
					}
				} else {
					selectedScenario = presets["victorian"]
				}
			}
		default:
			selectedScenario = presets["victorian"]
		}
	} else if scenarioFlag != "" {
		lower := strings.ToLower(strings.TrimSpace(scenarioFlag))
		if p, ok := presets[lower]; ok {
			selectedScenario = p
		} else {
			fmt.Printf("Generating custom scenario from premise: %q...\n", scenarioFlag)
			genSc, err := adventure.GenerateScenario(context.Background(), client, model, scenarioFlag)
			if err != nil {
				fmt.Printf("Warning: failed to generate scenario (%v), using Victorian Manor.\n", err)
				selectedScenario = presets["victorian"]
			} else {
				selectedScenario = *genSc
			}
		}
	} else {
		selectedScenario = presets["victorian"]
	}

	// Create game with scenario
	game := adventure.NewGameWithScenario(selectedScenario)

	// If load flag provided, restore saved state
	if loadSavePath != "" {
		if err := game.Load(loadSavePath); err != nil {
			fmt.Printf("Warning: failed to load save from %s: %v\n", loadSavePath, err)
		} else {
			fmt.Printf("Restored saved world from %s (Scenario: %s)\n", loadSavePath, game.ScenarioName)
		}
	}

	// Inject dynamic room generator matching the scenario
	game.AI_GenerateRoom = func(fromRoom *adventure.Room, direction string) *adventure.Room {
		systemPrompt := fmt.Sprintf(`You are designing a procedural text adventure game set in: %s.
Atmosphere: %s
Return a JSON object describing a new room. Structure:
{
  "name": "a short snake_case id like 'engine_deck'",
  "room_type": "a thematic room type appropriate for this setting",
  "base_prompt": "A comma separated list of atmospheric details",
  "items": ["list", "of", "items"], // optional
  "furniture": ["list", "of", "notable", "features", "or", "furniture"], // optional
  "secrets": ["a hidden detail", "a secret cache"] // optional, hidden info discoverable by searching
}`, game.ScenarioName, game.WorldPrompt)
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
			B: newRoom.ID, BDir: oppDir,
		}
		fromRoom.Doors[direction] = door
		newRoom.Doors[oppDir] = door

		return newRoom
	}

	// Inject the AI describer so rooms can be generated on first visit.
	game.AI_GenerateDescription = func(prompt string) string {
		sysPrompt := roomPrompt
		if sysPrompt == "" {
			sysPrompt = fmt.Sprintf("You are an atmospheric writer for an adventure setting: %s. Atmosphere: %s. Generate a static room description based on the provided tags. Keep it concise (2-3 sentences) and evocative.", game.ScenarioName, game.WorldPrompt)
		}
		resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
			Model: model,
			Messages: []openai.ChatCompletionMessage{
				{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
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
