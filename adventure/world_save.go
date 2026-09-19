package adventure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Flokey82/animus/pkg/memory"
)

// NPCSaveState serializes an individual NPC and its cognitive memory.
type NPCSaveState struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Location     string   `json:"location"`
	Persona      string   `json:"persona"`
	Disposition  int      `json:"disposition"`
	MaxHP        int      `json:"max_hp"`
	CurrentHP    int      `json:"current_hp"`
	Dead         bool     `json:"dead"`
	Memory       string   `json:"memory"`
	History      string   `json:"history"`
	EpisodicLogs []string `json:"episodic_logs,omitempty"`
}

// WorldSave represents the complete persistent snapshot of the adventure world and DM.
type WorldSave struct {
	Version      int                     `json:"version"`
	SavedAt      time.Time               `json:"saved_at"`
	SaveName     string                  `json:"save_name"`
	DMConfig     DMConfig                `json:"dm_config"`
	DMEpisodic   []memory.EpisodicEvent  `json:"dm_episodic"`
	DMMemories   string                  `json:"dm_memories"`
	CurrentRoom  string                  `json:"current_room"`
	PlayerHP     int                     `json:"player_hp"`
	Inventory    []string                `json:"inventory"`
	PlayerNotes  []string                `json:"player_notes"`
	Rooms        map[string]*Room        `json:"rooms"`
	NPCs         map[string]NPCSaveState `json:"npcs"`
	Timers       map[string]int          `json:"timers"`
}

// SaveWorld persists both the game world and the Dungeon Master's memory to disk.
func SaveWorld(filePath string, g *Game, dm *DungeonMaster) error {
	if filePath == "" {
		filePath = "saves/quicksave.json"
	}
	if !strings.HasSuffix(filePath, ".json") {
		filePath += ".json"
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	ws := WorldSave{
		Version:     1,
		SavedAt:     time.Now(),
		SaveName:    filepath.Base(filePath),
		CurrentRoom: g.CurrentRoomID,
		PlayerHP:    g.PlayerHP,
		Inventory:   g.Inventory,
		PlayerNotes: g.PlayerNotes,
		Rooms:       g.Rooms,
		Timers:      g.Timers,
		NPCs:        make(map[string]NPCSaveState),
	}

	// Capture NPC states
	for id, n := range g.NPCs {
		ns := NPCSaveState{
			ID:          n.ID,
			Name:        n.Name,
			Description: n.Description,
			Location:    n.Location,
			Persona:     n.Persona,
			Disposition: n.Disposition,
			MaxHP:       n.MaxHP,
			CurrentHP:   n.CurrentHP,
			Dead:        n.Dead,
			Memory:      n.Memory,
			History:     n.History,
		}
		if n.Agent != nil {
			recentLogs := n.Agent.Episodic.Recent(20)
			for _, l := range recentLogs {
				ns.EpisodicLogs = append(ns.EpisodicLogs, l.Content)
			}
		}
		ws.NPCs[id] = ns
	}

	// Capture Dungeon Master state
	if dm != nil && dm.Agent != nil {
		ws.DMConfig = dm.Config
		ws.DMEpisodic = dm.Agent.Episodic.Recent(50)
		ws.DMMemories = dm.Agent.Memory.ActiveMemoriesSummary()
	}

	if !strings.HasSuffix(filePath, ".json") {
		filePath += ".json"
	}

	if dir := filepath.Dir(filePath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(ws, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// LoadWorld restores a saved world state into active Game and DungeonMaster instances.
func LoadWorld(filePath string, g *Game, dm *DungeonMaster) error {
	if !strings.HasSuffix(filePath, ".json") {
		filePath += ".json"
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var ws WorldSave
	if err := json.Unmarshal(data, &ws); err != nil {
		return fmt.Errorf("invalid world save format: %w", err)
	}

	// Restore Game world
	g.CurrentRoomID = ws.CurrentRoom
	g.PlayerHP = ws.PlayerHP
	g.Inventory = ws.Inventory
	g.PlayerNotes = ws.PlayerNotes
	g.Rooms = ws.Rooms
	g.Timers = ws.Timers

	// Restore NPCs
	g.NPCs = make(map[string]*NPC)
	for id, ns := range ws.NPCs {
		npc := &NPC{
			ID:          ns.ID,
			Name:        ns.Name,
			Description: ns.Description,
			Location:    ns.Location,
			Persona:     ns.Persona,
			Disposition: ns.Disposition,
			MaxHP:       ns.MaxHP,
			CurrentHP:   ns.CurrentHP,
			Dead:        ns.Dead,
			Memory:      ns.Memory,
			History:     ns.History,
		}
		g.NPCs[id] = npc
	}

	// Restore Dungeon Master
	if dm != nil && dm.Agent != nil {
		dm.Config = ws.DMConfig
		dm.Agent.Name = ws.DMConfig.Name
		dm.Agent.Personality = ws.DMConfig.Personality
		dm.Agent.BaseIdentity = fmt.Sprintf("You are %s, %s.\n%s", ws.DMConfig.Name, ws.DMConfig.Title, ws.DMConfig.NarrationStyle)

		dm.Agent.Episodic.Clear()
		for _, ev := range ws.DMEpisodic {
			dm.Agent.Episodic.Log(ev.Category, ev.Content)
		}
		if ws.DMMemories != "" {
			dm.Agent.Memory.Add(ws.DMMemories, memory.TierLongTerm, 8.0, "restored_lore")
		}
		dm.RefreshTools()
	}

	return nil
}
