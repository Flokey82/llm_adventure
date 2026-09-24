package adventure

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/sashabaranov/go-openai"
)

// Scenario defines a thematic world setting with its narrative atmosphere,
// seed room templates, starting room, and wandering creature.
type Scenario struct {
	Name        string         `json:"name"`
	Atmosphere  string         `json:"atmosphere"`
	MonsterName string         `json:"monster_name"`
	MonsterDesc string         `json:"monster_desc"`
	DoorDescs   []string       `json:"door_descs"`
	UpDescs     []string       `json:"up_descs"`
	DownDescs   []string       `json:"down_descs"`
	Templates   []RoomTemplate `json:"templates"`
}

// PresetScenarios returns the built-in scenario templates that require zero LLM generation time.
func PresetScenarios() map[string]Scenario {
	return map[string]Scenario{
		"victorian": {
			Name:        "Victorian Manor",
			Atmosphere:  "A decaying Victorian manor shrouded in thick autumn mist, peeling wallpaper, creaking floorboards, and forgotten secrets.",
			MonsterName: "Grue",
			MonsterDesc: "A lurking, wordless horror of sharp shadows and many teeth.",
			DoorDescs:   []string{"creaky oak door", "heavy iron door", "ornate wooden door", "simple wooden door", "reinforced cellar door"},
			UpDescs:     []string{"grand oak staircase", "spiral staircase", "wooden attic ladder"},
			DownDescs:   []string{"stone cellar stairs", "cellar trapdoor", "narrow basement steps"},
			Templates:   roomTemplates,
		},
		"starship": {
			Name:        "Derelict Starship",
			Atmosphere:  "A silent, derelict deep-space freighter with flickering emergency sirens, cold titanium bulkheads, hissing atmospheric leaks, and dark crawlways.",
			MonsterName: "Xenomorphic Stalker",
			MonsterDesc: "A sleek, chitinous predator slinking silently through the overhead ventilation ducts.",
			DoorDescs:   []string{"pressurized airlock hatch", "pneumatic bulkhead door", "reinforced blast door", "sliding titanium door", "service hatch"},
			UpDescs:     []string{"grav-lift shaft", "vertical access ladder", "pressurized upper deck hatch"},
			DownDescs:   []string{"lower deck service hatch", "maintenance shaft ladder", "cargo lift"},
			Templates: []RoomTemplate{
				{Name: "airlock", Description: "pressurized airlock chamber, blinking red warning beacon, discarded space helmet, heavy airlock hatch", Items: []string{"emergency_beacon", "cutting_torch"}, MaxCount: 1},
				{Name: "command_bridge", Description: "command bridge, shattered viewscreen showing distant stars, dormant holotable, captain's console", Items: []string{"encryption_keycard"}, MaxCount: 1},
				{Name: "cryo_bay", Description: "frost-coated cryo pods, leaking coolant, humming life support console", Items: []string{"medical_stim"}, MaxCount: 2},
				{Name: "reactor_core", Description: "pulsing antimatter reactor chamber, elevated catwalks, high radiation warning signs", Items: []string{"coolant_canister"}, MaxCount: 1},
				{Name: "hydroponics", Description: "overgrown zero-g greenhouse, bioluminescent alien vines, broken glass enclosures", Items: []string{"strange_fruit"}, MaxCount: 2},
				{Name: "crew_quarters", Description: "ransacked living quarters, floating personal logs, magnetic boots", Items: []string{"datapad"}, MaxCount: 4},
				{Name: "cargo_bay", Description: "vast pressurized cargo bay, magnetic freight containers, heavy cargo loader", Items: []string{"heavy_wrench"}, MaxCount: 2},
				{Name: "maintenance_shaft", Description: "cramped service conduits, sparking wiring, crawlspace tunnels", Items: []string{"wire_cutters"}, MaxCount: 4},
			},
		},
		"cyberpunk": {
			Name:        "Cyberpunk Sprawl",
			Atmosphere:  "A neon-drenched dystopian megacity underworld with acid rain, holographic advertisements, dirty noodle stalls, and illicit black clinics.",
			MonsterName: "Rogue Cyber-Assassin",
			MonsterDesc: "A heavily augmented mercenary clad in active-camo chrome, vibrating blade unsheathed.",
			DoorDescs:   []string{"hydraulic steel door", "reinforced security gate", "grated metal shutter", "neon-lit sliding door", "bulletproof glass barrier"},
			UpDescs:     []string{"fire escape ladder", "rooftop maintenance stairs", "grated service elevator"},
			DownDescs:   []string{"subway escalator", "drainage maintenance shaft", "basement access stairs"},
			Templates: []RoomTemplate{
				{Name: "rain_slicked_alley", Description: "steamy alleyway lit by flickering neon signs, puddles reflecting holo-ads, buzzing surveillance drones", Items: []string{"broken_datashard", "stray_drone_battery"}, MaxCount: 1},
				{Name: "noodle_bar", Description: "cramped noodle shop, steam rising from synthetic broth, synth-jazz playing softly", Items: []string{"chopsticks", "credits_chip"}, MaxCount: 2},
				{Name: "netrunner_den", Description: "dark basement filled with glowing server racks, VR deck cables, cooling fans", Items: []string{"deck_cable"}, MaxCount: 2},
				{Name: "chop_shop", Description: "cluttered cyberware clinic, operating chair, surgical lasers, cybernetic limb parts", Items: []string{"neural_interface"}, MaxCount: 1},
				{Name: "corporate_plaza", Description: "monolithic glass skyscraper lobby, automated security turrets, sleek marble floor", Items: []string{"corp_id_badge"}, MaxCount: 1},
				{Name: "nightclub", Description: "thumping bass, flashing strobe lights, chrome bar counter, synthetic intoxicants", Items: []string{"neon_cocktail"}, MaxCount: 2},
				{Name: "black_market", Description: "underground subway bazaar, illegal tech vendors, encrypted datashards", Items: []string{"emp_grenade"}, MaxCount: 2},
			},
		},
		"sunken": {
			Name:        "Sunken Galleon",
			Atmosphere:  "An ancient pirate shipwreck settled upon a glowing coral reef beneath the cold ocean waves, air pockets trapped in rotting timber cabins.",
			MonsterName: "Drowned Captain",
			MonsterDesc: "A barnacle-encrusted pirate captain clutching a rusted cutlass with ghostly glowing eyes.",
			DoorDescs:   []string{"rotted timber hatch", "waterlogged cabin door", "barnacle-encrusted grating", "rusted iron hatch"},
			UpDescs:     []string{"ship rigging ladder", "companionway stairs", "hatch to upper weather deck"},
			DownDescs:   []string{"ballast ladder", "bilge access hatch", "cargo hold grating"},
			Templates: []RoomTemplate{
				{Name: "captains_cabin", Description: "tilted wooden cabin, sea water sloshing around boots, rusted sea chest, waterlogged navigation charts", Items: []string{"flintlock_pistol", "brass_compass"}, MaxCount: 1},
				{Name: "gun_deck", Description: "row of rusted bronze cannons, loose cannonballs rolling in silt, hanging fishing nets", Items: []string{"rusty_cutlass"}, MaxCount: 2},
				{Name: "cargo_hold", Description: "shattered rum barrels, gold doubloons half-buried in sand, barnacles", Items: []string{"gold_doubloon"}, MaxCount: 2},
				{Name: "coral_reef", Description: "sunlit breach in the ship's hull leading to vibrant, swaying coral and schools of silver fish", Items: []string{"sea_shell"}, MaxCount: 3},
				{Name: "crews_mess", Description: "decaying wooden mess table, pewter tankards, skeletal remains clinging to a post", Items: []string{"empty_bottle"}, MaxCount: 2},
				{Name: "crows_nest", Description: "collapsed mast covered in sea kelp, offering a view of the vast blue ocean abyss", Items: []string{"spyglass"}, MaxCount: 1},
			},
		},
	}
}

