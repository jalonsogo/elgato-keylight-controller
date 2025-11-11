package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/grandcat/zeroconf"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00"))

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2)

	buttonStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#666666")).
			Foreground(lipgloss.Color("#AAAAAA")).
			Padding(0, 1)

	buttonFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.ThickBorder()).
				BorderForeground(lipgloss.Color("#FFFFFF")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(0, 1)

	buttonActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.DoubleBorder()).
				BorderForeground(lipgloss.Color("#00FF00")).
				Background(lipgloss.Color("#003300")).
				Foreground(lipgloss.Color("#00FF00")).
				Bold(true).
				Padding(0, 1)
)

// Config structure
type Config struct {
	Lights             map[string]string `json:"lights"`
	LastBrightness     int               `json:"lastBrightness"`
	LastTemperature    int               `json:"lastTemperature"`
	LastSelectedLight  string            `json:"lastSelectedLight"`
}

// Light state
type LightState struct {
	On          int `json:"on"`
	Brightness  int `json:"brightness"`
	Temperature int `json:"temperature"`
}

type LightsResponse struct {
	Lights []LightState `json:"lights"`
}

// Light selection mode
type lightMode int

const (
	allLights lightMode = iota
	light1
	light2
)

// Control focus
type controlFocus int

const (
	focusToggle controlFocus = iota
	focusTurnOff
	focusTurnOn
	focusBrightness
	focusTemperature
)

// Model
type model struct {
	config              *Config
	lights              map[string]string
	lightsList          []string // ordered list of light names
	selectedLightMode   lightMode
	focusedControl      controlFocus
	brightnessValue     int
	temperatureValue    int
	message             string
	quitting            bool
}

func initialModel() model {
	config := loadConfig()
	if config.Lights == nil {
		config.Lights = make(map[string]string)
	}

	// Create ordered list of lights
	lightsList := make([]string, 0, len(config.Lights))
	for name := range config.Lights {
		lightsList = append(lightsList, name)
	}

	// Set defaults if not configured
	if config.LastBrightness == 0 {
		config.LastBrightness = 50
	}
	if config.LastTemperature == 0 {
		config.LastTemperature = 4000
	}

	return model{
		config:              config,
		lights:              config.Lights,
		lightsList:          lightsList,
		selectedLightMode:   allLights,
		focusedControl:      focusToggle,
		brightnessValue:     config.LastBrightness,
		temperatureValue:    config.LastTemperature,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Light selection shortcuts
		switch msg.String() {
		case "a":
			m.selectedLightMode = allLights
			m.message = "✓ Controlling all lights"
			return m, nil
		case "1":
			if len(m.lightsList) >= 1 {
				m.selectedLightMode = light1
				m.message = fmt.Sprintf("✓ Controlling %s", m.lightsList[0])
			}
			return m, nil
		case "2":
			if len(m.lightsList) >= 2 {
				m.selectedLightMode = light2
				m.message = fmt.Sprintf("✓ Controlling %s", m.lightsList[1])
			}
			return m, nil
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "d":
			m.message = "Discovering lights..."
			go discoverLights(&m)
			return m, nil
		}

		// Normal navigation
		switch msg.String() {
		case "up", "k":
			// Move up through control groups
			if m.focusedControl == focusBrightness {
				m.focusedControl = focusToggle // Go to action buttons
			} else if m.focusedControl == focusTemperature {
				m.focusedControl = focusBrightness
			}
		case "down", "j":
			// Move down through control groups
			if m.focusedControl <= focusTurnOn {
				m.focusedControl = focusBrightness // From action buttons to brightness
			} else if m.focusedControl == focusBrightness {
				m.focusedControl = focusTemperature
			}
		case "left", "h":
			// Navigate between action buttons or adjust sliders
			if m.focusedControl == focusTurnOff {
				m.focusedControl = focusToggle
			} else if m.focusedControl == focusTurnOn {
				m.focusedControl = focusTurnOff
			} else if m.focusedControl == focusBrightness {
				// Adjust brightness
				m.brightnessValue -= 5
				if m.brightnessValue < 3 {
					m.brightnessValue = 3
				}
			} else if m.focusedControl == focusTemperature {
				// Adjust temperature
				m.temperatureValue -= 200
				if m.temperatureValue < 2900 {
					m.temperatureValue = 2900
				}
			}
		case "right", "l":
			// Navigate between action buttons or adjust sliders
			if m.focusedControl == focusToggle {
				m.focusedControl = focusTurnOff
			} else if m.focusedControl == focusTurnOff {
				m.focusedControl = focusTurnOn
			} else if m.focusedControl == focusBrightness {
				// Adjust brightness
				m.brightnessValue += 5
				if m.brightnessValue > 100 {
					m.brightnessValue = 100
				}
			} else if m.focusedControl == focusTemperature {
				// Adjust temperature
				m.temperatureValue += 200
				if m.temperatureValue > 7000 {
					m.temperatureValue = 7000
				}
			}
		case "enter", " ":
			return m.activateControl()
		}
	}

	return m, nil
}

