package erp

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Version info
const (
	Version = "1.0.0"
	Author  = "Mikel Calvo"
	Year    = "2025"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)

	vpnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	internetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF9500")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262"))

	creditStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)
)

// View represents different screens
type View int

const (
	ViewMain View = iota
	ViewAttributes
	ViewItems
	ViewTemplates
	ViewGroups
	ViewBrands
	ViewAttrDetail
	ViewItemDetail
	ViewCreateAttr
	ViewCreateItem
	ViewCreateTemplate
	ViewConfirmDelete
)

// MenuItem for the main menu
type MenuItem struct {
	title       string
	description string
	view        View
}

func (i MenuItem) Title() string       { return i.title }
func (i MenuItem) Description() string { return i.description }
func (i MenuItem) FilterValue() string { return i.title }

// ListItem for resource lists
type ListItem struct {
	name    string
	details string
}

func (i ListItem) Title() string       { return i.name }
func (i ListItem) Description() string { return i.details }
func (i ListItem) FilterValue() string { return i.name }

// Model is the main TUI model
type Model struct {
	client       *Client
	view         View
	prevView     View
	width        int
	height       int
	mainMenu     list.Model
	currentList  list.Model
	inputs       []textinput.Model
	focusIndex   int
	message      string
	messageType  string
	loading      bool
	selectedItem string
	itemData     map[string]interface{}
}

// Messages
type connectedMsg struct {
	mode string
	url  string
	user string
}

type errorMsg struct {
	err error
}

type dataLoadedMsg struct {
	items []ListItem
}

type itemDetailMsg struct {
	data map[string]interface{}
}

type actionDoneMsg struct {
	message string
}

// NewTUI creates a new TUI model
func NewTUI(client *Client) Model {
	menuItems := []list.Item{
		MenuItem{"Attributes", "Manage item attributes", ViewAttributes},
		MenuItem{"Items", "View all items", ViewItems},
		MenuItem{"Templates", "Manage item templates", ViewTemplates},
		MenuItem{"Groups", "Manage item groups", ViewGroups},
		MenuItem{"Brands", "Manage brands", ViewBrands},
	}

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = selectedStyle
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))

	mainMenu := list.New(menuItems, delegate, 0, 0)
	mainMenu.Title = client.Config.Brand
	mainMenu.SetShowStatusBar(false)
	mainMenu.SetFilteringEnabled(false)
	mainMenu.Styles.Title = titleStyle

	return Model{
		client:   client,
		view:     ViewMain,
		mainMenu: mainMenu,
		loading:  true,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.detectConnection(),
	)
}

func (m Model) detectConnection() tea.Cmd {
	return func() tea.Msg {
		m.client.DetectConnection()

		fullURL := fmt.Sprintf("%s/api/method/frappe.auth.get_logged_user", m.client.ActiveURL)
		req, _ := m.client.HTTPClient.Get(fullURL)
		if req != nil {
			defer req.Body.Close()
		}

		return connectedMsg{
			mode: m.client.Mode,
			url:  m.client.ActiveURL,
			user: "",
		}
	}
}

func (m Model) loadAttributes() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Item%20Attribute?limit_page_length=0", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					items = append(items, ListItem{name: name, details: "Attribute"})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

