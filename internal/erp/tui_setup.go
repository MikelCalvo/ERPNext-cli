package erp

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SetupStep represents the current step in the setup wizard
type SetupStep int

const (
	SetupWelcome SetupStep = iota
	SetupURL
	SetupAPIKey
	SetupAPISecret
	SetupVPN
	SetupValidating
	SetupSuccess
	SetupError
)

// SetupModel is the model for the setup wizard
type SetupModel struct {
	step       SetupStep
	inputs     []textinput.Model
	focusIndex int
	width      int
	height     int
	err        error
	spinner    spinner.Model
	user       string // Authenticated username after validation
	mode       string // Connection mode after validation
	activeURL  string // URL used for connection
}

// Setup wizard styles
var (
	setupTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	setupBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2).
			Width(60)

	setupLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	setupHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	setupSuccessStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#04B575")).
				Bold(true)

	setupErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF4444")).
			Bold(true)

	setupHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))
)

// Messages for setup wizard
type setupValidateMsg struct {
	success bool
	user    string
	mode    string
	url     string
	err     error
}

type setupSaveMsg struct {
	success bool
	err     error
}

// NewSetupTUI creates a new setup wizard model
func NewSetupTUI() SetupModel {
	// Create 6 text inputs
	inputs := make([]textinput.Model, 6)

	// ERP URL
	inputs[0] = textinput.New()
	inputs[0].Placeholder = "https://your-erp.example.com"
	inputs[0].CharLimit = 256
	inputs[0].Width = 50

	// API Key
	inputs[1] = textinput.New()
	inputs[1].Placeholder = "API Key from User Settings > API Access"
	inputs[1].CharLimit = 64
	inputs[1].Width = 50

	// API Secret
	inputs[2] = textinput.New()
	inputs[2].Placeholder = "API Secret (generated with the key)"
	inputs[2].CharLimit = 64
	inputs[2].Width = 50
	inputs[2].EchoMode = textinput.EchoPassword

	// VPN URL (optional)
	inputs[3] = textinput.New()
	inputs[3].Placeholder = "http://192.168.1.100:8000 (optional)"
	inputs[3].CharLimit = 256
	inputs[3].Width = 50

	// Nginx Cookie (optional)
	inputs[4] = textinput.New()
	inputs[4].Placeholder = "Cookie value for reverse proxy (optional)"
	inputs[4].CharLimit = 256
	inputs[4].Width = 50

	// Nginx Cookie Name (optional)
	inputs[5] = textinput.New()
	inputs[5].Placeholder = "auth_cookie (default)"
	inputs[5].CharLimit = 64
	inputs[5].Width = 50

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	return SetupModel{
		step:    SetupWelcome,
		inputs:  inputs,
		spinner: s,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			if m.step == SetupValidating {
				return m, nil // Can't cancel during validation
			}
			return m, tea.Quit

		case "enter":
			return m.handleEnter()

		case "tab", "down":
			if m.step >= SetupURL && m.step <= SetupVPN {
				m.focusIndex++
				if m.focusIndex > 5 {
					m.focusIndex = 0
				}
				return m, m.updateInputFocus()
			}

		case "shift+tab", "up":
			if m.step >= SetupURL && m.step <= SetupVPN {
				m.focusIndex--
				if m.focusIndex < 0 {
					m.focusIndex = 5
				}
				return m, m.updateInputFocus()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case setupValidateMsg:
		if msg.success {
			m.step = SetupSuccess
			m.user = msg.user
			m.mode = msg.mode
			m.activeURL = msg.url
		} else {
			m.step = SetupError
			m.err = msg.err
		}
		return m, nil

	case setupSaveMsg:
		if msg.success {
			return m, tea.Quit
		}
		m.step = SetupError
		m.err = msg.err
		return m, nil
	}

	// Update text inputs when in form steps
	if m.step >= SetupURL && m.step <= SetupVPN {
		cmd := m.updateInputs(msg)
		return m, cmd
	}

	return m, nil
}

func (m *SetupModel) handleEnter() (tea.Model, tea.Cmd) {
	switch m.step {
	case SetupWelcome:
		m.step = SetupURL
		m.focusIndex = 0
		m.inputs[0].Focus()
		return m, textinput.Blink

	case SetupURL, SetupAPIKey, SetupAPISecret, SetupVPN:
		// Validate required fields
		if m.inputs[0].Value() == "" {
			m.focusIndex = 0
			return m, m.updateInputFocus()
		}
		if m.inputs[1].Value() == "" {
			m.focusIndex = 1
			return m, m.updateInputFocus()
		}
		if m.inputs[2].Value() == "" {
			m.focusIndex = 2
			return m, m.updateInputFocus()
		}

		// Start validation
		m.step = SetupValidating
		return m, tea.Batch(
			m.spinner.Tick,
			m.validateCredentials(),
		)

	case SetupSuccess:
		// Save config and exit
		return m, m.saveConfig()

	case SetupError:
		// Go back to URL step to retry
		m.step = SetupURL
		m.focusIndex = 0
		m.err = nil
		return m, m.updateInputFocus()
	}

	return m, nil
}

func (m *SetupModel) updateInputFocus() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds[i] = m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return tea.Batch(cmds...)
}

