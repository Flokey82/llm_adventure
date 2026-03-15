package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Flokey82/llm_adventure/adventure"
	"github.com/sashabaranov/go-openai"
)

func main() {
	config := openai.DefaultConfig("sk-no-key-required")
	config.BaseURL = "http://localhost:11434/v1"
	client := openai.NewClientWithConfig(config)
	model := "granite4"

	fmt.Println("Testing Room Generation")

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
			fmt.Printf("Error generation: %v\n", err)
			return nil
		}
		
		fmt.Printf("Raw LLM output:\n%s\n", resp.Choices[0].Message.Content)

		var roomSpec struct {
			Name       string   `json:"name"`
			RoomType   string   `json:"room_type"`
			BasePrompt string   `json:"base_prompt"`
			Items      []string `json:"items"`
			Furniture  []string `json:"furniture"`
			Secrets    []string `json:"secrets"`
		}
		if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &roomSpec); err != nil {
			fmt.Printf("Error unmarshalling: %v\n", err)
			return nil
		}

		newID := roomSpec.Name
		
		newRoom := &adventure.Room{
			ID:         newID,
			RoomType:   roomSpec.RoomType,
			BasePrompt: roomSpec.BasePrompt,
			Items:      roomSpec.Items,
			Furniture:  roomSpec.Furniture,
			Secrets:    roomSpec.Secrets,
			Doors:      make(map[string]*adventure.Door),
		}
		return newRoom
	}

	game.AI_GenerateDescription = func(prompt string) string {
		fmt.Printf("Description Prompt: %s\n", prompt)
		return "Mock description from AI"
	}
	
	room := game.Rooms[game.CurrentRoomID]
	newRoom := game.AI_GenerateRoom(room, "north")
	if newRoom != nil {
		fmt.Printf("Parsed Room object:\n%+v\n", newRoom)
		
		// Test Look
		game.Rooms[newRoom.ID] = newRoom
		game.CurrentRoomID = newRoom.ID
		fmt.Printf("\nLook Result:\n%s\n", game.Look())
		
		// Test Search
		fmt.Printf("\nSearch Result:\n%s\n", game.Search())
	}
}