func (m model) activateControl() (tea.Model, tea.Cmd) {
	switch m.focusedControl {
	case focusToggle:
		return m.toggleLights()
	case focusTurnOff:
		// Turn off selected lights
		ips := m.getSelectedLightIPs()
		offState := 0
		for _, ip := range ips {
			setLight(ip, &offState, nil, nil)
		}
		m.message = "✓ Lights turned off"
		return m, nil
	case focusTurnOn:
		// Turn on selected lights
		ips := m.getSelectedLightIPs()
		onState := 1
		for _, ip := range ips {
			setLight(ip, &onState, nil, nil)
		}
		m.message = "✓ Lights turned on"
		return m, nil
	case focusBrightness:
		// Apply brightness to selected lights
		ips := m.getSelectedLightIPs()
		success := true
		for _, ip := range ips {
			if err := setLight(ip, nil, &m.brightnessValue, nil); err != nil {
				m.message = fmt.Sprintf("✗ Error setting brightness")
				success = false
				break
			}
		}
		if success {
			m.config.LastBrightness = m.brightnessValue
			saveConfig(m.config)
			m.message = fmt.Sprintf("✓ Brightness set to %d%%", m.brightnessValue)
		}
		return m, nil
	case focusTemperature:
		// Apply temperature to selected lights
		ips := m.getSelectedLightIPs()
		success := true
		for _, ip := range ips {
			if err := setLight(ip, nil, nil, &m.temperatureValue); err != nil {
				m.message = fmt.Sprintf("✗ Error setting temperature")
				success = false
				break
			}
		}
		if success {
			m.config.LastTemperature = m.temperatureValue
			saveConfig(m.config)
			m.message = fmt.Sprintf("✓ Temperature set to %dK", m.temperatureValue)
		}
		return m, nil
	}
	return m, nil
}

func (m model) toggleLights() (tea.Model, tea.Cmd) {
	ips := m.getSelectedLightIPs()
	errorCount := 0
	successCount := 0

	for _, ip := range ips {
		if err := toggleLight(ip); err != nil {
			errorCount++
		} else {
			successCount++
		}
	}

	if errorCount == 0 && successCount > 0 {
		m.message = fmt.Sprintf("✓ %d light(s) toggled", successCount)
	} else if errorCount > 0 {
		m.message = fmt.Sprintf("✗ Error toggling lights")
	}

	return m, nil
}

func (m model) getSelectedLightIPs() []string {
	var ips []string

	switch m.selectedLightMode {
	case allLights:
		for _, ip := range m.lights {
			ips = append(ips, ip)
		}
	case light1:
		if len(m.lightsList) >= 1 {
			ips = append(ips, m.lights[m.lightsList[0]])
		}
	case light2:
		if len(m.lightsList) >= 2 {
			ips = append(ips, m.lights[m.lightsList[1]])
		}
	}

	return ips
}

func (m model) View() string {
	if m.quitting {
		return boxStyle.Render("Goodbye!\n")
	}

	return m.renderUnifiedView()
}

