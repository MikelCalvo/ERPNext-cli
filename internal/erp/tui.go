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
	Version = "1.6.1"
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
	// New views for TUI parity
	ViewDashboard
	ViewWarehouses
	ViewStock
	ViewStockDetail
	ViewStockReceive
	ViewStockTransfer
	ViewStockIssue
	ViewSerials
	ViewSerialDetail
	ViewCreateSerial
	ViewSuppliers
	ViewSupplierDetail
	ViewCreateSupplier
	ViewPurchaseOrders
	ViewPODetail
	ViewCreatePO
	ViewAddPOItem
	ViewPurchaseInvoices
	ViewPIDetail
	ViewCreatePI
	ViewConfirmAction
	// Sales module views
	ViewCustomers
	ViewCustomerDetail
	ViewCreateCustomer
	ViewQuotations
	ViewQuotationDetail
	ViewCreateQuotation
	ViewAddQuotationItem
	ViewSalesOrders
	ViewSODetail
	ViewCreateSO
	ViewCreateSOFromQuotation
	ViewAddSOItem
	ViewSalesInvoices
	ViewSIDetail
	ViewCreateSalesInvoice
	// Delivery Notes views
	ViewDeliveryNotes
	ViewDNDetail
	ViewCreateDN
	// Purchase Receipts views
	ViewPurchaseReceipts
	ViewPRDetail
	ViewCreatePR
	// Payment Entry views
	ViewPayments
	ViewPaymentDetail
	ViewCreatePayment
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
	// New fields for extended functionality
	dashboardData *ReportData
	formData      map[string]string
	confirmAction string
	confirmMsg    string
	listData      []map[string]interface{} // Raw data for detail views
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

type dashboardLoadedMsg struct {
	data *ReportData
}

type stockDataMsg struct {
	items []map[string]interface{}
}

type formSubmittedMsg struct {
	success bool
	message string
}

