package adventure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Flokey82/animus/pkg/engine"
	"github.com/Flokey82/animus/pkg/llm"
	"github.com/Flokey82/animus/pkg/tools"
	"github.com/sashabaranov/go-openai"
)

// DungeonMaster is an autonomous Animus cognitive agent that runs the text adventure world.
type DungeonMaster struct {
	Game   *Game
	Agent  *engine.Agent
	Config DMConfig
}

// NewDungeonMaster creates a Dungeon Master agent wired to the game instance.
func NewDungeonMaster(g *Game, llmCfg llm.Config, customCfg ...DMConfig) *DungeonMaster {
	dmCfg := DefaultDMConfig()
	if len(customCfg) > 0 && customCfg[0].Name != "" {
		dmCfg = customCfg[0]
	}

	identity := fmt.Sprintf(`You are %s, %s.
%s
Your role:
1. Impartially interpret player intents and run world actions (move, discover_room, spawn_item, open_door, attack, etc.).
2. Deliver vivid, atmospheric, present-tense, second-person narration ("You step into...").
3. Keep narrations concise (2-4 sentences). Do not break character or mention game engine internals.`, dmCfg.Name, dmCfg.Title, dmCfg.NarrationStyle)

	dmAgent := engine.NewAgent(engine.AgentConfig{
		Name:         dmCfg.Name,
		BaseIdentity: identity,
		Personality:  dmCfg.Personality,
		LLMConfig:    llmCfg,
	})

	dm := &DungeonMaster{
		Game:   g,
		Agent:  dmAgent,
		Config: dmCfg,
	}

	// Seed DM world memory
	dm.Agent.Memory.Add(fmt.Sprintf("A treacherous underground dungeon governed by %s holds lost relics and sleeping perils.", dmCfg.Name), "core", 10.0, "lore", "dungeon")

	// Dynamic tool provider based on current room context
	dm.Agent.ContextualTagsProvider = func() []string {
		return []string{"dungeon", "action"}
	}

	dm.RefreshTools()
	return dm
}

// RefreshTools updates the tools available in the DM's registry based on current game state.
func (dm *DungeonMaster) RefreshTools() {
	dm.Agent.Tools = tools.NewRegistry()
	adventTools := BuildAdventureTools(dm.Game)
	for _, t := range adventTools {
		dm.Agent.Tools.Register(t)
	}
}

// isDualModel reports whether a separate fast tool model is configured alongside the creative primary model.
func (dm *DungeonMaster) isDualModel() bool {
	cfg := dm.Agent.LLMClient.Config()
	return cfg.ToolModel != "" && cfg.ToolModel != cfg.Model
}

// buildDistilledNarrationPrompt creates a compact, focused creative writing prompt (~80-120 tokens)
// instead of pumping thousands of tokens of internal cognitive state, traits, and drives to the large model.
func (dm *DungeonMaster) buildDistilledNarrationPrompt(playerAction, outcome string) []openai.ChatCompletionMessage {
	room := dm.Game.Rooms[dm.Game.CurrentRoomID]
	roomInfo := dm.Game.CurrentRoomID
	if room != nil {
		roomType := room.RoomType
		if roomType == "" {
			roomType = room.ID
		}
		if room.BasePrompt != "" {
			roomInfo = fmt.Sprintf("%s (%s)", roomType, room.BasePrompt)
		} else {
			roomInfo = roomType
		}
	}

	sysPrompt := fmt.Sprintf("You are %s, the Dungeon Master (%s style).\nDeliver vivid, 1-2 sentence atmospheric second-person narration (\"You...\"). Keep it concise. Do not break character.",
		dm.Config.Name, dm.Config.Archetype)

	userPrompt := fmt.Sprintf("Location: %s\nPlayer Action: %q\nOutcome: %s\nTask: Narrate the atmospheric consequence in 1-2 sentences.",
		roomInfo, playerAction, outcome)

	return []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: sysPrompt},
		{Role: openai.ChatMessageRoleUser, Content: userPrompt},
	}
}

