package adventure

import (
	"context"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/sashabaranov/go-openai"
)

// RunTUIWithLLM launches the Text User Interface (TUI) for the game and integrates it with the LLM.
// This function sets up the layout, handles user input, and manages interactions between the game state and the LLM.
// The TUI includes panels for narration, event logs, inventory, room view, and a map.
func (g *Game) RunTUIWithLLM(client LLMClient, model string) error {
	app := tview.NewApplication()

	// Panels
	narration := tview.NewTextView()
	narration.SetDynamicColors(true)
	narration.SetBorder(true).SetTitle("Narration")

	eventLog := tview.NewTextView()
	eventLog.SetBorder(true).SetTitle("Event Log")

	roomView := tview.NewTextView()
	roomView.SetDynamicColors(true)
	roomView.SetBorder(true).SetTitle("Room")

	invView := tview.NewTextView()
	invView.SetBorder(true).SetTitle("Inventory")

	mapView := tview.NewTextView()
	mapView.SetDynamicColors(true)
	mapView.SetBorder(true).SetTitle("Map")

	helpView := tview.NewTextView()
	helpView.SetBorder(true).SetTitle("Help")

	input := tview.NewInputField().SetLabel(": ")

	// Layout: left column = narration + event log; right column = room/inv/map/help; bottom input
	rightTop := tview.NewFlex().SetDirection(tview.FlexRow)
	rightTop.AddItem(roomView, 0, 2, false)
	rightTop.AddItem(invView, 0, 1, false)
	rightTop.AddItem(mapView, 0, 1, false)
	rightTop.AddItem(helpView, 3, 0, false)

	left := tview.NewFlex().SetDirection(tview.FlexRow)
	left.AddItem(narration, 0, 3, false)
	left.AddItem(eventLog, 0, 1, false)

	mainFlex := tview.NewFlex()
	mainFlex.AddItem(left, 0, 3, false)
	mainFlex.AddItem(rightTop, 0, 1, false)

	layout := tview.NewFlex().SetDirection(tview.FlexRow)
	layout.AddItem(mainFlex, 0, 1, false)
	layout.AddItem(input, 1, 0, true)

	// Conversation state for LLM
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: `You are the narrator of a text adventure game. Use the 'look' tool immediately when the game starts or room changes. Rely on tool outputs for state. Do not invent items or exits. Describe scene atmospherically. Call tools rather than saying you did something.
			
Keep persistent notes about the player using update_player_notes (e.g. if they are covered in poop, smelling of lavender, or have a specific injury). These notes will be visible to you in the Look tool output.`,
		},
		{Role: openai.ChatMessageRoleUser, Content: "Start the game. Look around."},
	}

	// Helper function to update static views
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
			desc := d.Description
			if desc == "" {
				desc = "door"
			}
			if d.Open {
				status = "open"
			}
			// Reveal destination only when open
			if d.Open {
				if other, _, ok := d.OtherSide(g.CurrentRoomID); ok {
					doors[dir] = fmt.Sprintf("open %s -> %s", desc, other)
					continue
				}
			}
			if d.Locked {
				doors[dir] = fmt.Sprintf("locked %s", desc)
			} else {
				doors[dir] = fmt.Sprintf("%s %s", status, desc)
			}
		}
		// Show generated narrative if available, otherwise fall back to base prompt
		narrative := room.Narrative
		if narrative == "" {
			narrative = room.BasePrompt
		}
		
		details := ""
		if len(room.Details) > 0 {
			details = "\n" + strings.Join(room.Details, " ")
		}

		fmt.Fprintf(roomView, "[yellow]%s%s\n\n[white]Items: %s\nDoors: %s", narrative, details, tview.Escape(fmt.Sprintf("%v", room.Items)), tview.Escape(fmt.Sprintf("%v", doors)))

		// Update the inventory view
		invView.Clear()
		fmt.Fprintf(invView, "%s", tview.Escape(fmt.Sprintf("%v", g.Inventory)))

		// Update the map view with a small ASCII thumbnail using room coordinates
		mapView.Clear()
		fmt.Fprintf(mapView, "Z-Level: %d\n", room.Z)
		// Build position map and bounds
		pos := map[string]*Room{}
		minX, maxX, minY, maxY := 0, 0, 0, 0
		first := true
		for _, r := range g.Rooms {
			if r.Z != room.Z {
				continue
			}
			key := fmt.Sprintf("%d,%d", r.X, r.Y)
			pos[key] = r
			if first {
				minX, maxX, minY, maxY = r.X, r.X, r.Y, r.Y
				first = false
				continue
			}
			if r.X < minX {
				minX = r.X
			}
			if r.X > maxX {
				maxX = r.X
			}
			if r.Y < minY {
				minY = r.Y
			}
			if r.Y > maxY {
				maxY = r.Y
			}
		}
		// Draw rows (y increases downward) and highlight rooms with NPCs
		hasNPC := func(roomID string) bool {
			for _, n := range g.NPCs {
				if n.Location == roomID {
					return true
				}
			}
			return false
		}
		for y := minY; y <= maxY; y++ {
			line := ""
			for x := minX; x <= maxX; x++ {
				key := fmt.Sprintf("%d,%d", x, y)
				if r, ok := pos[key]; ok {
					if r.ID == g.CurrentRoomID {
						line += "[green]*[-]"
					} else if hasNPC(r.ID) {
						line += "[red]M[-]"
					} else {
						line += "[white]o[-]"
					}
				} else {
					line += " "
				}
				line += " "
			}
			fmt.Fprintln(mapView, line)
		}
	}

	appendNarration := func(s string) {
		fmt.Fprintln(narration, tview.Escape(s))
		narration.ScrollToEnd()
	}
	appendUser := func(s string) {
		// Show user input in a distinct color, but escape text to avoid markup
		fmt.Fprintf(narration, "[cyan]%s\n", tview.Escape(s))
		narration.ScrollToEnd()
	}
	appendEvent := func(s string) {
		fmt.Fprintln(eventLog, tview.Escape(s))
		eventLog.ScrollToEnd()
	}

	// Initial render
	appendNarration("--- Connected to LLM TUI ---")
	appendNarration(g.Look())
	updateViews()

	processing := false

	// Use Pages to allow modal overlays (chooser)
	pages := tview.NewPages()
	pages.AddPage("main", layout, true, true)

	// Chooser list (created lazily)
	var chooser *tview.List
	showChooser := func(options []string, onSelect func(string)) {
		if chooser == nil {
			chooser = tview.NewList().ShowSecondaryText(false)
			chooser.SetBorder(true).SetTitle("Choose an option")
			chooser.SetSelectedBackgroundColor(tcell.ColorBlue)
		} else {
			chooser.Clear()
		}
		for i, o := range options {
			opt := o
			idx := i
			label := fmt.Sprintf("%d) %s", idx+1, opt)
			chooser.AddItem(label, "", 0, func() {
				onSelect(opt)
				pages.RemovePage("chooser")
				app.SetFocus(input)
			})
		}
		// Cancel option
		chooser.AddItem("Cancel", "", 0, func() {
			pages.RemovePage("chooser")
			app.SetFocus(input)
		})
		chooser.SetDoneFunc(func() {
			pages.RemovePage("chooser")
			app.SetFocus(input)
			appendEvent("Choice cancelled.")
		})

		// Center the chooser in a simple flex
		modal := tview.NewFlex().SetDirection(tview.FlexRow)
		modal.AddItem(nil, 0, 1, false)
		inner := tview.NewFlex()
		inner.AddItem(nil, 0, 1, false)
		inner.AddItem(chooser, 60, 0, true)
		inner.AddItem(nil, 0, 1, false)
		modal.AddItem(inner, 10, 0, true)
		modal.AddItem(nil, 0, 1, false)

		pages.AddPage("chooser", modal, true, true)
		app.SetFocus(chooser)
	}

	// Run a request to the model, handle tool calls, then final narration
	runModel := func(userInput string) {
		processing = true

		// Simple pruning to avoid unbounded context growth: keep system prompt and
		// last N messages (here N=10).
		if len(messages) > 12 {
			sys := messages[0]
			recent := messages[len(messages)-10:]
			messages = append([]openai.ChatCompletionMessage{sys}, recent...)
		}

		// Try quick local command first to bypass LLM for clear intents
		handled, out, ambiguous := g.ExecuteQuickCommand(userInput)
		if handled {
			if len(ambiguous) > 0 {
				// Show a chooser modal so the player can select
				app.QueueUpdateDraw(func() {
					appendEvent(out)
					showChooser(ambiguous, func(opt string) {
						res := g.TakeItem(opt)
						// Record user and result. We use 'user' role for the result
						// to avoid protocol violations (tool role requires ID).
						messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userInput})
						messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: "ACTION RESULT: " + res})
						app.QueueUpdateDraw(func() { appendNarration(res); updateViews() })
					})
				})
				processing = false
				return
			}
			// Not ambiguous: record user and result so the model sees state,
			// then ask the model to generate narration. Note: we use 'user' role
			// for the action result to keep the protocol simple for local actions.
			messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userInput})
			messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: "ACTION RESULT: " + out})

			// Advance world state (tick) after the player's quick action
			g.Tick()

			// Request narration from the model without tools
			respN, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{Model: model, Messages: messages})
			if err != nil {
				app.QueueUpdateDraw(func() { appendEvent("LLM error: " + err.Error()) })
				processing = false
				return
			}
			if len(respN.Choices) == 0 {
				app.QueueUpdateDraw(func() { appendEvent("LLM returned no choices") })
				processing = false
				return
			}
			msgN := respN.Choices[0].Message
			messages = append(messages, msgN)
			app.QueueUpdateDraw(func() { appendNarration(msgN.Content); updateViews() })
			processing = false
			return
		}

		// Add user message
		messages = append(messages, openai.ChatCompletionMessage{Role: openai.ChatMessageRoleUser, Content: userInput})

		// Send request
		req := openai.ChatCompletionRequest{Model: model, Messages: messages, Tools: Tools()}
		resp, err := client.CreateChatCompletion(context.Background(), req)
		if err != nil {
			app.QueueUpdateDraw(func() { appendEvent("LLM error: " + err.Error()) })
			processing = false
			return
		}

		if len(resp.Choices) == 0 {
			app.QueueUpdateDraw(func() { appendEvent("LLM returned no choices") })
			processing = false
			return
		}

		msg := resp.Choices[0].Message
		messages = append(messages, msg)

		// If model called tools, execute them and feed back outputs
		if len(msg.ToolCalls) > 0 {
			toolMsgs, logs := g.ExecuteToolCallsFromMessage(msg)
			messages = append(messages, toolMsgs...)

			for _, l := range logs {
				app.QueueUpdateDraw(func() { appendEvent("[tool] " + l) })
			}

			// Advance world state (tick) after tool execution
			g.Tick()

			// Ask model to generate narration now that tools have updated state
			resp2, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{Model: model, Messages: messages})
			if err != nil {
				app.QueueUpdateDraw(func() { appendEvent("LLM error: " + err.Error()) })
				processing = false
				return
			}
			if len(resp2.Choices) == 0 {
				app.QueueUpdateDraw(func() { appendEvent("LLM returned no choices (after tool execution)") })
				processing = false
				return
			}
			msg2 := resp2.Choices[0].Message
			messages = append(messages, msg2)
			app.QueueUpdateDraw(func() { appendNarration(msg2.Content); updateViews() })
			processing = false
			return
		}

		// No tools called; display model content
		app.QueueUpdateDraw(func() { appendNarration(msg.Content); updateViews() })
		processing = false
	}

	// Input handling
	input.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		if processing {
			return
		}
		cmd := strings.TrimSpace(input.GetText())
		input.SetText("")
		if cmd == "" {
			return
		}
		// If chooser modal is active, ignore input (focus will be on chooser)
		if pages.HasPage("chooser") {
			appendEvent("Please make a choice in the dialog, or press Esc to cancel.")
			return
		}

		// Local commands short-circuit
		switch {
		case cmd == "quit" || cmd == "exit":
			app.Stop()
			return
		}

		// Show user input in narration (colored)
		appendUser("=> " + cmd)

		// Run LLM flow in goroutine
		go runModel(cmd)
	})

	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Basic history/navigation could be added here
		return event
	})

	if err := app.SetRoot(layout, true).EnableMouse(true).Run(); err != nil {
		return err
	}
	return nil
}
