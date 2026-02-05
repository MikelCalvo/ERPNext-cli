package erp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// loadWarehouses fetches all warehouses
func (m Model) loadWarehouses() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Warehouse?limit_page_length=0&fields=[\"name\",\"warehouse_name\",\"is_group\",\"parent_warehouse\"]", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					isGroup, _ := im["is_group"].(float64)
					parent := im["parent_warehouse"]

					var detail string
					if isGroup == 1 {
						detail = "Group"
					} else {
						detail = "Warehouse"
					}
					if parent != nil && parent != "" {
						detail += fmt.Sprintf(" (parent: %s)", parent)
					}
					items = append(items, ListItem{name: name, details: detail})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadStock fetches all items with stock
func (m Model) loadStock() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Item?limit_page_length=0&fields=[\"name\",\"item_name\",\"stock_uom\"]", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					itemName := fmt.Sprintf("%v", im["item_name"])
					uom := fmt.Sprintf("%v", im["stock_uom"])
					detail := fmt.Sprintf("%s (%s)", itemName, uom)
					items = append(items, ListItem{name: name, details: detail})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadStockDetail fetches stock detail for an item
func (m Model) loadStockDetail(itemCode string) tea.Cmd {
	return func() tea.Msg {
		filters := [][]interface{}{
			{"item_code", "=", itemCode},
		}
		encodedFilter, err := encodeFilters(filters)
		if err != nil {
			return errorMsg{err}
		}

		result, err := m.client.Request("GET", "Bin?filters="+encodedFilter+"&fields=[\"warehouse\",\"actual_qty\",\"reserved_qty\",\"ordered_qty\",\"stock_value\"]", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []map[string]interface{}
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					items = append(items, im)
				}
			}
		}
		return stockDataMsg{items}
	}
}

// renderStockDetail renders the stock detail view
func (m Model) renderStockDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Stock: "+m.selectedItem) + "\n\n")

	if len(m.listData) == 0 {
		b.WriteString("  No stock found for this item\n")
	} else {
		totalQty := 0.0
		totalValue := 0.0

		for _, bin := range m.listData {
			wh := fmt.Sprintf("%v", bin["warehouse"])
			actualQty, _ := bin["actual_qty"].(float64)
			reservedQty, _ := bin["reserved_qty"].(float64)
			orderedQty, _ := bin["ordered_qty"].(float64)
			stockValue, _ := bin["stock_value"].(float64)

			totalQty += actualQty
			totalValue += stockValue

			b.WriteString(fmt.Sprintf("  %s %s\n", boxStyle.BorderForeground(nil).Render(""), wh))
			b.WriteString(fmt.Sprintf("     Actual: %.0f", actualQty))
			if reservedQty > 0 {
				b.WriteString(fmt.Sprintf(" | Reserved: %.0f", reservedQty))
			}
			if orderedQty > 0 {
				b.WriteString(fmt.Sprintf(" | Ordered: %.0f", orderedQty))
			}
			if stockValue > 0 {
				b.WriteString(fmt.Sprintf(" | Value: %s", m.client.FormatCurrency(stockValue)))
			}
			b.WriteString("\n")
		}

		if len(m.listData) > 1 {
			b.WriteString(fmt.Sprintf("\n  Total: %.0f units | Value: %s\n", totalQty, m.client.FormatCurrency(totalValue)))
		}
	}

	return boxStyle.Render(b.String())
}

