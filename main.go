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
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:9123/elgato/lights", ip))
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

	client := &http.Client{Timeout: 2 * time.Second}
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

// Fast toggle without status check - for quick button presses
func toggleLightFast(ip string) error {
	// Retry up to 3 times for Loupedeck/automation reliability
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		err := toggleLightAttempt(ip)
		if err == nil {
			return nil
		}
		lastErr = err

		// Small delay before retry (only if not last attempt)
		if attempt < 3 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return fmt.Errorf("failed after 3 attempts: %w", lastErr)
}

func toggleLightAttempt(ip string) error {
	// Get current state quickly with 2 second timeout for reliability
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://%s:9123/elgato/lights", ip))
	if err != nil {
		return fmt.Errorf("failed to get light state: %w", err)
	}
	defer resp.Body.Close()

	var lightsResp LightsResponse
	if err := json.NewDecoder(resp.Body).Decode(&lightsResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(lightsResp.Lights) == 0 {
		return fmt.Errorf("no lights in response")
	}

	// Toggle
	newState := 0
	if lightsResp.Lights[0].On == 0 {
		newState = 1
	}

	// Build and send request inline for speed
	payload := map[string]interface{}{
		"lights": []map[string]interface{}{
			{"on": newState},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	req, err := http.NewRequest("PUT", fmt.Sprintf("http://%s:9123/elgato/lights", ip), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp2, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send toggle request: %w", err)
	}
	defer resp2.Body.Close()

	if resp2.StatusCode != 200 {
		return fmt.Errorf("API returned status %d", resp2.StatusCode)
	}

	return nil
}

func main() {
	// Check if CLI command is provided
	if len(os.Args) > 1 {
		handleCLI()
		return
	}

	// No CLI args - run TUI
	runTUI()
}

func runTUI() {
	// Check if lights are configured
	config := loadConfig()
	if len(config.Lights) == 0 {
		fmt.Println("No lights configured. Running discovery...")
		fmt.Println("Please wait...")

		// Run discovery in blocking mode on first run
		discovered := runDiscovery()
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

func runDiscovery() map[string]string {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		fmt.Println("Error: Failed to create resolver")
		return nil
	}

	entries := make(chan *zeroconf.ServiceEntry)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	err = resolver.Browse(ctx, "_elg._tcp", "local.", entries)
	if err != nil {
		fmt.Println("Error: Failed to discover")
		return nil
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
	return discovered
}

func handleCLI() {
	config := loadConfig()

	// Check if lights are configured
	if len(config.Lights) == 0 && os.Args[1] != "detect" && os.Args[1] != "help" {
		fmt.Println("No lights configured. Please run: keylight detect")
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "on":
		cliTurnOn(config)
	case "off":
		cliTurnOff(config)
	case "bright":
		cliBrightness(config)
	case "temp":
		cliTemperature(config)
	case "list":
		cliList(config)
	case "detect":
		cliDetect()
	case "status":
		cliStatus(config)
	case "help":
		cliHelp()
	default:
		// Check if it's a light name or ID
		cliSpecificLight(config, command)
	}
}

// CLI Commands

func cliTurnOn(config *Config) {
	// Use goroutines for parallel execution
	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(config.Lights))

	for name, ip := range config.Lights {
		go func(n, i string) {
			onState := 1
			err := setLight(i, &onState, nil, nil)
			results <- result{name: n, err: err}
		}(name, ip)
	}

	// Collect results
	for i := 0; i < len(config.Lights); i++ {
		r := <-results
		if r.err != nil {
			fmt.Printf("✗ Failed to turn on %s\n", r.name)
		} else {
			fmt.Printf("✓ Turned on %s\n", r.name)
		}
	}
}

func cliTurnOff(config *Config) {
	// Use goroutines for parallel execution
	type result struct {
		name string
		err  error
	}
	results := make(chan result, len(config.Lights))

	for name, ip := range config.Lights {
		go func(n, i string) {
			offState := 0
			err := setLight(i, &offState, nil, nil)
			results <- result{name: n, err: err}
		}(name, ip)
	}

	// Collect results
	for i := 0; i < len(config.Lights); i++ {
		r := <-results
		if r.err != nil {
			fmt.Printf("✗ Failed to turn off %s\n", r.name)
		} else {
			fmt.Printf("✓ Turned off %s\n", r.name)
		}
	}
}

func cliBrightness(config *Config) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: keylight bright [+|-|=|value]")
		os.Exit(1)
	}

	action := os.Args[2]

	switch action {
	case "+":
		// Increase brightness by 5%
		for name, ip := range config.Lights {
			state, err := getLightState(ip)
			if err != nil {
				fmt.Printf("✗ Failed to get state for %s\n", name)
				continue
			}
			newBright := state.Brightness + 5
			if newBright > 100 {
				newBright = 100
			}
			if err := setLight(ip, nil, &newBright, nil); err != nil {
				fmt.Printf("✗ Failed to adjust %s\n", name)
			} else {
				fmt.Printf("✓ %s brightness: %d%%\n", name, newBright)
			}
		}
	case "-":
		// Decrease brightness by 5%
		for name, ip := range config.Lights {
			state, err := getLightState(ip)
			if err != nil {
				fmt.Printf("✗ Failed to get state for %s\n", name)
				continue
			}
			newBright := state.Brightness - 5
			if newBright < 3 {
				newBright = 3
			}
			if err := setLight(ip, nil, &newBright, nil); err != nil {
				fmt.Printf("✗ Failed to adjust %s\n", name)
			} else {
				fmt.Printf("✓ %s brightness: %d%%\n", name, newBright)
			}
		}
	case "=":
		// Equalize all lights to the average brightness
		totalBright := 0
		count := 0
		for _, ip := range config.Lights {
			state, err := getLightState(ip)
			if err == nil {
				totalBright += state.Brightness
				count++
			}
		}
		if count == 0 {
			fmt.Println("✗ Could not read any lights")
			return
		}
		avgBright := totalBright / count
		fmt.Printf("Setting all lights to %d%%\n", avgBright)
		for name, ip := range config.Lights {
			if err := setLight(ip, nil, &avgBright, nil); err != nil {
				fmt.Printf("✗ Failed to set %s\n", name)
			} else {
				fmt.Printf("✓ %s brightness: %d%%\n", name, avgBright)
			}
		}
	default:
		// Set specific value
		var brightness int
		n, err := fmt.Sscanf(action, "%d", &brightness)
		if n == 1 && err == nil {
			if brightness < 3 || brightness > 100 {
				fmt.Println("Brightness must be between 3 and 100")
				os.Exit(1)
			}
			for name, ip := range config.Lights {
				if err := setLight(ip, nil, &brightness, nil); err != nil {
					fmt.Printf("✗ Failed to set %s\n", name)
				} else {
					fmt.Printf("✓ %s brightness: %d%%\n", name, brightness)
				}
			}
			config.LastBrightness = brightness
			saveConfig(config)
		} else {
			fmt.Println("Invalid brightness value")
			os.Exit(1)
		}
	}
}

func cliTemperature(config *Config) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: keylight temp [+|-|=|value]")
		os.Exit(1)
	}

	action := os.Args[2]

	switch action {
	case "+":
		// Increase temperature by 200K
		for name, ip := range config.Lights {
			state, err := getLightState(ip)
			if err != nil {
				fmt.Printf("✗ Failed to get state for %s\n", name)
				continue
			}
			currentTemp := int(1000000 / state.Temperature)
			newTemp := currentTemp + 200
			if newTemp > 7000 {
				newTemp = 7000
			}
			if err := setLight(ip, nil, nil, &newTemp); err != nil {
				fmt.Printf("✗ Failed to adjust %s\n", name)
			} else {
				fmt.Printf("✓ %s temperature: %dK\n", name, newTemp)
			}
		}
	case "-":
		// Decrease temperature by 200K
		for name, ip := range config.Lights {
			state, err := getLightState(ip)
			if err != nil {
				fmt.Printf("✗ Failed to get state for %s\n", name)
				continue
			}
			currentTemp := int(1000000 / state.Temperature)
			newTemp := currentTemp - 200
			if newTemp < 2900 {
				newTemp = 2900
			}
			if err := setLight(ip, nil, nil, &newTemp); err != nil {
				fmt.Printf("✗ Failed to adjust %s\n", name)
			} else {
				fmt.Printf("✓ %s temperature: %dK\n", name, newTemp)
			}
		}
	case "=":
		// Equalize all lights to the average temperature
		totalTemp := 0
		count := 0
		for _, ip := range config.Lights {
			state, err := getLightState(ip)
			if err == nil {
				totalTemp += int(1000000 / state.Temperature)
				count++
			}
		}
		if count == 0 {
			fmt.Println("✗ Could not read any lights")
			return
		}
		avgTemp := totalTemp / count
		fmt.Printf("Setting all lights to %dK\n", avgTemp)
		for name, ip := range config.Lights {
			if err := setLight(ip, nil, nil, &avgTemp); err != nil {
				fmt.Printf("✗ Failed to set %s\n", name)
			} else {
				fmt.Printf("✓ %s temperature: %dK\n", name, avgTemp)
			}
		}
	default:
		// Set specific value
		var temperature int
		n, err := fmt.Sscanf(action, "%d", &temperature)
		if n == 1 && err == nil {
			if temperature < 2900 || temperature > 7000 {
				fmt.Println("Temperature must be between 2900K and 7000K")
				os.Exit(1)
			}
			for name, ip := range config.Lights {
				if err := setLight(ip, nil, nil, &temperature); err != nil {
					fmt.Printf("✗ Failed to set %s\n", name)
				} else {
					fmt.Printf("✓ %s temperature: %dK\n", name, temperature)
				}
			}
			config.LastTemperature = temperature
			saveConfig(config)
		} else {
			fmt.Println("Invalid temperature value")
			os.Exit(1)
		}
	}
}

