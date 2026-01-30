package erp

import (
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"
)

// ReportData holds all dashboard metrics
type ReportData struct {
	// Stock
	TotalItems      int
	TotalStockValue float64
	ZeroStockItems  int

	// Purchasing
	DraftPOs       int
	DraftPOValue   float64
	PendingPOs     int
	PendingPOValue float64
	UnpaidInvoices int
	UnpaidValue    float64
	TopSuppliers   []SupplierStat

	// System
	TotalSuppliers  int
	TotalWarehouses int
	TotalGroups     int

	// Errors (for partial data display)
	Errors []string
}

// SupplierStat holds supplier statistics
type SupplierStat struct {
	Name    string
	POCount int
	Value   float64
}

// CmdReport handles report commands
func (c *Client) CmdReport(args []string) error {
	if len(args) == 0 {
		return c.reportSummary()
	}

	switch args[0] {
	case "summary", "dashboard":
		return c.reportSummary()
	case "stock":
		return c.reportStock()
	case "purchases":
		return c.reportPurchases()
	default:
		fmt.Println("Usage: erp-cli report [subcommand]")
		fmt.Println("Subcommands:")
		fmt.Println("  (none)      Executive dashboard (default)")
		fmt.Println("  summary     Alias for dashboard")
		fmt.Println("  stock       Detailed stock report")
		fmt.Println("  purchases   Detailed purchasing report")
		return nil
	}
}

// reportSummary displays the executive dashboard
func (c *Client) reportSummary() error {
	fmt.Printf("%sLoading dashboard...%s\n", Blue, Reset)

	var wg sync.WaitGroup
	var mu sync.Mutex
	data := &ReportData{}

	wg.Add(3)
	go func() {
		defer wg.Done()
		c.fetchStockMetrics(data, &mu)
	}()
	go func() {
		defer wg.Done()
		c.fetchPurchaseMetrics(data, &mu)
	}()
	go func() {
		defer wg.Done()
		c.fetchSystemMetrics(data, &mu)
	}()
	wg.Wait()

	return c.renderDashboard(data)
}

// fetchStockMetrics fetches stock-related metrics
func (c *Client) fetchStockMetrics(data *ReportData, mu *sync.Mutex) {
	// Total items
	result, err := c.Request("GET", "Item?limit_page_length=0&fields=[\"name\"]", nil)
	if err == nil {
		if items, ok := result["data"].([]interface{}); ok {
			mu.Lock()
			data.TotalItems = len(items)
			mu.Unlock()
		}
	} else {
		mu.Lock()
		data.Errors = append(data.Errors, "Failed to fetch items")
		mu.Unlock()
	}

	// Stock value from Bin
	result, err = c.Request("GET", "Bin?limit_page_length=0&fields=[\"stock_value\",\"actual_qty\"]", nil)
	if err == nil {
		if bins, ok := result["data"].([]interface{}); ok {
			totalValue := 0.0
			zeroStock := 0
			for _, bin := range bins {
				if m, ok := bin.(map[string]interface{}); ok {
					if val, ok := m["stock_value"].(float64); ok {
						totalValue += val
					}
					if qty, ok := m["actual_qty"].(float64); ok && qty == 0 {
						zeroStock++
					}
				}
			}
			mu.Lock()
			data.TotalStockValue = totalValue
			data.ZeroStockItems = zeroStock
			mu.Unlock()
		}
	} else {
		mu.Lock()
		data.Errors = append(data.Errors, "Failed to fetch stock data")
		mu.Unlock()
	}
}