// NewTUI creates a new TUI model
func NewTUI(client *Client) Model {
	menuItems := []list.Item{
		MenuItem{"Dashboard", "Executive summary & KPIs", ViewDashboard},
		MenuItem{"Attributes", "Manage item attributes", ViewAttributes},
		MenuItem{"Items", "View all items", ViewItems},
		MenuItem{"Templates", "Manage item templates", ViewTemplates},
		MenuItem{"Groups", "Manage item groups", ViewGroups},
		MenuItem{"Brands", "Manage brands", ViewBrands},
		MenuItem{"Warehouses", "View warehouses", ViewWarehouses},
		MenuItem{"Stock", "Stock levels & operations", ViewStock},
		MenuItem{"Serial Numbers", "Track serialized items", ViewSerials},
		MenuItem{"Customers", "Manage customers", ViewCustomers},
		MenuItem{"Quotations", "Sales quotations", ViewQuotations},
		MenuItem{"Sales Orders", "SO workflow", ViewSalesOrders},
		MenuItem{"Sales Invoices", "Customer invoices", ViewSalesInvoices},
		MenuItem{"Delivery Notes", "Shipments from SO", ViewDeliveryNotes},
		MenuItem{"Suppliers", "Manage suppliers", ViewSuppliers},
		MenuItem{"Purchase Orders", "PO workflow", ViewPurchaseOrders},
		MenuItem{"Purchase Invoices", "Invoice management", ViewPurchaseInvoices},
		MenuItem{"Purchase Receipts", "Goods received from PO", ViewPurchaseReceipts},
		MenuItem{"Payments", "Receive/Pay invoices", ViewPayments},
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
		formData: make(map[string]string),
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
		var endpoint string
		if templatesOnly {
			// Only templates (has_variants=1)
			endpoint = "Item?limit_page_length=0&filters=%5B%5B%22has_variants%22%2C%22%3D%22%2C1%5D%5D"
		} else {
			// Only regular items, exclude templates (has_variants=0)
			endpoint = "Item?limit_page_length=0&filters=%5B%5B%22has_variants%22%2C%22%3D%22%2C0%5D%5D"
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
		case "ctrl+c":
			return m, tea.Quit

		case "q":
			// 'q' for quit at main, or create from quotation at sales orders
			if m.view == ViewMain {
				return m, tea.Quit
			}
			// Check for sales 'q' key (create from quotation) before going back
			result, cmd := m.handleSalesKeys("q")
			if cmd != nil {
				return result, cmd
			}
			m.view = ViewMain
			return m, nil

		case "esc":
			switch m.view {
			case ViewMain:
				// Do nothing at main
			case ViewAttrDetail:
				m.view = ViewAttributes
			case ViewItemDetail:
				if m.prevView == ViewTemplates {
					m.view = ViewTemplates
				} else {
					m.view = ViewItems
				}
			case ViewStockDetail:
				m.view = ViewStock
			case ViewSerialDetail:
				m.view = ViewSerials
			case ViewSupplierDetail:
				m.view = ViewSuppliers
			case ViewPODetail:
				m.view = ViewPurchaseOrders
			case ViewPIDetail:
				m.view = ViewPurchaseInvoices
			case ViewCustomerDetail:
				m.view = ViewCustomers
			case ViewQuotationDetail:
				m.view = ViewQuotations
			case ViewSODetail:
				m.view = ViewSalesOrders
			case ViewSIDetail:
				m.view = ViewSalesInvoices
			case ViewDNDetail:
				m.view = ViewDeliveryNotes
			case ViewPRDetail:
				m.view = ViewPurchaseReceipts
			case ViewPaymentDetail:
				m.view = ViewPayments
			case ViewCreateSupplier, ViewCreateSerial, ViewStockReceive,
				ViewStockTransfer, ViewStockIssue, ViewCreatePO,
				ViewAddPOItem, ViewCreatePI, ViewCreatePR,
				ViewCreateCustomer, ViewCreateQuotation, ViewAddQuotationItem,
				ViewCreateSO, ViewCreateSOFromQuotation, ViewAddSOItem, ViewCreateSalesInvoice,
				ViewCreateDN, ViewCreatePayment:
				// Form views go back to their parent
				if m.prevView != 0 {
					m.view = m.prevView
				} else {
					m.view = ViewMain
				}
			case ViewConfirmDelete, ViewConfirmAction:
				m.view = m.prevView
			default:
				m.view = ViewMain
			}
			return m, nil

		case "enter":
			return m.handleEnter()

		case "d":
			if m.view != ViewMain && m.view != ViewConfirmDelete && m.view != ViewConfirmAction {
				// Handle delete for list views
				switch m.view {
				case ViewAttributes, ViewItems, ViewTemplates, ViewGroups, ViewBrands,
					ViewSuppliers, ViewSerials, ViewCustomers:
					if item, ok := m.currentList.SelectedItem().(ListItem); ok {
						m.selectedItem = item.name
						m.prevView = m.view
						m.view = ViewConfirmDelete
						return m, nil
					}
				case ViewSupplierDetail, ViewSerialDetail, ViewCustomerDetail:
					m.prevView = m.view
					m.view = ViewConfirmDelete
					return m, nil
				}
			}

		case "y":
			if m.view == ViewConfirmDelete {
				m.view = m.prevView
				return m, m.handleDeleteForView()
			}
			if m.view == ViewConfirmAction {
				return m, m.handleConfirmAction(true)
			}

		case "n":
			if m.view == ViewConfirmDelete {
				m.view = m.prevView
				return m, nil
			}
			if m.view == ViewConfirmAction {
				return m, m.handleConfirmAction(false)
			}
			// Handle 'n' for new in list views
			result, cmd := m.handlePurchasingKeys("n")
			if cmd != nil {
				return result, cmd
			}
			result, cmd = m.handleStockKeys("n")
			if cmd != nil {
				return result, cmd
			}
			result, cmd = m.handleSalesKeys("n")
			if cmd != nil {
				return result, cmd
			}

		case "r":
			// Handle 'r' for receive in stock views
			if m.view == ViewStock || m.view == ViewStockDetail {
				result, cmd := m.handleStockKeys("r")
				if cmd != nil {
					return result, cmd
				}
			}
			// Otherwise refresh
			if m.view == ViewDashboard {
				return m.refreshCurrentView()
			}
			return m.refreshCurrentView()

		case "t":
			// Handle 't' for transfer in stock views
			result, cmd := m.handleStockKeys("t")
			if cmd != nil {
				return result, cmd
			}

		case "i":
			// Handle 'i' for issue in stock views or invoice from SO
			result, cmd := m.handleStockKeys("i")
			if cmd != nil {
				return result, cmd
			}
			result, cmd = m.handleSalesKeys("i")
			if cmd != nil {
				return result, cmd
			}

		case "a":
			// Handle 'a' for add item in PO/SO/Quotation detail
			result, cmd := m.handlePurchasingKeys("a")
			if cmd != nil {
				return result, cmd
			}
			result, cmd = m.handleSalesKeys("a")
			if cmd != nil {
				return result, cmd
			}

		case "s":
			// Handle 's' for submit in PO/PI/SO/SI/Quotation detail
			result, cmd := m.handlePurchasingKeys("s")
			if cmd != nil {
				return result, cmd
			}
			result, cmd = m.handleSalesKeys("s")
			if cmd != nil {
				return result, cmd
			}

		case "x":
			// Handle 'x' for cancel in PO/PI/SO/SI/Quotation detail
			result, cmd := m.handlePurchasingKeys("x")
			if cmd != nil {
				return result, cmd
			}
			result, cmd = m.handleSalesKeys("x")
			if cmd != nil {
				return result, cmd
			}

		case "o":
			// Handle 'o' for create SO from Quotation in Quotation detail
			result, cmd := m.handleSalesKeys("o")
			if cmd != nil {
				return result, cmd
			}
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

		m.setListTitle()
		return m, nil

	case itemDetailMsg:
		m.loading = false
		m.itemData = msg.data
		return m, nil

	case actionDoneMsg:
		m.message = msg.message
		m.messageType = "success"
		return m.refreshCurrentView()

	case dashboardLoadedMsg:
		m.loading = false
		m.dashboardData = msg.data
		return m, nil

	case stockDataMsg:
		m.loading = false
		m.listData = msg.items
		return m, nil

	case formSubmittedMsg:
		m.loading = false
		if msg.success {
			m.message = msg.message
			m.messageType = "success"
			return m.refreshCurrentView()
		}
		m.message = msg.message
		m.messageType = "error"
		return m, nil
	}

	var cmd tea.Cmd
	switch m.view {
	case ViewMain:
		m.mainMenu, cmd = m.mainMenu.Update(msg)
	case ViewAttributes, ViewItems, ViewTemplates, ViewGroups, ViewBrands,
		ViewWarehouses, ViewStock, ViewSerials, ViewSuppliers,
		ViewPurchaseOrders, ViewPurchaseInvoices, ViewPurchaseReceipts,
		ViewCustomers, ViewQuotations, ViewSalesOrders, ViewSalesInvoices, ViewDeliveryNotes,
		ViewPayments:
		m.currentList, cmd = m.currentList.Update(msg)
	case ViewCreateSupplier, ViewCreateSerial, ViewStockReceive, ViewStockTransfer, ViewStockIssue,
		ViewCreatePO, ViewAddPOItem, ViewCreatePI, ViewCreatePR,
		ViewCreateCustomer, ViewCreateQuotation, ViewAddQuotationItem,
		ViewCreateSO, ViewCreateSOFromQuotation, ViewAddSOItem, ViewCreateSalesInvoice,
		ViewCreateDN, ViewCreatePayment:
		cmd = m.updateFormInputs(msg)
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
			case ViewDashboard:
				return m, m.loadDashboard()
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
			case ViewWarehouses:
				return m, m.loadWarehouses()
			case ViewStock:
				return m, m.loadStock()
			case ViewSerials:
				return m, m.loadSerials("")
			case ViewSuppliers:
				return m, m.loadSuppliers()
			case ViewPurchaseOrders:
				return m, m.loadPurchaseOrders()
			case ViewPurchaseInvoices:
				return m, m.loadPurchaseInvoices()
			case ViewCustomers:
				return m, m.loadCustomers()
			case ViewQuotations:
				return m, m.loadQuotations()
			case ViewSalesOrders:
				return m, m.loadSalesOrders()
			case ViewSalesInvoices:
				return m, m.loadSalesInvoices()
			case ViewDeliveryNotes:
				return m, m.loadDeliveryNotes()
			case ViewPurchaseReceipts:
				return m, m.loadPurchaseReceipts()
			case ViewPayments:
				return m, m.loadPayments()
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
			m.prevView = m.view
			m.view = ViewItemDetail
			m.loading = true
			return m, m.loadItemDetail(item.name)
		}

	case ViewStock:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewStockDetail
			m.loading = true
			return m, m.loadStockDetail(item.name)
		}

	case ViewSerials:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewSerialDetail
			m.loading = true
			return m, m.loadSerialDetail(item.name)
		}

	case ViewSuppliers:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewSupplierDetail
			m.loading = true
			return m, m.loadSupplierDetail(item.name)
		}

	case ViewPurchaseOrders:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewPODetail
			m.loading = true
			return m, m.loadPODetail(item.name)
		}

	case ViewPurchaseInvoices:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewPIDetail
			m.loading = true
			return m, m.loadPIDetail(item.name)
		}

	case ViewCustomers:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewCustomerDetail
			m.loading = true
			return m, m.loadCustomerDetail(item.name)
		}

	case ViewQuotations:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewQuotationDetail
			m.loading = true
			return m, m.loadQuotationDetail(item.name)
		}

	case ViewSalesOrders:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewSODetail
			m.loading = true
			return m, m.loadSODetail(item.name)
		}

	case ViewSalesInvoices:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewSIDetail
			m.loading = true
			return m, m.loadSIDetail(item.name)
		}

	case ViewDeliveryNotes:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewDNDetail
			m.loading = true
			return m, m.loadDNDetail(item.name)
		}

	case ViewPurchaseReceipts:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewPRDetail
			m.loading = true
			return m, m.loadPRDetail(item.name)
		}

	case ViewPayments:
		if item, ok := m.currentList.SelectedItem().(ListItem); ok {
			m.selectedItem = item.name
			m.view = ViewPaymentDetail
			m.loading = true
			return m, m.loadPaymentDetail(item.name)
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
	case ViewDashboard:
		return m, m.loadDashboard()
	case ViewWarehouses:
		return m, m.loadWarehouses()
	case ViewStock:
		return m, m.loadStock()
	case ViewSerials:
		return m, m.loadSerials("")
	case ViewSuppliers:
		return m, m.loadSuppliers()
	case ViewPurchaseOrders:
		return m, m.loadPurchaseOrders()
	case ViewPurchaseInvoices:
		return m, m.loadPurchaseInvoices()
	case ViewCustomers:
		return m, m.loadCustomers()
	case ViewQuotations:
		return m, m.loadQuotations()
	case ViewSalesOrders:
		return m, m.loadSalesOrders()
	case ViewSalesInvoices:
		return m, m.loadSalesInvoices()
	case ViewDeliveryNotes:
		return m, m.loadDeliveryNotes()
	case ViewPurchaseReceipts:
		return m, m.loadPurchaseReceipts()
	case ViewPayments:
		return m, m.loadPayments()
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
	case ViewAttributes, ViewItems, ViewTemplates, ViewGroups, ViewBrands,
		ViewWarehouses, ViewStock, ViewSerials, ViewSuppliers,
		ViewPurchaseOrders, ViewPurchaseInvoices, ViewPurchaseReceipts,
		ViewCustomers, ViewQuotations, ViewSalesOrders, ViewSalesInvoices, ViewDeliveryNotes,
		ViewPayments:
		if m.loading {
			content = "\n  Loading..."
		} else {
			content = m.currentList.View()
		}
	case ViewAttrDetail, ViewItemDetail:
		content = m.renderDetail()
	case ViewConfirmDelete:
		content = m.renderConfirmDelete()
	case ViewDashboard:
		content = m.renderDashboard()
	case ViewStockDetail:
		content = m.renderStockDetail()
	case ViewSerialDetail:
		content = m.renderSerialDetail()
	case ViewSupplierDetail:
		content = m.renderSupplierDetail()
	case ViewPODetail:
		content = m.renderPODetail()
	case ViewPIDetail:
		content = m.renderPIDetail()
	case ViewCreateSupplier:
		content = m.renderCreateSupplier()
	case ViewCreateSerial:
		content = m.renderCreateSerial()
	case ViewStockReceive:
		content = m.renderStockReceive()
	case ViewStockTransfer:
		content = m.renderStockTransfer()
	case ViewStockIssue:
		content = m.renderStockIssue()
	case ViewCreatePO:
		content = m.renderCreatePO()
	case ViewAddPOItem:
		content = m.renderAddPOItem()
	case ViewCreatePI:
		content = m.renderCreatePI()
	case ViewConfirmAction:
		content = m.renderConfirmAction()
	// Sales module views
	case ViewCustomerDetail:
		content = m.renderCustomerDetail()
	case ViewCreateCustomer:
		content = m.renderCreateCustomer()
	case ViewQuotationDetail:
		content = m.renderQuotationDetail()
	case ViewCreateQuotation:
		content = m.renderCreateQuotation()
	case ViewAddQuotationItem:
		content = m.renderAddQuotationItem()
	case ViewSODetail:
		content = m.renderSODetail()
	case ViewCreateSO:
		content = m.renderCreateSO()
	case ViewCreateSOFromQuotation:
		content = m.renderCreateSOFromQuotation()
	case ViewAddSOItem:
		content = m.renderAddSOItem()
	case ViewSIDetail:
		content = m.renderSIDetail()
	case ViewCreateSalesInvoice:
		content = m.renderCreateSI()
	// Delivery Notes views
	case ViewDNDetail:
		content = m.renderDNDetail()
	case ViewCreateDN:
		content = m.renderCreateDN()
	// Purchase Receipts views
	case ViewPRDetail:
		content = m.renderPRDetail()
	case ViewCreatePR:
		content = m.renderCreatePR()
	// Payment Entry views
	case ViewPaymentDetail:
		content = m.renderPaymentDetail()
	case ViewCreatePayment:
		content = m.renderCreatePayment()
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
	case ViewWarehouses:
		help = "↑/↓: navigate • r: refresh • /: search • esc: back"
	case ViewStock:
		help = "↑/↓: navigate • enter: detail • r: receive • t: transfer • i: issue • esc: back"
	case ViewSerials:
		help = "↑/↓: navigate • enter: detail • n: new • d: delete • /: search • esc: back"
	case ViewSuppliers:
		help = "↑/↓: navigate • enter: detail • n: new • d: delete • /: search • esc: back"
	case ViewPurchaseOrders:
		help = "↑/↓: navigate • enter: detail • n: new PO • /: search • esc: back"
	case ViewPurchaseInvoices:
		help = "↑/↓: navigate • enter: detail • /: search • esc: back"
	case ViewAttrDetail, ViewItemDetail:
		help = "esc: back • d: delete"
	case ViewStockDetail:
		help = "esc: back • r: receive • t: transfer • i: issue"
	case ViewSerialDetail, ViewSupplierDetail:
		help = "esc: back • d: delete"
	case ViewPIDetail:
		help = "esc: back • s: submit • x: cancel • p: create payment"
	// Sales views
	case ViewCustomers:
		help = "↑/↓: navigate • enter: detail • n: new • d: delete • /: search • esc: back"
	case ViewQuotations:
		help = "↑/↓: navigate • enter: detail • n: new • /: search • esc: back"
	case ViewSalesOrders:
		help = "↑/↓: navigate • enter: detail • n: new • q: from quotation • /: search • esc: back"
	case ViewSalesInvoices:
		help = "↑/↓: navigate • enter: detail • n: new • /: search • esc: back"
	case ViewCustomerDetail:
		help = "esc: back • d: delete"
	case ViewQuotationDetail:
		help = "esc: back • a: add item • s: submit • x: cancel • o: create SO"
	case ViewSODetail:
		help = "esc: back • a: add item • s: submit • x: cancel • i: create invoice • r: create DN"
	case ViewSIDetail:
		help = "esc: back • s: submit • x: cancel • p: create payment"
	case ViewDeliveryNotes:
		help = "↑/↓: navigate • enter: detail • n: new from SO • /: search • esc: back"
	case ViewDNDetail:
		help = "esc: back • s: submit • x: cancel"
	case ViewPurchaseReceipts:
		help = "↑/↓: navigate • enter: detail • n: new from PO • /: search • esc: back"
	case ViewPRDetail:
		help = "esc: back • s: submit • x: cancel"
	case ViewPayments:
		help = "↑/↓: navigate • enter: detail • /: search • esc: back"
	case ViewPaymentDetail:
		help = "esc: back • s: submit • x: cancel"
	case ViewPODetail:
		help = "esc: back • a: add item • s: submit • x: cancel PO • r: create PR"
	case ViewDashboard:
		help = "r: refresh • esc: back"
	case ViewConfirmDelete, ViewConfirmAction:
		help = "y: confirm • n: cancel"
	case ViewCreateSupplier, ViewCreateSerial, ViewStockReceive, ViewStockTransfer,
		ViewStockIssue, ViewCreatePO, ViewAddPOItem, ViewCreatePI, ViewCreatePR,
		ViewCreateCustomer, ViewCreateQuotation, ViewAddQuotationItem,
		ViewCreateSO, ViewCreateSOFromQuotation, ViewAddSOItem, ViewCreateSalesInvoice,
		ViewCreateDN, ViewCreatePayment:
		help = "tab: next field • enter: submit • esc: cancel"
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