// GenerateScenario uses the LLM to generate a customized scenario from a free-text description.
func GenerateScenario(ctx context.Context, client LLMClient, model, userDescription string) (*Scenario, error) {
	systemPrompt := `You are an expert game designer creating a text adventure setting.
Given the player's scenario premise, return a JSON object defining the scenario.
You MUST provide between 6 and 8 unique rooms in the "templates" list.
Structure:
{
  "name": "Short Setting Name",
  "atmosphere": "1-2 sentence atmospheric description for the DM/narrator",
  "monster_name": "Name of a wandering threat/monster",
  "monster_desc": "Short evocative description of the monster",
  "door_descs": ["thematic horizontal door 1", "thematic horizontal door 2", "thematic horizontal door 3"],
  "up_descs": ["thematic vertical ascent 1 (e.g. stairs, ladder, lift)", "thematic vertical ascent 2"],
  "down_descs": ["thematic vertical descent 1 (e.g. hatch, stairs, shaft)", "thematic vertical descent 2"],
  "templates": [
    {
      "name": "start_room_id",
      "description": "atmospheric sensory details",
      "items": ["item1", "item2"],
      "max_count": 1
    },
    {
      "name": "second_room_id",
      "description": "atmospheric sensory details",
      "items": ["item"],
      "max_count": 2
    },
    {
      "name": "third_room_id",
      "description": "atmospheric sensory details",
      "items": ["item"],
      "max_count": 2
    },
    {
      "name": "fourth_room_id",
      "description": "atmospheric sensory details",
      "items": ["item"],
      "max_count": 2
    },
    {
      "name": "fifth_room_id",
      "description": "atmospheric sensory details",
      "items": ["item"],
      "max_count": 2
    },
    {
      "name": "sixth_room_id",
      "description": "atmospheric sensory details",
      "items": ["item"],
      "max_count": 1
    }
  ]
}
The first room in "templates" MUST be the starting room with max_count: 1.`

	userMsg := fmt.Sprintf("Premise: %s", userDescription)

	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userMsg},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONObject,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("scenario generation failed: %w", err)
	}
	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from scenario generator")
	}

	var sc Scenario
	if err := json.Unmarshal([]byte(resp.Choices[0].Message.Content), &sc); err != nil {
		return nil, fmt.Errorf("failed to parse scenario JSON: %w (raw: %s)", err, resp.Choices[0].Message.Content)
	}

	if len(sc.Templates) == 0 {
		return nil, fmt.Errorf("generated scenario contained no room templates")
	}

	// Backfill vertical passages if the LLM left them empty
	if len(sc.UpDescs) == 0 {
		sc.UpDescs = []string{"flight of stairs leading up", "vertical access ladder", "maintenance shaft"}
	}
	if len(sc.DownDescs) == 0 {
		sc.DownDescs = []string{"flight of stairs leading down", "descent ladder", "lower hatch"}
	}

	// Backfill generic rooms if small model generated fewer than 4 templates
	if len(sc.Templates) < 4 {
		sc.Templates = append(sc.Templates,
			RoomTemplate{Name: "side_chamber", Description: fmt.Sprintf("a shadowy side chamber in %s", sc.Name), MaxCount: 2},
			RoomTemplate{Name: "narrow_corridor", Description: fmt.Sprintf("a narrow connecting passage in %s", sc.Name), MaxCount: 3},
			RoomTemplate{Name: "hidden_recess", Description: fmt.Sprintf("a forgotten alcove tucked away in %s", sc.Name), MaxCount: 1},
		)
	}

	return &sc, nil
}

