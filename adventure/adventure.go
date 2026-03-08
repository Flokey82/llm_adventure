// adventure.go
// This file defines the core game logic, including the `Game` struct and its methods.
// It manages the game state, player actions, NPC interactions, and time-based events.

package adventure

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// Game represents the state of the game, including the player's current location,
// inventory, rooms, NPCs, and timers for time-based events.
type Game struct {
	CurrentRoomID          string                                        // ID of the room the player is currently in
	Inventory              []string                                      // List of items the player is carrying
	Rooms                  map[string]*Room                              // Map of room IDs to Room objects
	AI_GenerateDescription func(prompt string) string                    // Function to generate room descriptions
	AI_CharacterChat       func(systemPrompt, userMessage string) string // Function for NPC dialogue
	NPCs                   map[string]*NPC                               // Map of NPC IDs to NPC objects
	Timers                 map[string]int                                // Timers for time-based events, keyed by strings
}

// NPC represents a non-player character in the world.
// NPCs have a location, a persona for dialogue, and a disposition towards the player.
type NPC struct {
	ID          string // Unique identifier for the NPC
	Name        string // Display name of the NPC
	Location    string // ID of the room the NPC is currently in
	Persona     string // System prompt for character dialogue
	Disposition int    // 0 = Hostile, 100 = Friendly
}

// NewGame creates a new game instance. Optionally, a seed can be provided for deterministic map generation.
func NewGame(seed ...int64) *Game {
	// Generate the map based on the provided seed or use a random seed
	var rooms map[string]*Room
	if len(seed) > 0 {
		rooms = GenerateMapSeeded(seed[0])
	} else {
		rooms = GenerateMap()
	}

	// Seed the global random number generator for NPC movement and other behaviors
	rand.Seed(time.Now().UnixNano())

	// Initialize the game state
	g := &Game{
		CurrentRoomID: "entry_hall",
		Rooms:         rooms,
		NPCs:          map[string]*NPC{},
		Timers:        map[string]int{},
	}

	// Add an example wandering monster (Grue) to a non-start room
	for id := range rooms {
		if id != "entry_hall" {
			g.NPCs["grue"] = &NPC{ID: "grue", Name: "Grue", Location: id, Persona: "A lurking, wordless horror that moves silently.", Disposition: 0}
			break
		}
	}
	return g
}

// Tick advances the game state by one step. It handles timers, NPC movement, and other time-based events.
// Returns a slice of human-readable event strings describing what happened during the tick.
func (g *Game) Tick() []string {
	var events []string

	// Handle timers
	for k, v := range g.Timers {
		if v <= 0 {
			continue
		}
		g.Timers[k] = v - 1
		if g.Timers[k] <= 0 {
			if callback, exists := timerCallbacks[strings.Split(k, ":")[0]]; exists {
				callback(g, k)
			}
			delete(g.Timers, k)
		}
	}

	// Move NPCs randomly, with special behavior for the Grue
	for _, n := range g.NPCs {
		if n.Location == "" {
			continue
		}
		moveChance := 25
		if strings.EqualFold(n.ID, "grue") {
			moveChance = 60
		}
		if rand.Intn(100) >= moveChance {
			continue
		}
		room, ok := g.Rooms[n.Location]
		if !ok {
			continue
		}
		var neighbors []string
		for _, d := range room.Doors {
			if d == nil {
				continue
			}
			if other, _, ok := d.OtherSide(room.ID); ok {
				neighbors = append(neighbors, other)
			}
		}
		if len(neighbors) == 0 {
			continue
		}
		choice := neighbors[rand.Intn(len(neighbors))]
		n.Location = choice
		events = append(events, fmt.Sprintf("%s moves to %s.", n.Name, choice))
	}

	return events
}

