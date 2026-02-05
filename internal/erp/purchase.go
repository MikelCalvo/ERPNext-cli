package erp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// PurchaseOrderItem represents an item in a Purchase Order
type PurchaseOrderItem struct {
	ItemCode string  `json:"item_code"`
	Qty      float64 `json:"qty"`
	Rate     float64 `json:"rate,omitempty"`
}

// PurchaseOrder represents an ERPNext Purchase Order
type PurchaseOrder struct {
	Name            string              `json:"name,omitempty"`
	Supplier        string              `json:"supplier"`
	TransactionDate string              `json:"transaction_date,omitempty"`
	Company         string              `json:"company,omitempty"`
	Status          string              `json:"status,omitempty"`
	DocStatus       int                 `json:"docstatus,omitempty"`
	GrandTotal      float64             `json:"grand_total,omitempty"`
	Items           []PurchaseOrderItem `json:"items,omitempty"`
}

// PurchaseInvoiceItem represents an item in a Purchase Invoice
type PurchaseInvoiceItem struct {
	ItemCode       string  `json:"item_code"`
	Qty            float64 `json:"qty"`
	Rate           float64 `json:"rate,omitempty"`
	PurchaseOrder  string  `json:"purchase_order,omitempty"`
	PODetail       string  `json:"po_detail,omitempty"`
	ExpenseAccount string  `json:"expense_account,omitempty"`
}

// PurchaseInvoice represents an ERPNext Purchase Invoice
type PurchaseInvoice struct {
	Name        string                `json:"name,omitempty"`
	Supplier    string                `json:"supplier"`
	PostingDate string                `json:"posting_date,omitempty"`
	Company     string                `json:"company,omitempty"`
	DocStatus   int                   `json:"docstatus,omitempty"`
	GrandTotal  float64               `json:"grand_total,omitempty"`
	Items       []PurchaseInvoiceItem `json:"items,omitempty"`
}

// CmdPO handles Purchase Order commands
func (c *Client) CmdPO(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli po <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create, add-item, submit, cancel")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli po list")
		fmt.Println("  erp-cli po list --supplier=\"Intel\" --status=Draft")
		fmt.Println("  erp-cli po get PUR-ORD-2025-00001")
		fmt.Println("  erp-cli po create \"Intel Corporation\"")
		fmt.Println("  erp-cli po add-item PUR-ORD-2025-00001 CPU-I7 10 --rate=450")
		fmt.Println("  erp-cli po submit PUR-ORD-2025-00001")
		fmt.Println("  erp-cli po cancel PUR-ORD-2025-00001")
		return nil
	}

	switch args[0] {
	case "list":
		opts := parsePOListOptions(args[1:])
		return c.poList(opts)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli po get <name>")
		}
		return c.poGet(args[1])
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli po create <supplier>")
		}
		return c.poCreate(args[1])
	case "add-item":
		if len(args) < 4 {
			return fmt.Errorf("usage: erp-cli po add-item <po_name> <item_code> <qty> [--rate=X]")
		}
		qty, err := strconv.ParseFloat(args[3], 64)
		if err != nil {
			return fmt.Errorf("invalid quantity: %s", args[3])
		}
		rate := 0.0
		for _, arg := range args[4:] {
			if len(arg) > 7 && arg[:7] == "--rate=" {
				rate, _ = strconv.ParseFloat(arg[7:], 64)
			}
		}
		return c.poAddItem(args[1], args[2], qty, rate)
	case "submit":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli po submit <name>")
		}
		return c.poSubmit(args[1])
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli po cancel <name>")
		}
		return c.poCancel(args[1])
	default:
		return fmt.Errorf("unknown po subcommand: %s", args[0])
	}
}

type poListOptions struct {
	supplier string
	status   string
}

func parsePOListOptions(args []string) poListOptions {
	opts := poListOptions{}
	for _, arg := range args {
		if len(arg) > 11 && arg[:11] == "--supplier=" {
			opts.supplier = arg[11:]
		}
		if len(arg) > 9 && arg[:9] == "--status=" {
			opts.status = arg[9:]
		}
	}
	return opts
}

