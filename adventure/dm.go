package adventure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Flokey82/animus/pkg/engine"
	"github.com/Flokey82/animus/pkg/llm"
	"github.com/Flokey82/animus/pkg/tools"
	"github.com/Flokey82/animus/pkg/traits"
	"github.com/sashabaranov/go-openai"
)

// DungeonMaster is an autonomous Animus cognitive agent that runs the text adventure world.
type DungeonMaster struct {
	Game  *Game
	Agent *engine.Agent
}

// NewDungeonMaster creates a Dungeon Master agent wired to the game instance.
func NewDungeonMaster(g *Game, llmCfg llm.Config) *DungeonMaster {
	dmAgent := engine.NewAgent(engine.AgentConfig{
		Name: "Dungeon Master",
		BaseIdentity: `You are the dark fantasy Dungeon Master governing this text adventure.
Your role:
1. Impartially interpret player intents and run world actions (move, discover_room, spawn_item, open_door, attack, etc.).
2. Deliver vivid, atmospheric, present-tense, second-person narration ("You step into...").
3. Keep narrations concise (2-4 sentences). Do not break character or mention game engine internals.`,
		Personality: traits.OCEAN{
			Openness:          90.0, // Highly imaginative & creative
			Conscientiousness: 85.0, // Methodical rule enforcement
			Extraversion:      50.0,
			Agreeableness:     60.0, // Fair and impartial
			Neuroticism:       20.0, // Calm & composed
		},
		LLMConfig: llmCfg,
	})

	dm := &DungeonMaster{
		Game:  g,
		Agent: dmAgent,
	}

	// Seed DM world memory
	dm.Agent.Memory.Add("A treacherous underground dungeon holds lost relics and sleeping perils.", "core", 10.0, "lore", "dungeon")

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