// NewGameWithScenario creates a new game configured with the given scenario and optional seed.
func NewGameWithScenario(sc Scenario, seed ...int64) *Game {
	var currentSeed int64
	if len(seed) > 0 {
		currentSeed = seed[0]
	} else {
		currentSeed = time.Now().UnixNano()
	}

	rooms, startID := GenerateMapWithTemplates(currentSeed, sc.Templates, sc.DoorDescs, sc.UpDescs, sc.DownDescs)
	rand.Seed(time.Now().UnixNano())

	g := &Game{
		CurrentRoomID:  startID,
		Rooms:          rooms,
		NPCs:           map[string]*NPC{},
		Timers:         map[string]int{},
		TimerCallbacks: make(map[string]func(*Game, string)),
		PlayerHP:       100,
		PlayerMaxHP:    100,
		ScenarioName:   sc.Name,
		WorldPrompt:    sc.Atmosphere,
	}

	// Add the scenario's wandering monster in a non-starting room
	monsterName := sc.MonsterName
	if monsterName == "" {
		monsterName = "Grue"
	}
	monsterDesc := sc.MonsterDesc
	if monsterDesc == "" {
		monsterDesc = "A terrifying shadow lurking in the dark."
	}
	monsterID := strings.ToLower(strings.ReplaceAll(monsterName, " ", "_"))

	for id := range rooms {
		if id != startID {
			g.NPCs[monsterID] = &NPC{
				ID:          monsterID,
				Name:        monsterName,
				Description: monsterDesc,
				Location:    id,
				Persona:     fmt.Sprintf("A lurking threat in %s that moves silently and strikes from darkness.", sc.Name),
				Disposition: 0,
				MaxHP:       50,
				CurrentHP:   50,
			}
			break
		}
	}

	return g
}