func (m model) renderUnifiedView() string {
	width := 97  // Content width inside box

	// Helper to create separator
	separator := func() string {
		line := ""
		for i := 0; i < width; i++ {
			line += "─"
		}
		return dimStyle.Render(line)
	}

	var content string
	content += "\n"  // Top padding

	// Title line with version and discover button
	titleLeft := "Control Elgato Lights  v0.9.1"
	titleRight := "(d Detect Lights)"
	padding := width - len(titleLeft) - len(titleRight)
	titleLine := titleLeft + lipgloss.NewStyle().Width(padding).Render("") + titleRight
	content += titleLine + "\n\n"

	content += separator() + "\n\n"

	// Light selection box
	content += m.renderLightSelectionBox() + "\n"

	content += separator() + "\n\n"

	// Control tools box
	content += m.renderControlsBox() + "\n"

	content += separator() + "\n\n"

	// Help
	help := dimStyle.Render("↑/↓: navigate rows • ←/→: buttons/adjust • Enter: apply • a: all • 1/2: select • d: discover • q: quit")
	content += help + "\n"

	// Message
	if m.message != "" {
		content += successStyle.Render(m.message) + "\n"
	}

	content += "\n"  // Bottom padding

	return boxStyle.Render(content)
}

func (m model) renderLightSelectionBox() string {
	var content string

	// Check how many lights are on for "All Lights" indicator
	lightsOn := 0
	totalLights := len(m.lightsList)

	for _, name := range m.lightsList {
		ip := m.lights[name]
		state, err := getLightState(ip)
		if err == nil && state.On == 1 {
			lightsOn++
		}
	}

	// All Lights indicator
	var allIndicator string
	if lightsOn == 0 {
		allIndicator = "○" // All off
	} else if lightsOn == totalLights {
		allIndicator = "●" // All on
	} else {
		allIndicator = "◐" // Partial
	}

	// Selection arrow
	allArrow := "▹ "
	if m.selectedLightMode == allLights {
		allArrow = "▶ "
	}

	// Style based on state
	allLineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	if lightsOn == 0 {
		allLineStyle = dimStyle
	}

	content += allArrow + allLineStyle.Render(fmt.Sprintf("%s - (a) All Lights", allIndicator)) + "\n"

	// Individual lights - show arrow when selected OR when All is selected
	for i, name := range m.lightsList {
		ip := m.lights[name]
		state, err := getLightState(ip)

		var indicator string
		var statusText string
		var lineStyle lipgloss.Style

		if err == nil {
			if state.On == 1 {
				indicator = "●"
				statusText = fmt.Sprintf("On / %d%% / %dK", state.Brightness, int(1000000/state.Temperature))
				// Bright white for on lights
				lineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
			} else {
				indicator = "○"
				statusText = fmt.Sprintf("Off / %d%% / %dK", state.Brightness, int(1000000/state.Temperature))
				// Dimmed for off lights
				lineStyle = dimStyle
			}
		} else {
			indicator = "○"
			statusText = "Offline"
			lineStyle = dimStyle
		}

		// Show arrow when this light is selected OR when All is selected
		arrow := "  "
		if m.selectedLightMode == allLights ||
		   (i == 0 && m.selectedLightMode == light1) ||
		   (i == 1 && m.selectedLightMode == light2) {
			arrow = "▶ "
		}

		content += arrow + lineStyle.Render(fmt.Sprintf("%s - (%d) %s (%s)", indicator, i+1, name, statusText)) + "\n"
	}

	return content
}

