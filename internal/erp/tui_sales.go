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

// loadCustomers fetches all customers
func (m Model) loadCustomers() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Customer?limit_page_length=0&fields=[\"name\",\"customer_name\",\"customer_group\",\"disabled\"]", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					group, _ := im["customer_group"].(string)
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

// loadCustomerDetail fetches customer detail
func (m Model) loadCustomerDetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Customer/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderCustomerDetail renders the customer detail view
func (m Model) renderCustomerDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Customer: "+m.selectedItem) + "\n\n")

	fields := []struct {
		key   string
		label string
	}{
		{"customer_name", "Name"},
		{"customer_group", "Group"},
		{"customer_type", "Type"},
		{"territory", "Territory"},
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

// initCreateCustomerForm initializes the create customer form
func (m *Model) initCreateCustomerForm() {
	m.inputs = make([]textinput.Model, 3)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Customer Name"
	m.inputs[0].Focus()

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Customer Group (optional)"

	m.inputs[2] = textinput.New()
	m.inputs[2].Placeholder = "Territory (optional)"

	m.focusIndex = 0
}

// renderCreateCustomer renders the create customer form
func (m Model) renderCreateCustomer() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Customer ") + "\n\n")

	labels := []string{"Name:", "Group:", "Territory:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// submitCreateCustomer submits the create customer form
func (m Model) submitCreateCustomer() tea.Cmd {
	return func() tea.Msg {
		name := m.inputs[0].Value()
		group := m.inputs[1].Value()
		territory := m.inputs[2].Value()

		if name == "" {
			return formSubmittedMsg{false, "Customer name is required"}
		}

		body := map[string]interface{}{
			"customer_name": name,
		}
		if group != "" {
			body["customer_group"] = group
		}
		if territory != "" {
			body["territory"] = territory
		}

		result, err := m.client.Request("POST", "Customer", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Customer created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create customer"}
	}
}

// deleteCustomer deletes a customer
func (m Model) deleteCustomer(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		_, err := m.client.Request("DELETE", "Customer/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}
		return actionDoneMsg{fmt.Sprintf("Deleted: %s", name)}
	}
}

// loadQuotations fetches all quotations
func (m Model) loadQuotations() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Quotation?limit_page_length=100&fields=[\"name\",\"party_name\",\"transaction_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					customer := fmt.Sprintf("%v", im["party_name"])
					status, _ := im["status"].(string)
					total, _ := im["grand_total"].(float64)

					statusBadge := renderStatusBadge(status)
					detail := fmt.Sprintf("%s | %s | %s", customer, statusBadge, m.client.FormatCurrency(total))
					items = append(items, ListItem{name: name, details: detail, amount: total, status: status})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadQuotationDetail fetches quotation detail
func (m Model) loadQuotationDetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Quotation/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderQuotationDetail renders the quotation detail view
func (m Model) renderQuotationDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Quotation: "+m.selectedItem) + "\n\n")

	b.WriteString(fmt.Sprintf("  Customer: %v\n", m.itemData["party_name"]))
	b.WriteString(fmt.Sprintf("  Date: %v\n", m.itemData["transaction_date"]))
	b.WriteString(fmt.Sprintf("  Valid Till: %v\n", m.itemData["valid_till"]))

	status, _ := m.itemData["status"].(string)
	statusStyle := helpStyle
	switch status {
	case "Draft":
		statusStyle = internetStyle
	case "Open", "Submitted":
		statusStyle = vpnStyle
	case "Ordered":
		statusStyle = successStyle
	case "Lost", "Cancelled", "Expired":
		statusStyle = errorStyle
	}
	b.WriteString(fmt.Sprintf("  Status: %s\n", statusStyle.Render(status)))

	grandTotal, _ := m.itemData["grand_total"].(float64)
	b.WriteString(fmt.Sprintf("  Total: %s\n", m.client.FormatCurrency(grandTotal)))

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

// initCreateQuotationForm initializes the create quotation form
func (m *Model) initCreateQuotationForm() {
	m.inputs = make([]textinput.Model, 1)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Customer Name"
	m.inputs[0].Focus()

	m.focusIndex = 0
}

// renderCreateQuotation renders the create quotation form
func (m Model) renderCreateQuotation() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Quotation ") + "\n\n")

	b.WriteString("  Customer:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	b.WriteString(helpStyle.Render("  After creation, add items with 'a' key"))

	return boxStyle.Render(b.String())
}

// submitCreateQuotation submits the create quotation form
func (m Model) submitCreateQuotation() tea.Cmd {
	return func() tea.Msg {
		customer := m.inputs[0].Value()

		if customer == "" {
			return formSubmittedMsg{false, "Customer is required"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		today := time.Now().Format("2006-01-02")
		validTill := time.Now().AddDate(0, 0, 30).Format("2006-01-02")

		body := map[string]interface{}{
			"quotation_to":     "Customer",
			"party_name":       customer,
			"transaction_date": today,
			"valid_till":       validTill,
			"company":          company,
			"items":            []interface{}{},
		}

		result, err := m.client.Request("POST", "Quotation", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Quotation created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create quotation"}
	}
}

// initAddQuotationItemForm initializes the add quotation item form
func (m *Model) initAddQuotationItemForm() {
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

// renderAddQuotationItem renders the add quotation item form
func (m Model) renderAddQuotationItem() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Add Item to Quotation: "+m.selectedItem) + "\n\n")

	labels := []string{"Item Code:", "Quantity:", "Rate:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// submitAddQuotationItem submits the add quotation item form
func (m Model) submitAddQuotationItem() tea.Cmd {
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

		qtnName := m.selectedItem
		encoded := url.PathEscape(qtnName)
		result, err := m.client.Request("GET", "Quotation/"+encoded, nil)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		data, ok := result["data"].(map[string]interface{})
		if !ok {
			return formSubmittedMsg{false, "Quotation not found"}
		}

		docStatus, _ := data["docstatus"].(float64)
		if docStatus != 0 {
			return formSubmittedMsg{false, "Cannot add items to submitted/cancelled quotation"}
		}

		var existingItems []map[string]interface{}
		if items, ok := data["items"].([]interface{}); ok {
			for _, item := range items {
				if im, ok := item.(map[string]interface{}); ok {
					existingItems = append(existingItems, im)
				}
			}
		}

		newItem := map[string]interface{}{
			"item_code": itemCode,
			"qty":       qty,
		}
		if rate > 0 {
			newItem["rate"] = rate
		}
		existingItems = append(existingItems, newItem)

		body := map[string]interface{}{
			"items": existingItems,
		}

		_, err = m.client.Request("PUT", "Quotation/"+encoded, body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		return formSubmittedMsg{true, fmt.Sprintf("Item added to Quotation: %s", qtnName)}
	}
}

// submitQuotation submits a quotation
func (m Model) submitQuotation(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.submitDocument("Quotation", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Quotation submitted: %s", name)}
	}
}

// cancelQuotation cancels a quotation
func (m Model) cancelQuotation(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.cancelDocument("Quotation", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Quotation cancelled: %s", name)}
	}
}

// loadSalesOrders fetches all sales orders
func (m Model) loadSalesOrders() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Sales%20Order?limit_page_length=100&fields=[\"name\",\"customer\",\"transaction_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					customer := fmt.Sprintf("%v", im["customer"])
					status, _ := im["status"].(string)
					total, _ := im["grand_total"].(float64)

					statusBadge := renderStatusBadge(status)
					detail := fmt.Sprintf("%s | %s | %s", customer, statusBadge, m.client.FormatCurrency(total))
					items = append(items, ListItem{name: name, details: detail, amount: total, status: status})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadSODetail fetches sales order detail
func (m Model) loadSODetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Sales%20Order/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderSODetail renders the sales order detail view
func (m Model) renderSODetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Sales Order: "+m.selectedItem) + "\n\n")

	b.WriteString(fmt.Sprintf("  Customer: %v\n", m.itemData["customer"]))
	b.WriteString(fmt.Sprintf("  Date: %v\n", m.itemData["transaction_date"]))
	b.WriteString(fmt.Sprintf("  Delivery Date: %v\n", m.itemData["delivery_date"]))

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

// initCreateSOForm initializes the create SO form
func (m *Model) initCreateSOForm() {
	m.inputs = make([]textinput.Model, 1)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Customer Name"
	m.inputs[0].Focus()

	m.focusIndex = 0
}

// renderCreateSO renders the create SO form
func (m Model) renderCreateSO() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Sales Order ") + "\n\n")

	b.WriteString("  Customer:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	b.WriteString(helpStyle.Render("  After creation, add items with 'a' key"))

	return boxStyle.Render(b.String())
}

// submitCreateSO submits the create SO form
func (m Model) submitCreateSO() tea.Cmd {
	return func() tea.Msg {
		customer := m.inputs[0].Value()

		if customer == "" {
			return formSubmittedMsg{false, "Customer is required"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		today := time.Now().Format("2006-01-02")

		body := map[string]interface{}{
			"customer":         customer,
			"transaction_date": today,
			"delivery_date":    today,
			"company":          company,
			"items":            []interface{}{},
		}

		result, err := m.client.Request("POST", "Sales%20Order", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("SO created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create SO"}
	}
}

// initCreateSOFromQuotationForm initializes the create SO from quotation form
func (m *Model) initCreateSOFromQuotationForm() {
	m.inputs = make([]textinput.Model, 1)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Quotation Name (e.g., QTN-00001)"
	m.inputs[0].Focus()

	m.focusIndex = 0
}

// renderCreateSOFromQuotation renders the create SO from quotation form
func (m Model) renderCreateSOFromQuotation() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create SO from Quotation ") + "\n\n")

	b.WriteString("  Quotation:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	b.WriteString(helpStyle.Render("  Enter the name of a submitted Quotation"))

	return boxStyle.Render(b.String())
}

// submitCreateSOFromQuotation submits the create SO from quotation form
func (m Model) submitCreateSOFromQuotation() tea.Cmd {
	return func() tea.Msg {
		qtnName := m.inputs[0].Value()

		if qtnName == "" {
			return formSubmittedMsg{false, "Quotation name is required"}
		}

		encoded := url.PathEscape(qtnName)
		result, err := m.client.Request("GET", "Quotation/"+encoded, nil)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		qtnData, ok := result["data"].(map[string]interface{})
		if !ok {
			return formSubmittedMsg{false, "Quotation not found"}
		}

		docStatus, _ := qtnData["docstatus"].(float64)
		if docStatus != 1 {
			return formSubmittedMsg{false, "Quotation must be submitted first"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		today := time.Now().Format("2006-01-02")

		var soItems []map[string]interface{}
		if items, ok := qtnData["items"].([]interface{}); ok {
			for _, item := range items {
				if im, ok := item.(map[string]interface{}); ok {
					soItems = append(soItems, map[string]interface{}{
						"item_code":       im["item_code"],
						"qty":             im["qty"],
						"rate":            im["rate"],
						"delivery_date":   today,
						"prevdoc_docname": qtnName,
						"quotation_item":  im["name"],
					})
				}
			}
		}

		if len(soItems) == 0 {
			return formSubmittedMsg{false, "No items found in quotation"}
		}

		body := map[string]interface{}{
			"customer":         qtnData["party_name"],
			"transaction_date": today,
			"delivery_date":    today,
			"company":          company,
			"items":            soItems,
		}

		result, err = m.client.Request("POST", "Sales%20Order", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("SO created: %s (from %s)", data["name"], qtnName)}
		}

		return formSubmittedMsg{false, "Failed to create SO"}
	}
}

// initAddSOItemForm initializes the add SO item form
func (m *Model) initAddSOItemForm() {
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

// renderAddSOItem renders the add SO item form
func (m Model) renderAddSOItem() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Add Item to SO: "+m.selectedItem) + "\n\n")

	labels := []string{"Item Code:", "Quantity:", "Rate:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	return boxStyle.Render(b.String())
}

// submitAddSOItem submits the add SO item form
func (m Model) submitAddSOItem() tea.Cmd {
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

		soName := m.selectedItem
		encoded := url.PathEscape(soName)
		result, err := m.client.Request("GET", "Sales%20Order/"+encoded, nil)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		data, ok := result["data"].(map[string]interface{})
		if !ok {
			return formSubmittedMsg{false, "SO not found"}
		}

		docStatus, _ := data["docstatus"].(float64)
		if docStatus != 0 {
			return formSubmittedMsg{false, "Cannot add items to submitted/cancelled SO"}
		}

		var existingItems []map[string]interface{}
		if items, ok := data["items"].([]interface{}); ok {
			for _, item := range items {
				if im, ok := item.(map[string]interface{}); ok {
					existingItems = append(existingItems, im)
				}
			}
		}

		newItem := map[string]interface{}{
			"item_code":     itemCode,
			"qty":           qty,
			"delivery_date": data["delivery_date"],
		}
		if rate > 0 {
			newItem["rate"] = rate
		}
		existingItems = append(existingItems, newItem)

		body := map[string]interface{}{
			"items": existingItems,
		}

		_, err = m.client.Request("PUT", "Sales%20Order/"+encoded, body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		return formSubmittedMsg{true, fmt.Sprintf("Item added to SO: %s", soName)}
	}
}

// submitSO submits a sales order
func (m Model) submitSO(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.submitDocument("Sales Order", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("SO submitted: %s", name)}
	}
}

// cancelSO cancels a sales order
func (m Model) cancelSO(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.cancelDocument("Sales Order", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("SO cancelled: %s", name)}
	}
}

// loadSalesInvoices fetches all sales invoices
func (m Model) loadSalesInvoices() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Sales%20Invoice?limit_page_length=100&fields=[\"name\",\"customer\",\"posting_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					customer := fmt.Sprintf("%v", im["customer"])
					status, _ := im["status"].(string)
					total, _ := im["grand_total"].(float64)

					statusBadge := renderStatusBadge(status)
					detail := fmt.Sprintf("%s | %s | %s", customer, statusBadge, m.client.FormatCurrency(total))
					items = append(items, ListItem{name: name, details: detail, amount: total, status: status})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadSIDetail fetches sales invoice detail
func (m Model) loadSIDetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Sales%20Invoice/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderSIDetail renders the sales invoice detail view
func (m Model) renderSIDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Sales Invoice: "+m.selectedItem) + "\n\n")

	b.WriteString(fmt.Sprintf("  Customer: %v\n", m.itemData["customer"]))
	b.WriteString(fmt.Sprintf("  Date: %v\n", m.itemData["posting_date"]))

	status, _ := m.itemData["status"].(string)
	statusStyle := helpStyle
	switch status {
	case "Draft":
		statusStyle = internetStyle
	case "Paid":
		statusStyle = successStyle
	case "Unpaid", "Overdue":
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

	if items, ok := m.itemData["items"].([]interface{}); ok && len(items) > 0 {
		b.WriteString(fmt.Sprintf("\n  %s\n", selectedStyle.Render("Items:")))
		for _, item := range items {
			if im, ok := item.(map[string]interface{}); ok {
				itemCode := fmt.Sprintf("%v", im["item_code"])
				qty, _ := im["qty"].(float64)
				rate, _ := im["rate"].(float64)
				amount, _ := im["amount"].(float64)
				so := im["sales_order"]

				line := fmt.Sprintf("    - %s: %.0f x %s = %s", itemCode, qty, m.client.FormatCurrency(rate), m.client.FormatCurrency(amount))
				if so != nil && so != "" {
					line += fmt.Sprintf(" (SO: %s)", so)
				}
				b.WriteString(line + "\n")
			}
		}
	}

	return boxStyle.Render(b.String())
}

// initCreateSIForm initializes the create SI from SO form
func (m *Model) initCreateSIForm() {
	m.inputs = make([]textinput.Model, 1)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Sales Order Name (e.g., SAL-ORD-2025-00001)"
	m.inputs[0].Focus()

	m.focusIndex = 0
}

// renderCreateSI renders the create SI form
func (m Model) renderCreateSI() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Invoice from SO ") + "\n\n")

	b.WriteString("  Sales Order:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	b.WriteString(helpStyle.Render("  Enter the name of a submitted Sales Order"))

	return boxStyle.Render(b.String())
}

// submitCreateSI submits the create SI form
func (m Model) submitCreateSI() tea.Cmd {
	return func() tea.Msg {
		soName := m.inputs[0].Value()

		if soName == "" {
			return formSubmittedMsg{false, "Sales Order name is required"}
		}

		encoded := url.PathEscape(soName)
		result, err := m.client.Request("GET", "Sales%20Order/"+encoded, nil)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		soData, ok := result["data"].(map[string]interface{})
		if !ok {
			return formSubmittedMsg{false, "Sales order not found"}
		}

		docStatus, _ := soData["docstatus"].(float64)
		if docStatus != 1 {
			return formSubmittedMsg{false, "Sales order must be submitted first"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		today := time.Now().Format("2006-01-02")

		var invoiceItems []map[string]interface{}
		if items, ok := soData["items"].([]interface{}); ok {
			for _, item := range items {
				if im, ok := item.(map[string]interface{}); ok {
					invoiceItems = append(invoiceItems, map[string]interface{}{
						"item_code":   im["item_code"],
						"qty":         im["qty"],
						"rate":        im["rate"],
						"sales_order": soName,
						"so_detail":   im["name"],
					})
				}
			}
		}

		if len(invoiceItems) == 0 {
			return formSubmittedMsg{false, "No items found in sales order"}
		}

		body := map[string]interface{}{
			"customer":     soData["customer"],
			"posting_date": today,
			"company":      company,
			"items":        invoiceItems,
		}

		result, err = m.client.Request("POST", "Sales%20Invoice", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Invoice created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create invoice"}
	}
}

// submitSI submits a sales invoice
func (m Model) submitSI(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.submitDocument("Sales Invoice", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Invoice submitted: %s", name)}
	}
}

// cancelSI cancels a sales invoice
func (m Model) cancelSI(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.cancelDocument("Sales Invoice", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Invoice cancelled: %s", name)}
	}
}

// loadDeliveryNotes fetches all delivery notes
func (m Model) loadDeliveryNotes() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Delivery%20Note?limit_page_length=100&fields=[\"name\",\"customer\",\"posting_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					customer := fmt.Sprintf("%v", im["customer"])
					status, _ := im["status"].(string)
					total, _ := im["grand_total"].(float64)

					statusBadge := renderStatusBadge(status)
					detail := fmt.Sprintf("%s | %s | %s", customer, statusBadge, m.client.FormatCurrency(total))
					items = append(items, ListItem{name: name, details: detail, amount: total, status: status})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadDNDetail fetches delivery note detail
func (m Model) loadDNDetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Delivery%20Note/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderDNDetail renders the delivery note detail view
func (m Model) renderDNDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Delivery Note: "+m.selectedItem) + "\n\n")

	b.WriteString(fmt.Sprintf("  Customer: %v\n", m.itemData["customer"]))
	b.WriteString(fmt.Sprintf("  Date: %v\n", m.itemData["posting_date"]))

	status, _ := m.itemData["status"].(string)
	statusStyle := helpStyle
	switch status {
	case "Draft":
		statusStyle = internetStyle
	case "Completed":
		statusStyle = successStyle
	case "Cancelled", "Return Issued":
		statusStyle = errorStyle
	}
	b.WriteString(fmt.Sprintf("  Status: %s\n", statusStyle.Render(status)))

	grandTotal, _ := m.itemData["grand_total"].(float64)
	b.WriteString(fmt.Sprintf("  Total: %s\n", m.client.FormatCurrency(grandTotal)))

	if items, ok := m.itemData["items"].([]interface{}); ok && len(items) > 0 {
		b.WriteString(fmt.Sprintf("\n  %s\n", selectedStyle.Render("Items:")))
		for _, item := range items {
			if im, ok := item.(map[string]interface{}); ok {
				itemCode := fmt.Sprintf("%v", im["item_code"])
				qty, _ := im["qty"].(float64)
				rate, _ := im["rate"].(float64)
				amount, _ := im["amount"].(float64)
				so := im["against_sales_order"]

				line := fmt.Sprintf("    - %s: %.0f x %s = %s", itemCode, qty, m.client.FormatCurrency(rate), m.client.FormatCurrency(amount))
				if so != nil && so != "" {
					line += fmt.Sprintf(" (SO: %s)", so)
				}
				b.WriteString(line + "\n")
			}
		}
	}

	return boxStyle.Render(b.String())
}

// initCreateDNForm initializes the create DN from SO form
func (m *Model) initCreateDNForm() {
	m.inputs = make([]textinput.Model, 1)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Sales Order Name (e.g., SAL-ORD-2025-00001)"
	m.inputs[0].Focus()

	m.focusIndex = 0
}

// renderCreateDN renders the create DN form
func (m Model) renderCreateDN() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Delivery Note from SO ") + "\n\n")

	b.WriteString("  Sales Order:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	b.WriteString(helpStyle.Render("  Enter the name of a submitted Sales Order"))

	return boxStyle.Render(b.String())
}

// submitCreateDN submits the create DN form
func (m Model) submitCreateDN() tea.Cmd {
	return func() tea.Msg {
		soName := m.inputs[0].Value()

		if soName == "" {
			return formSubmittedMsg{false, "Sales Order name is required"}
		}

		encoded := url.PathEscape(soName)
		result, err := m.client.Request("GET", "Sales%20Order/"+encoded, nil)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		soData, ok := result["data"].(map[string]interface{})
		if !ok {
			return formSubmittedMsg{false, "Sales order not found"}
		}

		docStatus, _ := soData["docstatus"].(float64)
		if docStatus != 1 {
			return formSubmittedMsg{false, "Sales order must be submitted first"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		today := time.Now().Format("2006-01-02")

		var dnItems []map[string]interface{}
		if items, ok := soData["items"].([]interface{}); ok {
			for _, item := range items {
				if im, ok := item.(map[string]interface{}); ok {
					dnItems = append(dnItems, map[string]interface{}{
						"item_code":           im["item_code"],
						"qty":                 im["qty"],
						"rate":                im["rate"],
						"against_sales_order": soName,
						"so_detail":           im["name"],
					})
				}
			}
		}

		if len(dnItems) == 0 {
			return formSubmittedMsg{false, "No items found in sales order"}
		}

		body := map[string]interface{}{
			"customer":     soData["customer"],
			"posting_date": today,
			"company":      company,
			"items":        dnItems,
		}

		result, err = m.client.Request("POST", "Delivery%20Note", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Delivery Note created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create delivery note"}
	}
}

// submitDN submits a delivery note
func (m Model) submitDN(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.submitDocument("Delivery Note", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Delivery Note submitted: %s", name)}
	}
}

// cancelDN cancels a delivery note
func (m Model) cancelDN(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.cancelDocument("Delivery Note", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Delivery Note cancelled: %s", name)}
	}
}

// loadPayments fetches all payment entries
func (m Model) loadPayments() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.Request("GET", "Payment%20Entry?limit_page_length=100&fields=[\"name\",\"payment_type\",\"party_type\",\"party\",\"paid_amount\",\"posting_date\",\"status\",\"docstatus\"]&order_by=creation%20desc", nil)
		if err != nil {
			return errorMsg{err}
		}

		var items []ListItem
		if data, ok := result["data"].([]interface{}); ok {
			for _, item := range data {
				if im, ok := item.(map[string]interface{}); ok {
					name := fmt.Sprintf("%v", im["name"])
					paymentType, _ := im["payment_type"].(string)
					party := fmt.Sprintf("%v", im["party"])
					status, _ := im["status"].(string)
					amount, _ := im["paid_amount"].(float64)

					typeIcon := "↓"
					if paymentType == "Pay" {
						typeIcon = "↑"
					}

					statusBadge := renderStatusBadge(status)
					detail := fmt.Sprintf("%s %s | %s | %s", typeIcon, party, statusBadge, m.client.FormatCurrency(amount))
					items = append(items, ListItem{name: name, details: detail, amount: amount, status: status})
				}
			}
		}
		return dataLoadedMsg{items}
	}
}

// loadPaymentDetail fetches payment entry detail
func (m Model) loadPaymentDetail(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		result, err := m.client.Request("GET", "Payment%20Entry/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return itemDetailMsg{data}
		}
		return errorMsg{fmt.Errorf("no data found")}
	}
}

// renderPaymentDetail renders the payment entry detail view
func (m Model) renderPaymentDetail() string {
	if m.loading {
		return "\n  Loading..."
	}

	if m.itemData == nil {
		return "\n  No data"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(" Payment Entry: "+m.selectedItem) + "\n\n")

	paymentType, _ := m.itemData["payment_type"].(string)
	typeStr := "Receive (from Customer)"
	if paymentType == "Pay" {
		typeStr = "Pay (to Supplier)"
	}
	b.WriteString(fmt.Sprintf("  Type: %s\n", typeStr))
	b.WriteString(fmt.Sprintf("  Party Type: %v\n", m.itemData["party_type"]))
	b.WriteString(fmt.Sprintf("  Party: %v\n", m.itemData["party"]))
	b.WriteString(fmt.Sprintf("  Date: %v\n", m.itemData["posting_date"]))

	status, _ := m.itemData["status"].(string)
	statusStyle := helpStyle
	switch status {
	case "Draft":
		statusStyle = internetStyle
	case "Submitted":
		statusStyle = successStyle
	case "Cancelled":
		statusStyle = errorStyle
	}
	b.WriteString(fmt.Sprintf("  Status: %s\n", statusStyle.Render(status)))

	paidAmount, _ := m.itemData["paid_amount"].(float64)
	b.WriteString(fmt.Sprintf("  Paid Amount: %s\n", m.client.FormatCurrency(paidAmount)))

	if mop, ok := m.itemData["mode_of_payment"]; ok && mop != nil && mop != "" {
		b.WriteString(fmt.Sprintf("  Mode of Payment: %v\n", mop))
	}

	if refs, ok := m.itemData["references"].([]interface{}); ok && len(refs) > 0 {
		b.WriteString(fmt.Sprintf("\n  %s\n", selectedStyle.Render("References:")))
		for _, ref := range refs {
			if r, ok := ref.(map[string]interface{}); ok {
				refDoctype := r["reference_doctype"]
				refName := r["reference_name"]
				allocated, _ := r["allocated_amount"].(float64)
				b.WriteString(fmt.Sprintf("    - %s: %s (Allocated: %s)\n", refDoctype, refName, m.client.FormatCurrency(allocated)))
			}
		}
	}

	return boxStyle.Render(b.String())
}

// initCreatePaymentForm initializes the create payment form
func (m *Model) initCreatePaymentForm(invoiceName string, paymentType string) {
	m.inputs = make([]textinput.Model, 2)

	m.inputs[0] = textinput.New()
	if paymentType == "Receive" {
		m.inputs[0].Placeholder = "Sales Invoice Name"
	} else {
		m.inputs[0].Placeholder = "Purchase Invoice Name"
	}
	m.inputs[0].SetValue(invoiceName)
	m.inputs[0].Focus()

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Amount (leave empty for full amount)"

	m.focusIndex = 0
	m.formData["payment_type"] = paymentType
}

// renderCreatePayment renders the create payment form
func (m Model) renderCreatePayment() string {
	var b strings.Builder

	paymentType := m.formData["payment_type"]
	if paymentType == "Receive" {
		b.WriteString(titleStyle.Render(" Create Payment (Receive) ") + "\n\n")
		b.WriteString("  Sales Invoice:\n")
	} else {
		b.WriteString(titleStyle.Render(" Create Payment (Pay) ") + "\n\n")
		b.WriteString("  Purchase Invoice:\n")
	}
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	b.WriteString("  Amount (optional):\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[1].View()))

	b.WriteString(helpStyle.Render("  Leave amount empty to pay the full outstanding balance"))

	return boxStyle.Render(b.String())
}

// submitCreatePayment submits the create payment form
func (m Model) submitCreatePayment() tea.Cmd {
	return func() tea.Msg {
		invoiceName := m.inputs[0].Value()
		amountStr := m.inputs[1].Value()
		paymentType := m.formData["payment_type"]

		if invoiceName == "" {
			return formSubmittedMsg{false, "Invoice name is required"}
		}

		amount := 0.0
		if amountStr != "" {
			var err error
			amount, err = strconv.ParseFloat(amountStr, 64)
			if err != nil {
				return formSubmittedMsg{false, "Invalid amount"}
			}
		}

		// Determine invoice type based on payment type
		isReceive := paymentType == "Receive"
		invoiceDoctype := "Purchase%20Invoice"
		invoiceLabel := "Purchase invoice"
		partyType := "Supplier"
		partyField := "supplier"
		refDoctype := "Purchase Invoice"
		if isReceive {
			invoiceDoctype = "Sales%20Invoice"
			invoiceLabel = "Sales invoice"
			partyType = "Customer"
			partyField = "customer"
			refDoctype = "Sales Invoice"
		}

		// Get the invoice
		encoded := url.PathEscape(invoiceName)
		result, err := m.client.Request("GET", invoiceDoctype+"/"+encoded, nil)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		invoiceData, ok := result["data"].(map[string]interface{})
		if !ok {
			return formSubmittedMsg{false, invoiceLabel + " not found"}
		}

		docStatus, _ := invoiceData["docstatus"].(float64)
		if docStatus != 1 {
			return formSubmittedMsg{false, invoiceLabel + " must be submitted first"}
		}

		outstanding, _ := invoiceData["outstanding_amount"].(float64)
		if outstanding <= 0 {
			return formSubmittedMsg{false, invoiceLabel + " has no outstanding amount"}
		}

		paidAmount := outstanding
		if amount > 0 {
			if amount > outstanding {
				return formSubmittedMsg{false, fmt.Sprintf("Amount exceeds outstanding balance of %s", m.client.FormatCurrency(outstanding))}
			}
			paidAmount = amount
		}

		party, _ := invoiceData[partyField].(string)
		grandTotal, _ := invoiceData["grand_total"].(float64)

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		body := map[string]interface{}{
			"payment_type": paymentType,
			"party_type":   partyType,
			"party":        party,
			"paid_amount":  paidAmount,
			"posting_date": time.Now().Format("2006-01-02"),
			"company":      company,
			"references": []map[string]interface{}{
				{
					"reference_doctype":  refDoctype,
					"reference_name":     invoiceName,
					"total_amount":       grandTotal,
					"outstanding_amount": outstanding,
					"allocated_amount":   paidAmount,
				},
			},
		}

		result, err = m.client.Request("POST", "Payment%20Entry", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Payment created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create payment"}
	}
}

// submitPayment submits a payment entry
func (m Model) submitPayment(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.submitDocument("Payment Entry", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Payment submitted: %s", name)}
	}
}

// cancelPayment cancels a payment entry
func (m Model) cancelPayment(name string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.cancelDocument("Payment Entry", name)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}
		return formSubmittedMsg{true, fmt.Sprintf("Payment cancelled: %s", name)}
	}
}

// handleSalesKeys handles keyboard shortcuts for sales views
func (m *Model) handleSalesKeys(key string) (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewCustomers:
		switch key {
		case "n":
			m.initCreateCustomerForm()
			m.view = ViewCreateCustomer
			return m, nil
		case "d":
			if item, ok := m.currentList.SelectedItem().(ListItem); ok {
				m.selectedItem = item.name
				m.prevView = m.view
				m.view = ViewConfirmDelete
				return m, nil
			}
		}

	case ViewQuotations:
		switch key {
		case "n":
			m.initCreateQuotationForm()
			m.view = ViewCreateQuotation
			return m, nil
		}

	case ViewQuotationDetail:
		switch key {
		case "a":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.initAddQuotationItemForm()
					m.view = ViewAddQuotationItem
					return m, nil
				}
			}
		case "s":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.confirmAction = "submit_quotation"
					m.confirmMsg = fmt.Sprintf("Submit Quotation %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "x":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.confirmAction = "cancel_quotation"
					m.confirmMsg = fmt.Sprintf("Cancel Quotation %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "o":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.initCreateSOFromQuotationForm()
					m.inputs[0].SetValue(m.selectedItem)
					m.view = ViewCreateSOFromQuotation
					return m, nil
				}
			}
		}

	case ViewSalesOrders:
		switch key {
		case "n":
			m.initCreateSOForm()
			m.view = ViewCreateSO
			return m, nil
		case "q":
			m.initCreateSOFromQuotationForm()
			m.view = ViewCreateSOFromQuotation
			return m, nil
		}

	case ViewSODetail:
		switch key {
		case "a":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.initAddSOItemForm()
					m.view = ViewAddSOItem
					return m, nil
				}
			}
		case "s":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.confirmAction = "submit_so"
					m.confirmMsg = fmt.Sprintf("Submit SO %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "x":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.confirmAction = "cancel_so"
					m.confirmMsg = fmt.Sprintf("Cancel SO %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "i":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.initCreateSIForm()
					m.inputs[0].SetValue(m.selectedItem)
					m.view = ViewCreateSalesInvoice
					return m, nil
				}
			}
		case "r":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.initCreateDNForm()
					m.inputs[0].SetValue(m.selectedItem)
					m.view = ViewCreateDN
					return m, nil
				}
			}
		}

	case ViewDeliveryNotes:
		switch key {
		case "n":
			m.initCreateDNForm()
			m.view = ViewCreateDN
			return m, nil
		}

	case ViewDNDetail:
		switch key {
		case "s":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.confirmAction = "submit_dn"
					m.confirmMsg = fmt.Sprintf("Submit Delivery Note %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "x":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.confirmAction = "cancel_dn"
					m.confirmMsg = fmt.Sprintf("Cancel Delivery Note %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		}

	case ViewSalesInvoices:
		switch key {
		case "n":
			m.initCreateSIForm()
			m.view = ViewCreateSalesInvoice
			return m, nil
		}

	case ViewSIDetail:
		switch key {
		case "s":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.confirmAction = "submit_si"
					m.confirmMsg = fmt.Sprintf("Submit Invoice %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "x":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.confirmAction = "cancel_si"
					m.confirmMsg = fmt.Sprintf("Cancel Invoice %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "p":
			// Create payment for submitted invoice with outstanding amount
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					if outstanding, ok := m.itemData["outstanding_amount"].(float64); ok && outstanding > 0 {
						m.initCreatePaymentForm(m.selectedItem, "Receive")
						m.prevView = m.view
						m.view = ViewCreatePayment
						return m, nil
					}
				}
			}
		}

	case ViewPaymentDetail:
		switch key {
		case "s":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 0 {
					m.confirmAction = "submit_payment"
					m.confirmMsg = fmt.Sprintf("Submit Payment %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		case "x":
			if m.itemData != nil {
				if docStatus, ok := m.itemData["docstatus"].(float64); ok && docStatus == 1 {
					m.confirmAction = "cancel_payment"
					m.confirmMsg = fmt.Sprintf("Cancel Payment %s?", m.selectedItem)
					m.prevView = m.view
					m.view = ViewConfirmAction
					return m, nil
				}
			}
		}
	}

	return m, nil
}
