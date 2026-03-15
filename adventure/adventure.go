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

type Game struct {
	CurrentRoomID          string                                // ID of the room the player is currently in
	Inventory              []string                              // List of items the player is carrying
	Rooms                  map[string]*Room                      // Map of room IDs to Room objects
	AI_GenerateDescription func(prompt string) string            // Function to generate room descriptions
	AI_GenerateRoom        func(fromRoom *Room, direction string) *Room // Function to generate new rooms
	AI_CharacterChat       func(npc *NPC, userMessage string) string // Function for NPC structured dialogue
	NPCs                   map[string]*NPC                       // Map of NPC IDs to NPC objects
	Timers                 map[string]int                                // Timers for time-based events, keyed by strings
	TimerCallbacks         map[string]func(*Game, string)                // Callbacks for timers
	PlayerNotes            []string                              // Persistent notes about the player (e.g. "covered in poop")
	PlayerHP               int                                   // Current hitpoints of the player
	PlayerMaxHP            int                                   // Maximum hitpoints of the player
}

// NPC represents a non-player character in the world.
// NPCs have a location, a persona for dialogue, and a disposition towards the player.
type NPC struct {
	ID          string // Unique identifier for the NPC
	Name        string // Display name of the NPC
	Description string // Atmosphere description of the NPC
	Location    string // ID of the room the NPC is currently in
	Persona     string // System prompt for character dialogue
	Disposition int    // 0 = Hostile, 100 = Friendly
	MaxHP       int    // Maximum hitpoints
	CurrentHP   int    // Current hitpoints
	Dead        bool   // True if the NPC is dead
	Memory      string // Condensed memory of past interactions
	History     string // Lore or history related to the world
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
		CurrentRoomID:  "entry_hall",
		Rooms:          rooms,
		NPCs:           map[string]*NPC{},
		Timers:         map[string]int{},
		TimerCallbacks: make(map[string]func(*Game, string)),
		PlayerHP:       100,
		PlayerMaxHP:    100,
	}

	// Add an example wandering monster (Grue) to a non-start room
	for id := range rooms {
		if id != "entry_hall" {
			g.NPCs["grue"] = &NPC{
				ID:          "grue",
				Name:        "Grue",
				Description: "A terrifying shadow with many sharp teeth.",
				Location:    id,
				Persona:     "A lurking, wordless horror that moves silently.",
				Disposition: 0,
				MaxHP:       50,
				CurrentHP:   50,
			}
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
			if callback, exists := g.TimerCallbacks[strings.Split(k, ":")[0]]; exists {
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

// DiscoverRoom attempts to generate and connect a new room in a given direction if one does not exist.
func (g *Game) DiscoverRoom(direction string) string {
	room := g.Rooms[g.CurrentRoomID]
	if door, exists := room.Doors[direction]; exists && door != nil {
		return fmt.Sprintf("There is already a visible path to the %s.", direction)
	}

	if g.AI_GenerateRoom != nil {
		newRoom := g.AI_GenerateRoom(room, direction)
		if newRoom != nil {
			g.Rooms[newRoom.ID] = newRoom
			g.CurrentRoomID = newRoom.ID
			return fmt.Sprintf("You discover a path to the %s and arrive at a new area... %s", direction, newRoom.BasePrompt)
		}
	}
	return fmt.Sprintf("You search to the %s, but find nothing new.", direction)
}

// SpawnItem spawns a new item in the current room.
func (g *Game) SpawnItem(itemName string, reasoning string) string {
	room := g.Rooms[g.CurrentRoomID]
	
	// Normalize item name for simplicity
	itemName = normalizeItemName(itemName)

	room.Items = append(room.Items, itemName)
	
	logMsg := fmt.Sprintf("You discover %s. (%s)", itemName, reasoning)
	if reasoning == "" {
		logMsg = fmt.Sprintf("You discover %s.", itemName)
	}
	return logMsg
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
	// call into the injected character chat, passing the NPC directly.
	resp := g.AI_CharacterChat(found, message)
	return fmt.Sprintf("%s replies: %s", found.Name, resp)
}

// Helper function to generate a room narrative if not already present.
func (g *Game) generateRoomNarrative(room *Room) string {
	if room.Narrative == "" && g.AI_GenerateDescription != nil {
		typeInfo := ""
		if room.RoomType != "" {
			typeInfo = fmt.Sprintf("It is a %s. ", room.RoomType)
		}
		furnInfo := ""
		if len(room.Furniture) > 0 {
			furnInfo = fmt.Sprintf("It contains: %s. ", strings.Join(room.Furniture, ", "))
		}
		prompt := fmt.Sprintf("Describe a room with these properties: %s%s%s. Keep it atmospheric but concise (under 50 words).", typeInfo, furnInfo, room.BasePrompt)
		room.Narrative = g.AI_GenerateDescription(prompt)
		room.Visited = true
	}
	if room.Narrative != "" {
		return room.Narrative
	}
	return room.BasePrompt
}

// AddRoomDetail adds a permanent narrative detail to the current room.
func (g *Game) AddRoomDetail(detail string) string {
	room := g.Rooms[g.CurrentRoomID]
	room.Details = append(room.Details, detail)
	return fmt.Sprintf("Added room detail: %s", detail)
}

// UpdatePlayerNotes replaces the current player notes with a new list.
func (g *Game) UpdatePlayerNotes(notes []string) string {
	g.PlayerNotes = notes
	return fmt.Sprintf("Updated player notes: %v", notes)
}

// Search attempts to find hidden secrets in the room.
func (g *Game) Search() string {
	room := g.Rooms[g.CurrentRoomID]
	if len(room.Secrets) > 0 {
		secrets := strings.Join(room.Secrets, " ")
		// Remove secrets so they aren't repeatedly discovered
		room.Secrets = nil
		return fmt.Sprintf("You search the area and discover: %s", secrets)
	}
	return "You search the area, but find nothing unusual."
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
		desc := d.Description
		if desc == "" {
			desc = "door"
		}
		if d.Open {
			status = "open"
			if other, _, ok := d.OtherSide(room.ID); ok {
				doors[dir] = fmt.Sprintf("open %s -> %s", desc, other)
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
			doors[dir] = fmt.Sprintf("locked %s", desc)
		} else {
			doors[dir] = fmt.Sprintf("%s %s", status, desc)
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
			state := "neutral"
			if n.Disposition <= 33 {
				state = "hostile"
			} else if n.Disposition >= 67 {
				state = "friendly"
			}
			if n.Dead {
				state = "dead"
			}
			present = append(present, fmt.Sprintf("%s (%s)", n.Name, state))
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

	detailsStr := ""
	if len(room.Details) > 0 {
		detailsStr = fmt.Sprintf("\nROOM DETAILS: %s", strings.Join(room.Details, " "))
	}

	notesStr := ""
	if len(g.PlayerNotes) > 0 {
		notesStr = fmt.Sprintf("\nPLAYER STATE/NOTES: %s", strings.Join(g.PlayerNotes, "; "))
	}

	desc := fmt.Sprintf("CURRENT ROOM NARRATIVE: %s%s%s\nPRESENT: %s\nVISIBLE ITEMS: %v\nDOORS: %v\nGLIMPSES: %s", narrative, detailsStr, notesStr, presentStr, room.Items, doors, strings.Join(glimpses, " "))
	return desc
}
// SpawnNPC creates a new NPC and places it in the current room.
func (g *Game) SpawnNPC(id, name, desc, persona string, disposition, hp int, history string) string {
	if _, exists := g.NPCs[id]; exists {
		return fmt.Sprintf("NPC with ID %s already exists.", id)
	}
	g.NPCs[id] = &NPC{
		ID:          id,
		Name:        name,
		Description: desc,
		Location:    g.CurrentRoomID,
		Persona:     persona,
		Disposition: disposition,
		MaxHP:       hp,
		CurrentHP:   hp,
		History:     history,
	}
	return fmt.Sprintf("Spawned NPC: %s. %s", name, desc)
}

// UpdateNPC updates the properties of an existing NPC.
func (g *Game) UpdateNPC(id, desc, memory string, disposition int, history string) string {
	npc, exists := g.NPCs[id]
	if !exists {
		return fmt.Sprintf("NPC with ID %s not found.", id)
	}
	if desc != "" {
		npc.Description = desc
	}
	if memory != "" {
		npc.Memory = memory
	}
	if disposition != -1 { // Use -1 as "no change" sentinel if needed, or check if provided
		npc.Disposition = disposition
	}
	if history != "" {
		npc.History = history
	}
	return fmt.Sprintf("Updated NPC: %s.", npc.Name)
}

// Attack deals damage to a target NPC.
func (g *Game) Attack(targetName, reasoning string) string {
	var target *NPC
	for _, n := range g.NPCs {
		if (strings.EqualFold(n.Name, targetName) || strings.EqualFold(n.ID, targetName)) && n.Location == g.CurrentRoomID {
			target = n
			break
		}
	}
	if target == nil {
		return fmt.Sprintf("You don't see %s here to attack.", targetName)
	}
	if target.Dead {
		return fmt.Sprintf("%s is already dead.", target.Name)
	}

	// Simple combat logic: deal random damage
	damage := rand.Intn(20) + 5
	target.CurrentHP -= damage
	if target.CurrentHP <= 0 {
		target.CurrentHP = 0
		target.Dead = true
		return fmt.Sprintf("You attack %s: %s. You deal %d damage and KILL them!", target.Name, reasoning, damage)
	}

	// Hostile response
	target.Disposition = 0

	return fmt.Sprintf("You attack %s: %s. You deal %d damage. They look %s.", target.Name, reasoning, damage, "furious")
}

// Resurrect brings a dead NPC back to life.
func (g *Game) Resurrect(id, reasoning string) string {
	npc, exists := g.NPCs[id]
	if !exists {
		return fmt.Sprintf("NPC with ID %s not found.", id)
	}
	if !npc.Dead {
		return fmt.Sprintf("%s is already alive.", npc.Name)
	}
	npc.Dead = false
	npc.CurrentHP = npc.MaxHP / 2 // Restore half HP
	return fmt.Sprintf("You resurrect %s: %s. They gasp as life returns to their body.", npc.Name, reasoning)
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