// narrateAction generates atmospheric narration using either the large creative model or fast tool model.
func (dm *DungeonMaster) narrateAction(ctx context.Context, playerAction, outcome string, escalateToBigModel bool) (string, error) {
	messages := dm.buildDistilledNarrationPrompt(playerAction, outcome)
	cfg := dm.Agent.LLMClient.Config()

	var modelToUse string
	if escalateToBigModel || !dm.isDualModel() {
		modelToUse = cfg.Model
	} else {
		modelToUse = cfg.ToolModel
	}

	return dm.Agent.LLMClient.Chat(ctx, messages, llm.WithModel(modelToUse), llm.WithTemperature(dm.Config.Temperature))
}

// classifyNeedsCreative uses the small model to rapidly classify whether an ambiguous player input warrants the 26B creative model.
// classifyNeedsCreative uses the small model to rapidly classify whether an action or text warrants the 26B creative model.
func (dm *DungeonMaster) classifyNeedsCreative(ctx context.Context, playerInput, outcome string) bool {
	content := playerInput
	if outcome != "" {
		content = fmt.Sprintf("Action: %s\nOutcome: %s", playerInput, outcome)
	} else {
		words := strings.Fields(playerInput)
		if len(words) <= 2 {
			return false
		}
	}

	prompt := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "Classify if the player's action and outcome requires dramatic creative narration (YES), or is a simple routine mechanic like opening an ordinary door or picking up an item (NO). Reply with ONLY 'YES' or 'NO'.",
		},
		{Role: openai.ChatMessageRoleUser, Content: content},
	}

	resp, err := dm.Agent.LLMClient.Chat(ctx, prompt,
		llm.WithModel(dm.Agent.LLMClient.Config().ToolModel),
		llm.WithMaxTokens(5),
		llm.WithTemperature(0.0),
	)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToUpper(resp), "YES")
}