// loadSerials fetches serial numbers
func (m Model) loadSerials(itemCode string) tea.Cmd {
	return func() tea.Msg {
		endpoint := "Serial%20No?limit_page_length=100&fields=[\"name\",\"item_code\",\"warehouse\",\"status\"]&order_by=creation%20desc"
		if itemCode != "" {
			filters := [][]interface{}{
				{"item_code", "=", itemCode},
			}
			encodedFilter, err := encodeFilters(filters)
			if err != nil {
				return errorMsg{err}
			}
			endpoint += "&filters=" + encodedFilter
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
					status, _ := im["status"].(string)
					itemCode := fmt.Sprintf("%v", im["item_code"])

					statusIcon := ""
					switch status {
					case "Active":
						statusIcon = "Active"
					case "Delivered":
						statusIcon = "Delivered"
					default:
						statusIcon = status
					}

					detail := fmt.Sprintf("%s | %s", itemCode, statusIcon)
					items = append(items, ListItem{name: name, details: detail})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadSerialDetail fetches serial number detail
func (m Model) loadSerialDetail(serialNo string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(serialNo)
		result, err := m.client.Request("GET", "Serial%20No/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderSerialDetail renders the serial number detail view
func (m Model) renderSerialDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Serial: "+m.selectedItem) + "\n\n")

	fields := []struct {
		key   string
		label string
	}{
		{"item_code", "Item"},
		{"item_name", "Item Name"},
		{"status", "Status"},
		{"warehouse", "Warehouse"},
		{"batch_no", "Batch"},
		{"supplier", "Supplier"},
		{"purchase_date", "Purchase Date"},
		{"warranty_expiry_date", "Warranty Expiry"},
	}

	for _, f := range fields {
		if val, ok := m.itemData[f.key]; ok && val != nil && val != "" {
			b.WriteString(fmt.Sprintf("  %s: %v\n", f.label, val))
		}
	}

	return boxStyle.Render(b.String())
}

// initStockReceiveForm initializes the stock receive form
func (m *Model) initStockReceiveForm() {
	m.inputs = make([]textinput.Model, 4)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Item Code"
	m.inputs[0].Focus()
	m.inputs[0].SetValue(m.selectedItem)

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Quantity"

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "Warehouse"

	m.inputs[3] = textinput.New()
	m.inputs[3].Placeholder = "Rate (optional)"

	m.focusIndex = 1 // Start at quantity since item is pre-filled
}

// initStockTransferForm initializes the stock transfer form
func (m *Model) initStockTransferForm() {
	m.inputs = make([]textinput.Model, 4)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Item Code"
	m.inputs[0].Focus()
	m.inputs[0].SetValue(m.selectedItem)

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Quantity"

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "From Warehouse"

	m.inputs[3] = textinput.New()
	m.inputs[3].Placeholder = "To Warehouse"

	m.focusIndex = 1
}

// initStockIssueForm initializes the stock issue form
func (m *Model) initStockIssueForm() {
	m.inputs = make([]textinput.Model, 3)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Item Code"
	m.inputs[0].Focus()
	m.inputs[0].SetValue(m.selectedItem)

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Quantity"

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "Warehouse"

	m.focusIndex = 1
}

// initCreateSerialForm initializes the create serial form
func (m *Model) initCreateSerialForm() {
	m.inputs = make([]textinput.Model, 3)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Serial Number"
	m.inputs[0].Focus()

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Item Code"

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "Supplier (optional)"

	m.focusIndex = 0
}

// renderStockReceive renders the stock receive form
func (m Model) renderStockReceive() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Receive Stock ") + "\n\n")

	labels := []string{"Item Code:", "Quantity:", "Warehouse:", "Rate:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// renderStockTransfer renders the stock transfer form
func (m Model) renderStockTransfer() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Transfer Stock ") + "\n\n")

	labels := []string{"Item Code:", "Quantity:", "From Warehouse:", "To Warehouse:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// renderStockIssue renders the stock issue form
func (m Model) renderStockIssue() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Issue Stock ") + "\n\n")

	labels := []string{"Item Code:", "Quantity:", "Warehouse:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// renderCreateSerial renders the create serial form
func (m Model) renderCreateSerial() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Serial Number ") + "\n\n")

	labels := []string{"Serial Number:", "Item Code:", "Supplier:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// submitStockReceive submits the stock receive form
func (m Model) submitStockReceive() tea.Cmd {
	return func() tea.Msg {
		itemCode := m.inputs[0].Value()
		qtyStr := m.inputs[1].Value()
		warehouse := m.inputs[2].Value()
		rateStr := m.inputs[3].Value()

		if itemCode == "" || qtyStr == "" || warehouse == "" {
			return formSubmittedMsg{false, "Item, quantity and warehouse are required"}
		}

		qty, err := strconv.ParseFloat(qtyStr, 64)
		if err != nil {
			return formSubmittedMsg{false, "Invalid quantity"}
		}

		rate := 0.0
		if rateStr != "" {
			rate, _ = strconv.ParseFloat(rateStr, 64)
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		item := map[string]interface{}{
			"item_code":   itemCode,
			"qty":         qty,
			"t_warehouse": warehouse,
		}
		if rate > 0 {
			item["basic_rate"] = rate
		}

		body := map[string]interface{}{
			"stock_entry_type": "Material Receipt",
			"company":          company,
			"items":            []interface{}{item},
		}

		result, err := m.client.Request("POST", "Stock%20Entry", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			entryName := fmt.Sprintf("%v", data["name"])
			m.client.submitStockEntry(entryName)
			return formSubmittedMsg{true, fmt.Sprintf("Stock received: %s", entryName)}
		}

		return formSubmittedMsg{false, "Failed to create stock entry"}
	}
}

// submitStockTransfer submits the stock transfer form
func (m Model) submitStockTransfer() tea.Cmd {
	return func() tea.Msg {
		itemCode := m.inputs[0].Value()
		qtyStr := m.inputs[1].Value()
		fromWarehouse := m.inputs[2].Value()
		toWarehouse := m.inputs[3].Value()

		if itemCode == "" || qtyStr == "" || fromWarehouse == "" || toWarehouse == "" {
			return formSubmittedMsg{false, "All fields are required"}
		}

		qty, err := strconv.ParseFloat(qtyStr, 64)
		if err != nil {
			return formSubmittedMsg{false, "Invalid quantity"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		body := map[string]interface{}{
			"stock_entry_type": "Material Transfer",
			"company":          company,
			"items": []interface{}{
				map[string]interface{}{
					"item_code":   itemCode,
					"qty":         qty,
					"s_warehouse": fromWarehouse,
					"t_warehouse": toWarehouse,
				},
			},
		}

		result, err := m.client.Request("POST", "Stock%20Entry", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			entryName := fmt.Sprintf("%v", data["name"])
			m.client.submitStockEntry(entryName)
			return formSubmittedMsg{true, fmt.Sprintf("Stock transferred: %s", entryName)}
		}

		return formSubmittedMsg{false, "Failed to create stock entry"}
	}
}

// submitStockIssue submits the stock issue form
func (m Model) submitStockIssue() tea.Cmd {
	return func() tea.Msg {
		itemCode := m.inputs[0].Value()
		qtyStr := m.inputs[1].Value()
		warehouse := m.inputs[2].Value()

		if itemCode == "" || qtyStr == "" || warehouse == "" {
			return formSubmittedMsg{false, "All fields are required"}
		}

		qty, err := strconv.ParseFloat(qtyStr, 64)
		if err != nil {
			return formSubmittedMsg{false, "Invalid quantity"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		body := map[string]interface{}{
			"stock_entry_type": "Material Issue",
			"company":          company,
			"items": []interface{}{
				map[string]interface{}{
					"item_code":   itemCode,
					"qty":         qty,
					"s_warehouse": warehouse,
				},
			},
		}

		result, err := m.client.Request("POST", "Stock%20Entry", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			entryName := fmt.Sprintf("%v", data["name"])
			m.client.submitStockEntry(entryName)
			return formSubmittedMsg{true, fmt.Sprintf("Stock issued: %s", entryName)}
		}

		return formSubmittedMsg{false, "Failed to create stock entry"}
	}
}

// submitCreateSerial submits the create serial form
func (m Model) submitCreateSerial() tea.Cmd {
	return func() tea.Msg {
		serialNo := m.inputs[0].Value()
		itemCode := m.inputs[1].Value()
		supplier := m.inputs[2].Value()

		if serialNo == "" || itemCode == "" {
			return formSubmittedMsg{false, "Serial number and item code are required"}
		}

		body := map[string]interface{}{
			"serial_no": serialNo,
			"item_code": itemCode,
		}
		if supplier != "" {
			body["supplier"] = supplier
		}

		_, err := m.client.Request("POST", "Serial%20No", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		return formSubmittedMsg{true, fmt.Sprintf("Serial number created: %s", serialNo)}
	}
}

// deleteSerial deletes a serial number
func (m Model) deleteSerial(serialNo string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(serialNo)
		_, err := m.client.Request("DELETE", "Serial%20No/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}
		return actionDoneMsg{fmt.Sprintf("Deleted: %s", serialNo)}
	}
}

// handleStockKeys handles keyboard shortcuts for stock views
func (m *Model) handleStockKeys(key string) (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewStock:
		switch key {
		case "r":
			if item, ok := m.currentList.SelectedItem().(ListItem); ok {
				m.selectedItem = item.name
				m.initStockReceiveForm()
				m.view = ViewStockReceive
				return m, nil
			}
		case "t":
			if item, ok := m.currentList.SelectedItem().(ListItem); ok {
				m.selectedItem = item.name
				m.initStockTransferForm()
				m.view = ViewStockTransfer
				return m, nil
			}
		case "i":
			if item, ok := m.currentList.SelectedItem().(ListItem); ok {
				m.selectedItem = item.name
				m.initStockIssueForm()
				m.view = ViewStockIssue
				return m, nil
			}
		}

	case ViewStockDetail:
		switch key {
		case "r":
			m.initStockReceiveForm()
			m.view = ViewStockReceive
			return m, nil
		case "t":
			m.initStockTransferForm()
			m.view = ViewStockTransfer
			return m, nil
		case "i":
			m.initStockIssueForm()
			m.view = ViewStockIssue
			return m, nil
		}

	case ViewSerials:
		switch key {
		case "n":
			m.initCreateSerialForm()
			m.view = ViewCreateSerial
			return m, nil
		case "d":
			if item, ok := m.currentList.SelectedItem().(ListItem); ok {
				m.selectedItem = item.name
				m.prevView = m.view
				m.view = ViewConfirmDelete
				return m, nil
			}
		}
	}

	return m, nil
}
