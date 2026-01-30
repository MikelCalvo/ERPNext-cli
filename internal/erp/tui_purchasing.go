package erp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// loadSuppliers fetches all suppliers
func (m Model) loadSuppliers() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Supplier?limit_page_length=0&fields=[\"name\",\"supplier_name\",\"supplier_group\",\"disabled\"]", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					group, _ := im["supplier_group"].(string)
					disabled, _ := im["disabled"].(float64)

					detail := group
					if disabled == 1 {
						detail += " [disabled]"
					}
					items = append(items, ListItem{name: name, details: detail})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadSupplierDetail fetches supplier detail
func (m Model) loadSupplierDetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Supplier/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderSupplierDetail renders the supplier detail view
func (m Model) renderSupplierDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Supplier: "+m.selectedItem) + "\n\n")

	fields := []struct {
		key   string
		label string
	}{
		{"supplier_name", "Name"},
		{"supplier_group", "Group"},
		{"supplier_type", "Type"},
		{"country", "Country"},
		{"default_currency", "Currency"},
	}

	for _, f := range fields {
		if val, ok := m.itemData[f.key]; ok && val != nil && val != "" {
			b.WriteString(fmt.Sprintf("  %s: %v\n", f.label, val))
		}
	}

	if disabled, ok := m.itemData["disabled"]; ok && disabled == float64(1) {
		b.WriteString(fmt.Sprintf("\n  %s\n", errorStyle.Render("DISABLED")))
	}

	return boxStyle.Render(b.String())
}

// initCreateSupplierForm initializes the create supplier form
func (m *Model) initCreateSupplierForm() {
	m.inputs = make([]textinput.Model, 3)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Supplier Name"
	m.inputs[0].Focus()

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Supplier Group (optional)"

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "Country (optional)"

	m.focusIndex = 0
}

