package adventure

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// Tools returns a list of tool definitions that the LLM can call to interact with the game environment.
// Each tool represents a specific action the player can perform, such as moving, interacting with items, or talking to NPCs.
func Tools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "move",
				Description: "Move the player in a direction. Only works if door is open.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						// The direction in which the player wants to move. Must be one of the specified values.
						"direction": {Type: jsonschema.String, Enum: []string{"north", "south", "east", "west"}},
					},
					Required: []string{"direction"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "open_door",
				Description: "Open a door in a specific direction.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						// The direction of the door to open. Must be one of the specified values.
						"direction": {Type: jsonschema.String, Enum: []string{"north", "south", "east", "west"}},
					},
					Required: []string{"direction"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "take_item",
				Description: "Pick up an item from the room.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						// The name of the item to pick up.
						"item_name": {Type: jsonschema.String},
					},
					Required: []string{"item_name"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "use_item",
				Description: "Use an item on a target (use item_name on target_name).",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						// The name of the item to use.
						"item_name": {Type: jsonschema.String},
						// The name of the target on which the item is used.
						"target_name": {Type: jsonschema.String},
					},
					Required: []string{"item_name", "target_name"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "talk_to",
				Description: "Talk to an NPC with a message. NPCs have personas and may respond.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						// The name of the NPC to talk to.
						"npc_name": {Type: jsonschema.String},
						// The message to send to the NPC.
						"message": {Type: jsonschema.String},
					},
					Required: []string{"npc_name", "message"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "look",
				Description: "Get the current visual description of the room and items.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
				},
			},
		},
	}
}