func (m model) renderControlsBox() string {
	var content string

	// Get scope text
	scopeText := "All"
	if m.selectedLightMode == light1 && len(m.lightsList) >= 1 {
		scopeText = m.lightsList[0]
		if len(scopeText) > 15 {
			scopeText = scopeText[:12] + "..."
		}
	} else if m.selectedLightMode == light2 && len(m.lightsList) >= 2 {
		scopeText = m.lightsList[1]
		if len(scopeText) > 15 {
			scopeText = scopeText[:12] + "..."
		}
	}

	// Action buttons on same line - use JoinHorizontal for proper alignment
	// Toggle button - uses thick border when focused
	toggleText := fmt.Sprintf(" TOGGLE %s ", scopeText)
	var toggleBtn string
	if m.focusedControl == focusToggle {
		toggleBtn = buttonFocusedStyle.Render(toggleText)
	} else {
		toggleBtn = buttonStyle.Render(toggleText)
	}

	// Turn Off button - uses thick border when focused
	var turnOffBtn string
	if m.focusedControl == focusTurnOff {
		turnOffBtn = buttonFocusedStyle.Render(" TURN OFF All ")
	} else {
		turnOffBtn = buttonStyle.Render(" TURN OFF All ")
	}

	// Turn On button - uses thick border when focused
	var turnOnBtn string
	if m.focusedControl == focusTurnOn {
		turnOnBtn = buttonFocusedStyle.Render(" TURN ON ALL ")
	} else {
		turnOnBtn = buttonStyle.Render(" TURN ON ALL ")
	}

	// Join buttons horizontally to form a row
	buttonsRow := lipgloss.JoinHorizontal(lipgloss.Top, toggleBtn, " ", turnOffBtn, " ", turnOnBtn)
	content += buttonsRow + "\n"

	// Brightness control
	content += m.renderBrightnessControl() + "\n"

	// Temperature control
	content += m.renderTemperatureControl() + "\n"

	return content
}

func (m model) renderControl(label string, focus controlFocus) string {
	if m.focusedControl == focus {
		return buttonFocusedStyle.Render(label)
	}
	return buttonStyle.Render(label)
}

func (m model) renderBrightnessControl() string {
	barWidth := 50
	percentage := float64(m.brightnessValue-3) / float64(100-3)
	filled := int(percentage * float64(barWidth))

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			// Brightness gradient (dark gray to bright white)
			brightness := int(50 + (float64(i)/float64(barWidth))*205)
			color := fmt.Sprintf("#%02x%02x%02x", brightness, brightness, brightness)
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("█")
		} else {
			bar += dimStyle.Render("░")
		}
	}

	var btnLabel string
	if m.focusedControl == focusBrightness {
		btnLabel = buttonFocusedStyle.Render("   Brightness   ")
	} else {
		btnLabel = buttonStyle.Render("   Brightness   ")
	}

	valueStr := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true).
		Render(fmt.Sprintf("%d%%", m.brightnessValue))

	// Create the bar and value as a single line
	barAndValue := "   " + bar + "      " + valueStr

	// Use JoinHorizontal to align button and bar vertically centered
	return lipgloss.JoinHorizontal(lipgloss.Center, btnLabel, barAndValue)
}

func (m model) renderTemperatureControl() string {
	barWidth := 50
	percentage := float64(m.temperatureValue-2900) / float64(7000-2900)
	filled := int(percentage * float64(barWidth))

	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			// Temperature gradient (warm orange to cool blue)
			tempPercent := float64(i) / float64(barWidth)
			r := int(255 - (tempPercent * 100))
			g := int(180 - (tempPercent * 50))
			b := int(100 + (tempPercent * 155))
			color := fmt.Sprintf("#%02x%02x%02x", r, g, b)
			bar += lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("█")
		} else {
			bar += dimStyle.Render("░")
		}
	}

	var btnLabel string
	if m.focusedControl == focusTemperature {
		btnLabel = buttonFocusedStyle.Render("   Temperature  ")
	} else {
		btnLabel = buttonStyle.Render("   Temperature  ")
	}

	valueStr := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00FF00")).
		Bold(true).
		Render(fmt.Sprintf("%dK", m.temperatureValue))

	// Create the bar and value as a single line
	barAndValue := "   " + bar + "      " + valueStr

	// Use JoinHorizontal to align button and bar vertically centered
	return lipgloss.JoinHorizontal(lipgloss.Center, btnLabel, barAndValue)
}


// Config management
func getConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "keylight", "config.json")
}

