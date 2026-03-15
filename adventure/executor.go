// executor.go
// This file handles the execution of tool calls from LLM messages. It processes
// the tool calls, executes the corresponding game actions, and returns the results.

package adventure

import (
	"encoding/json"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

// ExecuteToolCallsFromMessage processes tool calls found in the provided LLM message.
// Parameters:
//   - msg: An openai.ChatCompletionMessage containing tool calls to execute. Each tool call
//     includes a function name and arguments in JSON format.
//
// Returns:
// - outMsgs: A slice of ChatCompletionMessages with the results of the tool executions.
// - logs: A slice of human-readable log entries describing the executed actions.
func (g *Game) ExecuteToolCallsFromMessage(msg openai.ChatCompletionMessage) ([]openai.ChatCompletionMessage, []string) {
	var outMsgs []openai.ChatCompletionMessage
	var logs []string

	// If there are no tool calls, return empty results
	if len(msg.ToolCalls) == 0 {
		return outMsgs, logs
	}

	// Iterate over each tool call and execute the corresponding action
	for _, toolCall := range msg.ToolCalls {
		var args map[string]interface{}
		json.Unmarshal([]byte(toolCall.Function.Arguments), &args) // Parse arguments from JSON

		logs = append(logs, fmt.Sprintf("Executing tool: %s %v", toolCall.Function.Name, args))

		var toolOutput string
		switch toolCall.Function.Name {
		case "discover_room":
			if dir, ok := args["direction"].(string); ok {
				toolOutput = g.DiscoverRoom(dir)
			} else {
				toolOutput = "invalid arguments to discover_room"
			}
		case "spawn_item":
			if name, ok := args["item_name"].(string); ok {
				reasoning, _ := args["reasoning"].(string) 
				toolOutput = g.SpawnItem(name, reasoning)
			} else {
				toolOutput = "invalid arguments to spawn_item"
			}
		case "move":
			// Move the player in a specified direction
			if dir, ok := args["direction"].(string); ok {
				toolOutput = g.Move(dir)
			} else {
				toolOutput = "invalid arguments to move"
			}
		case "open_door":
			// Open a door in a specified direction
			if dir, ok := args["direction"].(string); ok {
				toolOutput = g.OpenDoor(dir)
			} else {
				toolOutput = "invalid arguments to open_door"
			}
		case "take_item":
			// Take an item from the current room
			if name, ok := args["item_name"].(string); ok {
				toolOutput = g.TakeItem(name)
			} else {
				toolOutput = "invalid arguments to take_item"
			}
		case "use_item":
			// Use an item on a target
			if item, ok := args["item_name"].(string); ok {
				if target, ok2 := args["target_name"].(string); ok2 {
					toolOutput = g.UseItem(item, target)
				} else {
					toolOutput = "invalid arguments to use_item"
				}
			} else {
				toolOutput = "invalid arguments to use_item"
			}
		case "talk_to":
			// Talk to an NPC
			if npc, ok := args["npc_name"].(string); ok {
				if msg, ok2 := args["message"].(string); ok2 {
					toolOutput = g.TalkTo(npc, msg)
				} else {
					toolOutput = "invalid arguments to talk_to"
				}
			} else {
				toolOutput = "invalid arguments to talk_to"
			}
		case "search":
			// Search the current room
			toolOutput = g.Search()
		case "look":
			// Look around the current room
			toolOutput = g.Look()
		case "add_room_detail":
			// Add a permanent narrative detail to the current room
			if detail, ok := args["detail"].(string); ok {
				toolOutput = g.AddRoomDetail(detail)
			} else {
				toolOutput = "invalid arguments to add_room_detail"
			}
		case "update_player_notes":
			// Update the persistent notes about the player
			if notesRaw, ok := args["notes"].([]interface{}); ok {
				var notes []string
				for _, n := range notesRaw {
					if ns, ok := n.(string); ok {
						notes = append(notes, ns)
					}
				}
				toolOutput = g.UpdatePlayerNotes(notes)
			} else {
				toolOutput = "invalid arguments to update_player_notes"
			}
		default:
			// Handle unknown tools
			toolOutput = "unknown tool"
		}

		// Append the tool output as a ChatCompletionMessage
		outMsgs = append(outMsgs, openai.ChatCompletionMessage{
			Role:       openai.ChatMessageRoleTool,
			Content:    toolOutput,
			ToolCallID: toolCall.ID,
		})
		logs = append(logs, toolOutput)
	}

	return outMsgs, logs
}