// fetchPurchaseMetrics fetches purchasing-related metrics
func (c *Client) fetchPurchaseMetrics(data *ReportData, mu *sync.Mutex) {
	// Draft POs (docstatus=0)
	filter := url.QueryEscape(`[["docstatus","=",0]]`)
	result, err := c.Request("GET", "Purchase%20Order?limit_page_length=0&filters="+filter+"&fields=[\"name\",\"grand_total\",\"supplier\"]", nil)
	if err == nil {
		if pos, ok := result["data"].([]interface{}); ok {
			total := 0.0
			for _, po := range pos {
				if m, ok := po.(map[string]interface{}); ok {
					if val, ok := m["grand_total"].(float64); ok {
						total += val
					}
				}
			}
			mu.Lock()
			data.DraftPOs = len(pos)
			data.DraftPOValue = total
			mu.Unlock()
		}
	}

	// Pending POs (To Receive and Bill or To Receive)
	filter = url.QueryEscape(`[["docstatus","=",1],["status","in",["To Receive and Bill","To Receive"]]]`)
	result, err = c.Request("GET", "Purchase%20Order?limit_page_length=0&filters="+filter+"&fields=[\"name\",\"grand_total\",\"supplier\"]", nil)
	if err == nil {
		if pos, ok := result["data"].([]interface{}); ok {
			total := 0.0
			supplierCounts := make(map[string]int)
			supplierValues := make(map[string]float64)
			for _, po := range pos {
				if m, ok := po.(map[string]interface{}); ok {
					if val, ok := m["grand_total"].(float64); ok {
						total += val
					}
					if supplier, ok := m["supplier"].(string); ok {
						supplierCounts[supplier]++
						if val, ok := m["grand_total"].(float64); ok {
							supplierValues[supplier] += val
						}
					}
				}
			}
			mu.Lock()
			data.PendingPOs = len(pos)
			data.PendingPOValue = total
			mu.Unlock()
		}
	}

	// All submitted POs for top suppliers calculation
	filter = url.QueryEscape(`[["docstatus","=",1]]`)
	result, err = c.Request("GET", "Purchase%20Order?limit_page_length=0&filters="+filter+"&fields=[\"supplier\",\"grand_total\"]", nil)
	if err == nil {
		if pos, ok := result["data"].([]interface{}); ok {
			supplierCounts := make(map[string]int)
			supplierValues := make(map[string]float64)
			for _, po := range pos {
				if m, ok := po.(map[string]interface{}); ok {
					if supplier, ok := m["supplier"].(string); ok {
						supplierCounts[supplier]++
						if val, ok := m["grand_total"].(float64); ok {
							supplierValues[supplier] += val
						}
					}
				}
			}

			// Convert to slice and sort
			var suppliers []SupplierStat
			for name, count := range supplierCounts {
				suppliers = append(suppliers, SupplierStat{
					Name:    name,
					POCount: count,
					Value:   supplierValues[name],
				})
			}
			sort.Slice(suppliers, func(i, j int) bool {
				return suppliers[i].POCount > suppliers[j].POCount
			})

			// Keep top 5
			if len(suppliers) > 5 {
				suppliers = suppliers[:5]
			}

			mu.Lock()
			data.TopSuppliers = suppliers
			mu.Unlock()
		}
	}

	// Unpaid invoices (outstanding_amount > 0)
	filter = url.QueryEscape(`[["docstatus","=",1],["outstanding_amount",">",0]]`)
	result, err = c.Request("GET", "Purchase%20Invoice?limit_page_length=0&filters="+filter+"&fields=[\"name\",\"outstanding_amount\"]", nil)
	if err == nil {
		if invoices, ok := result["data"].([]interface{}); ok {
			total := 0.0
			for _, inv := range invoices {
				if m, ok := inv.(map[string]interface{}); ok {
					if val, ok := m["outstanding_amount"].(float64); ok {
						total += val
					}
				}
			}
			mu.Lock()
			data.UnpaidInvoices = len(invoices)
			data.UnpaidValue = total
			mu.Unlock()
		}
	}
}

// fetchSystemMetrics fetches system-wide metrics
func (c *Client) fetchSystemMetrics(data *ReportData, mu *sync.Mutex) {
	// Suppliers
	result, err := c.Request("GET", "Supplier?limit_page_length=0&fields=[\"name\"]", nil)
	if err == nil {
		if items, ok := result["data"].([]interface{}); ok {
			mu.Lock()
			data.TotalSuppliers = len(items)
			mu.Unlock()
		}
	}

	// Warehouses
	result, err = c.Request("GET", "Warehouse?limit_page_length=0&fields=[\"name\"]", nil)
	if err == nil {
		if items, ok := result["data"].([]interface{}); ok {
			mu.Lock()
			data.TotalWarehouses = len(items)
			mu.Unlock()
		}
	}

	// Item Groups
	result, err = c.Request("GET", "Item%20Group?limit_page_length=0&fields=[\"name\"]", nil)
	if err == nil {
		if items, ok := result["data"].([]interface{}); ok {
			mu.Lock()
			data.TotalGroups = len(items)
			mu.Unlock()
		}
	}
}

