// quickmatch.go
// This file handles quick command parsing and execution for the game. It interprets
// user input and executes corresponding actions without relying on an LLM.

package adventure

import (
	"fmt"
	"strings"
)

// ExecuteQuickCommand interprets clear, unambiguous user input locally (no LLM).
// If it recognizes an action, it executes it against the Game state and returns:
//   - handled: true if the input was recognized and processed, false otherwise.
//   - output: the result of the executed action, or an empty string if unhandled.
//   - ambiguousMatches: a list of potential matches if the input was ambiguous.
//     If non-empty, the caller should prompt the user to choose one of the matches.
//     No state change is applied when ambiguousMatches is non-empty.
func (g *Game) ExecuteQuickCommand(input string) (bool, string, []string) {
	s := strings.TrimSpace(strings.ToLower(input))
	if s == "" {
		return false, "", nil
	}

	// Handle exact simple commands like "look" or "inventory"
	switch s {
	case "look", "l":
		return true, g.Look(), nil
	case "inventory", "inv":
		return true, fmt.Sprintf("Inventory: %v", g.Inventory), nil
	}

	// Handle movement commands like "move north" or "north"
	movePrefixes := []string{"move ", "go ", "walk "} // Common verbs for movement
	for _, p := range movePrefixes {
		if strings.HasPrefix(s, p) {
			dir := strings.TrimSpace(strings.TrimPrefix(s, p))
			return true, g.Move(dir), nil
		}
	}
	// Allow single-word directions like "north"
	switch s {
	case "north", "south", "east", "west":
		return true, g.Move(s), nil
	}

	// Handle door commands like "open north" or "unlock door"
	if strings.HasPrefix(s, "open ") || strings.HasPrefix(s, "unlock ") {
		// Strip leading verb and trailing word "door" if present
		rest := s
		if strings.HasPrefix(rest, "open ") {
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "open "))
		} else {
			rest = strings.TrimSpace(strings.TrimPrefix(rest, "unlock "))
		}
		rest = strings.TrimSuffix(rest, " door")
		rest = strings.TrimSpace(rest)
		if rest == "" {
			return false, "", nil
		}
		return true, g.OpenDoor(rest), nil
	}

	// Handle item commands like "take key" or "pick up lantern"
	takePrefixes := []string{"take ", "get ", "pick up ", "grab "} // Common verbs for taking items
	for _, p := range takePrefixes {
		if strings.HasPrefix(s, p) {
			item := strings.TrimSpace(strings.TrimPrefix(s, p))
			if item == "" {
				return false, "", nil
			}
			// Find matching items in the room
			matches := []string{}
			for _, it := range g.Rooms[g.CurrentRoomID].Items {
				ni := strings.ToLower(strings.ReplaceAll(it, " ", "_"))
				q := strings.ToLower(strings.ReplaceAll(item, " ", "_"))
				if ni == q || strings.Contains(ni, q) || strings.Contains(q, ni) {
					matches = append(matches, it)
				}
			}
			if len(matches) == 0 {
				return true, "You don't see that item here.", nil
			}
			if len(matches) == 1 {
				out := g.TakeItem(matches[0])
				return true, out, nil
			}
			// Handle ambiguous matches
			prompt := fmt.Sprintf("Multiple items match '%s': %v. Please choose.", item, matches)
			return true, prompt, matches
		}
	}

	// If no command was recognized, return unhandled
	return false, "", nil
}
