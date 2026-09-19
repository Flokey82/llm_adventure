package adventure

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Flokey82/animus/pkg/traits"
)

// DMConfig defines the identity, narrative voice, and personality of a Dungeon Master.
type DMConfig struct {
	Name           string       `json:"name"`            // e.g. "Malakor the Grim", "Elowen the Whimsical"
	Title          string       `json:"title"`           // e.g. "The Shadow Chronicler", "The Iron Arbiter"
	Archetype      string       `json:"archetype"`       // e.g. "Gothic Horror", "Grimdark Tactician", "Cryptic Oracle"
	NarrationStyle string       `json:"narration_style"` // Detailed guidance on prose rhythm, sensory details, and dilemmas
	Personality    traits.OCEAN `json:"personality"`     // Five-factor personality scores
	Temperature    float32      `json:"temperature"`     // Sampling temperature (0.6 - 0.9)
}

// DefaultDMConfig returns the baseline impartial dark fantasy Dungeon Master.
func DefaultDMConfig() DMConfig {
	return DMConfig{
		Name:      "The Chronicler",
		Title:     "Keeper of the Depths",
		Archetype: "Dark Fantasy",
		NarrationStyle: `You are an impartial, atmospheric dark fantasy Dungeon Master.
- Deliver vivid, present-tense, second-person prose ("You step into...").
- Keep descriptions concise (2-4 sentences).
- Emphasize ominous dungeon ambiance, tactile textures (cold stone, dripping water, rusted iron), and rising tension.
- Never ask "What do you do?", never list menu options, and never break character.`,
		Personality: traits.OCEAN{
			Openness:          85.0,
			Conscientiousness: 85.0,
			Extraversion:      50.0,
			Agreeableness:     65.0,
			Neuroticism:       20.0,
		},
		Temperature: 0.7,
	}
}

// DMArchetype bundles thematic presets for procedural DM generation.
type DMArchetype struct {
	Archetype      string
	TitlePool      []string
	NarrationStyle string
	OCEANBias      traits.OCEAN
	BaseTemp       float32
}

var dmArchetypes = []DMArchetype{
	{
		Archetype: "Gothic Horror",
		TitlePool: []string{"The Bleak Narrator", "The Crypt Weaver", "Harbinger of Gloom"},
		NarrationStyle: `You are a gothic horror narrator. Focus on decaying grandeur, oppressive dread, psychological suspense, and creeping shadows. Describe scenes with melancholic poetry and subtle morbid undertones.`,
		OCEANBias: traits.OCEAN{
			Openness:          95.0,
			Conscientiousness: 80.0,
			Extraversion:      35.0,
			Agreeableness:     55.0,
			Neuroticism:       45.0,
		},
		BaseTemp: 0.8,
	},
	{
		Archetype: "Grimdark Tactician",
		TitlePool: []string{"The Iron Arbiter", "Master of Blades", "The Unforgiving"},
		NarrationStyle: `You are a brutal, pragmatic dungeon referee. Describe consequences with sharp physical realism, brutal weight, fatigue, blood, and unforgiving tactical perils. Praise clever survival; offer no mercy to carelessness.`,
		OCEANBias: traits.OCEAN{
			Openness:          65.0,
			Conscientiousness: 95.0,
			Extraversion:      40.0,
			Agreeableness:     40.0,
			Neuroticism:       15.0,
		},
		BaseTemp: 0.65,
	},
	{
		Archetype: "Cryptic Oracle",
		TitlePool: []string{"The Blind Seer", "Voice of the Astral Veil", "The Rune Carver"},
		NarrationStyle: `You are an ancient oracle speaking in enigmatic fragments and celestial metaphors. Treat dungeon events as predetermined threads of fate. Weave hints of forgotten epochs, arcane prophecies, and strange echoes.`,
		OCEANBias: traits.OCEAN{
			Openness:          98.0,
			Conscientiousness: 70.0,
			Extraversion:      30.0,
			Agreeableness:     70.0,
			Neuroticism:       30.0,
		},
		BaseTemp: 0.85,
	},
	{
		Archetype: "Mythic Epic",
		TitlePool: []string{"The Skald of Ages", "Keeper of the Saga", "The Golden Herald"},
		NarrationStyle: `You are a high fantasy epic bard. Paint scenes with grandeur, heroic resonance, ancient oaths, and monumental stakes. Evoke legends of dragons, forgotten kings, and valorous endurance against the dark.`,
		OCEANBias: traits.OCEAN{
			Openness:          90.0,
			Conscientiousness: 75.0,
			Extraversion:      80.0,
			Agreeableness:     80.0,
			Neuroticism:       20.0,
		},
		BaseTemp: 0.75,
	},
	{
		Archetype: "Trickster Faerie",
		TitlePool: []string{"The Goblin Jester", "The Cheshire Watcher", "Lord of the Gilded Thorn"},
		NarrationStyle: `You are a mercurial, sly trickster referee. Narration is whimsical yet venomous, finding dark irony and bizarre humor in the dungeon's horrors. Tempt the player with mischievous curio and uncanny riddles.`,
		OCEANBias: traits.OCEAN{
			Openness:          95.0,
			Conscientiousness: 40.0,
			Extraversion:      75.0,
			Agreeableness:     50.0,
			Neuroticism:       35.0,
		},
		BaseTemp: 0.85,
	},
}