// renderDashboard displays the dashboard
func (c *Client) renderDashboard(data *ReportData) error {
	fmt.Print("\033[H\033[2J") // Clear screen

	// Header
	fmt.Println()
	fmt.Printf("%s══════════════════════════════════════════════════════════════%s\n", Cyan, Reset)
	fmt.Printf("%s                    ERPNEXT DASHBOARD                         %s\n", Cyan, Reset)
	fmt.Printf("%s══════════════════════════════════════════════════════════════%s\n", Cyan, Reset)
	fmt.Println()

	// Stock Section
	fmt.Printf("%s┌─ STOCK ─────────────────────────────────────────────────────┐%s\n", Yellow, Reset)
	fmt.Printf("%s│%s  Items totales:        %-6d                                %s│%s\n", Yellow, Reset, data.TotalItems, Yellow, Reset)
	fmt.Printf("%s│%s  Valor inventario:     $%-12.2f                       %s│%s\n", Yellow, Reset, data.TotalStockValue, Yellow, Reset)
	if data.ZeroStockItems > 0 {
		fmt.Printf("%s│%s  Sin stock:            %s%-3d ⚠%s                               %s│%s\n", Yellow, Reset, Red, data.ZeroStockItems, Reset, Yellow, Reset)
	} else {
		fmt.Printf("%s│%s  Sin stock:            %-6d                                %s│%s\n", Yellow, Reset, data.ZeroStockItems, Yellow, Reset)
	}
	fmt.Printf("%s└─────────────────────────────────────────────────────────────┘%s\n", Yellow, Reset)
	fmt.Println()

	// Purchasing Section
	fmt.Printf("%s┌─ COMPRAS ───────────────────────────────────────────────────┐%s\n", Yellow, Reset)
	fmt.Printf("%s│%s  POs en borrador:      %-3d ($%.2f)                   %s│%s\n", Yellow, Reset, data.DraftPOs, data.DraftPOValue, Yellow, Reset)
	fmt.Printf("%s│%s  POs por recibir:      %-3d ($%.2f)                   %s│%s\n", Yellow, Reset, data.PendingPOs, data.PendingPOValue, Yellow, Reset)
	if data.UnpaidInvoices > 0 {
		fmt.Printf("%s│%s  Facturas pendientes:  %s%-3d ($%.2f)%s                   %s│%s\n", Yellow, Reset, Red, data.UnpaidInvoices, data.UnpaidValue, Reset, Yellow, Reset)
	} else {
		fmt.Printf("%s│%s  Facturas pendientes:  %-3d                                  %s│%s\n", Yellow, Reset, data.UnpaidInvoices, Yellow, Reset)
	}
	fmt.Printf("%s│%s                                                             %s│%s\n", Yellow, Reset, Yellow, Reset)
	if len(data.TopSuppliers) > 0 {
		fmt.Printf("%s│%s  %sTop Proveedores:%s                                           %s│%s\n", Yellow, Reset, Cyan, Reset, Yellow, Reset)
		for i, s := range data.TopSuppliers {
			name := s.Name
			if len(name) > 25 {
				name = name[:22] + "..."
			}
			fmt.Printf("%s│%s    %d. %-25s %3d POs                   %s│%s\n", Yellow, Reset, i+1, name, s.POCount, Yellow, Reset)
		}
	}
	fmt.Printf("%s└─────────────────────────────────────────────────────────────┘%s\n", Yellow, Reset)
	fmt.Println()

	// System Section
	fmt.Printf("%s┌─ SISTEMA ──────────────────────────────────────────────────┐%s\n", Yellow, Reset)
	fmt.Printf("%s│%s  Proveedores:   %-4d                                        %s│%s\n", Yellow, Reset, data.TotalSuppliers, Yellow, Reset)
	fmt.Printf("%s│%s  Almacenes:     %-4d                                        %s│%s\n", Yellow, Reset, data.TotalWarehouses, Yellow, Reset)
	fmt.Printf("%s│%s  Grupos:        %-4d                                        %s│%s\n", Yellow, Reset, data.TotalGroups, Yellow, Reset)
	fmt.Printf("%s└─────────────────────────────────────────────────────────────┘%s\n", Yellow, Reset)
	fmt.Println()

	// Footer
	modeStr := "VPN"
	if c.Mode == "internet" {
		modeStr = "Internet"
	}
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("Generado: %s | Modo: %s%s%s\n", timestamp, Cyan, modeStr, Reset)

	// Show errors if any
	if len(data.Errors) > 0 {
		fmt.Println()
		fmt.Printf("%sAdvertencias:%s\n", Yellow, Reset)
		for _, err := range data.Errors {
			fmt.Printf("  - %s\n", err)
		}
	}

	return nil
}

