package erp

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// PaymentReference represents a reference to an invoice in a Payment Entry
type PaymentReference struct {
	ReferenceDoctype  string  `json:"reference_doctype"`
	ReferenceName     string  `json:"reference_name"`
	TotalAmount       float64 `json:"total_amount"`
	OutstandingAmount float64 `json:"outstanding_amount"`
	AllocatedAmount   float64 `json:"allocated_amount"`
}

// PaymentEntry represents an ERPNext Payment Entry
type PaymentEntry struct {
	Name            string             `json:"name,omitempty"`
	PaymentType     string             `json:"payment_type"`
	PartyType       string             `json:"party_type"`
	Party           string             `json:"party"`
	PaidAmount      float64            `json:"paid_amount"`
	PaidFromAccount string             `json:"paid_from,omitempty"`
	PaidToAccount   string             `json:"paid_to,omitempty"`
	ModeOfPayment   string             `json:"mode_of_payment,omitempty"`
	PostingDate     string             `json:"posting_date,omitempty"`
	Status          string             `json:"status,omitempty"`
	DocStatus       int                `json:"docstatus,omitempty"`
	References      []PaymentReference `json:"references,omitempty"`
}

// CmdPayment handles Payment Entry commands
func (c *Client) CmdPayment(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli payment <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, receive, pay, submit, cancel")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli payment list")
		fmt.Println("  erp-cli payment list --type=receive --party=\"Acme Corp\"")
		fmt.Println("  erp-cli payment list --type=pay --status=Draft")
		fmt.Println("  erp-cli payment get PE-00001")
		fmt.Println("  erp-cli payment receive ACC-SINV-2025-00001")
		fmt.Println("  erp-cli payment receive ACC-SINV-2025-00001 --amount=500")
		fmt.Println("  erp-cli payment pay ACC-PINV-2025-00001")
		fmt.Println("  erp-cli payment pay ACC-PINV-2025-00001 --amount=1000")
		fmt.Println("  erp-cli payment submit PE-00001")
		fmt.Println("  erp-cli payment cancel PE-00001")
		return nil
	}

	switch args[0] {
	case "list":
		opts := parsePaymentListOptions(args[1:])
		return c.paymentList(opts)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli payment get <name>")
		}
		return c.paymentGet(args[1])
	case "receive":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli payment receive <si_name> [--amount=X]")
		}
		amount := 0.0
		for _, arg := range args[2:] {
			if len(arg) > 9 && arg[:9] == "--amount=" {
				amount, _ = strconv.ParseFloat(arg[9:], 64)
			}
		}
		return c.paymentReceive(args[1], amount)
	case "pay":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli payment pay <pi_name> [--amount=X]")
		}
		amount := 0.0
		for _, arg := range args[2:] {
			if len(arg) > 9 && arg[:9] == "--amount=" {
				amount, _ = strconv.ParseFloat(arg[9:], 64)
			}
		}
		return c.paymentPay(args[1], amount)
	case "submit":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli payment submit <name>")
		}
		return c.paymentSubmit(args[1])
	case "cancel":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli payment cancel <name>")
		}
		return c.paymentCancel(args[1])
	default:
		return fmt.Errorf("unknown payment subcommand: %s", args[0])
	}
}

type paymentListOptions struct {
	party       string
	paymentType string
	status      string
}

func parsePaymentListOptions(args []string) paymentListOptions {
	opts := paymentListOptions{}
	for _, arg := range args {
		if len(arg) > 8 && arg[:8] == "--party=" {
			opts.party = arg[8:]
		}
		if len(arg) > 7 && arg[:7] == "--type=" {
			opts.paymentType = arg[7:]
		}
		if len(arg) > 9 && arg[:9] == "--status=" {
			opts.status = arg[9:]
		}
	}
	return opts
}

