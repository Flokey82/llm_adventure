package adventure

import (
	"context"

	"github.com/Flokey82/animus/pkg/tools"
	"github.com/sashabaranov/go-openai"
)

// AdventureToolAdapter wraps Game tool definitions and actions into animus.AgentTool.
type AdventureToolAdapter struct {
	toolDef openai.Tool
	game    *Game
	handler func(g *Game, args map[string]any) string
	tags    []string
}

func (a *AdventureToolAdapter) Definition() openai.Tool {
	return a.toolDef
}

func (a *AdventureToolAdapter) Execute(ctx context.Context, args map[string]any) (string, error) {
	out := a.handler(a.game, args)
	return tools.MakeSuccessPayload(out), nil
}

func (a *AdventureToolAdapter) Tags() []string {
	return a.tags
}

// BuildAdventureTools converts all available game actions into animus.AgentTools.
func BuildAdventureTools(g *Game) []tools.AgentTool {
	defs := Tools(g)
	defMap := make(map[string]openai.Tool)
	for _, d := range defs {
		defMap[d.Function.Name] = d
	}

	handlers := map[string]func(g *Game, args map[string]any) string{
		"move": func(g *Game, args map[string]any) string {
			dir, _ := args["direction"].(string)
			return g.Move(dir)
		},
		"discover_room": func(g *Game, args map[string]any) string {
			dir, _ := args["direction"].(string)
			return g.DiscoverRoom(dir)
		},
		"spawn_item": func(g *Game, args map[string]any) string {
			name, _ := args["item_name"].(string)
			reasoning, _ := args["reasoning"].(string)
			return g.SpawnItem(name, reasoning)
		},
		"open_door": func(g *Game, args map[string]any) string {
			dir, _ := args["direction"].(string)
			return g.OpenDoor(dir)
		},
		"take_item": func(g *Game, args map[string]any) string {
			item, _ := args["item_name"].(string)
			return g.TakeItem(item)
		},
		"use_item": func(g *Game, args map[string]any) string {
			item, _ := args["item_name"].(string)
			target, _ := args["target_name"].(string)
			return g.UseItem(item, target)
		},
		"drop_item": func(g *Game, args map[string]any) string {
			item, _ := args["item_name"].(string)
			return g.DropItem(item)
		},
		"search": func(g *Game, args map[string]any) string {
			return g.Search()
		},
		"look": func(g *Game, args map[string]any) string {
			return g.Look()
		},
		"add_room_detail": func(g *Game, args map[string]any) string {
			detail, _ := args["detail"].(string)
			return g.AddRoomDetail(detail)
		},
		"attack": func(g *Game, args map[string]any) string {
			target, _ := args["target_name"].(string)
			reasoning, _ := args["reasoning"].(string)
			return g.Attack(target, reasoning)
		},
		"resurrect": func(g *Game, args map[string]any) string {
			id, _ := args["npc_id"].(string)
			reasoning, _ := args["reasoning"].(string)
			return g.Resurrect(id, reasoning)
		},
		"talk_to": func(g *Game, args map[string]any) string {
			npc, _ := args["npc_name"].(string)
			msg, _ := args["message"].(string)
			return g.TalkTo(npc, msg)
		},
		"spawn_npc": func(g *Game, args map[string]any) string {
			id, _ := args["npc_id"].(string)
			name, _ := args["name"].(string)
			desc, _ := args["description"].(string)
			persona, _ := args["persona"].(string)
			disp, _ := args["disposition"].(float64)
			hp, _ := args["hp"].(float64)
			history, _ := args["history"].(string)
			return g.SpawnNPC(id, name, desc, persona, int(disp), int(hp), history)
		},
	}

	var res []tools.AgentTool
	for name, h := range handlers {
		if d, ok := defMap[name]; ok {
			res = append(res, &AdventureToolAdapter{
				toolDef: d,
				game:    g,
				handler: h,
				tags:    []string{"dungeon", "action"},
			})
		}
	}
	return res
}