// Move attempts to move the player in the specified direction. Returns a message describing the result.
func (g *Game) Move(direction string) string {
	room := g.Rooms[g.CurrentRoomID]
	door, exists := room.Doors[direction]
	if !exists || door == nil {
		return "There is no door in that direction."
	}
	if !door.Open {
		return "The door is closed."
	}

	otherRoom, _, ok := door.OtherSide(g.CurrentRoomID)
	if !ok || otherRoom == "" {
		return "Error: door has no destination."
	}
	prev := g.CurrentRoomID
	g.CurrentRoomID = otherRoom
	return fmt.Sprintf("You moved %s from %s to %s.", direction, prev, g.CurrentRoomID)
}

// OpenDoor attempts to open a door in the specified direction. Returns a message describing the result.
func (g *Game) OpenDoor(direction string) string {
	room := g.Rooms[g.CurrentRoomID]
	door, exists := room.Doors[direction]
	if !exists || door == nil {
		return "There is no door there to open."
	}
	// If the door is locked, check for the rusty_key in the inventory
	if door.Locked {
		hasKey := false
		for _, item := range g.Inventory {
			if item == "rusty_key" {
				hasKey = true
				break
			}
		}
		if !hasKey {
			return "It's locked. You probably need a key."
		}
		// Unlock and open the door
		door.Locked = false
	}

	door.Open = true
	return fmt.Sprintf("You opened the %s door.", direction)
}

// DropItem removes an item from the player's inventory and places it in the current room.
func (g *Game) DropItem(itemName string) string {
	// Find the item in the inventory (case-insensitive)
	idx := -1
	for i, it := range g.Inventory {
		if strings.EqualFold(it, itemName) || strings.Contains(strings.ToLower(it), strings.ToLower(itemName)) {
			idx = i
			break
		}
	}
	if idx == -1 {
		return "You don't have that item."
	}
	item := g.Inventory[idx]
	// Remove the item from the inventory and add it to the room
	g.Inventory = append(g.Inventory[:idx], g.Inventory[idx+1:]...)
	room := g.Rooms[g.CurrentRoomID]
	room.Items = append(room.Items, item)
	return fmt.Sprintf("You dropped %s.", item)
}

func (g *Game) TakeItem(itemName string) string {
	room := g.Rooms[g.CurrentRoomID]
	// Use the helper function to find the item
	idx, item := findItem(room.Items, itemName)
	if idx == -1 {
		return "You don't see that item here."
	}
	// Remove the item from the room and add it to the inventory
	room.Items = append(room.Items[:idx], room.Items[idx+1:]...)
	g.Inventory = append(g.Inventory, item)
	return fmt.Sprintf("Taken %s. Added to inventory.", item)
}

// ItemBehavior defines a function type for item-specific behaviors.
type ItemBehavior func(g *Game, target string) string

// Map of item names to their behaviors.
var itemBehaviors = map[string]ItemBehavior{
	"rusty_key": func(g *Game, target string) string {
		tg := normalizeItemName(target)
		room := g.Rooms[g.CurrentRoomID]
		if d, ok := room.Doors[tg]; ok && d != nil {
			if d.Locked {
				d.Locked = false
				d.Open = true
				return fmt.Sprintf("You unlock and open the %s door with the rusty key.", tg)
			}
			return "That door isn't locked."
		}
		for dir, d := range room.Doors {
			if d != nil && d.Locked {
				d.Locked = false
				d.Open = true
				return fmt.Sprintf("You unlock and open the %s door with the rusty key.", dir)
			}
		}
		return "You jiggle the key, but nothing obvious happens."
	},
	"ham": func(g *Game, target string) string {
		for _, n := range g.NPCs {
			if strings.EqualFold(n.Location, g.CurrentRoomID) && strings.Contains(normalizeItemName(n.Name), "dog") {
				n.Disposition = 100
				return "The dog happily devours the ham and wags its tail."
			}
		}
		return "There's no dog here to feed."
	},
}

func (g *Game) UseItem(item, target string) string {
	it := normalizeItemName(item)
	if behavior, exists := itemBehaviors[it]; exists {
		return behavior(g, target)
	}
	return "That doesn't seem to work."
}