func (m *SetupModel) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))
	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}
	return tea.Batch(cmds...)
}

func (m SetupModel) validateCredentials() tea.Cmd {
	return func() tea.Msg {
		url := strings.TrimSuffix(m.inputs[0].Value(), "/")
		apiKey := m.inputs[1].Value()
		apiSecret := m.inputs[2].Value()
		vpnURL := strings.TrimSuffix(m.inputs[3].Value(), "/")

		// Try VPN URL first if provided
		if vpnURL != "" {
			user, err := validateConnection(vpnURL, apiKey, apiSecret)
			if err == nil {
				return setupValidateMsg{
					success: true,
					user:    user,
					mode:    "vpn",
					url:     vpnURL,
				}
			}
		}

		// Try main URL
		user, err := validateConnection(url, apiKey, apiSecret)
		if err != nil {
			return setupValidateMsg{
				success: false,
				err:     err,
			}
		}

		return setupValidateMsg{
			success: true,
			user:    user,
			mode:    "internet",
			url:     url,
		}
	}
}

func validateConnection(url, apiKey, apiSecret string) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	fullURL := fmt.Sprintf("%s/api/method/frappe.auth.get_logged_user", url)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", apiKey, apiSecret))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", fmt.Errorf("authentication failed: invalid API key or secret")
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("server error: HTTP %d", resp.StatusCode)
	}

	// Parse response to get username
	var result map[string]interface{}
	if err := decodeJSON(resp.Body, &result); err != nil {
		return "", fmt.Errorf("invalid response from server")
	}

	if msg, ok := result["message"].(string); ok && msg != "" {
		return msg, nil
	}

	return "Unknown", nil
}

func (m SetupModel) saveConfig() tea.Cmd {
	return func() tea.Msg {
		url := strings.TrimSuffix(m.inputs[0].Value(), "/")
		apiKey := m.inputs[1].Value()
		apiSecret := m.inputs[2].Value()
		vpnURL := strings.TrimSuffix(m.inputs[3].Value(), "/")
		nginxCookie := m.inputs[4].Value()
		nginxCookieName := m.inputs[5].Value()

		// Build config content
		var sb strings.Builder
		sb.WriteString("# ERPNext CLI Configuration\n")
		sb.WriteString("# Generated by setup wizard\n\n")

		sb.WriteString("# ERPNext URL (required)\n")
		sb.WriteString(fmt.Sprintf("ERP_URL=%s\n\n", url))

		sb.WriteString("# API Credentials (required)\n")
		sb.WriteString(fmt.Sprintf("ERP_API_KEY=%s\n", apiKey))
		sb.WriteString(fmt.Sprintf("ERP_API_SECRET=%s\n\n", apiSecret))

		if vpnURL != "" {
			sb.WriteString("# VPN/Direct URL (optional, tried first)\n")
			sb.WriteString(fmt.Sprintf("ERP_VPN=%s\n\n", vpnURL))
		}

		if nginxCookie != "" {
			sb.WriteString("# Nginx cookie for reverse proxy auth\n")
			sb.WriteString(fmt.Sprintf("NGINX_COOKIE=%s\n", nginxCookie))
			if nginxCookieName != "" {
				sb.WriteString(fmt.Sprintf("NGINX_COOKIE_NAME=%s\n", nginxCookieName))
			} else {
				sb.WriteString("NGINX_COOKIE_NAME=auth_cookie\n")
			}
			sb.WriteString("\n")
		}

		sb.WriteString("# Optional: Company name (auto-detected if empty)\n")
		sb.WriteString("# ERP_COMPANY=Your Company Name\n\n")

		sb.WriteString("# Optional: Custom branding for TUI\n")
		sb.WriteString("# ERP_BRAND=ERPNext CLI\n")

		// Write to .erp-config in current directory
		err := os.WriteFile(".erp-config", []byte(sb.String()), 0600)
		if err != nil {
			return setupSaveMsg{success: false, err: err}
		}

		return setupSaveMsg{success: true}
	}
}

func (m SetupModel) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	switch m.step {
	case SetupWelcome:
		content = m.renderWelcome()
	case SetupURL, SetupAPIKey, SetupAPISecret, SetupVPN:
		content = m.renderForm()
	case SetupValidating:
		content = m.renderValidating()
	case SetupSuccess:
		content = m.renderSuccess()
	case SetupError:
		content = m.renderError()
	}

	return content
}

func (m SetupModel) renderWelcome() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(setupTitleStyle.Render("  Welcome to ERPNext CLI  "))
	sb.WriteString("\n\n")

	welcomeText := `No configuration file found.
Let's set up your connection to ERPNext.

You'll need:
  * Your ERPNext URL
  * API Key & Secret (User Settings > API Access)

`
	sb.WriteString(welcomeText)
	sb.WriteString(setupHelpStyle.Render("[Enter] Continue    [Esc] Cancel"))

	return setupBoxStyle.Render(sb.String())
}

