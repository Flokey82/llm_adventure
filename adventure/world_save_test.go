package adventure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Flokey82/animus/pkg/llm"
)

func TestWorldSaveAndLoad(t *testing.T) {
	game := NewGame()
	dmConfig := GenerateRandomDMConfig(101)
	dm := NewDungeonMaster(game, llm.Config{MockMode: true}, dmConfig)

	// Mutate game state
	game.Rooms["crypt_vault"] = &Room{
		ID:        "crypt_vault",
		RoomType:  "Crypt Vault",
		Narrative: "A dark stone chamber with resting sarcophagi.",
	}
	game.CurrentRoomID = "crypt_vault"
	game.Inventory = append(game.Inventory, "rusty_key", "bone_dagger")
	game.PlayerHP = 85
	game.PlayerNotes = []string{"smells of brimstone", "wounded left shoulder"}

	// Mutate DM memory and episodic log
	dm.Agent.Episodic.Log("player_action", "Player defeated the gargoyle")
	dm.Agent.Memory.Add("Player spared the goblin scout.", "core", 9.0, "goblin", "mercy")

	// Save world
	tmpDir, err := os.MkdirTemp("", "world_save_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	savePath := filepath.Join(tmpDir, "adventure_save.json")
	if err := SaveWorld(savePath, game, dm); err != nil {
		t.Fatalf("failed to save world: %v", err)
	}

	// Create fresh game and DM
	newGame := NewGame()
	newDM := NewDungeonMaster(newGame, llm.Config{MockMode: true})

	// Load world
	if err := LoadWorld(savePath, newGame, newDM); err != nil {
		t.Fatalf("failed to load world: %v", err)
	}

	// Assertions
	if newGame.CurrentRoomID != "crypt_vault" {
		t.Errorf("expected room 'crypt_vault', got %q", newGame.CurrentRoomID)
	}
	if len(newGame.Inventory) != 2 || newGame.Inventory[0] != "rusty_key" {
		t.Errorf("inventory mismatch: %+v", newGame.Inventory)
	}
	if newGame.PlayerHP != 85 {
		t.Errorf("expected HP 85, got %d", newGame.PlayerHP)
	}
	if newDM.Config.Name != dmConfig.Name {
		t.Errorf("DM name mismatch: got %q, want %q", newDM.Config.Name, dmConfig.Name)
	}

	// Verify episodic logs and memories restored
	logs := newDM.Agent.Episodic.FullLogText()
	if !strings.Contains(logs, "defeated the gargoyle") {
		t.Errorf("expected episodic log restored in new DM, got:\n%s", logs)
	}
	mem := newDM.Agent.Memory.ActiveMemoriesSummary()
	if !strings.Contains(mem, "spared the goblin scout") {
		t.Errorf("expected memory restored in new DM, got:\n%s", mem)
	}
}
