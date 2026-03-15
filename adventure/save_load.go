package adventure

import (
	"encoding/json"
	"os"
)

// Save serializes the Game state to a JSON file.
// Note: Function pointers like AI_GenerateDescription and AI_CharacterChat
// are ignored by json.Marshal, which is exactly what we want.
func (g *Game) Save(filename string) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

// Load reads the Game state from a JSON file and populates the current Game instance.
// Note: This does not overwrite the injected function pointers, preserving AI capabilities.
func (g *Game) Load(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	// We unmarshal into a temporary Game object so we don't accidentally
	// wipe out the function pointers defined on the active `g` instance.
	var tempGame Game
	if err := json.Unmarshal(data, &tempGame); err != nil {
		return err
	}

	// Copy the loaded state to the active Game instance
	g.CurrentRoomID = tempGame.CurrentRoomID
	g.Inventory = tempGame.Inventory
	g.Rooms = tempGame.Rooms
	g.NPCs = tempGame.NPCs
	g.Timers = tempGame.Timers
	g.PlayerNotes = tempGame.PlayerNotes
	// Do not override TimerCallbacks or AI_ functions

	return nil
}
