package adventure

import (
	"context"
	"strings"
	"testing"

	"github.com/Flokey82/animus/pkg/llm"
)

func TestDungeonMasterCreationAndStep(t *testing.T) {
	game := NewGame()
	dm := NewDungeonMaster(game, llm.Config{
		MockMode: true,
	})

	if dm.Agent == nil {
		t.Fatalf("expected Agent to be initialized")
	}

	// Verify tools were registered
	allTools := dm.Agent.Tools.AllOpenAITools()
	if len(allTools) == 0 {
		t.Fatalf("expected DM tools to be registered")
	}

	hasMove := false
	for _, tool := range allTools {
		if tool.Function.Name == "move" {
			hasMove = true
			break
		}
	}
	if !hasMove {
		t.Errorf("expected 'move' tool in DM registry")
	}

	// Step test with quick command
	ctx := context.Background()
	narration, err := dm.Step(ctx, "look")
	if err != nil {
		t.Fatalf("unexpected step error: %v", err)
	}
	if narration == "" {
		t.Errorf("expected narration output from Step")
	}

	// Verify episodic logs in DM
	logs := dm.Agent.Episodic.FullLogText()
	if !strings.Contains(logs, "look") {
		t.Errorf("expected player action 'look' logged in DM episodic memory: %s", logs)
	}
}

func TestDualModelFastPathAndDistilledPrompt(t *testing.T) {
	game := NewGame()
	game.Inventory = append(game.Inventory, "iron_torch")

	dm := NewDungeonMaster(game, llm.Config{
		Model:     "Gemma-4-26B-A4B-it-MTP-GGUF",
		ToolModel: "granite-4.0-h-tiny-GGUF",
		MockMode:  true,
	})

	if !dm.isDualModel() {
		t.Errorf("expected isDualModel() to be true when ToolModel != Model")
	}

	ctx := context.Background()

	// Fast-path test: "inventory" should return instantly with no LLM overhead
	invOut, err := dm.Step(ctx, "inventory")
	if err != nil {
		t.Fatalf("unexpected error on inventory quick command: %v", err)
	}
	if !strings.Contains(invOut, "iron_torch") {
		t.Errorf("expected inventory to list iron_torch, got: %s", invOut)
	}

	// Distilled prompt test: ensure concise prompt size
	prompt := dm.buildDistilledNarrationPrompt("take rusty key", "You picked up the rusty key.")
	if len(prompt) != 2 {
		t.Fatalf("expected 2 messages in distilled prompt, got %d", len(prompt))
	}
	totalChars := len(prompt[0].Content) + len(prompt[1].Content)
	if totalChars > 500 {
		t.Errorf("distilled prompt is too long (%d chars), should be concise", totalChars)
	}
}