// Step processes a player turn, executing tools or quick commands, advancing world state, and returning narration.
func (dm *DungeonMaster) Step(ctx context.Context, playerInput string) (string, error) {
	playerInput = strings.TrimSpace(playerInput)
	if playerInput == "" {
		return "", nil
	}

	// Capture initial state snapshot before action
	prevRoomID := dm.Game.CurrentRoomID
	prevRoom := dm.Game.Rooms[prevRoomID]
	prevRoomVisited := prevRoom != nil && prevRoom.Visited
	prevHP := dm.Game.PlayerHP

	// 1. Try quick local command first for immediate low-latency intents
	handled, out, ambiguous := dm.Game.ExecuteQuickCommand(playerInput)
	if handled {
		if len(ambiguous) > 0 {
			return out, nil // Caller handles disambiguation
		}

		lowerInput := strings.ToLower(playerInput)

		// Fast-path A: Purely informational or rejection outputs require ZERO LLM calls (0ms latency)
		if lowerInput == "inventory" || lowerInput == "inv" || lowerInput == "i" ||
			lowerInput == "save" || lowerInput == "load" ||
			strings.HasPrefix(out, "There is no door") ||
			strings.HasPrefix(out, "The door is closed") ||
			strings.HasPrefix(out, "You don't see that item here") ||
			strings.HasPrefix(out, "Failed to save") ||
			strings.HasPrefix(out, "Failed to load") {
			dm.Agent.Episodic.Log("player_action", fmt.Sprintf("Quick action %q -> %s", playerInput, out))
			return out, nil
		}

		// Fast-path B: Looking at a room or direction (narrative is already in out)
		if lowerInput == "look" || lowerInput == "l" || strings.HasPrefix(lowerInput, "look ") || strings.HasPrefix(lowerInput, "peer ") || strings.HasPrefix(lowerInput, "glance ") {
			dm.Game.Tick()
			dm.Agent.Episodic.Log("player_action", fmt.Sprintf("Quick action %q -> looked", playerInput))
			return out, nil
		}

		// Advance world tick for state-changing quick commands (move, take, open)
		dm.Game.Tick()
		dm.Agent.Episodic.Log("player_action", fmt.Sprintf("Quick action %q -> %s", playerInput, out))
		dm.RefreshTools()

		currRoomID := dm.Game.CurrentRoomID
		currRoom := dm.Game.Rooms[currRoomID]

		// Escalation check:
		// Did we move into an unvisited room?
		isNewRoom := currRoomID != prevRoomID && currRoom != nil && (!prevRoomVisited && !currRoom.Visited)
		// Did HP change?
		hpChanged := dm.Game.PlayerHP != prevHP

		escalate := isNewRoom || hpChanged

		// Ask Granite to classify whether this quick action outcome warrants Gemma
		if !escalate && dm.isDualModel() {
			escalate = dm.classifyNeedsCreative(ctx, playerInput, out)
		}

		// For routine mechanical actions (e.g. picking up an item, opening an ordinary door),
		// we use the fast tool model (Granite), completely avoiding heavy Gemma calls unless escalated.
		narration, err := dm.narrateAction(ctx, playerInput, out, escalate)
		if err != nil {
			return out, nil
		}
		return narration, nil
	}

	// 2. Complex or agentic turn with tool calling
	dm.RefreshTools()

	// Query tool model using a lean prompt (no massive system prompt overhead)
	leanSystemPrompt := fmt.Sprintf("You are the world engine for %s. Execute appropriate tools (move, open_door, take_item, attack, talk_to, search, etc.) for the player's action. If no tool fits, answer concisely.", dm.Config.Name)

	var tags []string
	if dm.Agent.ContextualTagsProvider != nil {
		tags = dm.Agent.ContextualTagsProvider()
	}
	availableTools := dm.Agent.Tools.FilterByTags(tags...)

	toolMessages := []openai.ChatCompletionMessage{
		{Role: openai.ChatMessageRoleSystem, Content: leanSystemPrompt},
		{Role: openai.ChatMessageRoleUser, Content: fmt.Sprintf("Player action in %s: %s", dm.Game.CurrentRoomID, playerInput)},
	}

	resp, err := dm.Agent.LLMClient.ChatWithTools(ctx, toolMessages, availableTools)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from world engine")
	}

	choice := resp.Choices[0]
	hasToolCalls := len(choice.Message.ToolCalls) > 0

	if hasToolCalls {
		var toolOutputs []string
		escalate := false

		for _, tc := range choice.Message.ToolCalls {
			fnName := tc.Function.Name
			toolRes, execErr := dm.Agent.Tools.Execute(ctx, fnName, tc.Function.Arguments)
			if execErr != nil {
				toolRes = fmt.Sprintf("Tool error: %v", execErr)
			}
			dm.Agent.Episodic.Log("tool", fmt.Sprintf("Executed %s: %s -> %s", fnName, tc.Function.Arguments, toolRes))
			toolOutputs = append(toolOutputs, fmt.Sprintf("%s: %s", fnName, toolRes))

			// Check tool escalation triggers
			if fnName == "talk_to" || fnName == "attack" || fnName == "resurrect" ||
				fnName == "spawn_npc" || fnName == "discover_room" || fnName == "add_room_detail" {
				escalate = true
			}
		}

		dm.Game.Tick()

		// State diff escalation triggers
		currRoom := dm.Game.Rooms[dm.Game.CurrentRoomID]
		if dm.Game.CurrentRoomID != prevRoomID && currRoom != nil && !currRoom.Visited {
			escalate = true
		}
		if dm.Game.PlayerHP != prevHP {
			escalate = true
		}

		combinedOut := strings.Join(toolOutputs, "; ")

		// Ask Granite classifier if tool consequence warrants Gemma
		if !escalate && dm.isDualModel() {
			escalate = dm.classifyNeedsCreative(ctx, playerInput, combinedOut)
		}

		narration, err := dm.narrateAction(ctx, playerInput, combinedOut, escalate)
		if err != nil {
			return combinedOut, nil
		}
		return narration, nil
	}

	// 3. No tools called: check if creative narration is needed or return fast response
	content := choice.Message.Content
	if dm.isDualModel() && dm.classifyNeedsCreative(ctx, playerInput, "") {
		narration, err := dm.narrateAction(ctx, playerInput, "The player acts or speaks freely in the scene.", true)
		if err == nil && narration != "" {
			dm.Agent.Episodic.Log("chat", fmt.Sprintf("Player said: %q -> %s", playerInput, narration))
			return narration, nil
		}
	}

	dm.Agent.Episodic.Log("chat", fmt.Sprintf("Player said: %q -> %s", playerInput, content))
	return content, nil
}
