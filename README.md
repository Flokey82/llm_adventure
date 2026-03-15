# LLM Adventure

LLM Adventure is a text-based adventure game framework that leverages language models to generate dynamic narratives and interactions. The game is designed to be modular, extensible, and easy to integrate with AI systems for generating room descriptions, NPC dialogues, and more.

## Features

- **Dynamic Room Descriptions**: Rooms are described using AI-generated narratives based on their properties.
- **Persistent Player Notes**: The LLM maintains persistent notes about the player's state (e.g., "covered in poop", "smells like lavender") to ensure narrative continuity.
- **Interactive NPCs**: NPCs can respond to player input using AI-driven dialogue systems, and now possess persistent **hitpoints**, **memory**, and **world history**.
- **Combat & Life Mechanics**: Players can **attack** NPCs, potentially killing them. Dead NPCs remain in rooms as corpses and can sometimes be **resurrected**.
- **Selective Tool Exposure**: The LLM is dynamically presented only with tools that make sense for the current context (e.g., `talk_to` only appears if an NPC is present).
- **Dynamic Room Details**: Permanent narrative details added by the LLM (e.g., "stinky air") are visible in the TUI Room pane.
- **Flexible Game Logic**: Core mechanics like movement, item usage, and NPC interactions are implemented with extensibility in mind.
- **Text-Based UI**: Includes a terminal-based user interface with dedicated panes for narration, room info, inventory, and maps.
- **Integration with OpenAI**: Supports OpenAI's API for generating narratives and handling tool calls.

## TODO:

- [ ] Add more complex puzzles and item interactions.
- [x] Enhance the TUI with better formatting and user feedback.
- [x] Add more tools for AI to interact with the game state.
- [ ] Write comprehensive tests for game mechanics and AI integration.
- [x] Add more npcs and room types to create a richer game world.

## Project Structure

```
llm_adventure/
├── adventure/          # Core game logic
│   ├── adventure.go    # Main game mechanics
│   ├── client.go       # Defines the LLMClient interface
│   ├── executor.go     # Tool execution logic
│   ├── quickmatch.go   # Quick command parsing
│   ├── rooms.go        # Room and door definitions
│   ├── runner.go       # Interactive game loop
│   ├── save_load.go    # JSON serialization for game state
│   ├── tools.go        # Tool definitions for AI integration
│   ├── tui.go          # Text-based user interface
│   ├── tui_llm.go      # TUI with LLM integration
├── cmd/
│   └── llm_adventure/
│       └── main.go     # Entry point for the game
├── Makefile            # Build and Run scripts
├── go.mod              # Go module definition
├── LICENSE             # License file
```

## Getting Started

### Prerequisites

- Go 1.18 or later
- OpenAI API key (if using AI features)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/Flokey82/llm_adventure.git
   cd llm_adventure
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

### Running the Game

The project includes a `Makefile` for convenience. You can build and run the game directly using `make` commands.

To build the executable:
```bash
make build
```

To run the game with the TUI (default):
```bash
make run-tui
```

To run the alternative simple interactive mode:
```bash
make run-interactive
```

### Configuration

You can configure the game's LLM connection through command-line flags when running `main.go`.
Example:
```bash
go run cmd/llm_adventure/main.go -base-url="http://localhost:11434/v1" -model="granite4" tui-llm
```

You can also use custom system prompts by injecting custom AI functions in `cmd/llm_adventure/main.go` for generating room descriptions and NPC dialogues.

### Example Commands

- `look`: Examine the current room.
- `move <direction>`: Move to an adjacent room. (If no door exists, the game may dynamically generate a new area!)
- `open <direction>`: Open a door in the specified direction.
- `take <item>`: Pick up an item.
- `use <item> on <target>`: Use an item on a target.
- `attack <target>`: Attack an NPC or object.
- `talk <npc>`: Initiate a conversation with an NPC.
- `inventory`: View your inventory.
- `save`: Save the current game state.
- `load`: Load the last saved game state.

## Development

### Code Structure

- **Core Logic**: The `adventure` package contains the main game logic, including room definitions, NPC interactions, and game mechanics.
- **TUI**: The `tui.go` and `tui_llm.go` files implement the text-based user interface.
- **AI Integration**: The `tools.go` file defines tools that the AI can call to interact with the game state.

### Adding New Features

1. Define new game mechanics in the `adventure` package.
2. Update the TUI or AI integration as needed.
3. Test your changes by running the game.

### Testing

To run tests:
```bash
make test
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [OpenAI](https://openai.com) for providing the API used in this project.
- [tview](https://github.com/rivo/tview) for the text-based user interface library.

---

Enjoy your adventure!