func cliList(config *Config) {
	if len(config.Lights) == 0 {
		fmt.Println("No lights configured. Run: keylight detect")
		return
	}

	fmt.Println("Configured lights:")
	i := 1
	for name, ip := range config.Lights {
		fmt.Printf("  %d. %s (%s)\n", i, name, ip)
		i++
	}
}

func cliDetect() {
	fmt.Println("Discovering lights...")
	discovered := runDiscovery()

	if len(discovered) == 0 {
		fmt.Println("✗ No lights found")
		return
	}

	config := loadConfig()
	config.Lights = discovered
	saveConfig(config)
	fmt.Printf("\n✓ Discovered %d light(s)\n", len(discovered))
}

func cliStatus(config *Config) {
	if len(config.Lights) == 0 {
		fmt.Println("No lights configured. Run: keylight detect")
		return
	}

	fmt.Println("Light status:")
	for name, ip := range config.Lights {
		state, err := getLightState(ip)
		if err != nil {
			fmt.Printf("  %s: Offline\n", name)
			continue
		}

		status := "Off"
		if state.On == 1 {
			status = "On"
		}
		temp := int(1000000 / state.Temperature)
		fmt.Printf("  %s: %s | Brightness: %d%% | Temperature: %dK\n", name, status, state.Brightness, temp)
	}
}

