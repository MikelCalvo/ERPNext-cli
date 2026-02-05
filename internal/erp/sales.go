package erp

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// QuotationItem represents an item in a Quotation
type QuotationItem struct {
	ItemCode string  `json:"item_code"`
	Qty      float64 `json:"qty"`
	Rate     float64 `json:"rate,omitempty"`
}

// Quotation represents an ERPNext Quotation
type Quotation struct {
	Name        string          `json:"name,omitempty"`
	PartyName   string          `json:"party_name"`
	QuotationTo string          `json:"quotation_to"`
	ValidTill   string          `json:"valid_till,omitempty"`
	Company     string          `json:"company,omitempty"`
	Status      string          `json:"status,omitempty"`
	DocStatus   int             `json:"docstatus,omitempty"`
	GrandTotal  float64         `json:"grand_total,omitempty"`
	Items       []QuotationItem `json:"items,omitempty"`
}

// SalesOrderItem represents an item in a Sales Order
type SalesOrderItem struct {
	ItemCode     string  `json:"item_code"`
	Qty          float64 `json:"qty"`
	Rate         float64 `json:"rate,omitempty"`
	DeliveryDate string  `json:"delivery_date,omitempty"`
}

// SalesOrder represents an ERPNext Sales Order
type SalesOrder struct {
	Name            string           `json:"name,omitempty"`
	Customer        string           `json:"customer"`
	TransactionDate string           `json:"transaction_date,omitempty"`
	DeliveryDate    string           `json:"delivery_date,omitempty"`
	Company         string           `json:"company,omitempty"`
	Status          string           `json:"status,omitempty"`
	DocStatus       int              `json:"docstatus,omitempty"`
	GrandTotal      float64          `json:"grand_total,omitempty"`
	Items           []SalesOrderItem `json:"items,omitempty"`
}

// SalesInvoiceItem represents an item in a Sales Invoice
type SalesInvoiceItem struct {
	ItemCode   string  `json:"item_code"`
	Qty        float64 `json:"qty"`
	Rate       float64 `json:"rate,omitempty"`
	SalesOrder string  `json:"sales_order,omitempty"`
	SODetail   string  `json:"so_detail,omitempty"`
}

// SalesInvoice represents an ERPNext Sales Invoice
type SalesInvoice struct {
	Name        string             `json:"name,omitempty"`
	Customer    string             `json:"customer"`
	PostingDate string             `json:"posting_date,omitempty"`
	Company     string             `json:"company,omitempty"`
	DocStatus   int                `json:"docstatus,omitempty"`
	GrandTotal  float64            `json:"grand_total,omitempty"`
	Items       []SalesInvoiceItem `json:"items,omitempty"`
}

// CmdQuotation handles Quotation commands
func (c *Client) CmdQuotation(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli quotation <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create, add-item, submit, cancel")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli quotation list")
		fmt.Println("  erp-cli quotation list --customer=\"Acme\" --status=Draft")
		fmt.Println("  erp-cli quotation get QTN-00001")
		fmt.Println("  erp-cli quotation create \"Acme Corp\"")
		fmt.Println("  erp-cli quotation add-item QTN-00001 CPU-I7 10 --rate=450")
		fmt.Println("  erp-cli quotation submit QTN-00001")
		fmt.Println("  erp-cli quotation cancel QTN-00001")
		return nil
	}

	switch args[0] {
	case "list":
		opts := parseQuotationListOptions(args[1:])
		return c.quotationList(opts)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli quotation get <name>")
		}
		return c.quotationGet(args[1])
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli quotation create <customer>")
		}
		return c.quotationCreate(args[1])
	case "add-item":
		if len(args) < 4 {
			return fmt.Errorf("usage: erp-cli quotation add-item <name> <item_code> <qty> [--rate=X]")
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
		return c.quotationAddItem(args[1], args[2], qty, rate)
	case "submit":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli quotation submit <name>")
		}
		return c.quotationSubmit(args[1])
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli quotation cancel <name>")
		}
		return c.quotationCancel(args[1])
	default:
		return fmt.Errorf("unknown quotation subcommand: %s", args[0])
	}
}

type quotationListOptions struct {
	customer string
	status   string
}

func parseQuotationListOptions(args []string) quotationListOptions {
	opts := quotationListOptions{}
	for _, arg := range args {
		if len(arg) > 11 && arg[:11] == "--customer=" {
			opts.customer = arg[11:]
		}
		if len(arg) > 9 && arg[:9] == "--status=" {
			opts.status = arg[9:]
		}
	}
	return opts
}

