// executor.go
// This file handles the execution of tool calls from LLM messages. It processes
// the tool calls, executes the corresponding game actions, and returns the results.

package adventure

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
)

var (
	gemmaToolCallRegex = regexp.MustCompile(`(?s)<\|tool_call\|?>call:(\w+)\s*\{(.*?)\}\s*<tool_call\|?>`)
	unquotedKeyRegex   = regexp.MustCompile(`([{\s,])([a-zA-Z_]\w*)\s*:`)
)

func parseFlexibleJSON(raw string) map[string]interface{} {
	// 1. Replace Gemma quotation tokens
	cleaned := strings.ReplaceAll(raw, `<|"|>`, `"`)
	cleaned = strings.TrimSpace(cleaned)
	if !strings.HasPrefix(cleaned, "{") {
		cleaned = "{" + cleaned
	}
	if !strings.HasSuffix(cleaned, "}") {
		cleaned = cleaned + "}"
	}

	// 2. Try direct unmarshal first
	var res map[string]interface{}
	if err := json.Unmarshal([]byte(cleaned), &res); err == nil {
		return res
	}

	// 3. Ensure unquoted keys are quoted: {key:"val"} -> {"key":"val"}
	quotedKeys := unquotedKeyRegex.ReplaceAllString(cleaned, `$1"$2":`)
	if err := json.Unmarshal([]byte(quotedKeys), &res); err == nil {
		return res
	}

	return nil
}

// ExtractEmbeddedToolCalls finds and extracts any tool calls rendered as raw text tokens in content (e.g. by Gemma 4 / Hermes).
func ExtractEmbeddedToolCalls(content string) (string, []openai.ToolCall) {
	matches := gemmaToolCallRegex.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	var toolCalls []openai.ToolCall
	for i, idx := range matches {
		fnName := content[idx[2]:idx[3]]
		argsRaw := content[idx[4]:idx[5]]
		parsedArgs := parseFlexibleJSON(argsRaw)
		if parsedArgs == nil {
			parsedArgs = make(map[string]interface{})
		}
		argsBytes, _ := json.Marshal(parsedArgs)

		toolCalls = append(toolCalls, openai.ToolCall{
			ID:   fmt.Sprintf("embedded_call_%d", i),
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      fnName,
				Arguments: string(argsBytes),
			},
		})
	}

	cleanText := gemmaToolCallRegex.ReplaceAllString(content, "")
	cleanText = strings.TrimSpace(cleanText)
	return cleanText, toolCalls
}

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

	toolCalls := msg.ToolCalls
	if len(toolCalls) == 0 && msg.Content != "" {
		_, embedded := ExtractEmbeddedToolCalls(msg.Content)
		toolCalls = embedded
	}

	// If there are no tool calls, return empty results
	if len(toolCalls) == 0 {
		return outMsgs, logs
	}

	// Iterate over each tool call and execute the corresponding action
	for _, toolCall := range toolCalls {
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
		case "spawn_npc":
			name, _ := args["name"].(string)
			id, _ := args["npc_id"].(string)
			if id == "" && name != "" {
				id = strings.ToLower(strings.ReplaceAll(name, " ", "_"))
			}
			if id == "" {
				id = fmt.Sprintf("npc_%d", len(g.NPCs)+1)
			}
			desc, _ := args["description"].(string)
			persona, _ := args["persona"].(string)
			if persona == "" {
				persona = fmt.Sprintf("You are %s. %s", name, desc)
			}
			disposition := 50
			if dispositionRaw, ok := args["disposition"].(float64); ok {
				disposition = int(dispositionRaw)
			} else if dStr, ok := args["disposition"].(string); ok {
				dStr = strings.ToLower(dStr)
				switch {
				case strings.Contains(dStr, "hostile"), strings.Contains(dStr, "aggressive"), strings.Contains(dStr, "enemy"):
					disposition = 10
				case strings.Contains(dStr, "annoyed"), strings.Contains(dStr, "angry"), strings.Contains(dStr, "offended"):
					disposition = 30
				case strings.Contains(dStr, "friendly"), strings.Contains(dStr, "ally"), strings.Contains(dStr, "helpful"):
					disposition = 80
				default:
					disposition = 50
				}
			}
			hp := 10
			if hpRaw, ok := args["hp"].(float64); ok && hpRaw > 0 {
				hp = int(hpRaw)
			}
			history, _ := args["history"].(string)
			toolOutput = g.SpawnNPC(id, name, desc, persona, disposition, hp, history)
		case "update_npc":
			id, _ := args["npc_id"].(string)
			desc, _ := args["description"].(string)
			memory, _ := args["memory"].(string)
			dispositionRaw, ok := args["disposition"].(float64)
			disposition := -1
			if ok {
				disposition = int(dispositionRaw)
			}
			history, _ := args["history"].(string)
			toolOutput = g.UpdateNPC(id, desc, memory, disposition, history)
		case "attack":
			target, _ := args["target_name"].(string)
			reasoning, _ := args["reasoning"].(string)
			toolOutput = g.Attack(target, reasoning)
		case "resurrect":
			id, _ := args["npc_id"].(string)
			reasoning, _ := args["reasoning"].(string)
			toolOutput = g.Resurrect(id, reasoning)
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
			} else if noteStr, ok := args["notes"].(string); ok && noteStr != "" {
				g.PlayerNotes = append(g.PlayerNotes, noteStr)
				toolOutput = fmt.Sprintf("Player note added: %s", noteStr)
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