func cliHelp() {
	help := `Elgato Key Light Controller

USAGE:
  keylight                    Open TUI interface
  keylight [command] [args]   Run CLI command

COMMANDS:
  on                          Turn on all lights
  off                         Turn off all lights

  bright +                    Increase brightness by 5%
  bright -                    Decrease brightness by 5%
  bright =                    Equalize brightness across all lights
  bright <value>              Set brightness to specific value (3-100)

  temp +                      Increase temperature by 200K
  temp -                      Decrease temperature by 200K
  temp =                      Equalize temperature across all lights
  temp <value>                Set temperature to specific value (2900-7000)

  list                        Show all configured lights
  detect                      Discover lights on network
  status                      Show status of all lights

  <light_name|index>          Toggle specific light
  <light_name> <command>      Control specific light
                              Commands: on, off, bright [+|-|value], temp [+|-|value]

  help                        Show this help message

EXAMPLES:
  keylight on                 Turn on all lights
  keylight bright 50          Set all lights to 50% brightness
  keylight temp 4000          Set all lights to 4000K
  keylight bright =           Match brightness across all lights
  keylight 1                  Toggle light 1
  keylight 2 bright +         Increase light 2 brightness
  keylight "My Light" on      Turn on specific light
  keylight status             Check status of all lights
`
	fmt.Println(help)
}