func (c *Client) quotationList(opts quotationListOptions) error {
	fmt.Printf("%sFetching quotations...%s\n", Blue, Reset)

	filters := [][]interface{}{}
	if opts.customer != "" {
		filters = append(filters, []interface{}{"party_name", "like", fmt.Sprintf("%%%s%%", opts.customer)})
	}
	if opts.status != "" {
		filters = append(filters, []interface{}{"status", "=", opts.status})
	}

	endpoint := "Quotation?limit_page_length=0&fields=[\"name\",\"party_name\",\"transaction_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc"
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
			fmt.Printf("%sNo quotations found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sQuotations (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				customer := m["party_name"]
				date := m["transaction_date"]
				status := m["status"]
				total, _ := m["grand_total"].(float64)

				statusColor := Yellow
				if status == "Submitted" || status == "Ordered" {
					statusColor = Green
				} else if status == "Cancelled" || status == "Lost" || status == "Expired" {
					statusColor = Red
				}

				fmt.Printf("  %s - %s\n", name, customer)
				fmt.Printf("    Date: %s | Status: %s%s%s | Total: %s\n",
					date, statusColor, status, Reset, c.FormatCurrency(total))
			}
		}
	}
	return nil
}

func (c *Client) quotationGet(name string) error {
	fmt.Printf("%sFetching quotation: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Quotation/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sQuotation: %s%s\n", Cyan, name, Reset)

		fmt.Printf("  Customer: %s\n", data["party_name"])
		fmt.Printf("  Date: %s\n", data["transaction_date"])
		fmt.Printf("  Valid Till: %s\n", data["valid_till"])
		fmt.Printf("  Status: %s\n", data["status"])
		grandTotal, _ := data["grand_total"].(float64)
		fmt.Printf("  Total: %s\n", c.FormatCurrency(grandTotal))

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

func (c *Client) quotationCreate(customer string) error {
	fmt.Printf("%sCreating quotation for: %s%s\n", Blue, customer, Reset)

	company, err := c.GetCompany()
	if err != nil {
		return err
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

	result, err := c.Request("POST", "Quotation", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		qtnName := data["name"]
		fmt.Printf("%s✓ Quotation created: %s%s\n", Green, qtnName, Reset)
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Valid until: %s\n", validTill)
		fmt.Printf("  Use 'erp-cli quotation add-item %s <item> <qty>' to add items\n", qtnName)
	}

	return nil
}

func (c *Client) quotationAddItem(qtnName, itemCode string, qty, rate float64) error {
	fmt.Printf("%sAdding item to Quotation: %s%s\n", Blue, qtnName, Reset)
	fmt.Printf("  Item: %s\n", itemCode)
	fmt.Printf("  Quantity: %.0f\n", qty)

	encoded := url.PathEscape(qtnName)
	result, err := c.Request("GET", "Quotation/"+encoded, nil)
	if err != nil {
		return err
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("quotation not found")
	}

	docStatus, _ := data["docstatus"].(float64)
	if docStatus != 0 {
		return fmt.Errorf("cannot add items to submitted/cancelled quotation")
	}

	var existingItems []map[string]interface{}
	if items, ok := data["items"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				existingItems = append(existingItems, m)
			}
		}
	}

	newItem := map[string]interface{}{
		"item_code": itemCode,
		"qty":       qty,
	}
	if rate > 0 {
		newItem["rate"] = rate
		fmt.Printf("  Rate: %s\n", c.FormatCurrency(rate))
	}
	existingItems = append(existingItems, newItem)

	body := map[string]interface{}{
		"items": existingItems,
	}

	_, err = c.Request("PUT", "Quotation/"+encoded, body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Item added to Quotation: %s%s\n", Green, qtnName, Reset)
	return nil
}

func (c *Client) quotationSubmit(name string) error {
	fmt.Printf("%sSubmitting quotation: %s%s\n", Blue, name, Reset)

	err := c.submitDocument("Quotation", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Quotation submitted: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) quotationCancel(name string) error {
	fmt.Printf("%sCancelling quotation: %s%s\n", Blue, name, Reset)

	err := c.cancelDocument("Quotation", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Quotation cancelled: %s%s\n", Green, name, Reset)
	return nil
}

// CmdSO handles Sales Order commands
func (c *Client) CmdSO(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli so <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create, create-from-quotation, add-item, submit, cancel")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli so list")
		fmt.Println("  erp-cli so list --customer=\"Acme\" --status=Draft")
		fmt.Println("  erp-cli so get SAL-ORD-2025-00001")
		fmt.Println("  erp-cli so create \"Acme Corp\"")
		fmt.Println("  erp-cli so create-from-quotation QTN-00001")
		fmt.Println("  erp-cli so add-item SAL-ORD-2025-00001 CPU-I7 10 --rate=450")
		fmt.Println("  erp-cli so submit SAL-ORD-2025-00001")
		fmt.Println("  erp-cli so cancel SAL-ORD-2025-00001")
		return nil
	}

	switch args[0] {
	case "list":
		opts := parseSOListOptions(args[1:])
		return c.soList(opts)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli so get <name>")
		}
		return c.soGet(args[1])
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli so create <customer>")
		}
		return c.soCreate(args[1])
	case "create-from-quotation":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli so create-from-quotation <quotation_name>")
		}
		return c.soCreateFromQuotation(args[1])
	case "add-item":
		if len(args) < 4 {
			return fmt.Errorf("usage: erp-cli so add-item <so_name> <item_code> <qty> [--rate=X]")
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
		return c.soAddItem(args[1], args[2], qty, rate)
	case "submit":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli so submit <name>")
		}
		return c.soSubmit(args[1])
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli so cancel <name>")
		}
		return c.soCancel(args[1])
	default:
		return fmt.Errorf("unknown so subcommand: %s", args[0])
	}
}