func (c *Client) paymentList(opts paymentListOptions) error {
	fmt.Printf("%sFetching payment entries...%s\n", Blue, Reset)

	filters := []string{}
	if opts.party != "" {
		filters = append(filters, fmt.Sprintf(`["party","like","%%%s%%"]`, opts.party))
	}
	if opts.paymentType != "" {
		paymentTypeValue := "Receive"
		if opts.paymentType == "pay" {
			paymentTypeValue = "Pay"
		}
		filters = append(filters, fmt.Sprintf(`["payment_type","=","%s"]`, paymentTypeValue))
	}
	if opts.status != "" {
		filters = append(filters, fmt.Sprintf(`["status","=","%s"]`, opts.status))
	}

	endpoint := "Payment%20Entry?limit_page_length=0&fields=[\"name\",\"payment_type\",\"party_type\",\"party\",\"paid_amount\",\"posting_date\",\"status\",\"docstatus\"]&order_by=creation%20desc"
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
			fmt.Printf("%sNo payment entries found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sPayment Entries (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				paymentType, _ := m["payment_type"].(string)
				partyType := m["party_type"]
				party := m["party"]
				date := m["posting_date"]
				status, _ := m["status"].(string)
				amount, _ := m["paid_amount"].(float64)

				typeIcon := "↓ Receive"
				if paymentType == "Pay" {
					typeIcon = "↑ Pay"
				}

				statusColor := Yellow
				if status == "Submitted" {
					statusColor = Green
				} else if status == "Cancelled" {
					statusColor = Red
				}

				fmt.Printf("  %s - %s (%s: %s)\n", name, typeIcon, partyType, party)
				fmt.Printf("    Date: %s | Status: %s%s%s | Amount: %s\n",
					date, statusColor, status, Reset, c.FormatCurrency(amount))
			}
		}
	}
	return nil
}

func (c *Client) paymentGet(name string) error {
	fmt.Printf("%sFetching payment entry: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Payment%20Entry/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sPayment Entry: %s%s\n", Cyan, name, Reset)

		paymentType, _ := data["payment_type"].(string)
		typeStr := "Receive (from Customer)"
		if paymentType == "Pay" {
			typeStr = "Pay (to Supplier)"
		}
		fmt.Printf("  Type: %s\n", typeStr)
		fmt.Printf("  Party Type: %s\n", data["party_type"])
		fmt.Printf("  Party: %s\n", data["party"])
		fmt.Printf("  Date: %s\n", data["posting_date"])
		fmt.Printf("  Status: %s\n", data["status"])

		paidAmount, _ := data["paid_amount"].(float64)
		fmt.Printf("  Paid Amount: %s\n", c.FormatCurrency(paidAmount))

		if mop, ok := data["mode_of_payment"]; ok && mop != nil && mop != "" {
			fmt.Printf("  Mode of Payment: %s\n", mop)
		}

		if refs, ok := data["references"].([]interface{}); ok && len(refs) > 0 {
			fmt.Printf("\n  %sReferences:%s\n", Yellow, Reset)
			for _, ref := range refs {
				if r, ok := ref.(map[string]interface{}); ok {
					refDoctype := r["reference_doctype"]
					refName := r["reference_name"]
					allocated, _ := r["allocated_amount"].(float64)
					fmt.Printf("    - %s: %s (Allocated: %s)\n", refDoctype, refName, c.FormatCurrency(allocated))
				}
			}
		}
	}
	return nil
}