func cliSpecificLight(config *Config, lightIdentifier string) {
	// Try to find light by name or index
	var targetIP string
	var targetName string

	// Check if it's a numeric index
	var index int
	n, _ := fmt.Sscanf(lightIdentifier, "%d", &index)
	if n == 1 && index > 0 {
		// Find light by index
		i := 1
		for name, ip := range config.Lights {
			if i == index {
				targetIP = ip
				targetName = name
				break
			}
			i++
		}
	} else {
		// Find light by name
		for name, ip := range config.Lights {
			if name == lightIdentifier {
				targetIP = ip
				targetName = name
				break
			}
		}
	}

	if targetIP == "" {
		fmt.Printf("✗ Light '%s' not found. Use 'keylight list' to see available lights.\n", lightIdentifier)
		os.Exit(1)
	}

	// If no command specified, toggle the light (fast mode)
	if len(os.Args) < 3 {
		if err := toggleLightFast(targetIP); err != nil {
			fmt.Printf("✗ Failed to toggle %s: %v\n", targetName, err)
			os.Exit(1)
		} else {
			fmt.Printf("✓ Toggled %s\n", targetName)
		}
		return
	}

	command := os.Args[2]

	switch command {
	case "on":
		onState := 1
		if err := setLight(targetIP, &onState, nil, nil); err != nil {
			fmt.Printf("✗ Failed to turn on %s\n", targetName)
		} else {
			fmt.Printf("✓ Turned on %s\n", targetName)
		}
	case "off":
		offState := 0
		if err := setLight(targetIP, &offState, nil, nil); err != nil {
			fmt.Printf("✗ Failed to turn off %s\n", targetName)
		} else {
			fmt.Printf("✓ Turned off %s\n", targetName)
		}
	case "bright":
		if len(os.Args) < 4 {
			fmt.Println("Usage: keylight <light> bright [+|-|value]")
			os.Exit(1)
		}
		action := os.Args[3]
		switch action {
		case "+":
			state, err := getLightState(targetIP)
			if err != nil {
				fmt.Printf("✗ Failed to get state for %s\n", targetName)
				os.Exit(1)
			}
			newBright := state.Brightness + 5
			if newBright > 100 {
				newBright = 100
			}
			if err := setLight(targetIP, nil, &newBright, nil); err != nil {
				fmt.Printf("✗ Failed to adjust %s\n", targetName)
			} else {
				fmt.Printf("✓ %s brightness: %d%%\n", targetName, newBright)
			}
		case "-":
			state, err := getLightState(targetIP)
			if err != nil {
				fmt.Printf("✗ Failed to get state for %s\n", targetName)
				os.Exit(1)
			}
			newBright := state.Brightness - 5
			if newBright < 3 {
				newBright = 3
			}
			if err := setLight(targetIP, nil, &newBright, nil); err != nil {
				fmt.Printf("✗ Failed to adjust %s\n", targetName)
			} else {
				fmt.Printf("✓ %s brightness: %d%%\n", targetName, newBright)
			}
		default:
			var brightness int
			n, err := fmt.Sscanf(action, "%d", &brightness)
			if n == 1 && err == nil {
				if brightness < 3 || brightness > 100 {
					fmt.Println("Brightness must be between 3 and 100")
					os.Exit(1)
				}
				if err := setLight(targetIP, nil, &brightness, nil); err != nil {
					fmt.Printf("✗ Failed to set brightness for %s\n", targetName)
				} else {
					fmt.Printf("✓ %s brightness: %d%%\n", targetName, brightness)
				}
			} else {
				fmt.Println("Invalid brightness value")
				os.Exit(1)
			}
		}
	case "temp":
		if len(os.Args) < 4 {
			fmt.Println("Usage: keylight <light> temp [+|-|value]")
			os.Exit(1)
		}
		action := os.Args[3]
		switch action {
		case "+":
			state, err := getLightState(targetIP)
			if err != nil {
				fmt.Printf("✗ Failed to get state for %s\n", targetName)
				os.Exit(1)
			}
			currentTemp := int(1000000 / state.Temperature)
			newTemp := currentTemp + 200
			if newTemp > 7000 {
				newTemp = 7000
			}
			if err := setLight(targetIP, nil, nil, &newTemp); err != nil {
				fmt.Printf("✗ Failed to adjust %s\n", targetName)
			} else {
				fmt.Printf("✓ %s temperature: %dK\n", targetName, newTemp)
			}
		case "-":
			state, err := getLightState(targetIP)
			if err != nil {
				fmt.Printf("✗ Failed to get state for %s\n", targetName)
				os.Exit(1)
			}
			currentTemp := int(1000000 / state.Temperature)
			newTemp := currentTemp - 200
			if newTemp < 2900 {
				newTemp = 2900
			}
			if err := setLight(targetIP, nil, nil, &newTemp); err != nil {
				fmt.Printf("✗ Failed to adjust %s\n", targetName)
			} else {
				fmt.Printf("✓ %s temperature: %dK\n", targetName, newTemp)
			}
		default:
			var temperature int
			n, err := fmt.Sscanf(action, "%d", &temperature)
			if n == 1 && err == nil {
				if temperature < 2900 || temperature > 7000 {
					fmt.Println("Temperature must be between 2900K and 7000K")
					os.Exit(1)
				}
				if err := setLight(targetIP, nil, nil, &temperature); err != nil {
					fmt.Printf("✗ Failed to set temperature for %s\n", targetName)
				} else {
					fmt.Printf("✓ %s temperature: %dK\n", targetName, temperature)
				}
			} else {
				fmt.Println("Invalid temperature value")
				os.Exit(1)
			}
		}
	case "status":
		state, err := getLightState(targetIP)
		if err != nil {
			fmt.Printf("✗ %s: Offline\n", targetName)
		} else {
			status := "Off"
			if state.On == 1 {
				status = "On"
			}
			temp := int(1000000 / state.Temperature)
			fmt.Printf("%s: %s | Brightness: %d%% | Temperature: %dK\n", targetName, status, state.Brightness, temp)
		}
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Available commands: on, off, bright, temp, status")
		os.Exit(1)
	}
}
