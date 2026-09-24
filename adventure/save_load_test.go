package adventure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	savePath := filepath.Join(tmpDir, "test_save.json")

	presets := PresetScenarios()
	sc := presets["starship"]
	g := NewGameWithScenario(sc, 42)

	g.Inventory = []string{"plasma_torch", "stim_pack"}
	g.PlayerHP = 75
	g.PlayerMaxHP = 120
	g.PlayerNotes = []string{"smells like ozone"}

	// Save game
	if err := g.Save(savePath); err != nil {
		t.Fatalf("failed to save game: %v", err)
	}

	// Create a new blank game and load the saved state
	loaded := NewGame()
	if err := loaded.Load(savePath); err != nil {
		t.Fatalf("failed to load game: %v", err)
	}

	if loaded.ScenarioName != sc.Name {
		t.Errorf("expected ScenarioName %q, got %q", sc.Name, loaded.ScenarioName)
	}
	if loaded.WorldPrompt != sc.Atmosphere {
		t.Errorf("expected WorldPrompt %q, got %q", sc.Atmosphere, loaded.WorldPrompt)
	}
	if loaded.PlayerHP != 75 {
		t.Errorf("expected PlayerHP 75, got %d", loaded.PlayerHP)
	}
	if loaded.PlayerMaxHP != 120 {
		t.Errorf("expected PlayerMaxHP 120, got %d", loaded.PlayerMaxHP)
	}
	if len(loaded.Inventory) != 2 || loaded.Inventory[0] != "plasma_torch" {
		t.Errorf("inventory mismatch: %v", loaded.Inventory)
	}
	if len(loaded.PlayerNotes) != 1 || loaded.PlayerNotes[0] != "smells like ozone" {
		t.Errorf("player notes mismatch: %v", loaded.PlayerNotes)
	}
	if loaded.CurrentRoomID != g.CurrentRoomID {
		t.Errorf("room ID mismatch: %q vs %q", loaded.CurrentRoomID, g.CurrentRoomID)
	}
}

func TestQuickMatchSaveAndLoad(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "quickmatch_save_test.json")
	defer os.Remove(tmpFile)

	g := NewGame(123)
	g.Inventory = append(g.Inventory, "test_item")

	handled, out, _ := g.ExecuteQuickCommand("save " + tmpFile)
	if !handled || out == "" {
		t.Fatalf("expected save quick command to be handled, got handled=%v, out=%s", handled, out)
	}

	g2 := NewGame(456)
	handledLoad, outLoad, _ := g2.ExecuteQuickCommand("load " + tmpFile)
	if !handledLoad || outLoad == "" {
		t.Fatalf("expected load quick command to be handled, got handled=%v, out=%s", handledLoad, outLoad)
	}

	found := false
	for _, item := range g2.Inventory {
		if item == "test_item" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find test_item in loaded game inventory, got %v", g2.Inventory)
	}
}
