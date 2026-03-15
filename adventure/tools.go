package adventure

import (
	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

// Tools returns a list of tool definitions that the LLM can call to interact with the game environment.
// It selectively filters tools based on the current game state (e.g., presence of items or NPCs).
func Tools(g *Game) []openai.Tool {
	allTools := map[string]openai.Tool{
		"move": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "move",
				Description: "Move the player in a direction. Only works if an open door exists or there is a known open path.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"direction": {Type: jsonschema.String, Enum: []string{"north", "south", "east", "west", "up", "down"}},
					},
					Required: []string{"direction"},
				},
			},
		},
		"discover_room": {
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
		"spawn_item": {
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
		"open_door": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "open_door",
				Description: "Open a door in a specific direction.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"direction": {Type: jsonschema.String, Enum: []string{"north", "south", "east", "west", "up", "down"}},
					},
					Required: []string{"direction"},
				},
			},
		},
		"take_item": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "take_item",
				Description: "Pick up an item from the room.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"item_name": {Type: jsonschema.String},
					},
					Required: []string{"item_name"},
				},
			},
		},
		"use_item": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "use_item",
				Description: "Use an item on a target (use item_name on target_name).",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"item_name":   {Type: jsonschema.String},
						"target_name": {Type: jsonschema.String},
					},
					Required: []string{"item_name", "target_name"},
				},
			},
		},
		"talk_to": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "talk_to",
				Description: "Talk to an NPC with a message. NPCs have personas and may respond.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"npc_name": {Type: jsonschema.String},
						"message":  {Type: jsonschema.String},
					},
					Required: []string{"npc_name", "message"},
				},
			},
		},
		"search": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "search",
				Description: "Search the current room for hidden secrets, passages, or concealed information.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
				},
			},
		},
		"look": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "look",
				Description: "Get the current visual description of the room and items.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
				},
			},
		},
		"spawn_npc": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "spawn_npc",
				Description: "Dynamically generate and place a new NPC in the room.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"npc_id":      {Type: jsonschema.String, Description: "Unique ID (snake_case)."},
						"name":        {Type: jsonschema.String, Description: "Display name."},
						"description": {Type: jsonschema.String, Description: "Visual description."},
						"persona":     {Type: jsonschema.String, Description: "Dialogue persona/system prompt."},
						"disposition": {Type: jsonschema.Integer, Description: "0-100 (0=hostile, 100=friendly)."},
						"hp":          {Type: jsonschema.Integer, Description: "Starting/Max Hitpoints."},
						"history":     {Type: jsonschema.String, Description: "World history or background lore."},
					},
					Required: []string{"npc_id", "name", "description", "persona", "disposition", "hp"},
				},
			},
		},
		"update_npc": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "update_npc",
				Description: "Update an NPC's description, memory, or disposition after an event.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"npc_id":      {Type: jsonschema.String},
						"description": {Type: jsonschema.String},
						"memory":      {Type: jsonschema.String, Description: "Condensed memory of what just happened or what was told."},
						"disposition": {Type: jsonschema.Integer, Description: "New disposition 0-100."},
						"history":     {Type: jsonschema.String, Description: "New or updated history/lore."},
					},
					Required: []string{"npc_id"},
				},
			},
		},
		"attack": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "attack",
				Description: "Attack an NPC or object. Deals damage based on reasoning.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"target_name": {Type: jsonschema.String},
						"reasoning":   {Type: jsonschema.String, Description: "Why the player is attacking and how."},
					},
					Required: []string{"target_name"},
				},
			},
		},
		"resurrect": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "resurrect",
				Description: "Attempt to raise an NPC from the dead or restore a corpse.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"npc_id":    {Type: jsonschema.String},
						"reasoning": {Type: jsonschema.String, Description: "How the resurrection is performed."},
					},
					Required: []string{"npc_id"},
				},
			},
		},
		"update_player_notes": {
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "update_player_notes",
				Description: "Update the persistent notes about the player's state (e.g. 'covered in poop'). Use this when the player's status changes in a lasting way.",
				Parameters: jsonschema.Definition{
					Type: jsonschema.Object,
					Properties: map[string]jsonschema.Definition{
						"notes": {
							Type:  jsonschema.Array,
							Items: &jsonschema.Definition{Type: jsonschema.String},
							Description: "The complete new list of player notes. Replaces the old list.",
						},
					},
					Required: []string{"notes"},
				},
			},
		},
	}

	// Filter logic
	var tools []openai.Tool
	
	// Always available
	always := []string{"move", "discover_room", "open_door", "search", "look", "spawn_item", "spawn_npc", "update_player_notes"}
	for _, t := range always {
		tools = append(tools, allTools[t])
	}

	room := g.Rooms[g.CurrentRoomID]
	
	// Only if room has items
	if len(room.Items) > 0 {
		tools = append(tools, allTools["take_item"])
	}

	// Only if player has items
	if len(g.Inventory) > 0 {
		tools = append(tools, allTools["use_item"])
	}

	// NPC related tools
	hasNPC := false
	hasDeadNPC := false
	for _, n := range g.NPCs {
		if n.Location == g.CurrentRoomID {
			if n.Dead {
				hasDeadNPC = true
			} else {
				hasNPC = true
			}
		}
	}

	if hasNPC {
		tools = append(tools, allTools["talk_to"])
		tools = append(tools, allTools["attack"])
		tools = append(tools, allTools["update_npc"])
	}

	if hasDeadNPC {
		tools = append(tools, allTools["resurrect"])
		// You can also attack a corpse (e.g. to destroy it)
		if !hasNPC {
			tools = append(tools, allTools["attack"])
			tools = append(tools, allTools["update_npc"])
		}
	}

	return tools
}