var firstNames = []string{
	"Malakor", "Valen", "Morwen", "Ignis", "Zephyr", "Cassian", "Elowen", "Vesper",
	"Kaelen", "Sylas", "Thalor", "Nyx", "Orion", "Baelor", "Astrid", "Gideon",
	"Rowan", "Fenris", "Corvus", "Seraphina",
}

var epithets = []string{
	"the Grim", "the Silent", "the Watcher", "the Merciless", "the Bleak", "the Ancient",
	"the Wandering", "the Ironborn", "the Veiled", "the Unseen", "the Astrologer", "the Cryptic",
}

// GenerateRandomDMConfig procedurally creates a personalized Dungeon Master persona.
func GenerateRandomDMConfig(seed ...int64) DMConfig {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(seed) > 0 {
		r = rand.New(rand.NewSource(seed[0]))
	}

	name := fmt.Sprintf("%s %s", firstNames[r.Intn(len(firstNames))], epithets[r.Intn(len(epithets))])
	arch := dmArchetypes[r.Intn(len(dmArchetypes))]
	title := arch.TitlePool[r.Intn(len(arch.TitlePool))]

	// Apply subtle random jitter to OCEAN traits (+/- 10)
	jitter := func(v float64) float64 {
		res := v + float64(r.Intn(21)-10)
		if res < 0 {
			return 0
		}
		if res > 100 {
			return 100
		}
		return res
	}

	pers := traits.OCEAN{
		Openness:          jitter(arch.OCEANBias.Openness),
		Conscientiousness: jitter(arch.OCEANBias.Conscientiousness),
		Extraversion:      jitter(arch.OCEANBias.Extraversion),
		Agreeableness:     jitter(arch.OCEANBias.Agreeableness),
		Neuroticism:       jitter(arch.OCEANBias.Neuroticism),
	}

	return DMConfig{
		Name:           name,
		Title:          title,
		Archetype:      arch.Archetype,
		NarrationStyle: arch.NarrationStyle,
		Personality:    pers,
		Temperature:    arch.BaseTemp,
	}
}

// Save writes the DM configuration to disk as JSON.
func (c *DMConfig) Save(filePath string) error {
	if filePath == "" {
		filePath = fmt.Sprintf("dm_%s.json", strings.ToLower(strings.ReplaceAll(c.Name, " ", "_")))
	}
	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

// LoadDMConfig reads a DM configuration from disk.
func LoadDMConfig(filePath string) (*DMConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg DMConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse DM config: %w", err)
	}

	if cfg.Name == "" {
		cfg.Name = "The Chronicler"
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.7
	}
	return &cfg, nil
}
