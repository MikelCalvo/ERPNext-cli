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

		wg.Add(3)
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
		wg.Wait()

		return dashboardLoadedMsg{data}
	}
}

// renderDashboard renders the dashboard view
func (m Model) renderDashboard() string {
	if m.loading {
		return "\n  Loading dashboard..."
	}

	if m.dashboardData == nil {
		return "\n  No data available"
	}

	data := m.dashboardData
	var b strings.Builder

	// Header
	b.WriteString(titleStyle.Render(" ERPNEXT DASHBOARD "))
	b.WriteString("\n\n")

	// Stock Section
	stockBox := m.renderDashboardStock(data)
	b.WriteString(stockBox)
	b.WriteString("\n\n")

	// Purchasing Section
	purchaseBox := m.renderDashboardPurchases(data)
	b.WriteString(purchaseBox)
	b.WriteString("\n\n")

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

	return boxStyle.Render(b.String())
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

func (m Model) renderDashboardSystem(data *ReportData) string {
	var b strings.Builder
	b.WriteString(selectedStyle.Render("SYSTEM"))
	b.WriteString("\n\n")

	b.WriteString(fmt.Sprintf("  Suppliers:    %d\n", data.TotalSuppliers))
	b.WriteString(fmt.Sprintf("  Warehouses:   %d\n", data.TotalWarehouses))
	b.WriteString(fmt.Sprintf("  Item Groups:  %d\n", data.TotalGroups))

	return b.String()
}
