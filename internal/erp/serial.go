package erp

import (
	"fmt"
	"net/url"
	"strconv"
)

// CmdSerial handles serial number commands
func (c *Client) CmdSerial(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli serial <subcommand> [args...]")
		fmt.Println("Subcommands: create, list, get, create-batch")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli serial create SN-CPU-001 CPU-LGA1700-I7")
		fmt.Println("  erp-cli serial create SN-CPU-001 CPU-LGA1700-I7 --supplier=\"Intel Dist\"")
		fmt.Println("  erp-cli serial list CPU-LGA1700-I7")
		fmt.Println("  erp-cli serial get SN-CPU-001")
		fmt.Println("  erp-cli serial create-batch CPU-LGA1700-I7 SN-CPU 1 10")
		return nil
	}

	switch args[0] {
	case "create":
		if len(args) < 3 {
			return fmt.Errorf("usage: erp-cli serial create <serial_no> <item_code> [--supplier=X] [--batch=X]")
		}
		opts := parseSerialOptions(args[3:])
		return c.serialCreate(args[1], args[2], opts)
	case "list":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli serial list <item_code> [--warehouse=X]")
		}
		warehouse := ""
		for _, arg := range args[2:] {
			if len(arg) > 12 && arg[:12] == "--warehouse=" {
				warehouse = arg[12:]
			}
		}
		return c.serialList(args[1], warehouse)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli serial get <serial_no>")
		}
		return c.serialGet(args[1])
	case "create-batch":
		if len(args) < 5 {
			return fmt.Errorf("usage: erp-cli serial create-batch <item_code> <prefix> <start> <count>")
		}
		start, err := strconv.Atoi(args[3])
		if err != nil {
			return fmt.Errorf("invalid start number: %s", args[3])
		}
		count, err := strconv.Atoi(args[4])
		if err != nil {
			return fmt.Errorf("invalid count: %s", args[4])
		}
		return c.serialCreateBatch(args[1], args[2], start, count)
	default:
		return fmt.Errorf("unknown serial subcommand: %s", args[0])
	}
}

type serialOptions struct {
	supplier string
	batch    string
}

func parseSerialOptions(args []string) serialOptions {
	opts := serialOptions{}
	for _, arg := range args {
		if len(arg) > 11 && arg[:11] == "--supplier=" {
			opts.supplier = arg[11:]
		}
		if len(arg) > 8 && arg[:8] == "--batch=" {
			opts.batch = arg[8:]
		}
	}
	return opts
}

func (c *Client) serialCreate(serialNo, itemCode string, opts serialOptions) error {
	fmt.Printf("%sCreating serial number: %s%s\n", Blue, serialNo, Reset)
	fmt.Printf("  Item: %s\n", itemCode)

	body := map[string]interface{}{
		"serial_no": serialNo,
		"item_code": itemCode,
	}

	if opts.supplier != "" {
		body["supplier"] = opts.supplier
		fmt.Printf("  Supplier: %s\n", opts.supplier)
	}

	if opts.batch != "" {
		body["batch_no"] = opts.batch
		fmt.Printf("  Batch: %s\n", opts.batch)
	}

	_, err := c.Request("POST", "Serial%20No", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Serial number created: %s%s\n", Green, serialNo, Reset)
	return nil
}

func (c *Client) serialList(itemCode, warehouse string) error {
	fmt.Printf("%sFetching serial numbers for: %s%s\n", Blue, itemCode, Reset)

	filter := fmt.Sprintf(`[["item_code","=","%s"]]`, itemCode)
	if warehouse != "" {
		filter = fmt.Sprintf(`[["item_code","=","%s"],["warehouse","=","%s"]]`, itemCode, warehouse)
	}
	encodedFilter := url.QueryEscape(filter)

	result, err := c.Request("GET", "Serial%20No?filters="+encodedFilter+"&fields=[\"name\",\"warehouse\",\"status\",\"purchase_date\"]&limit_page_length=0", nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			fmt.Printf("%sNo serial numbers found for: %s%s\n", Yellow, itemCode, Reset)
			return nil
		}

		fmt.Printf("\n%sSerial Numbers (%d):%s\n", Cyan, len(data), Reset)

		active := make([]map[string]interface{}, 0)
		delivered := make([]map[string]interface{}, 0)
		other := make([]map[string]interface{}, 0)

		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				status, _ := m["status"].(string)
				switch status {
				case "Active":
					active = append(active, m)
				case "Delivered":
					delivered = append(delivered, m)
				default:
					other = append(other, m)
				}
			}
		}

		if len(active) > 0 {
			fmt.Printf("\n  %sActive (%d):%s\n", Green, len(active), Reset)
			for _, m := range active {
				printSerialLine(m)
			}
		}

		if len(delivered) > 0 {
			fmt.Printf("\n  %sDelivered (%d):%s\n", Yellow, len(delivered), Reset)
			for _, m := range delivered {
				printSerialLine(m)
			}
		}

		if len(other) > 0 {
			fmt.Printf("\n  %sOther (%d):%s\n", Blue, len(other), Reset)
			for _, m := range other {
				printSerialLine(m)
			}
		}
	}

	return nil
}

func printSerialLine(m map[string]interface{}) {
	name := m["name"]
	warehouse := m["warehouse"]
	purchaseDate := m["purchase_date"]

	line := fmt.Sprintf("    • %s", name)
	if warehouse != nil && warehouse != "" {
		line += fmt.Sprintf(" @ %s", warehouse)
	}
	if purchaseDate != nil && purchaseDate != "" {
		line += fmt.Sprintf(" (purchased: %s)", purchaseDate)
	}
	fmt.Println(line)
}

func (c *Client) serialGet(serialNo string) error {
	fmt.Printf("%sFetching serial number: %s%s\n", Blue, serialNo, Reset)

	encoded := url.PathEscape(serialNo)
	result, err := c.Request("GET", "Serial%20No/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("\n%sSerial Number: %s%s\n", Cyan, serialNo, Reset)

		fields := []struct {
			key   string
			label string
		}{
			{"item_code", "Item"},
			{"item_name", "Item Name"},
			{"status", "Status"},
			{"warehouse", "Warehouse"},
			{"batch_no", "Batch"},
			{"supplier", "Supplier"},
			{"purchase_date", "Purchase Date"},
			{"warranty_expiry_date", "Warranty Expiry"},
			{"company", "Company"},
		}

		for _, f := range fields {
			if val, ok := data[f.key]; ok && val != nil && val != "" {
				fmt.Printf("  %s: %v\n", f.label, val)
			}
		}
	}

	return nil
}

func (c *Client) serialCreateBatch(itemCode, prefix string, start, count int) error {
	fmt.Printf("%sCreating %d serial numbers...%s\n", Blue, count, Reset)
	fmt.Printf("  Item: %s\n", itemCode)
	fmt.Printf("  Pattern: %s%03d - %s%03d\n", prefix, start, prefix, start+count-1)

	created := 0
	failed := 0

	for i := 0; i < count; i++ {
		serialNo := fmt.Sprintf("%s%03d", prefix, start+i)

		body := map[string]interface{}{
			"serial_no": serialNo,
			"item_code": itemCode,
		}

		_, err := c.Request("POST", "Serial%20No", body)
		if err != nil {
			failed++
			fmt.Printf("  %s✗ Failed: %s (%s)%s\n", Red, serialNo, err, Reset)
		} else {
			created++
			fmt.Printf("  %s✓ Created: %s%s\n", Green, serialNo, Reset)
		}
	}

	fmt.Printf("\n%sSummary: %d created, %d failed%s\n", Cyan, created, failed, Reset)
	return nil
}