type soListOptions struct {
	customer string
	status   string
}

func parseSOListOptions(args []string) soListOptions {
	opts := soListOptions{}
	for _, arg := range args {
		if len(arg) > 11 && arg[:11] == "--customer=" {
			opts.customer = arg[11:]
		}
		if len(arg) > 9 && arg[:9] == "--status=" {
			opts.status = arg[9:]
		}
	}
	return opts
}

func (c *Client) soList(opts soListOptions) error {
	fmt.Printf("%sFetching sales orders...%s\n", Blue, Reset)

	filters := [][]interface{}{}
	if opts.customer != "" {
		filters = append(filters, []interface{}{"customer", "like", fmt.Sprintf("%%%s%%", opts.customer)})
	}
	if opts.status != "" {
		filters = append(filters, []interface{}{"status", "=", opts.status})
	}

	endpoint := "Sales%20Order?limit_page_length=0&fields=[\"name\",\"customer\",\"transaction_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc"
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
			fmt.Printf("%sNo sales orders found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sSales Orders (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				customer := m["customer"]
				date := m["transaction_date"]
				status := m["status"]
				total, _ := m["grand_total"].(float64)

				statusColor := Yellow
				if status == "Completed" || status == "To Deliver and Bill" {
					statusColor = Green
				} else if status == "Cancelled" {
					statusColor = Red
				}

				fmt.Printf("  %s - %s\n", name, customer)
				fmt.Printf("    Date: %s | Status: %s%s%s | Total: %s\n",
					date, statusColor, status, Reset, c.FormatCurrency(total))
			}
		}
	}
	return nil
}

func (c *Client) soGet(name string) error {
	fmt.Printf("%sFetching sales order: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Sales%20Order/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sSales Order: %s%s\n", Cyan, name, Reset)

		fmt.Printf("  Customer: %s\n", data["customer"])
		fmt.Printf("  Date: %s\n", data["transaction_date"])
		fmt.Printf("  Delivery Date: %s\n", data["delivery_date"])
		fmt.Printf("  Status: %s\n", data["status"])
		grandTotal, _ := data["grand_total"].(float64)
		fmt.Printf("  Total: %s\n", c.FormatCurrency(grandTotal))

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

func (c *Client) soCreate(customer string) error {
	fmt.Printf("%sCreating sales order for: %s%s\n", Blue, customer, Reset)

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	body := map[string]interface{}{
		"customer":         customer,
		"transaction_date": today,
		"delivery_date":    today,
		"company":          company,
		"items":            []interface{}{},
	}

	result, err := c.Request("POST", "Sales%20Order", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		soName := data["name"]
		fmt.Printf("%s✓ Sales Order created: %s%s\n", Green, soName, Reset)
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli so add-item %s <item> <qty>' to add items\n", soName)
	}

	return nil
}

func (c *Client) soCreateFromQuotation(qtnName string) error {
	fmt.Printf("%sCreating sales order from Quotation: %s%s\n", Blue, qtnName, Reset)

	encoded := url.PathEscape(qtnName)
	result, err := c.Request("GET", "Quotation/"+encoded, nil)
	if err != nil {
		return err
	}

	qtnData, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("quotation not found")
	}

	docStatus, _ := qtnData["docstatus"].(float64)
	if docStatus != 1 {
		return fmt.Errorf("quotation must be submitted first")
	}

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	var soItems []map[string]interface{}
	if items, ok := qtnData["items"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				soItems = append(soItems, map[string]interface{}{
					"item_code":       m["item_code"],
					"qty":             m["qty"],
					"rate":            m["rate"],
					"delivery_date":   today,
					"prevdoc_docname": qtnName,
					"quotation_item":  m["name"],
				})
			}
		}
	}

	if len(soItems) == 0 {
		return fmt.Errorf("no items found in quotation")
	}

	body := map[string]interface{}{
		"customer":         qtnData["party_name"],
		"transaction_date": today,
		"delivery_date":    today,
		"company":          company,
		"items":            soItems,
	}

	result, err = c.Request("POST", "Sales%20Order", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		soName := data["name"]
		fmt.Printf("%s✓ Sales Order created: %s%s\n", Green, soName, Reset)
		fmt.Printf("  From Quotation: %s\n", qtnName)
		fmt.Printf("  Items: %d\n", len(soItems))
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli so submit %s' to submit\n", soName)
	}

	return nil
}

