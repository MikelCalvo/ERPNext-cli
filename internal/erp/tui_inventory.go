package erp

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// =============================================================================
// CREATE GROUP
// =============================================================================

// initCreateGroupForm initializes the create group form
func (m *Model) initCreateGroupForm() {
	m.inputs = make([]textinput.Model, 2)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Group Name *"
	m.inputs[0].Focus()

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Parent Group (optional)"

	m.focusIndex = 0
}

// renderCreateGroup renders the create group form
func (m Model) renderCreateGroup() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Item Group ") + "\n\n")

	labels := []string{"Group Name: *", "Parent Group:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	b.WriteString(helpStyle.Render("  * Required field"))

	return boxStyle.Render(b.String())
}

// submitCreateGroup submits the create group form
func (m Model) submitCreateGroup() tea.Cmd {
	return func() tea.Msg {
		name := m.inputs[0].Value()
		parent := m.inputs[1].Value()

		if name == "" {
			return formSubmittedMsg{false, "Group name is required"}
		}

		body := map[string]interface{}{
			"item_group_name": name,
		}
		if parent != "" {
			body["parent_item_group"] = parent
		}

		result, err := m.client.Request("POST", "Item%20Group", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Group created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create group"}
	}
}

// =============================================================================
// CREATE BRAND
// =============================================================================

// initCreateBrandForm initializes the create brand form
func (m *Model) initCreateBrandForm() {
	m.inputs = make([]textinput.Model, 2)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Brand Name *"
	m.inputs[0].Focus()

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Description (optional)"

	m.focusIndex = 0
}

// renderCreateBrand renders the create brand form
func (m Model) renderCreateBrand() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Brand ") + "\n\n")

	labels := []string{"Brand Name: *", "Description:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	b.WriteString(helpStyle.Render("  * Required field"))

	return boxStyle.Render(b.String())
}

// submitCreateBrand submits the create brand form
func (m Model) submitCreateBrand() tea.Cmd {
	return func() tea.Msg {
		name := m.inputs[0].Value()
		description := m.inputs[1].Value()

		if name == "" {
			return formSubmittedMsg{false, "Brand name is required"}
		}

		body := map[string]interface{}{
			"brand": name,
		}
		if description != "" {
			body["description"] = description
		}

		result, err := m.client.Request("POST", "Brand", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Brand created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create brand"}
	}
}

// =============================================================================
// CREATE WAREHOUSE
// =============================================================================

// initCreateWarehouseForm initializes the create warehouse form
func (m *Model) initCreateWarehouseForm() {
	m.inputs = make([]textinput.Model, 2)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Warehouse Name *"
	m.inputs[0].Focus()

	m.inputs[1] = textinput.New()
	m.inputs[1].Placeholder = "Parent Warehouse (optional)"

	m.focusIndex = 0
}

// renderCreateWarehouse renders the create warehouse form
func (m Model) renderCreateWarehouse() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Warehouse ") + "\n\n")

	labels := []string{"Warehouse Name: *", "Parent Warehouse:"}
	for i, input := range m.inputs {
		b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	b.WriteString(helpStyle.Render("  * Required field"))

	return boxStyle.Render(b.String())
}

// submitCreateWarehouse submits the create warehouse form
func (m Model) submitCreateWarehouse() tea.Cmd {
	return func() tea.Msg {
		name := m.inputs[0].Value()
		parent := m.inputs[1].Value()

		if name == "" {
			return formSubmittedMsg{false, "Warehouse name is required"}
		}

		company, err := m.client.GetCompany()
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		body := map[string]interface{}{
			"warehouse_name": name,
			"company":        company,
		}
		if parent != "" {
			body["parent_warehouse"] = parent
		}

		result, err := m.client.Request("POST", "Warehouse", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Warehouse created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create warehouse"}
	}
}

// =============================================================================
// CREATE VARIANT
// =============================================================================

// initCreateVariantForm initializes the create variant from template form
func (m *Model) initCreateVariantForm() {
	// Get template attributes
	attrs, _ := m.itemData["attributes"].([]interface{})
	numAttrs := len(attrs)
	if numAttrs == 0 {
		numAttrs = 1 // At least one field
	}

	m.inputs = make([]textinput.Model, numAttrs)

	for i := 0; i < numAttrs; i++ {
		m.inputs[i] = textinput.New()
		if i < len(attrs) {
			if attr, ok := attrs[i].(map[string]interface{}); ok {
				attrName := fmt.Sprintf("%v", attr["attribute"])
				m.inputs[i].Placeholder = attrName + " value *"
			}
		}
		if i == 0 {
			m.inputs[i].Focus()
		}
	}

	m.focusIndex = 0
}

// renderCreateVariant renders the create variant form
func (m Model) renderCreateVariant() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Variant from: "+m.selectedItem) + "\n\n")

	// Get template attributes
	attrs, _ := m.itemData["attributes"].([]interface{})

	if len(attrs) == 0 {
		b.WriteString("  No attributes defined for this template\n")
		return boxStyle.Render(b.String())
	}

	for i, input := range m.inputs {
		label := "Attribute:"
		if i < len(attrs) {
			if attr, ok := attrs[i].(map[string]interface{}); ok {
				label = fmt.Sprintf("%v: *", attr["attribute"])
			}
		}
		b.WriteString(fmt.Sprintf("  %s\n", label))
		b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
	}

	b.WriteString(helpStyle.Render("  * Required fields - Enter values for each attribute"))

	return boxStyle.Render(b.String())
}

// submitCreateVariant submits the create variant form
func (m Model) submitCreateVariant() tea.Cmd {
	return func() tea.Msg {
		templateCode := m.selectedItem
		attrs, _ := m.itemData["attributes"].([]interface{})

		if len(attrs) == 0 {
			return formSubmittedMsg{false, "Template has no attributes defined"}
		}

		// Build attributes array
		var variantAttrs []map[string]interface{}
		for i, attr := range attrs {
			if i >= len(m.inputs) {
				break
			}
			value := m.inputs[i].Value()
			if value == "" {
				attrName := ""
				if a, ok := attr.(map[string]interface{}); ok {
					attrName = fmt.Sprintf("%v", a["attribute"])
				}
				return formSubmittedMsg{false, fmt.Sprintf("Value for %s is required", attrName)}
			}

			if a, ok := attr.(map[string]interface{}); ok {
				variantAttrs = append(variantAttrs, map[string]interface{}{
					"attribute":       a["attribute"],
					"attribute_value": value,
				})
			}
		}

		body := map[string]interface{}{
			"template":   templateCode,
			"attributes": variantAttrs,
		}

		result, err := m.client.Request("POST", "Item", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Variant created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create variant"}
	}
}

// =============================================================================
// CREATE ATTRIBUTE
// =============================================================================

// initCreateAttrForm initializes the create attribute form
func (m *Model) initCreateAttrForm(attrType string) {
	switch attrType {
	case "text":
		// Text attribute: name + values (comma separated)
		m.inputs = make([]textinput.Model, 2)

		m.inputs[0] = textinput.New()
		m.inputs[0].Placeholder = "Attribute Name *"
		m.inputs[0].Focus()

		m.inputs[1] = textinput.New()
		m.inputs[1].Placeholder = "Values (comma separated) *"

		m.formData["attr_type"] = "text"

	case "numeric":
		// Numeric attribute: name + from + to + increment
		m.inputs = make([]textinput.Model, 4)

		m.inputs[0] = textinput.New()
		m.inputs[0].Placeholder = "Attribute Name *"
		m.inputs[0].Focus()

		m.inputs[1] = textinput.New()
		m.inputs[1].Placeholder = "From Range *"

		m.inputs[2] = textinput.New()
		m.inputs[2].Placeholder = "To Range *"

		m.inputs[3] = textinput.New()
		m.inputs[3].Placeholder = "Increment *"

		m.formData["attr_type"] = "numeric"

	case "select":
		// Select attribute: name + values (comma separated)
		m.inputs = make([]textinput.Model, 2)

		m.inputs[0] = textinput.New()
		m.inputs[0].Placeholder = "Attribute Name *"
		m.inputs[0].Focus()

		m.inputs[1] = textinput.New()
		m.inputs[1].Placeholder = "Values (comma separated) *"

		m.formData["attr_type"] = "select"
	}

	m.focusIndex = 0
}

// renderCreateAttr renders the create attribute form
func (m Model) renderCreateAttr() string {
	var b strings.Builder

	attrType := m.formData["attr_type"]

	switch attrType {
	case "text":
		b.WriteString(titleStyle.Render(" Create Text Attribute ") + "\n\n")
		labels := []string{"Attribute Name: *", "Values (comma separated): *"}
		for i, input := range m.inputs {
			b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
			b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
		}
		b.WriteString(helpStyle.Render("  Example values: Red, Blue, Green"))

	case "numeric":
		b.WriteString(titleStyle.Render(" Create Numeric Attribute ") + "\n\n")
		labels := []string{"Attribute Name: *", "From Range: *", "To Range: *", "Increment: *"}
		for i, input := range m.inputs {
			b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
			b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
		}
		b.WriteString(helpStyle.Render("  Example: From 1, To 10, Increment 1"))

	case "select":
		b.WriteString(titleStyle.Render(" Create Select Attribute ") + "\n\n")
		labels := []string{"Attribute Name: *", "Values (comma separated): *"}
		for i, input := range m.inputs {
			b.WriteString(fmt.Sprintf("  %s\n", labels[i]))
			b.WriteString(fmt.Sprintf("  %s\n\n", input.View()))
		}
		b.WriteString(helpStyle.Render("  Example values: Small, Medium, Large"))
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  * Required fields"))

	return boxStyle.Render(b.String())
}

// submitCreateAttr submits the create attribute form
func (m Model) submitCreateAttr() tea.Cmd {
	return func() tea.Msg {
		attrType := m.formData["attr_type"]
		name := m.inputs[0].Value()

		if name == "" {
			return formSubmittedMsg{false, "Attribute name is required"}
		}

		body := map[string]interface{}{
			"attribute_name": name,
		}

		switch attrType {
		case "text", "select":
			valuesStr := m.inputs[1].Value()
			if valuesStr == "" {
				return formSubmittedMsg{false, "Values are required"}
			}

			// Parse comma-separated values
			values := strings.Split(valuesStr, ",")
			var attrValues []map[string]interface{}
			for _, v := range values {
				v = strings.TrimSpace(v)
				if v != "" {
					// Generate abbreviation (first 3 chars uppercase)
					abbr := v
					if len(abbr) > 3 {
						abbr = abbr[:3]
					}
					abbr = strings.ToUpper(abbr)

					attrValues = append(attrValues, map[string]interface{}{
						"attribute_value": v,
						"abbr":            abbr,
					})
				}
			}

			if len(attrValues) == 0 {
				return formSubmittedMsg{false, "At least one value is required"}
			}

			body["item_attribute_values"] = attrValues

		case "numeric":
			fromStr := m.inputs[1].Value()
			toStr := m.inputs[2].Value()
			incrStr := m.inputs[3].Value()

			if fromStr == "" || toStr == "" || incrStr == "" {
				return formSubmittedMsg{false, "All numeric fields are required"}
			}

			fromVal, err := strconv.ParseFloat(fromStr, 64)
			if err != nil {
				return formSubmittedMsg{false, "Invalid from range value"}
			}

			toVal, err := strconv.ParseFloat(toStr, 64)
			if err != nil {
				return formSubmittedMsg{false, "Invalid to range value"}
			}

			incr, err := strconv.ParseFloat(incrStr, 64)
			if err != nil {
				return formSubmittedMsg{false, "Invalid increment value"}
			}

			body["numeric_values"] = 1
			body["from_range"] = fromVal
			body["to_range"] = toVal
			body["increment"] = incr
		}

		result, err := m.client.Request("POST", "Item%20Attribute", body)
		if err != nil {
			return formSubmittedMsg{false, err.Error()}
		}

		if data, ok := result["data"].(map[string]interface{}); ok {
			return formSubmittedMsg{true, fmt.Sprintf("Attribute created: %s", data["name"])}
		}

		return formSubmittedMsg{false, "Failed to create attribute"}
	}
}

// =============================================================================
// CREATE PI FROM PO (Quick Action)
// =============================================================================

// initCreatePIFromPOForm initializes the create PI from PO form (pre-filled from current PO)
func (m *Model) initCreatePIFromPOForm() {
	m.inputs = make([]textinput.Model, 1)

	m.inputs[0] = textinput.New()
	m.inputs[0].Placeholder = "Purchase Order Name"
	m.inputs[0].SetValue(m.selectedItem) // Pre-fill with current PO
	m.inputs[0].Focus()

	m.focusIndex = 0
}

// renderCreatePIFromPO renders the create PI from PO form
func (m Model) renderCreatePIFromPO() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(" Create Invoice from PO ") + "\n\n")

	b.WriteString("  Purchase Order:\n")
	b.WriteString(fmt.Sprintf("  %s\n\n", m.inputs[0].View()))

	// Show PO details if available
	if m.itemData != nil {
		if supplier, ok := m.itemData["supplier"]; ok {
			b.WriteString(fmt.Sprintf("  Supplier: %v\n", supplier))
		}
		if total, ok := m.itemData["grand_total"].(float64); ok {
			b.WriteString(fmt.Sprintf("  Total: %s\n", m.client.FormatCurrency(total)))
		}
		if items, ok := m.itemData["items"].([]interface{}); ok {
			b.WriteString(fmt.Sprintf("  Items: %d\n", len(items)))
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("  Press Enter to create invoice"))

	return boxStyle.Render(b.String())
}

// submitCreatePIFromPO creates a Purchase Invoice from the current PO
func (m Model) submitCreatePIFromPO() tea.Cmd {
	return m.submitCreatePI() // Reuse existing PI creation logic
}

// =============================================================================
// INVENTORY KEY HANDLERS
// =============================================================================

// handleInventoryKeys handles keyboard shortcuts for inventory views
func (m *Model) handleInventoryKeys(key string) (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewAttributes:
		if key == "n" {
			// Show attribute type selection - default to text for now
			m.initCreateAttrForm("text")
			m.prevView = m.view
			m.view = ViewCreateAttrText
			return m, nil
		}

	case ViewGroups:
		if key == "n" {
			m.initCreateGroupForm()
			m.prevView = m.view
			m.view = ViewCreateGroup
			return m, nil
		}

	case ViewBrands:
		if key == "n" {
			m.initCreateBrandForm()
			m.prevView = m.view
			m.view = ViewCreateBrand
			return m, nil
		}

	case ViewWarehouses:
		if key == "n" {
			m.initCreateWarehouseForm()
			m.prevView = m.view
			m.view = ViewCreateWarehouse
			return m, nil
		}
	}

	return m, nil
}

// =============================================================================
// DELETE HANDLERS
// =============================================================================

// deleteGroup deletes an item group
func (m Model) deleteGroup(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		_, err := m.client.Request("DELETE", "Item%20Group/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}
		return actionDoneMsg{fmt.Sprintf("Deleted: %s", name)}
	}
}

// deleteBrand deletes a brand
func (m Model) deleteBrand(name string) tea.Cmd {
	return func() tea.Msg {
		encoded := url.PathEscape(name)
		_, err := m.client.Request("DELETE", "Brand/"+encoded, nil)
		if err != nil {
			return errorMsg{err}
		}
		return actionDoneMsg{fmt.Sprintf("Deleted: %s", name)}
	}
}
