package erp

import (
	"fmt"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// loadDashboard fetches dashboard data
func (m Model) loadDashboard() tea.Cmd {
	return func() tea.Msg {
		// Pre-fetch currency
		m.client.GetCurrency()

		var wg sync.WaitGroup
		var mu sync.Mutex
		data := &ReportData{}

		wg.Add(5)
		go func() {
			defer wg.Done()
			m.client.fetchStockMetrics(data, &mu)
		}()
		go func() {
			defer wg.Done()
			m.client.fetchPurchaseMetrics(data, &mu)
		}()
		go func() {
			defer wg.Done()
			m.client.fetchSystemMetrics(data, &mu)
		}()
		go func() {
			defer wg.Done()
			m.client.fetchSalesMetrics(data, &mu)
		}()
		go func() {
			defer wg.Done()
			m.client.fetchPaymentMetrics(data, &mu)
		}()
		wg.Wait()

		return dashboardLoadedMsg{data}
	}
}

// renderDashboard renders the dashboard view with scrollable viewport
func (m Model) renderDashboard() string {
	if m.loading {
		return fmt.Sprintf("\n  %s Loading dashboard...", m.spinner.View())
	}

	if m.dashboardData == nil {
		return "\n  No data available"
	}

	if !m.viewportReady {
		return "\n  Initializing..."
	}

	// Show viewport with scroll indicator
	var b strings.Builder
	b.WriteString(m.viewport.View())
	b.WriteString("\n")

	// Scroll indicator
	scrollPercent := m.viewport.ScrollPercent() * 100
	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		b.WriteString(helpStyle.Render(fmt.Sprintf("  ↑↓ scroll • %.0f%% ", scrollPercent)))
	}

	return b.String()
}

// renderDashboardContent returns the dashboard content for the viewport
func (m Model) renderDashboardContent() string {
	if m.dashboardData == nil {
		return "No data available"
	}

	data := m.dashboardData
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render(" ERPNEXT DASHBOARD "))
	b.WriteString("\n\n")

	// Stock Section
	stockBox := m.renderDashboardStock(data)
	b.WriteString(stockBox)
	b.WriteString("\n")

	// Sales Section
	salesBox := m.renderDashboardSales(data)
	b.WriteString(salesBox)
	b.WriteString("\n")

	// Purchasing Section
	purchaseBox := m.renderDashboardPurchases(data)
	b.WriteString(purchaseBox)
	b.WriteString("\n")

	// Payments Section
	paymentsBox := m.renderDashboardPayments(data)
	b.WriteString(paymentsBox)
	b.WriteString("\n")

	// System Section
	systemBox := m.renderDashboardSystem(data)
	b.WriteString(systemBox)
	b.WriteString("\n\n")

	// Footer
	modeStr := "VPN"
	if m.client.Mode == "internet" {
		modeStr = "Internet"
	}
	currencyStr := "USD"
	if m.client.Currency != nil {
		currencyStr = m.client.Currency.Code
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	b.WriteString(helpStyle.Render(fmt.Sprintf("Updated: %s | Mode: %s | Currency: %s", timestamp, modeStr, currencyStr)))

	// Errors
	if len(data.Errors) > 0 {
		b.WriteString("\n\n")
		b.WriteString(errorStyle.Render("Warnings:"))
		for _, err := range data.Errors {
			b.WriteString(fmt.Sprintf("\n  - %s", err))
		}
	}

	return b.String()
}

