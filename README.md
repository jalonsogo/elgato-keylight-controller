# Elgato Key Light Controller

A terminal-based UI application for controlling Elgato Key Lights, built with Go and Bubble Tea.

## Features

- 🔍 Auto-discovery of Elgato Key Lights on your network
- 💡 Control multiple lights individually or all at once
- 🎚️ Adjust brightness (3-100%)
- 🌡️ Adjust color temperature (2900K-7000K)
- 🎨 Beautiful TUI with RGB gradient visualizations
- ⚡ Fast and responsive controls

## Installation

### Prerequisites

- Go 1.24 or higher

### Build

```bash
go build -o keylight-go main.go
```

## Usage

Run the application:

```bash
./keylight-go
```

On first run, it will automatically discover Elgato Key Lights on your network.

### Controls

- **Arrow Keys**:
  - `↑`/`↓`: Navigate between control rows
  - `←`/`→`: Navigate between buttons or adjust sliders
- **Shortcuts**:
  - `a`: Select all lights
  - `1`/`2`: Select individual lights
  - `d`: Discover lights
  - `Enter`: Apply action
  - `q`: Quit

### Features

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
