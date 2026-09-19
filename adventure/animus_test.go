package adventure

import (
	"strings"
	"testing"

	"github.com/Flokey82/animus/pkg/engine"
	"github.com/Flokey82/animus/pkg/llm"
	"github.com/Flokey82/animus/pkg/traits"
)

func TestNPCWithAnimusAgent(t *testing.T) {
	game := NewGame()

	// Create an autonomous NPC agent using animus
	agent := engine.NewAgent(engine.AgentConfig{
		Name:         "Eldrin",
		BaseIdentity: "An ancient, enigmatic archivist guarding the dungeon secrets.",
		Personality: traits.OCEAN{
			Openness:          80,
			Conscientiousness: 85,
			Extraversion:      30, // Quiet
			Agreeableness:     70,
			Neuroticism:       20,
		},
		LLMConfig: llm.Config{
			MockMode: true,
		},
	})

	// Add seed memory to NPC agent
	agent.Memory.Add("The silver key is hidden beneath the mossy sarcophagus.", "core", 10.0, "key", "secret")

	npc := &NPC{
		ID:          "eldrin",
		Name:        "Eldrin the Archivist",
		Description: "A tall hooded figure surrounded by floating manuscripts.",
		Location:    "entry_hall",
		Persona:     "Wise and quiet.",
		Disposition: 70,
		Agent:       agent,
	}
	game.NPCs[npc.ID] = npc

	reply := game.TalkTo("Eldrin", "Greetings archivist, where is the silver key?")
	if !strings.Contains(reply, "Eldrin the Archivist replies:") {
		t.Fatalf("unexpected TalkTo reply: %s", reply)
	}

	// Verify agent logged the conversation into episodic memory
	logs := agent.Episodic.FullLogText()
	if !strings.Contains(logs, "where is the silver key?") {
		t.Errorf("expected player query logged in agent episodic memory: %s", logs)
	}
}