func (c *Client) poList(opts poListOptions) error {
	fmt.Printf("%sFetching purchase orders...%s\n", Blue, Reset)

	// Build filters
	filters := [][]interface{}{}
	if opts.supplier != "" {
		filters = append(filters, []interface{}{"supplier", "like", fmt.Sprintf("%%%s%%", opts.supplier)})
	}
	if opts.status != "" {
		filters = append(filters, []interface{}{"status", "=", opts.status})
	}

	endpoint := "Purchase%20Order?limit_page_length=0&fields=[\"name\",\"supplier\",\"transaction_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc"
	if len(filters) > 0 {
		encoded, err := encodeFilters(filters)
		if err != nil {
			return err
		}
		endpoint += "&filters=" + encoded
	}

	result, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			fmt.Printf("%sNo purchase orders found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sPurchase Orders (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				supplier := m["supplier"]
				date := m["transaction_date"]
				status := m["status"]
				total, _ := m["grand_total"].(float64)

				statusColor := Yellow
				if status == "Completed" || status == "To Receive and Bill" {
					statusColor = Green
				} else if status == "Cancelled" {
					statusColor = Red
				}

				fmt.Printf("  %s - %s\n", name, supplier)
				fmt.Printf("    Date: %s | Status: %s%s%s | Total: %s\n",
					date, statusColor, status, Reset, c.FormatCurrency(total))
			}
		}
	}
	return nil
}

func (c *Client) poGet(name string) error {
	fmt.Printf("%sFetching purchase order: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Purchase%20Order/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sPurchase Order: %s%s\n", Cyan, name, Reset)

		// Basic info
		fmt.Printf("  Supplier: %s\n", data["supplier"])
		fmt.Printf("  Date: %s\n", data["transaction_date"])
		fmt.Printf("  Status: %s\n", data["status"])
		grandTotal, _ := data["grand_total"].(float64)
		fmt.Printf("  Total: %s\n", c.FormatCurrency(grandTotal))

		// Items
		if items, ok := data["items"].([]interface{}); ok && len(items) > 0 {
			fmt.Printf("\n  %sItems:%s\n", Yellow, Reset)
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					itemCode := m["item_code"]
					qty, _ := m["qty"].(float64)
					rate, _ := m["rate"].(float64)
					amount, _ := m["amount"].(float64)
					fmt.Printf("    - %s: %.0f x %s = %s\n", itemCode, qty, c.FormatCurrency(rate), c.FormatCurrency(amount))
				}
			}
		}
	}
	return nil
}

func (c *Client) poCreate(supplier string) error {
	fmt.Printf("%sCreating purchase order for: %s%s\n", Blue, supplier, Reset)

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	body := map[string]interface{}{
		"supplier":         supplier,
		"transaction_date": today,
		"schedule_date":    today,
		"company":          company,
		"items":            []interface{}{},
	}

	result, err := c.Request("POST", "Purchase%20Order", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		poName := data["name"]
		fmt.Printf("%s✓ Purchase Order created: %s%s\n", Green, poName, Reset)
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli po add-item %s <item> <qty>' to add items\n", poName)
	}

	return nil
}