// TalkTo uses the injected AI_CharacterChat to have an NPC respond.
func (g *Game) TalkTo(npcName, message string) string {
	// find NPC by name (case-insensitive)
	var found *NPC
	for _, n := range g.NPCs {
		if strings.EqualFold(n.Name, npcName) || strings.EqualFold(n.ID, npcName) {
			found = n
			break
		}
	}
	if found == nil {
		return "You don't see that person here."
	}
	if g.AI_CharacterChat == nil {
		return "No dialogue system is available."
	}
	// call into the injected character chat. Use the NPC persona as system prompt.
	resp := g.AI_CharacterChat(found.Persona, message)
	// Optionally, we could modify disposition based on response, but keep simple.
	return fmt.Sprintf("%s replies: %s", found.Name, resp)
}

// Helper function to generate a room narrative if not already present.
func (g *Game) generateRoomNarrative(room *Room) string {
	if room.Narrative == "" && g.AI_GenerateDescription != nil {
		prompt := fmt.Sprintf("Describe a room with these properties: %s. Keep it atmospheric but concise (under 50 words).", room.BasePrompt)
		room.Narrative = g.AI_GenerateDescription(prompt)
		room.Visited = true
	}
	if room.Narrative != "" {
		return room.Narrative
	}
	return room.BasePrompt
}

// Helper function to build door descriptions and glimpses of adjacent rooms.
func (g *Game) buildDoorDescriptions(room *Room) (map[string]string, []string) {
	doors := map[string]string{}
	var glimpses []string
	for dir, d := range room.Doors {
		if d == nil {
			doors[dir] = "(missing)"
			continue
		}
		status := "closed"
		if d.Open {
			status = "open"
			if other, _, ok := d.OtherSide(room.ID); ok {
				doors[dir] = fmt.Sprintf("%s -> %s", status, other)
				if or := g.Rooms[other]; or != nil {
					if or.Narrative != "" {
						glimpses = append(glimpses, fmt.Sprintf("To the %s you glimpse: %s", dir, or.Narrative))
					} else if or.BasePrompt != "" {
						glimpses = append(glimpses, fmt.Sprintf("To the %s you glimpse: %s", dir, or.BasePrompt))
					}
				}
				continue
			}
		}
		if d.Locked {
			doors[dir] = "locked"
		} else {
			doors[dir] = status
		}
	}
	return doors, glimpses
}

// Helper function to list NPCs present in the current room.
func (g *Game) listNPCsInRoom(room *Room) string {
	var present []string
	for _, n := range g.NPCs {
		if n == nil {
			continue
		}
		if strings.EqualFold(n.Location, room.ID) {
			mood := "neutral"
			if n.Disposition <= 33 {
				mood = "hostile"
			} else if n.Disposition >= 67 {
				mood = "friendly"
			}
			present = append(present, fmt.Sprintf("%s (%s)", n.Name, mood))
		}
	}
	if len(present) > 0 {
		return strings.Join(present, ", ")
	}
	return "none"
}

func (g *Game) Look() string {
	room := g.Rooms[g.CurrentRoomID]
	narrative := g.generateRoomNarrative(room)
	doors, glimpses := g.buildDoorDescriptions(room)
	presentStr := g.listNPCsInRoom(room)

	desc := fmt.Sprintf("CURRENT ROOM NARRATIVE: %s\nPRESENT: %s\nVISIBLE ITEMS: %v\nDOORS: %v\nGLIMPSES: %s", narrative, presentStr, room.Items, doors, strings.Join(glimpses, " "))
	return desc
}

// Helper function to normalize item names for matching.
func normalizeItemName(item string) string {
	item = strings.ToLower(item)
	item = strings.ReplaceAll(item, " ", "_")
	return item
}

// Helper function to find an item in a list by name or partial match.
func findItem(items []string, target string) (int, string) {
	normTarget := normalizeItemName(target)
	for i, item := range items {
		normItem := normalizeItemName(item)
		if normItem == normTarget || strings.Contains(normItem, normTarget) || strings.Contains(normTarget, normItem) {
			return i, item
		}
	}
	return -1, ""
}

// timerCallbacks is a global map that associates timer keys with their respective callback functions.
var timerCallbacks = make(map[string]func(*Game, string))