func (m SetupModel) renderForm() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(setupTitleStyle.Render("  Setup (1/1)  "))
	sb.WriteString("\n\n")

	// URL field
	sb.WriteString(setupLabelStyle.Render("ERPNext URL *"))
	sb.WriteString("\n")
	sb.WriteString(m.inputs[0].View())
	sb.WriteString("\n")
	sb.WriteString(setupHintStyle.Render("Example: https://erp.mycompany.com"))
	sb.WriteString("\n\n")

	// API Key field
	sb.WriteString(setupLabelStyle.Render("API Key *"))
	sb.WriteString("\n")
	sb.WriteString(m.inputs[1].View())
	sb.WriteString("\n")
	sb.WriteString(setupHintStyle.Render("Find in: User Settings > API Access"))
	sb.WriteString("\n\n")

	// API Secret field
	sb.WriteString(setupLabelStyle.Render("API Secret *"))
	sb.WriteString("\n")
	sb.WriteString(m.inputs[2].View())
	sb.WriteString("\n")
	sb.WriteString(setupHintStyle.Render("Generated with API Key"))
	sb.WriteString("\n\n")

	// VPN URL field (optional)
	sb.WriteString(setupLabelStyle.Render("VPN/Direct URL"))
	sb.WriteString(" ")
	sb.WriteString(setupHintStyle.Render("(optional)"))
	sb.WriteString("\n")
	sb.WriteString(m.inputs[3].View())
	sb.WriteString("\n")
	sb.WriteString(setupHintStyle.Render("Direct connection tried first"))
	sb.WriteString("\n\n")

	// Nginx Cookie field (optional)
	sb.WriteString(setupLabelStyle.Render("Nginx Cookie"))
	sb.WriteString(" ")
	sb.WriteString(setupHintStyle.Render("(optional)"))
	sb.WriteString("\n")
	sb.WriteString(m.inputs[4].View())
	sb.WriteString("\n")
	sb.WriteString(setupHintStyle.Render("For reverse proxy authentication"))
	sb.WriteString("\n\n")

	// Nginx Cookie Name field (optional)
	sb.WriteString(setupLabelStyle.Render("Cookie Name"))
	sb.WriteString(" ")
	sb.WriteString(setupHintStyle.Render("(optional)"))
	sb.WriteString("\n")
	sb.WriteString(m.inputs[5].View())
	sb.WriteString("\n")
	sb.WriteString(setupHintStyle.Render("Default: auth_cookie"))
	sb.WriteString("\n\n")

	sb.WriteString(setupHelpStyle.Render("[Tab] Next field    [Enter] Submit    [Esc] Cancel"))

	return setupBoxStyle.Render(sb.String())
}

func (m SetupModel) renderValidating() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(setupTitleStyle.Render("  Validating  "))
	sb.WriteString("\n\n")

	sb.WriteString(m.spinner.View())
	sb.WriteString(" Testing connection to ERPNext...")
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("URL: %s\n", m.inputs[0].Value()))
	sb.WriteString(fmt.Sprintf("API Key: %s...\n", truncate(m.inputs[1].Value(), 8)))

	return setupBoxStyle.Render(sb.String())
}

func (m SetupModel) renderSuccess() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(setupSuccessStyle.Render("  Setup Complete!  "))
	sb.WriteString("\n\n")

	sb.WriteString("Configuration saved to: ")
	sb.WriteString(setupLabelStyle.Render(".erp-config"))
	sb.WriteString("\n\n")

	sb.WriteString(fmt.Sprintf("Connected as: %s\n", setupLabelStyle.Render(m.user)))

	if m.mode == "vpn" {
		sb.WriteString("Mode: ")
		sb.WriteString(vpnStyle.Render("VPN direct"))
	} else {
		sb.WriteString("Mode: ")
		sb.WriteString(internetStyle.Render("Internet"))
	}
	sb.WriteString("\n\n")

	sb.WriteString(setupHelpStyle.Render("[Enter] Start ERPNext CLI"))

	return setupBoxStyle.Render(sb.String())
}

func (m SetupModel) renderError() string {
	var sb strings.Builder

	sb.WriteString("\n")
	sb.WriteString(setupErrorStyle.Render("  Connection Failed  "))
	sb.WriteString("\n\n")

	if m.err != nil {
		sb.WriteString(fmt.Sprintf("Error: %s\n\n", m.err.Error()))
	}

	sb.WriteString("Please check:\n")
	sb.WriteString("  * URL is correct and reachable\n")
	sb.WriteString("  * API Key and Secret are valid\n")
	sb.WriteString("  * Your ERPNext instance is running\n\n")

	sb.WriteString(setupHelpStyle.Render("[Enter] Try again    [Esc] Cancel"))

	return setupBoxStyle.Render(sb.String())
}

func truncate(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length]
}

// RunSetupTUI runs the setup wizard
func RunSetupTUI() error {
	p := tea.NewProgram(NewSetupTUI(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