func (c *Client) poAddItem(poName, itemCode string, qty, rate float64) error {
	fmt.Printf("%sAdding item to PO: %s%s\n", Blue, poName, Reset)
	fmt.Printf("  Item: %s\n", itemCode)
	fmt.Printf("  Quantity: %.0f\n", qty)

	// Get current PO
	encoded := url.PathEscape(poName)
	result, err := c.Request("GET", "Purchase%20Order/"+encoded, nil)
	if err != nil {
		return err
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("purchase order not found")
	}

	// Check if draft
	docStatus, _ := data["docstatus"].(float64)
	if docStatus != 0 {
		return fmt.Errorf("cannot add items to submitted/cancelled PO")
	}

	// Get existing items
	var existingItems []map[string]interface{}
	if items, ok := data["items"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				existingItems = append(existingItems, m)
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
		fmt.Printf("  Rate: %s\n", c.FormatCurrency(rate))
	}
	existingItems = append(existingItems, newItem)

	// Update PO
	body := map[string]interface{}{
		"items": existingItems,
	}

	_, err = c.Request("PUT", "Purchase%20Order/"+encoded, body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Item added to PO: %s%s\n", Green, poName, Reset)
	return nil
}

func (c *Client) poSubmit(name string) error {
	fmt.Printf("%sSubmitting purchase order: %s%s\n", Blue, name, Reset)

	err := c.submitDocument("Purchase Order", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Purchase Order submitted: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) poCancel(name string) error {
	fmt.Printf("%sCancelling purchase order: %s%s\n", Blue, name, Reset)

	err := c.cancelDocument("Purchase Order", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Purchase Order cancelled: %s%s\n", Green, name, Reset)
	return nil
}

// CmdPI handles Purchase Invoice commands
func (c *Client) CmdPI(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli pi <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create-from-po, submit, cancel")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli pi list")
		fmt.Println("  erp-cli pi list --supplier=\"Intel\" --status=Draft")
		fmt.Println("  erp-cli pi get ACC-PINV-2025-00001")
		fmt.Println("  erp-cli pi create-from-po PUR-ORD-2025-00001")
		fmt.Println("  erp-cli pi submit ACC-PINV-2025-00001")
		fmt.Println("  erp-cli pi cancel ACC-PINV-2025-00001")
		return nil
	}

	switch args[0] {
	case "list":
		opts := parsePIListOptions(args[1:])
		return c.piList(opts)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli pi get <name>")
		}
		return c.piGet(args[1])
	case "create-from-po":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli pi create-from-po <po_name>")
		}
		return c.piCreateFromPO(args[1])
	case "submit":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli pi submit <name>")
		}
		return c.piSubmit(args[1])
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli pi cancel <name>")
		}
		return c.piCancel(args[1])
	default:
		return fmt.Errorf("unknown pi subcommand: %s", args[0])
	}
}

type piListOptions struct {
	supplier string
	status   string
}

func parsePIListOptions(args []string) piListOptions {
	opts := piListOptions{}
	for _, arg := range args {
		if len(arg) > 11 && arg[:11] == "--supplier=" {
			opts.supplier = arg[11:]
		}
		if len(arg) > 9 && arg[:9] == "--status=" {
			opts.status = arg[9:]
		}
	}
	return opts
}

func (c *Client) piList(opts piListOptions) error {
	fmt.Printf("%sFetching purchase invoices...%s\n", Blue, Reset)

	// Build filters
	filters := [][]interface{}{}
	if opts.supplier != "" {
		filters = append(filters, []interface{}{"supplier", "like", fmt.Sprintf("%%%s%%", opts.supplier)})
	}
	if opts.status != "" {
		filters = append(filters, []interface{}{"status", "=", opts.status})
	}

	endpoint := "Purchase%20Invoice?limit_page_length=0&fields=[\"name\",\"supplier\",\"posting_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc"
	if len(filters) > 0 {
		encoded, err := encodeFilters(filters)
		if err != nil {
			return err
		}
		endpoint += "&filters=" + encoded
	}

	result, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			fmt.Printf("%sNo purchase invoices found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sPurchase Invoices (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				supplier := m["supplier"]
				date := m["posting_date"]
				status := m["status"]
				total, _ := m["grand_total"].(float64)

				statusColor := Yellow
				if status == "Paid" || status == "Submitted" {
					statusColor = Green
				} else if status == "Cancelled" {
					statusColor = Red
				}

				fmt.Printf("  %s - %s\n", name, supplier)
				fmt.Printf("    Date: %s | Status: %s%s%s | Total: %s\n",
					date, statusColor, status, Reset, c.FormatCurrency(total))
			}
		}
	}
	return nil
}

func (c *Client) piGet(name string) error {
	fmt.Printf("%sFetching purchase invoice: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Purchase%20Invoice/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sPurchase Invoice: %s%s\n", Cyan, name, Reset)

		// Basic info
		fmt.Printf("  Supplier: %s\n", data["supplier"])
		fmt.Printf("  Date: %s\n", data["posting_date"])
		fmt.Printf("  Status: %s\n", data["status"])
		grandTotal, _ := data["grand_total"].(float64)
		fmt.Printf("  Total: %s\n", c.FormatCurrency(grandTotal))

		// Items
		if items, ok := data["items"].([]interface{}); ok && len(items) > 0 {
			fmt.Printf("\n  %sItems:%s\n", Yellow, Reset)
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					itemCode := m["item_code"]
					qty, _ := m["qty"].(float64)
					rate, _ := m["rate"].(float64)
					amount, _ := m["amount"].(float64)
					po := m["purchase_order"]
					poStr := ""
					if po != nil && po != "" {
						poStr = fmt.Sprintf(" (PO: %s)", po)
					}
					fmt.Printf("    - %s: %.0f x %s = %s%s\n", itemCode, qty, c.FormatCurrency(rate), c.FormatCurrency(amount), poStr)
				}
			}
		}
	}
	return nil
}