func (m Model) loadItems(templatesOnly bool) tea.Cmd {
	return func() tea.Msg {
		endpoint := "Item?limit_page_length=0"
		if templatesOnly {
			endpoint = "Item?limit_page_length=0&filters=%5B%5B%22has_variants%22%2C%22%3D%22%2C1%5D%5D"
		}

		result, err := m.client.Request("GET", endpoint, nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					detail := "Item"
					if templatesOnly {
						detail = "Template"
					}
					items = append(items, ListItem{name: name, details: detail})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

func (m Model) loadGroups() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Item%20Group?limit_page_length=0", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					items = append(items, ListItem{name: name, details: "Group"})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

func (m Model) loadBrands() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Brand?limit_page_length=0", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					items = append(items, ListItem{name: name, details: "Brand"})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

func (m Model) loadItemDetail(code string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Item/"+code, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

func (m Model) loadAttrDetail(name string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Item%20Attribute/"+name, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

func (m Model) deleteItem(itemType, name string) tea.Cmd {
	return func() tea.Msg {
		var endpoint string
		switch itemType {
		case "attr":
			endpoint = "Item%20Attribute/" + name
		case "item", "template":
			endpoint = "Item/" + name
		case "group":
			endpoint = "Item%20Group/" + name
		case "brand":
			endpoint = "Brand/" + name
		}

		_, err := m.client.Request("DELETE", endpoint, nil)
		if err != nil {
			return errorMsg{err}
		}
		return actionDoneMsg{fmt.Sprintf("Deleted: %s", name)}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		m.message = ""
		m.messageType = ""

		switch msg.String() {
		case "ctrl+c", "q":
			if m.view == ViewMain {
				return m, tea.Quit
			}
			m.view = ViewMain
			return m, nil

		case "esc":
			if m.view != ViewMain {
				m.view = ViewMain
				return m, nil
			}

		case "enter":
			return m.handleEnter()

		case "d":
			if m.view != ViewMain && m.view != ViewConfirmDelete {
				if item, ok := m.currentList.SelectedItem().(ListItem); ok {
					m.selectedItem = item.name
					m.prevView = m.view
					m.view = ViewConfirmDelete
					return m, nil
				}
			}

		case "y":
			if m.view == ViewConfirmDelete {
				var itemType string
				switch m.prevView {
				case ViewAttributes:
					itemType = "attr"
				case ViewItems:
					itemType = "item"
				case ViewTemplates:
					itemType = "template"
				case ViewGroups:
					itemType = "group"
				case ViewBrands:
					itemType = "brand"
				}
				m.view = m.prevView
				return m, m.deleteItem(itemType, m.selectedItem)
			}

		case "n":
			if m.view == ViewConfirmDelete {
				m.view = m.prevView
				return m, nil
			}

		case "r":
			return m.refreshCurrentView()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		h := msg.Height - 8
		w := msg.Width - 4

		m.mainMenu.SetSize(w, h)
		if m.currentList.Items() != nil {
			m.currentList.SetSize(w, h)
		}

	case connectedMsg:
		m.loading = false
		m.client.Mode = msg.mode
		m.client.ActiveURL = msg.url
		return m, nil

	case errorMsg:
		m.loading = false
		m.message = msg.err.Error()
		m.messageType = "error"
		return m, nil

	case dataLoadedMsg:
		m.loading = false
		items := make([]list.Item, len(msg.items))
		for i, item := range msg.items {
			items[i] = item
		}

		delegate := list.NewDefaultDelegate()
		delegate.Styles.SelectedTitle = selectedStyle

		m.currentList = list.New(items, delegate, m.width-4, m.height-8)
		m.currentList.SetShowStatusBar(true)
		m.currentList.SetFilteringEnabled(true)

		switch m.view {
		case ViewAttributes:
			m.currentList.Title = "Attributes"
		case ViewItems:
			m.currentList.Title = "Items"
		case ViewTemplates:
			m.currentList.Title = "Templates"
		case ViewGroups:
			m.currentList.Title = "Groups"
		case ViewBrands:
			m.currentList.Title = "Brands"
		}
		m.currentList.Styles.Title = titleStyle
		return m, nil

	case itemDetailMsg:
		m.loading = false
		m.itemData = msg.data
		return m, nil

	case actionDoneMsg:
		m.message = msg.message
		m.messageType = "success"
		return m.refreshCurrentView()
	}

	var cmd tea.Cmd
	switch m.view {
	case ViewMain:
		m.mainMenu, cmd = m.mainMenu.Update(msg)
	case ViewAttributes, ViewItems, ViewTemplates, ViewGroups, ViewBrands:
		m.currentList, cmd = m.currentList.Update(msg)
	}

	return m, cmd
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewMain:
		if item, ok := m.mainMenu.SelectedItem().(MenuItem); ok {
			m.view = item.view
			m.loading = true

			switch item.view {
			case ViewAttributes:
				return m, m.loadAttributes()
			case ViewItems:
				return m, m.loadItems(false)
			case ViewTemplates:
				return m, m.loadItems(true)
			case ViewGroups:
				return m, m.loadGroups()
			case ViewBrands:
				return m, m.loadBrands()
			}
		}

	case ViewAttributes:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewAttrDetail
			m.loading = true
			return m, m.loadAttrDetail(item.name)
		}

	case ViewItems, ViewTemplates:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewItemDetail
			m.loading = true
			return m, m.loadItemDetail(item.name)
		}
	}

	return m, nil
}

func (m Model) refreshCurrentView() (tea.Model, tea.Cmd) {
	m.loading = true
	switch m.view {
	case ViewAttributes:
		return m, m.loadAttributes()
	case ViewItems:
		return m, m.loadItems(false)
	case ViewTemplates:
		return m, m.loadItems(true)
	case ViewGroups:
		return m, m.loadGroups()
	case ViewBrands:
		return m, m.loadBrands()
	}
	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var content string

	switch m.view {
	case ViewMain:
		content = m.mainMenu.View()
	case ViewAttributes, ViewItems, ViewTemplates, ViewGroups, ViewBrands:
		if m.loading {
			content = "\n  Loading..."
		} else {
			content = m.currentList.View()
		}
	case ViewAttrDetail, ViewItemDetail:
		content = m.renderDetail()
	case ViewConfirmDelete:
		content = m.renderConfirmDelete()
	}

	var b strings.Builder

	// Status bar
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n\n")

	// Content
	b.WriteString(content)

	// Message
	if m.message != "" {
		b.WriteString("\n\n")
		if m.messageType == "error" {
			b.WriteString(errorStyle.Render("Error: " + m.message))
		} else if m.messageType == "success" {
			b.WriteString(successStyle.Render("✓ " + m.message))
		}
	}

	// Help
	b.WriteString("\n\n")
	b.WriteString(m.renderHelp())

	// Credits
	b.WriteString("\n")
	b.WriteString(m.renderCredits())

	return b.String()
}

func (m Model) renderStatusBar() string {
	var mode string
	if m.client.Mode == "vpn" {
		mode = vpnStyle.Render("● VPN")
	} else {
		mode = internetStyle.Render("● Internet")
	}

	status := fmt.Sprintf(" %s | %s | %s ", m.client.Config.Brand, mode, m.client.ActiveURL)
	return statusBarStyle.Render(status)
}

func (m Model) renderHelp() string {
	var help string
	switch m.view {
	case ViewMain:
		help = "↑/↓: navigate • enter: select • q: quit"
	case ViewAttributes, ViewItems, ViewTemplates, ViewGroups, ViewBrands:
		help = "↑/↓: navigate • enter: view detail • d: delete • r: refresh • /: search • esc: back"
	case ViewAttrDetail, ViewItemDetail:
		help = "esc: back • d: delete"
	case ViewConfirmDelete:
		help = "y: confirm • n: cancel"
	}
	return helpStyle.Render(help)
}

func (m Model) renderCredits() string {
	return creditStyle.Render(fmt.Sprintf("Created by %s in %s • v%s", Author, Year, Version))
}

func (m Model) renderDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder

	if m.view == ViewAttrDetail {
		b.WriteString(titleStyle.Render(" Attribute: "+m.selectedItem) + "\n\n")

		if name, ok := m.itemData["attribute_name"]; ok {
			b.WriteString(fmt.Sprintf("  Name: %v\n", name))
		}
		if numeric, ok := m.itemData["numeric_values"]; ok {
			if numeric == float64(1) {
				b.WriteString("  Type: Numeric\n")
				b.WriteString(fmt.Sprintf("  Range: %v - %v (step: %v)\n",
					m.itemData["from_range"], m.itemData["to_range"], m.itemData["increment"]))
			} else {
				b.WriteString("  Type: List/Text\n")
			}
		}

		if values, ok := m.itemData["item_attribute_values"].([]interface{}); ok && len(values) > 0 {
			b.WriteString(fmt.Sprintf("\n  Values (%d):\n", len(values)))
			for i, v := range values {
				if i >= 10 {
					b.WriteString(fmt.Sprintf("  ... and %d more\n", len(values)-10))
					break
				}
				if vm, ok := v.(map[string]interface{}); ok {
					b.WriteString(fmt.Sprintf("    • %v (%v)\n", vm["attribute_value"], vm["abbr"]))
				}
			}
		}
	} else {
		b.WriteString(titleStyle.Render(" Item: "+m.selectedItem) + "\n\n")

		fields := []string{"item_code", "item_name", "item_group", "stock_uom"}
		labels := []string{"Code", "Name", "Group", "UoM"}

		for i, field := range fields {
			if val, ok := m.itemData[field]; ok {
				b.WriteString(fmt.Sprintf("  %s: %v\n", labels[i], val))
			}
		}

		if hv, ok := m.itemData["has_variants"]; ok && hv == float64(1) {
			b.WriteString("  Type: Template (with variants)\n")
		}

		if attrs, ok := m.itemData["attributes"].([]interface{}); ok && len(attrs) > 0 {
			b.WriteString(fmt.Sprintf("\n  Attributes (%d):\n", len(attrs)))
			for _, a := range attrs {
				if am, ok := a.(map[string]interface{}); ok {
					b.WriteString(fmt.Sprintf("    • %v\n", am["attribute"]))
				}
			}
		}
	}

	return boxStyle.Render(b.String())
}

func (m Model) renderConfirmDelete() string {
	content := fmt.Sprintf(`
  Delete "%s"?

  This action cannot be undone.

  [y] Yes, delete    [n] No, cancel
`, m.selectedItem)

	return boxStyle.Render(content)
}

// RunTUI starts the TUI
func RunTUI(client *Client) error {
	p := tea.NewProgram(NewTUI(client), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