func (c *Client) soAddItem(soName, itemCode string, qty, rate float64) error {
	fmt.Printf("%sAdding item to SO: %s%s\n", Blue, soName, Reset)
	fmt.Printf("  Item: %s\n", itemCode)
	fmt.Printf("  Quantity: %.0f\n", qty)

	encoded := url.PathEscape(soName)
	result, err := c.Request("GET", "Sales%20Order/"+encoded, nil)
	if err != nil {
		return err
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("sales order not found")
	}

	docStatus, _ := data["docstatus"].(float64)
	if docStatus != 0 {
		return fmt.Errorf("cannot add items to submitted/cancelled SO")
	}

	var existingItems []map[string]interface{}
	if items, ok := data["items"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				existingItems = append(existingItems, m)
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
		fmt.Printf("  Rate: %s\n", c.FormatCurrency(rate))
	}
	existingItems = append(existingItems, newItem)

	body := map[string]interface{}{
		"items": existingItems,
	}

	_, err = c.Request("PUT", "Sales%20Order/"+encoded, body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Item added to SO: %s%s\n", Green, soName, Reset)
	return nil
}

func (c *Client) soSubmit(name string) error {
	fmt.Printf("%sSubmitting sales order: %s%s\n", Blue, name, Reset)

	err := c.submitDocument("Sales Order", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Sales Order submitted: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) soCancel(name string) error {
	fmt.Printf("%sCancelling sales order: %s%s\n", Blue, name, Reset)

	err := c.cancelDocument("Sales Order", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Sales Order cancelled: %s%s\n", Green, name, Reset)
	return nil
}

// CmdSI handles Sales Invoice commands
func (c *Client) CmdSI(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli si <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create-from-so, submit, cancel")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli si list")
		fmt.Println("  erp-cli si list --customer=\"Acme\" --status=Draft")
		fmt.Println("  erp-cli si get ACC-SINV-2025-00001")
		fmt.Println("  erp-cli si create-from-so SAL-ORD-2025-00001")
		fmt.Println("  erp-cli si submit ACC-SINV-2025-00001")
		fmt.Println("  erp-cli si cancel ACC-SINV-2025-00001")
		return nil
	}

	switch args[0] {
	case "list":
		opts := parseSIListOptions(args[1:])
		return c.siList(opts)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli si get <name>")
		}
		return c.siGet(args[1])
	case "create-from-so":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli si create-from-so <so_name>")
		}
		return c.siCreateFromSO(args[1])
	case "submit":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli si submit <name>")
		}
		return c.siSubmit(args[1])
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli si cancel <name>")
		}
		return c.siCancel(args[1])
	default:
		return fmt.Errorf("unknown si subcommand: %s", args[0])
	}
}

type siListOptions struct {
	customer string
	status   string
}

func parseSIListOptions(args []string) siListOptions {
	opts := siListOptions{}
	for _, arg := range args {
		if len(arg) > 11 && arg[:11] == "--customer=" {
			opts.customer = arg[11:]
		}
		if len(arg) > 9 && arg[:9] == "--status=" {
			opts.status = arg[9:]
		}
	}
	return opts
}