// reportStock displays detailed stock report
func (c *Client) reportStock() error {
	fmt.Printf("%sGenerating stock report...%s\n\n", Blue, Reset)

	// Get all bins with stock data
	result, err := c.Request("GET", "Bin?limit_page_length=0&fields=[\"item_code\",\"warehouse\",\"actual_qty\",\"stock_value\",\"reserved_qty\",\"ordered_qty\"]&order_by=item_code", nil)
	if err != nil {
		return err
	}

	fmt.Printf("%s══════════════════════════════════════════════════════════════%s\n", Cyan, Reset)
	fmt.Printf("%s                    STOCK REPORT                              %s\n", Cyan, Reset)
	fmt.Printf("%s══════════════════════════════════════════════════════════════%s\n\n", Cyan, Reset)

	if bins, ok := result["data"].([]interface{}); ok {
		totalValue := 0.0
		totalQty := 0.0
		zeroStock := 0
		itemCount := make(map[string]bool)

		// Group by item
		itemData := make(map[string][]map[string]interface{})
		for _, bin := range bins {
			if m, ok := bin.(map[string]interface{}); ok {
				itemCode := fmt.Sprintf("%v", m["item_code"])
				itemCount[itemCode] = true
				itemData[itemCode] = append(itemData[itemCode], m)

				if qty, ok := m["actual_qty"].(float64); ok {
					totalQty += qty
					if qty == 0 {
						zeroStock++
					}
				}
				if val, ok := m["stock_value"].(float64); ok {
					totalValue += val
				}
			}
		}

		// Print summary
		fmt.Printf("%sSummary:%s\n", Yellow, Reset)
		fmt.Printf("  Total items with stock entries: %d\n", len(itemCount))
		fmt.Printf("  Total stock value: $%.2f\n", totalValue)
		fmt.Printf("  Total quantity: %.0f units\n", totalQty)
		fmt.Printf("  Zero stock entries: %d\n\n", zeroStock)

		// Print by warehouse
		warehouseData := make(map[string]float64)
		warehouseQty := make(map[string]float64)
		for _, bin := range bins {
			if m, ok := bin.(map[string]interface{}); ok {
				wh := fmt.Sprintf("%v", m["warehouse"])
				if val, ok := m["stock_value"].(float64); ok {
					warehouseData[wh] += val
				}
				if qty, ok := m["actual_qty"].(float64); ok {
					warehouseQty[wh] += qty
				}
			}
		}

		fmt.Printf("%sBy Warehouse:%s\n", Yellow, Reset)
		for wh, value := range warehouseData {
			qty := warehouseQty[wh]
			fmt.Printf("  📦 %s\n", wh)
			fmt.Printf("     Quantity: %.0f | Value: $%.2f\n", qty, value)
		}
	}

	fmt.Printf("\n%sGenerated: %s%s\n", Cyan, time.Now().Format("2006-01-02 15:04:05"), Reset)
	return nil
}

