package erp

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// updateFormInputs handles form input updates
func (m *Model) updateFormInputs(msg tea.Msg) tea.Cmd {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab", "down":
			m.focusIndex++
			if m.focusIndex >= len(m.inputs) {
				m.focusIndex = 0
			}
			return m.updateFocus()

		case "shift+tab", "up":
			m.focusIndex--
			if m.focusIndex < 0 {
				m.focusIndex = len(m.inputs) - 1
			}
			return m.updateFocus()

		case "enter":
			return m.submitCurrentForm()

		case "esc":
			m.view = m.prevView
			if m.prevView == ViewMain {
				m.view = ViewMain
			}
			return nil
		}
	}

	// Update the focused input
	if m.focusIndex < len(m.inputs) {
		var cmd tea.Cmd
		m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
		return cmd
	}

	return nil
}

// updateFocus updates which input has focus
func (m *Model) updateFocus() tea.Cmd {
	for i := range m.inputs {
		if i == m.focusIndex {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
	return nil
}

// submitCurrentForm submits the current form based on view
func (m *Model) submitCurrentForm() tea.Cmd {
	m.loading = true

	switch m.view {
	case ViewCreateSupplier:
		m.prevView = ViewSuppliers
		return m.submitCreateSupplier()
	case ViewCreateSerial:
		m.prevView = ViewSerials
		return m.submitCreateSerial()
	case ViewStockReceive:
		m.prevView = ViewStock
		return m.submitStockReceive()
	case ViewStockTransfer:
		m.prevView = ViewStock
		return m.submitStockTransfer()
	case ViewStockIssue:
		m.prevView = ViewStock
		return m.submitStockIssue()
	case ViewCreatePO:
		m.prevView = ViewPurchaseOrders
		return m.submitCreatePO()
	case ViewAddPOItem:
		m.prevView = ViewPODetail
		return m.submitAddPOItem()
	case ViewCreatePI:
		m.prevView = ViewPurchaseInvoices
		return m.submitCreatePI()
	// Sales forms
	case ViewCreateCustomer:
		m.prevView = ViewCustomers
		return m.submitCreateCustomer()
	case ViewCreateQuotation:
		m.prevView = ViewQuotations
		return m.submitCreateQuotation()
	case ViewAddQuotationItem:
		m.prevView = ViewQuotationDetail
		return m.submitAddQuotationItem()
	case ViewCreateSO:
		m.prevView = ViewSalesOrders
		return m.submitCreateSO()
	case ViewCreateSOFromQuotation:
		m.prevView = ViewSalesOrders
		return m.submitCreateSOFromQuotation()
	case ViewAddSOItem:
		m.prevView = ViewSODetail
		return m.submitAddSOItem()
	case ViewCreateSalesInvoice:
		m.prevView = ViewSalesInvoices
		return m.submitCreateSI()
	case ViewCreateDN:
		m.prevView = ViewDeliveryNotes
		return m.submitCreateDN()
	case ViewCreatePR:
		m.prevView = ViewPurchaseReceipts
		return m.submitCreatePR()
	case ViewCreatePayment:
		m.prevView = ViewPayments
		return m.submitCreatePayment()
	}

	return nil
}

// renderConfirmAction renders the confirm action dialog
func (m Model) renderConfirmAction() string {
	content := fmt.Sprintf(`
  %s

  This action may be irreversible.

  [y] Yes, proceed    [n] No, cancel
`, m.confirmMsg)

	return boxStyle.Render(content)
}

// handleConfirmAction handles the confirm action response
func (m *Model) handleConfirmAction(confirmed bool) tea.Cmd {
	if !confirmed {
		m.view = m.prevView
		return nil
	}

	m.loading = true
	m.view = m.prevView

	switch m.confirmAction {
	case "submit_po":
		return m.submitPO(m.selectedItem)
	case "cancel_po":
		return m.cancelPO(m.selectedItem)
	case "submit_pi":
		return m.submitPI(m.selectedItem)
	case "cancel_pi":
		return m.cancelPI(m.selectedItem)
	// Sales actions
	case "submit_quotation":
		return m.submitQuotation(m.selectedItem)
	case "cancel_quotation":
		return m.cancelQuotation(m.selectedItem)
	case "submit_so":
		return m.submitSO(m.selectedItem)
	case "cancel_so":
		return m.cancelSO(m.selectedItem)
	case "submit_si":
		return m.submitSI(m.selectedItem)
	case "cancel_si":
		return m.cancelSI(m.selectedItem)
	// Delivery Notes actions
	case "submit_dn":
		return m.submitDN(m.selectedItem)
	case "cancel_dn":
		return m.cancelDN(m.selectedItem)
	// Purchase Receipts actions
	case "submit_pr":
		return m.submitPR(m.selectedItem)
	case "cancel_pr":
		return m.cancelPR(m.selectedItem)
	// Payment Entry actions
	case "submit_payment":
		return m.submitPayment(m.selectedItem)
	case "cancel_payment":
		return m.cancelPayment(m.selectedItem)
	}

	return nil
}

// handleDeleteForView handles delete action for different views
func (m *Model) handleDeleteForView() tea.Cmd {
	switch m.prevView {
	case ViewAttributes:
		return m.deleteItem("attr", m.selectedItem)
	case ViewItems:
		return m.deleteItem("item", m.selectedItem)
	case ViewTemplates:
		return m.deleteItem("template", m.selectedItem)
	case ViewGroups:
		return m.deleteItem("group", m.selectedItem)
	case ViewBrands:
		return m.deleteItem("brand", m.selectedItem)
	case ViewSuppliers:
		return m.deleteSupplier(m.selectedItem)
	case ViewSerials:
		return m.deleteSerial(m.selectedItem)
	case ViewCustomers:
		return m.deleteCustomer(m.selectedItem)
	}
	return nil
}

// setListTitle sets the title for the current list based on view
func (m *Model) setListTitle() {
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
	case ViewWarehouses:
		m.currentList.Title = "Warehouses"
	case ViewStock:
		m.currentList.Title = "Stock"
	case ViewSerials:
		m.currentList.Title = "Serial Numbers"
	case ViewSuppliers:
		m.currentList.Title = "Suppliers"
	case ViewPurchaseOrders:
		m.currentList.Title = "Purchase Orders"
	case ViewPurchaseInvoices:
		m.currentList.Title = "Purchase Invoices"
	case ViewCustomers:
		m.currentList.Title = "Customers"
	case ViewQuotations:
		m.currentList.Title = "Quotations"
	case ViewSalesOrders:
		m.currentList.Title = "Sales Orders"
	case ViewSalesInvoices:
		m.currentList.Title = "Sales Invoices"
	case ViewDeliveryNotes:
		m.currentList.Title = "Delivery Notes"
	case ViewPurchaseReceipts:
		m.currentList.Title = "Purchase Receipts"
	case ViewPayments:
		m.currentList.Title = "Payments"
	}
	m.currentList.Styles.Title = titleStyle
}

// formatStatusBadge returns a styled status badge
func formatStatusBadge(status string) string {
	var style = helpStyle

	switch strings.ToLower(status) {
	case "draft":
		style = internetStyle
	case "active", "completed", "paid", "submitted":
		style = successStyle
	case "cancelled", "expired", "overdue":
		style = errorStyle
	case "pending", "to receive", "to receive and bill", "unpaid":
		style = vpnStyle
	}

	return style.Render(status)
}
