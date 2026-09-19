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

// Step processes a player turn, executing tools or quick commands, advancing world state, and returning narration.
func (dm *DungeonMaster) Step(ctx context.Context, playerInput string) (string, error) {
	playerInput = strings.TrimSpace(playerInput)
	if playerInput == "" {
		return "", nil
	}

	// 1. Try quick local command first for immediate low-latency intents
	handled, out, ambiguous := dm.Game.ExecuteQuickCommand(playerInput)
	if handled {
		if len(ambiguous) > 0 {
			return out, nil // Caller handles disambiguation
		}
		// Advance game world tick
		dm.Game.Tick()
		dm.Agent.Episodic.Log("player_action", fmt.Sprintf("Quick action %q -> %s", playerInput, out))

		// Ask DM to narrate consequence of action
		dm.RefreshTools()
		narrationPrompt := fmt.Sprintf("The player executed: %q. Action outcome: %s. Narrate the atmospheric consequence in 1-2 sentences.", playerInput, out)
		narration, err := dm.Agent.LLMClient.Chat(ctx, []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: dm.Agent.BuildSystemPrompt()},
			{Role: openai.ChatMessageRoleUser, Content: narrationPrompt},
		}, llm.WithTemperature(0.7))
		if err != nil {
			return out, nil
		}
		return narration, nil
	}

	// 2. Full agentic turn with contextual tool calling
	dm.RefreshTools()
	narration, err := dm.Agent.Chat(ctx, playerInput, "Player")
	if err != nil {
		return "", err
	}

	// Advance world tick
	dm.Game.Tick()
	return narration, nil
}
