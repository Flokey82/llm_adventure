package adventure

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// RunTUI launches a Text User Interface (TUI) for manual gameplay without the use of an LLM.
// This function sets up the layout, handles user input, and provides feedback to the player.
// The TUI includes panels for narration, room details, inventory, and help.
// Commands supported: `look`, `move <dir>`, `open <dir>`, `take <item>`, `quit`.
func (g *Game) RunTUI() error {
	app := tview.NewApplication()

	// Views
	narration := tview.NewTextView()
	narration.SetDynamicColors(true)
	narration.SetBorder(true)
	narration.SetTitle("Narration")

	roomView := tview.NewTextView()
	roomView.SetDynamicColors(true)
	roomView.SetBorder(true)
	roomView.SetTitle("Room")

	invView := tview.NewTextView()
	invView.SetBorder(true)
	invView.SetTitle("Inventory")

	helpView := tview.NewTextView()
	helpView.SetBorder(true)
	helpView.SetTitle("Help")

	input := tview.NewInputField().SetLabel(": ")

	// Layout: narration on top, below that room + inventory + help, input at bottom
	top := tview.NewFlex().SetDirection(tview.FlexColumn)
	top.AddItem(roomView, 0, 2, false)
	top.AddItem(invView, 0, 1, false)
	top.AddItem(helpView, 30, 0, false)

	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(narration, 0, 3, false)
	layout.AddItem(top, 0, 1, false)
	layout.AddItem(input, 1, 0, true)

	// Command history
	var history []string
	histIdx := -1

	// Helper function to append messages to the narration pane
	appendNarration := func(format string, a ...interface{}) {
		msg := fmt.Sprintf(format+"\n", a...)
		fmt.Fprint(narration, tview.Escape(msg))
		// Scroll to the bottom of the narration pane
		narration.ScrollToEnd()
	}

	// Helper function to append user input to the narration pane
	appendUser := func(s string) {
		fmt.Fprintf(narration, "[cyan]%s\n", tview.Escape(s))
		narration.ScrollToEnd()
	}

	// Helper function to update the room, inventory, and help views
	updateViews := func() {
		// Update the room view with the current room's narrative, items, and doors
		room := g.Rooms[g.CurrentRoomID]
		roomView.Clear()
		// Build readable doors info
		doors := map[string]string{}
		for dir, d := range room.Doors {
			if d == nil {
				doors[dir] = "(missing)"
				continue
			}
			status := "closed"
			if d.Open {
				status = "open"
			}
			if d.Open {
				if other, _, ok := d.OtherSide(g.CurrentRoomID); ok {
					doors[dir] = fmt.Sprintf("%s -> %s", status, other)
					continue
				}
			}
			if d.Locked {
				doors[dir] = "locked"
			} else {
				doors[dir] = status
			}
		}
		// Show generated narrative if available, otherwise fall back to base prompt
		narrative := room.Narrative
		if narrative == "" {
			narrative = room.BasePrompt
		}
		fmt.Fprintf(roomView, "[yellow]%s\n\n[white]Items: %s\nDoors: %s\n\n", narrative, tview.Escape(fmt.Sprintf("%v", room.Items)), tview.Escape(fmt.Sprintf("%v", doors)))

		// Update the inventory view
		invView.Clear()
		fmt.Fprintf(invView, "%s", tview.Escape(fmt.Sprintf("%v", g.Inventory)))

		// Update the help view with available commands
		helpView.Clear()
		fmt.Fprint(helpView, "Commands: look | move <dir> | open <dir> | take <item> | quit\nUse Up/Down to recall commands.")
	}

	// Handle user input
	input.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		cmd := strings.TrimSpace(input.GetText())
		input.SetText("")
		if cmd == "" {
			return
		}

		// Push the command to history
		history = append(history, cmd)
		histIdx = len(history)

		// Show user input in narration with color
		appendUser("=> " + cmd)

		// Parse and execute the command
		switch {
		case cmd == "quit" || cmd == "exit":
			app.Stop()
			return
		case cmd == "look":
			appendNarration(g.Look())
		case strings.HasPrefix(cmd, "move "):
			dir := strings.TrimSpace(strings.TrimPrefix(cmd, "move "))
			appendNarration(g.Move(dir))
		case strings.HasPrefix(cmd, "open "):
			dir := strings.TrimSpace(strings.TrimPrefix(cmd, "open "))
			appendNarration(g.OpenDoor(dir))
		case strings.HasPrefix(cmd, "take "):
			item := strings.TrimSpace(strings.TrimPrefix(cmd, "take "))
			appendNarration(g.TakeItem(item))
		default:
			appendNarration("Unknown command: %s", cmd)
		}

		updateViews()
	})

	// Support Up/Down arrow keys for navigating command history
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			if len(history) == 0 {
				return event
			}
			if histIdx > 0 {
				histIdx--
			} else {
				histIdx = 0
			}
			input.SetText(history[histIdx])
			return nil
		case tcell.KeyDown:
			if len(history) == 0 {
				return event
			}
			if histIdx < len(history)-1 {
				histIdx++
				input.SetText(history[histIdx])
			} else {
				histIdx = len(history)
				input.SetText("")
			}
			return nil
		}
		return event
	})

	// Initial content
	appendNarration("Welcome to the adventure. Type 'help' for commands.")
	appendNarration(g.Look())
	updateViews()

	if err := app.SetRoot(layout, true).EnableMouse(true).Run(); err != nil {
		return err
	}
	return nil
}