func (c *Client) siList(opts siListOptions) error {
	fmt.Printf("%sFetching sales invoices...%s\n", Blue, Reset)

	filters := [][]interface{}{}
	if opts.customer != "" {
		filters = append(filters, []interface{}{"customer", "like", fmt.Sprintf("%%%s%%", opts.customer)})
	}
	if opts.status != "" {
		filters = append(filters, []interface{}{"status", "=", opts.status})
	}

	endpoint := "Sales%20Invoice?limit_page_length=0&fields=[\"name\",\"customer\",\"posting_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc"
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
			fmt.Printf("%sNo sales invoices found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sSales Invoices (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				customer := m["customer"]
				date := m["posting_date"]
				status := m["status"]
				total, _ := m["grand_total"].(float64)

				statusColor := Yellow
				if status == "Paid" || status == "Submitted" {
					statusColor = Green
				} else if status == "Cancelled" {
					statusColor = Red
				}

				fmt.Printf("  %s - %s\n", name, customer)
				fmt.Printf("    Date: %s | Status: %s%s%s | Total: %s\n",
					date, statusColor, status, Reset, c.FormatCurrency(total))
			}
		}
	}
	return nil
}

func (c *Client) siGet(name string) error {
	fmt.Printf("%sFetching sales invoice: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Sales%20Invoice/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sSales Invoice: %s%s\n", Cyan, name, Reset)

		fmt.Printf("  Customer: %s\n", data["customer"])
		fmt.Printf("  Date: %s\n", data["posting_date"])
		fmt.Printf("  Status: %s\n", data["status"])
		grandTotal, _ := data["grand_total"].(float64)
		fmt.Printf("  Total: %s\n", c.FormatCurrency(grandTotal))

		if items, ok := data["items"].([]interface{}); ok && len(items) > 0 {
			fmt.Printf("\n  %sItems:%s\n", Yellow, Reset)
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					itemCode := m["item_code"]
					qty, _ := m["qty"].(float64)
					rate, _ := m["rate"].(float64)
					amount, _ := m["amount"].(float64)
					so := m["sales_order"]
					soStr := ""
					if so != nil && so != "" {
						soStr = fmt.Sprintf(" (SO: %s)", so)
					}
					fmt.Printf("    - %s: %.0f x %s = %s%s\n", itemCode, qty, c.FormatCurrency(rate), c.FormatCurrency(amount), soStr)
				}
			}
		}
	}
	return nil
}

func (c *Client) siCreateFromSO(soName string) error {
	fmt.Printf("%sCreating sales invoice from SO: %s%s\n", Blue, soName, Reset)

	encoded := url.PathEscape(soName)
	result, err := c.Request("GET", "Sales%20Order/"+encoded, nil)
	if err != nil {
		return err
	}

	soData, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("sales order not found")
	}

	docStatus, _ := soData["docstatus"].(float64)
	if docStatus != 1 {
		return fmt.Errorf("sales order must be submitted first")
	}

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	var invoiceItems []map[string]interface{}
	if items, ok := soData["items"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				invoiceItems = append(invoiceItems, map[string]interface{}{
					"item_code":   m["item_code"],
					"qty":         m["qty"],
					"rate":        m["rate"],
					"sales_order": soName,
					"so_detail":   m["name"],
				})
			}
		}
	}

	if len(invoiceItems) == 0 {
		return fmt.Errorf("no items found in sales order")
	}

	body := map[string]interface{}{
		"customer":     soData["customer"],
		"posting_date": today,
		"company":      company,
		"items":        invoiceItems,
	}

	result, err = c.Request("POST", "Sales%20Invoice", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		siName := data["name"]
		fmt.Printf("%s✓ Sales Invoice created: %s%s\n", Green, siName, Reset)
		fmt.Printf("  From SO: %s\n", soName)
		fmt.Printf("  Items: %d\n", len(invoiceItems))
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli si submit %s' to submit\n", siName)
	}

	return nil
}

func (c *Client) siSubmit(name string) error {
	fmt.Printf("%sSubmitting sales invoice: %s%s\n", Blue, name, Reset)

	err := c.submitDocument("Sales Invoice", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Sales Invoice submitted: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) siCancel(name string) error {
	fmt.Printf("%sCancelling sales invoice: %s%s\n", Blue, name, Reset)

	err := c.cancelDocument("Sales Invoice", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Sales Invoice cancelled: %s%s\n", Green, name, Reset)
	return nil
}