func (c *Client) paymentReceive(siName string, amount float64) error {
	fmt.Printf("%sCreating payment entry for Sales Invoice: %s%s\n", Blue, siName, Reset)

	// Get the Sales Invoice
	encoded := url.PathEscape(siName)
	result, err := c.Request("GET", "Sales%20Invoice/"+encoded, nil)
	if err != nil {
		return err
	}

	siData, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("sales invoice not found")
	}

	// Check if submitted
	docStatus, _ := siData["docstatus"].(float64)
	if docStatus != 1 {
		return fmt.Errorf("sales invoice must be submitted first")
	}

	// Check outstanding amount
	outstanding, _ := siData["outstanding_amount"].(float64)
	if outstanding <= 0 {
		return fmt.Errorf("sales invoice has no outstanding amount")
	}

	// Use provided amount or outstanding amount
	paidAmount := outstanding
	if amount > 0 {
		if amount > outstanding {
			return fmt.Errorf("amount exceeds outstanding balance of %s", c.FormatCurrency(outstanding))
		}
		paidAmount = amount
	}

	customer, _ := siData["customer"].(string)
	grandTotal, _ := siData["grand_total"].(float64)

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	body := map[string]interface{}{
		"payment_type": "Receive",
		"party_type":   "Customer",
		"party":        customer,
		"paid_amount":  paidAmount,
		"posting_date": today,
		"company":      company,
		"references": []map[string]interface{}{
			{
				"reference_doctype":  "Sales Invoice",
				"reference_name":     siName,
				"total_amount":       grandTotal,
				"outstanding_amount": outstanding,
				"allocated_amount":   paidAmount,
			},
		},
	}

	result, err = c.Request("POST", "Payment%20Entry", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		peName := data["name"]
		fmt.Printf("%s✓ Payment Entry created: %s%s\n", Green, peName, Reset)
		fmt.Printf("  Type: Receive (from Customer)\n")
		fmt.Printf("  Customer: %s\n", customer)
		fmt.Printf("  Amount: %s\n", c.FormatCurrency(paidAmount))
		fmt.Printf("  For Invoice: %s\n", siName)
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli payment submit %s' to submit\n", peName)
	}

	return nil
}

func (c *Client) paymentPay(piName string, amount float64) error {
	fmt.Printf("%sCreating payment entry for Purchase Invoice: %s%s\n", Blue, piName, Reset)

	// Get the Purchase Invoice
	encoded := url.PathEscape(piName)
	result, err := c.Request("GET", "Purchase%20Invoice/"+encoded, nil)
	if err != nil {
		return err
	}

	piData, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("purchase invoice not found")
	}

	// Check if submitted
	docStatus, _ := piData["docstatus"].(float64)
	if docStatus != 1 {
		return fmt.Errorf("purchase invoice must be submitted first")
	}

	// Check outstanding amount
	outstanding, _ := piData["outstanding_amount"].(float64)
	if outstanding <= 0 {
		return fmt.Errorf("purchase invoice has no outstanding amount")
	}

	// Use provided amount or outstanding amount
	paidAmount := outstanding
	if amount > 0 {
		if amount > outstanding {
			return fmt.Errorf("amount exceeds outstanding balance of %s", c.FormatCurrency(outstanding))
		}
		paidAmount = amount
	}

	supplier, _ := piData["supplier"].(string)
	grandTotal, _ := piData["grand_total"].(float64)

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	today := time.Now().Format("2006-01-02")

	body := map[string]interface{}{
		"payment_type": "Pay",
		"party_type":   "Supplier",
		"party":        supplier,
		"paid_amount":  paidAmount,
		"posting_date": today,
		"company":      company,
		"references": []map[string]interface{}{
			{
				"reference_doctype":  "Purchase Invoice",
				"reference_name":     piName,
				"total_amount":       grandTotal,
				"outstanding_amount": outstanding,
				"allocated_amount":   paidAmount,
			},
		},
	}

	result, err = c.Request("POST", "Payment%20Entry", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		peName := data["name"]
		fmt.Printf("%s✓ Payment Entry created: %s%s\n", Green, peName, Reset)
		fmt.Printf("  Type: Pay (to Supplier)\n")
		fmt.Printf("  Supplier: %s\n", supplier)
		fmt.Printf("  Amount: %s\n", c.FormatCurrency(paidAmount))
		fmt.Printf("  For Invoice: %s\n", piName)
		fmt.Printf("  Status: Draft\n")
		fmt.Printf("  Use 'erp-cli payment submit %s' to submit\n", peName)
	}

	return nil
}

func (c *Client) paymentSubmit(name string) error {
	fmt.Printf("%sSubmitting payment entry: %s%s\n", Blue, name, Reset)

	err := c.submitDocument("Payment Entry", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Payment Entry submitted: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) paymentCancel(name string) error {
	fmt.Printf("%sCancelling payment entry: %s%s\n", Blue, name, Reset)

	err := c.cancelDocument("Payment Entry", name)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Payment Entry cancelled: %s%s\n", Green, name, Reset)
	return nil
}