func (m Model) renderDashboardStock(data *ReportData) string {
	var b strings.Builder
	b.WriteString(selectedStyle.Render("STOCK"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Total Items:        %d\n", data.TotalItems))
	b.WriteString(fmt.Sprintf("  Inventory Value:    %s\n", m.client.FormatCurrency(data.TotalStockValue)))

	if data.ZeroStockItems > 0 {
		b.WriteString(fmt.Sprintf("  Zero Stock Items:   %s\n", errorStyle.Render(fmt.Sprintf("%d", data.ZeroStockItems))))
	} else {
		b.WriteString(fmt.Sprintf("  Zero Stock Items:   %d\n", data.ZeroStockItems))
	}

	return b.String()
}

func (m Model) renderDashboardPurchases(data *ReportData) string {
	var b strings.Builder
	b.WriteString(selectedStyle.Render("PURCHASES"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Draft POs:          %d (%s)\n", data.DraftPOs, m.client.FormatCurrency(data.DraftPOValue)))
	b.WriteString(fmt.Sprintf("  Pending POs:        %d (%s)\n", data.PendingPOs, m.client.FormatCurrency(data.PendingPOValue)))
	b.WriteString(fmt.Sprintf("  Completed POs:      %s\n", successStyle.Render(fmt.Sprintf("%d (%s)", data.CompletedPOs, m.client.FormatCurrency(data.CompletedPOValue)))))

	if data.UnpaidInvoices > 0 {
		b.WriteString(fmt.Sprintf("  Unpaid Invoices:    %s\n", errorStyle.Render(fmt.Sprintf("%d (%s)", data.UnpaidInvoices, m.client.FormatCurrency(data.UnpaidValue)))))
	} else {
		b.WriteString(fmt.Sprintf("  Unpaid Invoices:    %d\n", data.UnpaidInvoices))
	}

	if len(data.TopSuppliers) > 0 {
		b.WriteString("\n  Top Suppliers:\n")
		for i, s := range data.TopSuppliers {
			name := s.Name
			if len(name) > 25 {
				name = name[:22] + "..."
			}
			b.WriteString(fmt.Sprintf("    %d. %-25s %3d POs\n", i+1, name, s.POCount))
		}
	}

	return b.String()
}

func (m Model) renderDashboardSales(data *ReportData) string {
	var b strings.Builder
	b.WriteString(selectedStyle.Render("SALES"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Open Quotations:    %d\n", data.OpenQuotations))
	b.WriteString(fmt.Sprintf("  Pending SOs:        %d\n", data.PendingSOs))
	b.WriteString(fmt.Sprintf("  Completed SOs:      %s\n", successStyle.Render(fmt.Sprintf("%d (%s)", data.CompletedSOs, m.client.FormatCurrency(data.CompletedSOValue)))))

	if data.UnpaidSIs > 0 {
		b.WriteString(fmt.Sprintf("  Unpaid Invoices:    %s\n", errorStyle.Render(fmt.Sprintf("%d (%s)", data.UnpaidSIs, m.client.FormatCurrency(data.UnpaidSIValue)))))
	} else {
		b.WriteString(fmt.Sprintf("  Unpaid Invoices:    %d\n", data.UnpaidSIs))
	}

	return b.String()
}

func (m Model) renderDashboardPayments(data *ReportData) string {
	var b strings.Builder
	b.WriteString(selectedStyle.Render("PAYMENTS"))
	b.WriteString("\n\n")

	if data.TotalReceivables > 0 {
		b.WriteString(fmt.Sprintf("  Receivables:        %s\n", successStyle.Render(m.client.FormatCurrency(data.TotalReceivables))))
	} else {
		b.WriteString(fmt.Sprintf("  Receivables:        %s\n", m.client.FormatCurrency(0)))
	}

	if data.TotalPayables > 0 {
		b.WriteString(fmt.Sprintf("  Payables:           %s\n", errorStyle.Render(m.client.FormatCurrency(data.TotalPayables))))
	} else {
		b.WriteString(fmt.Sprintf("  Payables:           %s\n", m.client.FormatCurrency(0)))
	}

	return b.String()
}

func (m Model) renderDashboardSystem(data *ReportData) string {
	var b strings.Builder
	b.WriteString(selectedStyle.Render("SYSTEM"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Suppliers:    %d\n", data.TotalSuppliers))
	b.WriteString(fmt.Sprintf("  Customers:    %d\n", data.TotalCustomers))
	b.WriteString(fmt.Sprintf("  Warehouses:   %d\n", data.TotalWarehouses))
	b.WriteString(fmt.Sprintf("  Item Groups:  %d\n", data.TotalGroups))

	return b.String()
}