func (c *Client) piCreateFromPO(poName string) error {
	fmt.Printf("%sCreating purchase invoice from PO: %s%s\n", Blue, poName, Reset)

	// Get the PO
	encoded := url.PathEscape(poName)
	result, err := c.Request("GET", "Purchase%20Order/"+encoded, nil)
	if err != nil {
		return err
	}

	poData, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("purchase order not found")
	}

	// Check if submitted
	docStatus, _ := poData["docstatus"].(float64)
	if docStatus != 1 {
		return fmt.Errorf("purchase order must be submitted first")
	}

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	// Build invoice items from PO items
	var invoiceItems []map[string]interface{}
	if items, ok := poData["items"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				invoiceItems = append(invoiceItems, map[string]interface{}{
					"item_code":      m["item_code"],
					"qty":            m["qty"],
					"rate":           m["rate"],
					"purchase_order": poName,
					"po_detail":      m["name"],
				})
			}
		}
	}

	if len(invoiceItems) == 0 {
		return fmt.Errorf("no items found in purchase order")
	}

	body := map[string]interface{}{
		"supplier":     poData["supplier"],
		"posting_date": today,
		"company":      company,
		"items":        invoiceItems,
	}

	result, err = c.Request("POST", "Purchase%20Invoice", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		piName := data["name"]
		fmt.Printf("%s✓ Purchase Invoice created: %s%s\n", Green, piName, Reset)
		fmt.Printf("  From PO: %s\n", poName)
		fmt.Printf("  Items: %d\n", len(invoiceItems))
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli pi submit %s' to submit\n", piName)
	}

	return nil
}

func (c *Client) piSubmit(name string) error {
	fmt.Printf("%sSubmitting purchase invoice: %s%s\n", Blue, name, Reset)

	err := c.submitDocument("Purchase Invoice", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Purchase Invoice submitted: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) piCancel(name string) error {
	fmt.Printf("%sCancelling purchase invoice: %s%s\n", Blue, name, Reset)

	err := c.cancelDocument("Purchase Invoice", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Purchase Invoice cancelled: %s%s\n", Green, name, Reset)
	return nil
}

// submitDocument submits a document using frappe.client.submit
func (c *Client) submitDocument(doctype, name string) error {
	fullURL := fmt.Sprintf("%s/api/method/frappe.client.submit", c.ActiveURL)

	body := map[string]interface{}{
		"doc": map[string]interface{}{
			"doctype": doctype,
			"name":    name,
		},
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", c.Config.APIKey, c.Config.APISecret))
	req.Header.Set("Content-Type", "application/json")

	if c.Mode == "internet" && c.Config.NginxCookie != "" {
		req.AddCookie(&http.Cookie{Name: c.Config.NginxCookieName, Value: c.Config.NginxCookie})
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	_, err = parseAPIResponse(resp.StatusCode, respBody)
	if err != nil {
		return fmt.Errorf("submit failed: %w", err)
	}

	return nil
}

// cancelDocument cancels a document using frappe.client.cancel
func (c *Client) cancelDocument(doctype, name string) error {
	fullURL := fmt.Sprintf("%s/api/method/frappe.client.cancel", c.ActiveURL)

	body := map[string]interface{}{
		"doctype": doctype,
		"name":    name,
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", c.Config.APIKey, c.Config.APISecret))
	req.Header.Set("Content-Type", "application/json")

	if c.Mode == "internet" && c.Config.NginxCookie != "" {
		req.AddCookie(&http.Cookie{Name: c.Config.NginxCookieName, Value: c.Config.NginxCookie})
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	_, err = parseAPIResponse(resp.StatusCode, respBody)
	if err != nil {
		return fmt.Errorf("cancel failed: %w", err)
	}

	return nil
}
