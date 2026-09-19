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
