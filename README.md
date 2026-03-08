# LLM Adventure

LLM Adventure is a text-based adventure game framework that leverages language models to generate dynamic narratives and interactions. The game is designed to be modular, extensible, and easy to integrate with AI systems for generating room descriptions, NPC dialogues, and more.

## Features

- **Dynamic Room Descriptions**: Rooms are described using AI-generated narratives based on their properties.
- **Interactive NPCs**: NPCs can respond to player input using AI-driven dialogue systems.
- **Flexible Game Logic**: Core mechanics like movement, item usage, and NPC interactions are implemented with extensibility in mind.
- **Text-Based UI**: Includes a terminal-based user interface for manual play.
- **Integration with OpenAI**: Supports OpenAI's API for generating narratives and handling tool calls.

## Project Structure

```
llm_adventure/
├── adventure/          # Core game logic
│   ├── adventure.go    # Main game mechanics
│   ├── executor.go     # Tool execution logic
│   ├── quickmatch.go   # Quick command parsing
│   ├── rooms.go        # Room and door definitions
│   ├── runner.go       # Interactive game loop
│   ├── tools.go        # Tool definitions for AI integration
│   ├── tui.go          # Text-based user interface
│   ├── tui_llm.go      # TUI with LLM integration
├── main.go             # Entry point for the game
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

To start the game, run:
```bash
go run main.go
```

### Configuration

You can configure the game by modifying the `Game` struct in `adventure/adventure.go`. For example, you can inject custom AI functions for generating room descriptions and NPC dialogues.

### Example Commands

- `look`: Examine the current room.
- `move <direction>`: Move to an adjacent room.
- `open <direction>`: Open a door in the specified direction.
- `take <item>`: Pick up an item.
- `use <item> on <target>`: Use an item on a target.
- `inventory`: View your inventory.

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
go test ./...
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [OpenAI](https://openai.com) for providing the API used in this project.
- [tview](https://github.com/rivo/tview) for the text-based user interface library.

---

Enjoy your adventure!
