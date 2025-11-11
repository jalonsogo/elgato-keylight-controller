# Elgato Key Light Controller

A terminal-based application for controlling Elgato Key Lights, built with Go and Bubble Tea.

Supports both CLI commands for quick actions and an interactive TUI for detailed control.

## Features

- 🔍 Auto-discovery of Elgato Key Lights on your network
- 💡 Control multiple lights individually or all at once
- 🎚️ Adjust brightness (3-100%)
- 🌡️ Adjust color temperature (2900K-7000K)
- ⚡ CLI commands for quick control
- 🎨 Beautiful TUI with RGB gradient visualizations
- 🔄 Equalize settings across multiple lights

## Installation

### Prerequisites

- Go 1.24 or higher

### Build

```bash
go build -o keylight-go main.go
```

## Usage

### CLI Mode

Control your lights with simple commands:

```bash
# Turn lights on/off
keylight on                    # Turn on all lights
keylight off                   # Turn off all lights

# Brightness control
keylight bright +              # Increase brightness by 5%
keylight bright -              # Decrease brightness by 5%
keylight bright =              # Equalize brightness across all lights
keylight bright 50             # Set brightness to 50%

# Temperature control
keylight temp +                # Increase temperature by 200K
keylight temp -                # Decrease temperature by 200K
keylight temp =                # Equalize temperature across all lights
keylight temp 4000             # Set temperature to 4000K

# Information
keylight list                  # Show all configured lights
keylight detect                # Discover lights on network
keylight status                # Show status of all lights

# Control specific light
keylight "Elgato Key Light 1" on       # Turn on specific light
keylight "Elgato Key Light 1" bright 75  # Set specific light brightness
keylight 1 temp 3500           # Use index to control light

# Help
keylight help                  # Show all commands
```

### TUI Mode

Launch the interactive terminal UI:

```bash
keylight
```

On first run, it will automatically discover Elgato Key Lights on your network.

#### Controls

- **Arrow Keys**:
  - `↑`/`↓`: Navigate between control rows
  - `←`/`→`: Navigate between buttons or adjust sliders
- **Shortcuts**:
  - `a`: Select all lights
  - `1`/`2`: Select individual lights
  - `d`: Discover lights
  - `Enter`: Apply action
  - `q`: Quit

#### Features

- **Toggle**: Switch lights on/off
- **Turn Off/On**: Explicit power control
- **Brightness**: Adjust from 3% to 100% in 5% increments
- **Temperature**: Adjust from 2900K (warm) to 7000K (cool) in 200K steps

## Configuration

Settings are stored in `~/.config/keylight/config.json`:

```json
{
  "lights": {
    "Elgato Key Light 1": "192.168.1.100",
    "Elgato Key Light 2": "192.168.1.101"
  },
  "lastBrightness": 50,
  "lastTemperature": 4000
}
```

## Dependencies

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) - Terminal styling
- [Zeroconf](https://github.com/grandcat/zeroconf) - mDNS service discovery

## License

MIT