// reportPurchases displays detailed purchasing report
func (c *Client) reportPurchases() error {
	fmt.Printf("%sGenerating purchasing report...%s\n\n", Blue, Reset)

	fmt.Printf("%s══════════════════════════════════════════════════════════════%s\n", Cyan, Reset)
	fmt.Printf("%s                    PURCHASING REPORT                         %s\n", Cyan, Reset)
	fmt.Printf("%s══════════════════════════════════════════════════════════════%s\n\n", Cyan, Reset)

	// Draft POs
	fmt.Printf("%sDraft Purchase Orders:%s\n", Yellow, Reset)
	filter := url.QueryEscape(`[["docstatus","=",0]]`)
	result, err := c.Request("GET", "Purchase%20Order?limit_page_length=20&filters="+filter+"&fields=[\"name\",\"supplier\",\"grand_total\",\"transaction_date\"]&order_by=creation%20desc", nil)
	if err == nil {
		if pos, ok := result["data"].([]interface{}); ok {
			if len(pos) == 0 {
				fmt.Println("  No draft POs")
			} else {
				for _, po := range pos {
					if m, ok := po.(map[string]interface{}); ok {
						fmt.Printf("  - %s: %s ($%.2f) - %s\n",
							m["name"], m["supplier"], m["grand_total"], m["transaction_date"])
					}
				}
			}
		}
	}
	fmt.Println()

	// Pending POs
	fmt.Printf("%sPending Purchase Orders (To Receive):%s\n", Yellow, Reset)
	filter = url.QueryEscape(`[["docstatus","=",1],["status","in",["To Receive and Bill","To Receive"]]]`)
	result, err = c.Request("GET", "Purchase%20Order?limit_page_length=20&filters="+filter+"&fields=[\"name\",\"supplier\",\"grand_total\",\"status\"]&order_by=creation%20desc", nil)
	if err == nil {
		if pos, ok := result["data"].([]interface{}); ok {
			if len(pos) == 0 {
				fmt.Println("  No pending POs")
			} else {
				totalPending := 0.0
				for _, po := range pos {
					if m, ok := po.(map[string]interface{}); ok {
						val, _ := m["grand_total"].(float64)
						totalPending += val
						fmt.Printf("  - %s: %s ($%.2f) - %s\n",
							m["name"], m["supplier"], val, m["status"])
					}
				}
				fmt.Printf("  %sTotal pending: $%.2f%s\n", Cyan, totalPending, Reset)
			}
		}
	}
	fmt.Println()

	// Unpaid Invoices
	fmt.Printf("%sUnpaid Purchase Invoices:%s\n", Yellow, Reset)
	filter = url.QueryEscape(`[["docstatus","=",1],["outstanding_amount",">",0]]`)
	result, err = c.Request("GET", "Purchase%20Invoice?limit_page_length=20&filters="+filter+"&fields=[\"name\",\"supplier\",\"grand_total\",\"outstanding_amount\",\"posting_date\"]&order_by=posting_date%20desc", nil)
	if err == nil {
		if invoices, ok := result["data"].([]interface{}); ok {
			if len(invoices) == 0 {
				fmt.Println("  No unpaid invoices")
			} else {
				totalUnpaid := 0.0
				for _, inv := range invoices {
					if m, ok := inv.(map[string]interface{}); ok {
						outstanding, _ := m["outstanding_amount"].(float64)
						totalUnpaid += outstanding
						fmt.Printf("  - %s: %s (Outstanding: $%.2f of $%.2f) - %s\n",
							m["name"], m["supplier"], outstanding, m["grand_total"], m["posting_date"])
					}
				}
				fmt.Printf("  %sTotal outstanding: $%.2f%s\n", Red, totalUnpaid, Reset)
			}
		}
	}
	fmt.Println()

	// Supplier Statistics
	fmt.Printf("%sSupplier Statistics (by PO count):%s\n", Yellow, Reset)
	filter = url.QueryEscape(`[["docstatus","=",1]]`)
	result, err = c.Request("GET", "Purchase%20Order?limit_page_length=0&filters="+filter+"&fields=[\"supplier\",\"grand_total\"]", nil)
	if err == nil {
		if pos, ok := result["data"].([]interface{}); ok {
			supplierCounts := make(map[string]int)
			supplierValues := make(map[string]float64)
			for _, po := range pos {
				if m, ok := po.(map[string]interface{}); ok {
					if supplier, ok := m["supplier"].(string); ok {
						supplierCounts[supplier]++
						if val, ok := m["grand_total"].(float64); ok {
							supplierValues[supplier] += val
						}
					}
				}
			}

			// Convert to slice and sort
			var suppliers []SupplierStat
			for name, count := range supplierCounts {
				suppliers = append(suppliers, SupplierStat{
					Name:    name,
					POCount: count,
					Value:   supplierValues[name],
				})
			}
			sort.Slice(suppliers, func(i, j int) bool {
				return suppliers[i].POCount > suppliers[j].POCount
			})

			for i, s := range suppliers {
				if i >= 10 {
					fmt.Printf("  ... and %d more suppliers\n", len(suppliers)-10)
					break
				}
				fmt.Printf("  %2d. %-30s %3d POs  $%.2f\n", i+1, s.Name, s.POCount, s.Value)
			}
		}
	}

	fmt.Printf("\n%sGenerated: %s%s\n", Cyan, time.Now().Format("2006-01-02 15:04:05"), Reset)
	return nil
}