func loadConfig() *Config {
	configPath := getConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return &Config{
			Lights:          make(map[string]string),
			LastBrightness:  50,
			LastTemperature: 4000,
		}
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return &Config{
			Lights:          make(map[string]string),
			LastBrightness:  50,
			LastTemperature: 4000,
		}
	}

	return &config
}

func saveConfig(config *Config) {
	configPath := getConfigPath()
	os.MkdirAll(filepath.Dir(configPath), 0755)

	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configPath, data, 0644)
}

// Discovery
func discoverLights(m *model) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		m.message = "Error: Failed to create resolver"
		return
	}

	entries := make(chan *zeroconf.ServiceEntry)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	err = resolver.Browse(ctx, "_elg._tcp", "local.", entries)
	if err != nil {
		m.message = "Error: Failed to discover"
		return
	}

	discovered := make(map[string]string)
	go func() {
		for entry := range entries {
			if len(entry.AddrIPv4) > 0 {
				name := entry.Instance
				ip := entry.AddrIPv4[0].String()
				discovered[name] = ip
			}
		}
	}()

	<-ctx.Done()

	if len(discovered) > 0 {
		m.config.Lights = discovered
		m.lights = discovered
		saveConfig(m.config)
		m.message = fmt.Sprintf("✓ Discovered %d light(s)", len(discovered))
	} else {
		m.message = "⚠ No lights found"
	}
}

// API Functions
func getLightState(ip string) (*LightState, error) {
	resp, err := http.Get(fmt.Sprintf("http://%s:9123/elgato/lights", ip))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lightsResp LightsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lightsResp); err != nil {
		return nil, err
	}

	if len(lightsResp.Lights) > 0 {
		return &lightsResp.Lights[0], nil
	}
	return nil, fmt.Errorf("no lights in response")
}

func setLight(ip string, on *int, brightness *int, temperature *int) error {
	payload := make(map[string]interface{})
	lights := make([]map[string]interface{}, 1)
	light := make(map[string]interface{})

	if on != nil {
		light["on"] = *on
	}
	if brightness != nil {
		light["brightness"] = *brightness
	}
	if temperature != nil {
		// Convert from Kelvin to Elgato scale (inverted: 7000K=143, 2900K=344)
		elgatoTemp := int(1000000 / *temperature)
		light["temperature"] = elgatoTemp
	}

	lights[0] = light
	payload["lights"] = lights

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s:9123/elgato/lights", ip), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

func toggleLight(ip string) error {
	state, err := getLightState(ip)
	if err != nil {
		// If we can't get state, just try to turn on
		newState := 1
		return setLight(ip, &newState, nil, nil)
	}

	newState := 0
	if state.On == 0 {
		newState = 1
	}

	return setLight(ip, &newState, nil, nil)
}

func main() {
	// Check if lights are configured
	config := loadConfig()
	if len(config.Lights) == 0 {
		fmt.Println("No lights configured. Running discovery...")
		fmt.Println("Please wait...")

		// Run discovery in blocking mode on first run
		resolver, err := zeroconf.NewResolver(nil)
		if err != nil {
			fmt.Println("Error: Failed to create resolver")
			os.Exit(1)
		}

		entries := make(chan *zeroconf.ServiceEntry)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()

		err = resolver.Browse(ctx, "_elg._tcp", "local.", entries)
		if err != nil {
			fmt.Println("Error: Failed to discover")
			os.Exit(1)
		}

		discovered := make(map[string]string)
		go func() {
			for entry := range entries {
				if len(entry.AddrIPv4) > 0 {
					name := entry.Instance
					ip := entry.AddrIPv4[0].String()
					discovered[name] = ip
					fmt.Printf("Found: %s at %s\n", name, ip)
				}
			}
		}()

		<-ctx.Done()

		if len(discovered) == 0 {
			fmt.Println("No lights found. Make sure they are powered on.")
			os.Exit(1)
		}

		config.Lights = discovered
		saveConfig(config)
		fmt.Printf("\n✓ Discovered %d light(s)\n\n", len(discovered))
	}

	// Start TUI
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
