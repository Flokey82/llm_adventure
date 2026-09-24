package adventure

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestPresetScenarios(t *testing.T) {
	presets := PresetScenarios()
	expected := []string{"victorian", "starship", "cyberpunk", "sunken"}

	for _, name := range expected {
		sc, exists := presets[name]
		if !exists {
			t.Errorf("expected preset %q to exist", name)
			continue
		}
		if sc.Name == "" || sc.Atmosphere == "" || len(sc.Templates) == 0 {
			t.Errorf("preset %q is missing required metadata", name)
		}

		g := NewGameWithScenario(sc, 100)
		if g.ScenarioName != sc.Name {
			t.Errorf("expected scenario name %q, got %q", sc.Name, g.ScenarioName)
		}
		if g.CurrentRoomID == "" {
			t.Errorf("expected non-empty start room for scenario %q", name)
		}
		if len(g.Rooms) < 5 {
			t.Errorf("expected at least 5 rooms generated for %q, got %d", name, len(g.Rooms))
		}
	}
}

func TestScenarioReachability(t *testing.T) {
	presets := PresetScenarios()
	for name, sc := range presets {
		g := NewGameWithScenario(sc, 999)
		startRoom := g.Rooms[g.CurrentRoomID]
		if startRoom == nil {
			t.Fatalf("scenario %s start room %s not found in rooms map", name, g.CurrentRoomID)
		}

		// Simple BFS to verify reachability from start room
		visited := map[string]bool{g.CurrentRoomID: true}
		queue := []string{g.CurrentRoomID}

		for len(queue) > 0 {
			curr := queue[0]
			queue = queue[1:]
			r := g.Rooms[curr]
			for _, d := range r.Doors {
				if d == nil {
					continue
				}
				if other, _, ok := d.OtherSide(curr); ok {
					if !visited[other] {
						visited[other] = true
						queue = append(queue, other)
					}
				}
			}
		}

		// Ensure all rooms are reachable
		if len(visited) != len(g.Rooms) {
			t.Errorf("scenario %s: reachable rooms (%d) does not match total rooms (%d)", name, len(visited), len(g.Rooms))
		}
	}
}

func TestLiveScenarioGeneration(t *testing.T) {
	// Only run live test if Lemonade is reachable
	cfg := openai.DefaultConfig("sk-no-key-required")
	cfg.BaseURL = "http://192.168.86.208:8000/api/v1"
	client := openai.NewClientWithConfig(cfg)

	ctx := context.Background()
	sc, err := GenerateScenario(ctx, client, "granite-4.0-h-tiny-GGUF", "A volcanic subterranean dwarven forge overrun by magma elementals")
	if err != nil {
		t.Skipf("skipping live test due to server or network error: %v", err)
	}

	if sc.Name == "" {
		t.Errorf("expected generated scenario to have a name")
	}
	if len(sc.Templates) < 4 {
		t.Errorf("expected at least 4 generated templates, got %d", len(sc.Templates))
	}
	t.Logf("Successfully generated scenario: %s (Atmosphere: %s, Rooms: %d)", sc.Name, sc.Atmosphere, len(sc.Templates))

	g := NewGameWithScenario(*sc, 777)
	if g.CurrentRoomID == "" {
		t.Errorf("expected valid current room ID")
	}
	if len(g.Rooms) == 0 {
		t.Errorf("expected rooms to be generated from scenario")
	}
}
