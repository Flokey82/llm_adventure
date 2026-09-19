package adventure

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDMConfigGenerationAndPersistence(t *testing.T) {
	// 1. Test Procedural Generation
	dm1 := GenerateRandomDMConfig(42)
	dm2 := GenerateRandomDMConfig(42)

	if dm1.Name != dm2.Name || dm1.Archetype != dm2.Archetype {
		t.Errorf("expected deterministic generation with same seed, got %q vs %q", dm1.Name, dm2.Name)
	}

	if dm1.Name == "" || dm1.Title == "" || dm1.NarrationStyle == "" {
		t.Fatalf("generated DM has missing fields: %+v", dm1)
	}

	// 2. Test Save and Load
	tmpDir, err := os.MkdirTemp("", "dm_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	savePath := filepath.Join(tmpDir, "custom_dm.json")
	if err := dm1.Save(savePath); err != nil {
		t.Fatalf("failed to save DM config: %v", err)
	}

	loaded, err := LoadDMConfig(savePath)
	if err != nil {
		t.Fatalf("failed to load DM config: %v", err)
	}

	if loaded.Name != dm1.Name || loaded.Archetype != dm1.Archetype || loaded.Personality.Openness != dm1.Personality.Openness {
		t.Errorf("loaded DM does not match saved DM: got %+v, want %+v", loaded, dm1)
	}
}
