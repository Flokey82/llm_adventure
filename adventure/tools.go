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
				Description: "Move the player in a direction. Only works if an open door exists or there is a known open path.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						// The direction in which the player wants to move. Must be one of the specified values.
						"direction": {Type: jsonschema.String, Enum: []string{"north", "south", "east", "west", "up", "down"}},
					},
					Required: []string{"direction"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "discover_room",
				Description: "Discover or create a new path/room in a direction. Use this when the player searches for hidden paths, forces their way through an obstacle, or investigates a direction with no visible door.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"direction": {Type: jsonschema.String, Enum: []string{"north", "south", "east", "west", "up", "down"}},
						"reasoning": {Type: jsonschema.String, Description: "Brief explanation of why a new path makes sense here."},
					},
					Required: []string{"direction"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "spawn_item",
				Description: "Dynamically generate and place a new item in the room. Use this when the player searches a container, defeats an enemy, or if finding an item makes strong narrative sense.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"item_name": {Type: jsonschema.String, Description: "The name of the item to spawn (snake_case)."},
						"reasoning": {Type: jsonschema.String, Description: "Brief narrative reason for finding the item."},
					},
					Required: []string{"item_name"},
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
						"direction": {Type: jsonschema.String, Enum: []string{"north", "south", "east", "west", "up", "down"}},
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
				Name:        "search",
				Description: "Search the current room for hidden secrets, passages, or concealed information.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
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
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "update_player_notes",
				Description: "Update the persistent notes about the player's state (e.g. 'covered in poop'). Use this when the player's status changes in a lasting way.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"notes": {
							Type: jsonschema.Array,
							Items: &jsonschema.Definition{Type: jsonschema.String},
							Description: "The complete new list of player notes. Replaces the old list.",
						},
					},
					Required: []string{"notes"},
				},
			},
		},
	}
}
