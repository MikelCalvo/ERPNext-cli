package erp

import (
	"fmt"
	"net/url"
	"time"
)

// PurchaseReceiptItem represents an item in a Purchase Receipt
type PurchaseReceiptItem struct {
	ItemCode          string  `json:"item_code"`
	Qty               float64 `json:"qty"`
	Rate              float64 `json:"rate,omitempty"`
	PurchaseOrder     string  `json:"purchase_order,omitempty"`
	PurchaseOrderItem string  `json:"purchase_order_item,omitempty"`
}

// PurchaseReceipt represents an ERPNext Purchase Receipt
type PurchaseReceipt struct {
	Name        string                `json:"name,omitempty"`
	Supplier    string                `json:"supplier"`
	PostingDate string                `json:"posting_date,omitempty"`
	Company     string                `json:"company,omitempty"`
	DocStatus   int                   `json:"docstatus,omitempty"`
	GrandTotal  float64               `json:"grand_total,omitempty"`
	Items       []PurchaseReceiptItem `json:"items,omitempty"`
}

// CmdPR handles Purchase Receipt commands
func (c *Client) CmdPR(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli pr <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create-from-po, submit, cancel")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli pr list")
		fmt.Println("  erp-cli pr list --supplier=\"Intel\" --status=Draft")
		fmt.Println("  erp-cli pr get PREC-00001")
		fmt.Println("  erp-cli pr create-from-po PUR-ORD-2025-00001")
		fmt.Println("  erp-cli pr submit PREC-00001")
		fmt.Println("  erp-cli pr cancel PREC-00001")
		return nil
	}

	switch args[0] {
	case "list":
		opts := parsePRListOptions(args[1:])
		return c.prList(opts)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli pr get <name>")
		}
		return c.prGet(args[1])
	case "create-from-po":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli pr create-from-po <po_name>")
		}
		return c.prCreateFromPO(args[1])
	case "submit":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli pr submit <name>")
		}
		return c.prSubmit(args[1])
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli pr cancel <name>")
		}
		return c.prCancel(args[1])
	default:
		return fmt.Errorf("unknown pr subcommand: %s", args[0])
	}
}

type prListOptions struct {
	supplier string
	status   string
}

func parsePRListOptions(args []string) prListOptions {
	opts := prListOptions{}
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

func (c *Client) prList(opts prListOptions) error {
	fmt.Printf("%sFetching purchase receipts...%s\n", Blue, Reset)

	filters := [][]interface{}{}
	if opts.supplier != "" {
		filters = append(filters, []interface{}{"supplier", "like", fmt.Sprintf("%%%s%%", opts.supplier)})
	}
	if opts.status != "" {
		filters = append(filters, []interface{}{"status", "=", opts.status})
	}

	endpoint := "Purchase%20Receipt?limit_page_length=0&fields=[\"name\",\"supplier\",\"posting_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc"
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
			fmt.Printf("%sNo purchase receipts found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sPurchase Receipts (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				supplier := m["supplier"]
				date := m["posting_date"]
				status := m["status"]
				total, _ := m["grand_total"].(float64)

				statusColor := Yellow
				if status == "Completed" {
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

func (c *Client) prGet(name string) error {
	fmt.Printf("%sFetching purchase receipt: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Purchase%20Receipt/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sPurchase Receipt: %s%s\n", Cyan, name, Reset)

		fmt.Printf("  Supplier: %s\n", data["supplier"])
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

func (c *Client) prCreateFromPO(poName string) error {
	fmt.Printf("%sCreating purchase receipt from PO: %s%s\n", Blue, poName, Reset)

	encoded := url.PathEscape(poName)
	result, err := c.Request("GET", "Purchase%20Order/"+encoded, nil)
	if err != nil {
		return err
	}

	poData, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("purchase order not found")
	}

	docStatus, _ := poData["docstatus"].(float64)
	if docStatus != 1 {
		return fmt.Errorf("purchase order must be submitted first")
	}

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	var prItems []map[string]interface{}
	if items, ok := poData["items"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				prItems = append(prItems, map[string]interface{}{
					"item_code":           m["item_code"],
					"qty":                 m["qty"],
					"rate":                m["rate"],
					"purchase_order":      poName,
					"purchase_order_item": m["name"],
				})
			}
		}
	}

	if len(prItems) == 0 {
		return fmt.Errorf("no items found in purchase order")
	}

	body := map[string]interface{}{
		"supplier":     poData["supplier"],
		"posting_date": today,
		"company":      company,
		"items":        prItems,
	}

	result, err = c.Request("POST", "Purchase%20Receipt", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		prName := data["name"]
		fmt.Printf("%s✓ Purchase Receipt created: %s%s\n", Green, prName, Reset)
		fmt.Printf("  From PO: %s\n", poName)
		fmt.Printf("  Items: %d\n", len(prItems))
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli pr submit %s' to submit\n", prName)
	}

	return nil
}

func (c *Client) prSubmit(name string) error {
	fmt.Printf("%sSubmitting purchase receipt: %s%s\n", Blue, name, Reset)

	err := c.submitDocument("Purchase Receipt", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Purchase Receipt submitted: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) prCancel(name string) error {
	fmt.Printf("%sCancelling purchase receipt: %s%s\n", Blue, name, Reset)

	err := c.cancelDocument("Purchase Receipt", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Purchase Receipt cancelled: %s%s\n", Green, name, Reset)
	return nil
}
