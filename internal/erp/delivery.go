package erp

import (
	"fmt"
	"net/url"
	"time"
)

// DeliveryNoteItem represents an item in a Delivery Note
type DeliveryNoteItem struct {
	ItemCode          string  `json:"item_code"`
	Qty               float64 `json:"qty"`
	Rate              float64 `json:"rate,omitempty"`
	AgainstSalesOrder string  `json:"against_sales_order,omitempty"`
	SODetail          string  `json:"so_detail,omitempty"`
}

// DeliveryNote represents an ERPNext Delivery Note
type DeliveryNote struct {
	Name        string             `json:"name,omitempty"`
	Customer    string             `json:"customer"`
	PostingDate string             `json:"posting_date,omitempty"`
	Company     string             `json:"company,omitempty"`
	DocStatus   int                `json:"docstatus,omitempty"`
	GrandTotal  float64            `json:"grand_total,omitempty"`
	Items       []DeliveryNoteItem `json:"items,omitempty"`
}

// CmdDN handles Delivery Note commands
func (c *Client) CmdDN(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli dn <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create-from-so, submit, cancel")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli dn list")
		fmt.Println("  erp-cli dn list --customer=\"Acme\" --status=Draft")
		fmt.Println("  erp-cli dn get DN-00001")
		fmt.Println("  erp-cli dn create-from-so SAL-ORD-2025-00001")
		fmt.Println("  erp-cli dn submit DN-00001")
		fmt.Println("  erp-cli dn cancel DN-00001")
		return nil
	}

	switch args[0] {
	case "list":
		opts := parseDNListOptions(args[1:])
		return c.dnList(opts)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli dn get <name>")
		}
		return c.dnGet(args[1])
	case "create-from-so":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli dn create-from-so <so_name>")
		}
		return c.dnCreateFromSO(args[1])
	case "submit":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli dn submit <name>")
		}
		return c.dnSubmit(args[1])
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli dn cancel <name>")
		}
		return c.dnCancel(args[1])
	default:
		return fmt.Errorf("unknown dn subcommand: %s", args[0])
	}
}

type dnListOptions struct {
	customer string
	status   string
}

func parseDNListOptions(args []string) dnListOptions {
	opts := dnListOptions{}
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

func (c *Client) dnList(opts dnListOptions) error {
	fmt.Printf("%sFetching delivery notes...%s\n", Blue, Reset)

	filters := []string{}
	if opts.customer != "" {
		filters = append(filters, fmt.Sprintf(`["customer","like","%%%s%%"]`, opts.customer))
	}
	if opts.status != "" {
		filters = append(filters, fmt.Sprintf(`["status","=","%s"]`, opts.status))
	}

	endpoint := "Delivery%20Note?limit_page_length=0&fields=[\"name\",\"customer\",\"posting_date\",\"status\",\"grand_total\",\"docstatus\"]&order_by=creation%20desc"
	if len(filters) > 0 {
		filterStr := "[" + filters[0]
		for i := 1; i < len(filters); i++ {
			filterStr += "," + filters[i]
		}
		filterStr += "]"
		endpoint += "&filters=" + url.QueryEscape(filterStr)
	}

	result, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			fmt.Printf("%sNo delivery notes found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sDelivery Notes (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				customer := m["customer"]
				date := m["posting_date"]
				status := m["status"]
				total, _ := m["grand_total"].(float64)

				statusColor := Yellow
				if status == "Completed" {
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

func (c *Client) dnGet(name string) error {
	fmt.Printf("%sFetching delivery note: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Delivery%20Note/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sDelivery Note: %s%s\n", Cyan, name, Reset)

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
					so := m["against_sales_order"]
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

func (c *Client) dnCreateFromSO(soName string) error {
	fmt.Printf("%sCreating delivery note from SO: %s%s\n", Blue, soName, Reset)

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

	var dnItems []map[string]interface{}
	if items, ok := soData["items"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				dnItems = append(dnItems, map[string]interface{}{
					"item_code":           m["item_code"],
					"qty":                 m["qty"],
					"rate":                m["rate"],
					"against_sales_order": soName,
					"so_detail":           m["name"],
				})
			}
		}
	}

	if len(dnItems) == 0 {
		return fmt.Errorf("no items found in sales order")
	}

	body := map[string]interface{}{
		"customer":     soData["customer"],
		"posting_date": today,
		"company":      company,
		"items":        dnItems,
	}

	result, err = c.Request("POST", "Delivery%20Note", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		dnName := data["name"]
		fmt.Printf("%s✓ Delivery Note created: %s%s\n", Green, dnName, Reset)
		fmt.Printf("  From SO: %s\n", soName)
		fmt.Printf("  Items: %d\n", len(dnItems))
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli dn submit %s' to submit\n", dnName)
	}

	return nil
}

func (c *Client) dnSubmit(name string) error {
	fmt.Printf("%sSubmitting delivery note: %s%s\n", Blue, name, Reset)

	err := c.submitDocument("Delivery Note", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Delivery Note submitted: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) dnCancel(name string) error {
	fmt.Printf("%sCancelling delivery note: %s%s\n", Blue, name, Reset)

	err := c.cancelDocument("Delivery Note", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Delivery Note cancelled: %s%s\n", Green, name, Reset)
	return nil
}