// renderCreateSupplier renders the create supplier form
func (m Model) renderCreateSupplier() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Supplier ") + "\n\n")

	labels := []string{"Name:", "Group:", "Country:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// submitCreateSupplier submits the create supplier form
func (m Model) submitCreateSupplier() tea.Cmd {
	return func() tea.Msg {
		name := m.inputs[0].Value()
		group := m.inputs[1].Value()
		country := m.inputs[2].Value()

		if name == "" {
			return formSubmittedMsg{false, "Supplier name is required"}
		}

		body := map[string]interface{}{
			"supplier_name": name,
		}
		if group != "" {
			body["supplier_group"] = group
		}
		if country != "" {
			body["country"] = country
		}

		result, err := m.client.Request("POST", "Supplier", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Supplier created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create supplier"}
	}
}

// deleteSupplier deletes a supplier
func (m Model) deleteSupplier(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		_, err := m.client.Request("DELETE", "Supplier/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}
		return actionDoneMsg{fmt.Sprintf("Deleted: %s", name)}
	}
}

// loadPurchaseOrders fetches all purchase orders
func (m Model) loadPurchaseOrders() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Purchase%20Order?limit_page_length=100&fields=[\"name\",\"supplier\",\"transaction_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					supplier := fmt.Sprintf("%v", im["supplier"])
					status, _ := im["status"].(string)
					total, _ := im["grand_total"].(float64)

					statusIcon := ""
					switch status {
					case "Draft":
						statusIcon = "Draft"
					case "To Receive and Bill", "To Receive":
						statusIcon = "Pending"
					case "Completed":
						statusIcon = "Completed"
					case "Cancelled":
						statusIcon = "Cancelled"
					default:
						statusIcon = status
					}

					detail := fmt.Sprintf("%s | %s | %s", supplier, statusIcon, m.client.FormatCurrency(total))
					items = append(items, ListItem{name: name, details: detail})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadPODetail fetches purchase order detail
func (m Model) loadPODetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Purchase%20Order/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderPODetail renders the purchase order detail view
func (m Model) renderPODetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Purchase Order: "+m.selectedItem) + "\n\n")

	// Basic info
	b.WriteString(fmt.Sprintf("  Supplier: %v\n", m.itemData["supplier"]))
	b.WriteString(fmt.Sprintf("  Date: %v\n", m.itemData["transaction_date"]))

	status, _ := m.itemData["status"].(string)
	statusStyle := helpStyle
	switch status {
	case "Draft":
		statusStyle = internetStyle
	case "Completed":
		statusStyle = successStyle
	case "Cancelled":
		statusStyle = errorStyle
	}
	b.WriteString(fmt.Sprintf("  Status: %s\n", statusStyle.Render(status)))

	grandTotal, _ := m.itemData["grand_total"].(float64)
	b.WriteString(fmt.Sprintf("  Total: %s\n", m.client.FormatCurrency(grandTotal)))

	// Items
	if items, ok := m.itemData["items"].([]interface{}); ok && len(items) > 0 {
		b.WriteString(fmt.Sprintf("\n  %s\n", selectedStyle.Render("Items:")))
		for _, item := range items {
			if im, ok := item.(map[string]interface{}); ok {
				itemCode := fmt.Sprintf("%v", im["item_code"])
				qty, _ := im["qty"].(float64)
				rate, _ := im["rate"].(float64)
				amount, _ := im["amount"].(float64)
				b.WriteString(fmt.Sprintf("    - %s: %.0f x %s = %s\n", itemCode, qty, m.client.FormatCurrency(rate), m.client.FormatCurrency(amount)))
			}
		}
	}

	return boxStyle.Render(b.String())
}

// initCreatePOForm initializes the create PO form
func (m *Model) initCreatePOForm() {
	m.inputs = make([]textinput.Model, 1)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Supplier Name"
	m.inputs[0].Focus()

	m.focusIndex = 0
}

// renderCreatePO renders the create PO form
func (m Model) renderCreatePO() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Purchase Order ") + "\n\n")

	b.WriteString("  Supplier:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	b.WriteString(helpStyle.Render("  After creation, add items with 'a' key"))

	return boxStyle.Render(b.String())
}

// submitCreatePO submits the create PO form
func (m Model) submitCreatePO() tea.Cmd {
	return func() tea.Msg {
		supplier := m.inputs[0].Value()

		if supplier == "" {
			return formSubmittedMsg{false, "Supplier is required"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		today := time.Now().Format("2006-01-02")

		body := map[string]interface{}{
			"supplier":         supplier,
			"transaction_date": today,
			"schedule_date":    today,
			"company":          company,
			"items":            []interface{}{},
		}

		result, err := m.client.Request("POST", "Purchase%20Order", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("PO created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create PO"}
	}
}

// initAddPOItemForm initializes the add PO item form
func (m *Model) initAddPOItemForm() {
	m.inputs = make([]textinput.Model, 3)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Item Code"
	m.inputs[0].Focus()

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Quantity"

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "Rate (optional)"

	m.focusIndex = 0
}

// renderAddPOItem renders the add PO item form
func (m Model) renderAddPOItem() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Add Item to PO: "+m.selectedItem) + "\n\n")

	labels := []string{"Item Code:", "Quantity:", "Rate:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// submitAddPOItem submits the add PO item form
func (m Model) submitAddPOItem() tea.Cmd {
	return func() tea.Msg {
		itemCode := m.inputs[0].Value()
		qtyStr := m.inputs[1].Value()
		rateStr := m.inputs[2].Value()

		if itemCode == "" || qtyStr == "" {
			return formSubmittedMsg{false, "Item code and quantity are required"}
		}

		qty, err := strconv.ParseFloat(qtyStr, 64)
		if err != nil {
			return formSubmittedMsg{false, "Invalid quantity"}
		}

		rate := 0.0
		if rateStr != "" {
			rate, _ = strconv.ParseFloat(rateStr, 64)
		}

		// Get current PO
		poName := m.selectedItem
		encoded := url.PathEscape(poName)
		result, err := m.client.Request("GET", "Purchase%20Order/"+encoded, nil)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		data, ok := result["data"].(map[string]interface{})
		if !ok {
			return formSubmittedMsg{false, "PO not found"}
		}

		// Check if draft
		docStatus, _ := data["docstatus"].(float64)
		if docStatus != 0 {
			return formSubmittedMsg{false, "Cannot add items to submitted/cancelled PO"}
		}

		// Get existing items
		var existingItems []map[string]interface{}
		if items, ok := data["items"].([]interface{}); ok {
			for _, item := range items {
				if im, ok := item.(map[string]interface{}); ok {
					existingItems = append(existingItems, im)
				}
			}
		}

		// Add new item
		newItem := map[string]interface{}{
			"item_code":     itemCode,
			"qty":           qty,
			"schedule_date": data["schedule_date"],
		}
		if rate > 0 {
			newItem["rate"] = rate
		}
		existingItems = append(existingItems, newItem)

		// Update PO
		body := map[string]interface{}{
			"items": existingItems,
		}

		_, err = m.client.Request("PUT", "Purchase%20Order/"+encoded, body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		return formSubmittedMsg{true, fmt.Sprintf("Item added to PO: %s", poName)}
	}
}

// submitPO submits a purchase order
func (m Model) submitPO(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.submitDocument("Purchase Order", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("PO submitted: %s", name)}
	}
}

// cancelPO cancels a purchase order
func (m Model) cancelPO(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.cancelDocument("Purchase Order", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("PO cancelled: %s", name)}
	}
}

// loadPurchaseInvoices fetches all purchase invoices
func (m Model) loadPurchaseInvoices() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Purchase%20Invoice?limit_page_length=100&fields=[\"name\",\"supplier\",\"posting_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					supplier := fmt.Sprintf("%v", im["supplier"])
					status, _ := im["status"].(string)
					total, _ := im["grand_total"].(float64)

					statusIcon := ""
					switch status {
					case "Draft":
						statusIcon = "Draft"
					case "Unpaid":
						statusIcon = "Unpaid"
					case "Paid":
						statusIcon = "Paid"
					case "Cancelled":
						statusIcon = "Cancelled"
					default:
						statusIcon = status
					}

					detail := fmt.Sprintf("%s | %s | %s", supplier, statusIcon, m.client.FormatCurrency(total))
					items = append(items, ListItem{name: name, details: detail})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadPIDetail fetches purchase invoice detail
func (m Model) loadPIDetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Purchase%20Invoice/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderPIDetail renders the purchase invoice detail view
func (m Model) renderPIDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Purchase Invoice: "+m.selectedItem) + "\n\n")

	// Basic info
	b.WriteString(fmt.Sprintf("  Supplier: %v\n", m.itemData["supplier"]))
	b.WriteString(fmt.Sprintf("  Date: %v\n", m.itemData["posting_date"]))

	status, _ := m.itemData["status"].(string)
	statusStyle := helpStyle
	switch status {
	case "Draft":
		statusStyle = internetStyle
	case "Paid":
		statusStyle = successStyle
	case "Unpaid":
		statusStyle = errorStyle
	case "Cancelled":
		statusStyle = errorStyle
	}
	b.WriteString(fmt.Sprintf("  Status: %s\n", statusStyle.Render(status)))

	grandTotal, _ := m.itemData["grand_total"].(float64)
	b.WriteString(fmt.Sprintf("  Total: %s\n", m.client.FormatCurrency(grandTotal)))

	if outstanding, ok := m.itemData["outstanding_amount"].(float64); ok && outstanding > 0 {
		b.WriteString(fmt.Sprintf("  Outstanding: %s\n", errorStyle.Render(m.client.FormatCurrency(outstanding))))
	}

	// Items
	if items, ok := m.itemData["items"].([]interface{}); ok && len(items) > 0 {
		b.WriteString(fmt.Sprintf("\n  %s\n", selectedStyle.Render("Items:")))
		for _, item := range items {
			if im, ok := item.(map[string]interface{}); ok {
				itemCode := fmt.Sprintf("%v", im["item_code"])
				qty, _ := im["qty"].(float64)
				rate, _ := im["rate"].(float64)
				amount, _ := im["amount"].(float64)
				po := im["purchase_order"]

				line := fmt.Sprintf("    - %s: %.0f x %s = %s", itemCode, qty, m.client.FormatCurrency(rate), m.client.FormatCurrency(amount))
				if po != nil && po != "" {
					line += fmt.Sprintf(" (PO: %s)", po)
				}
				b.WriteString(line + "\n")
			}
		}
	}

	return boxStyle.Render(b.String())
}

// initCreatePIForm initializes the create PI from PO form
func (m *Model) initCreatePIForm() {
	m.inputs = make([]textinput.Model, 1)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Purchase Order Name (e.g., PUR-ORD-2025-00001)"
	m.inputs[0].Focus()

	m.focusIndex = 0
}

// renderCreatePI renders the create PI form
func (m Model) renderCreatePI() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Invoice from PO ") + "\n\n")

	b.WriteString("  Purchase Order:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	b.WriteString(helpStyle.Render("  Enter the name of a submitted Purchase Order"))

	return boxStyle.Render(b.String())
}

// submitCreatePI submits the create PI form
func (m Model) submitCreatePI() tea.Cmd {
	return func() tea.Msg {
		poName := m.inputs[0].Value()

		if poName == "" {
			return formSubmittedMsg{false, "Purchase Order name is required"}
		}

		// Get the PO
		encoded := url.PathEscape(poName)
		result, err := m.client.Request("GET", "Purchase%20Order/"+encoded, nil)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		poData, ok := result["data"].(map[string]interface{})
		if !ok {
			return formSubmittedMsg{false, "Purchase order not found"}
		}

		// Check if submitted
		docStatus, _ := poData["docstatus"].(float64)
		if docStatus != 1 {
			return formSubmittedMsg{false, "Purchase order must be submitted first"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		today := time.Now().Format("2006-01-02")

		// Build invoice items from PO items
		var invoiceItems []map[string]interface{}
		if items, ok := poData["items"].([]interface{}); ok {
			for _, item := range items {
				if im, ok := item.(map[string]interface{}); ok {
					invoiceItems = append(invoiceItems, map[string]interface{}{
						"item_code":      im["item_code"],
						"qty":            im["qty"],
						"rate":           im["rate"],
						"purchase_order": poName,
						"po_detail":      im["name"],
					})
				}
			}
		}

		if len(invoiceItems) == 0 {
			return formSubmittedMsg{false, "No items found in purchase order"}
		}

		body := map[string]interface{}{
			"supplier":     poData["supplier"],
			"posting_date": today,
			"company":      company,
			"items":        invoiceItems,
		}

		result, err = m.client.Request("POST", "Purchase%20Invoice", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Invoice created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create invoice"}
	}
}

// submitPI submits a purchase invoice
func (m Model) submitPI(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.submitDocument("Purchase Invoice", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Invoice submitted: %s", name)}
	}
}

// cancelPI cancels a purchase invoice
func (m Model) cancelPI(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.cancelDocument("Purchase Invoice", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Invoice cancelled: %s", name)}
	}
}

// handlePurchasingKeys handles keyboard shortcuts for purchasing views
func (m *Model) handlePurchasingKeys(key string) (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewSuppliers:
		switch key {
		case "n":
			m.initCreateSupplierForm()
			m.view = ViewCreateSupplier
			return m, nil
		case "d":
			if item, ok := m.currentList.SelectedItem().(ListItem); ok {
				m.selectedItem = item.name
				m.prevView = m.view
				m.view = ViewConfirmDelete
				return m, nil
			}
		}

	case ViewPurchaseOrders:
		switch key {
		case "n":
			m.initCreatePOForm()
			m.view = ViewCreatePO
			return m, nil
		}

	case ViewPODetail:
		switch key {
		case "a":
			// Add item - only if draft
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.initAddPOItemForm()
					m.view = ViewAddPOItem
					return m, nil
				}
			}
		case "s":
			// Submit
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.confirmAction = "submit_po"
					m.confirmMsg = fmt.Sprintf("Submit PO %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "x":
			// Cancel
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.confirmAction = "cancel_po"
					m.confirmMsg = fmt.Sprintf("Cancel PO %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		}

	case ViewPurchaseInvoices:
		switch key {
		case "n":
			m.initCreatePIForm()
			m.view = ViewCreatePI
			return m, nil
		}

	case ViewPIDetail:
		switch key {
		case "s":
			// Submit
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.confirmAction = "submit_pi"
					m.confirmMsg = fmt.Sprintf("Submit Invoice %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "x":
			// Cancel
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.confirmAction = "cancel_pi"
					m.confirmMsg = fmt.Sprintf("Cancel Invoice %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		}
	}

	return m, nil
}
