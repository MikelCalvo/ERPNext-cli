package erp

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Supplier represents an ERPNext Supplier
type Supplier struct {
	Name          string `json:"name,omitempty"`
	SupplierName  string `json:"supplier_name"`
	SupplierGroup string `json:"supplier_group,omitempty"`
	Country       string `json:"country,omitempty"`
	Disabled      int    `json:"disabled,omitempty"`
}

// CmdSupplier handles supplier commands
func (c *Client) CmdSupplier(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli supplier <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create, delete")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli supplier list")
		fmt.Println("  erp-cli supplier get \"Intel Corporation\"")
		fmt.Println("  erp-cli supplier create \"New Supplier\" --group=\"Services\"")
		fmt.Println("  erp-cli supplier delete \"Old Supplier\"")
		return nil
	}

	switch args[0] {
	case "list":
		return c.supplierList()
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli supplier get <name>")
		}
		return c.supplierGet(args[1])
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli supplier create <name> [--group=X] [--country=X]")
		}
		opts := parseSupplierOptions(args[2:])
		return c.supplierCreate(args[1], opts)
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli supplier delete <name>")
		}
		return c.supplierDelete(args[1])
	default:
		return fmt.Errorf("unknown supplier subcommand: %s", args[0])
	}
}

type supplierOptions struct {
	group   string
	country string
}

func parseSupplierOptions(args []string) supplierOptions {
	opts := supplierOptions{}
	for _, arg := range args {
		if len(arg) > 8 && arg[:8] == "--group=" {
			opts.group = arg[8:]
		}
		if len(arg) > 10 && arg[:10] == "--country=" {
			opts.country = arg[10:]
		}
	}
	return opts
}

func (c *Client) supplierList() error {
	fmt.Printf("%sFetching suppliers...%s\n", Blue, Reset)

	result, err := c.Request("GET", "Supplier?limit_page_length=0&fields=[\"name\",\"supplier_name\",\"supplier_group\",\"country\",\"disabled\"]", nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			fmt.Printf("%sNo suppliers found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sSuppliers (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				supplierName := m["supplier_name"]
				group := m["supplier_group"]
				disabled, _ := m["disabled"].(float64)

				status := ""
				if disabled == 1 {
					status = fmt.Sprintf(" %s[disabled]%s", Red, Reset)
				}

				if supplierName != nil && supplierName != name {
					fmt.Printf("  %s (%s)%s", name, supplierName, status)
				} else {
					fmt.Printf("  %s%s", name, status)
				}

				if group != nil && group != "" {
					fmt.Printf(" - %s%s%s", Yellow, group, Reset)
				}
				fmt.Println()
			}
		}
	}
	return nil
}

func (c *Client) supplierGet(name string) error {
	fmt.Printf("%sFetching supplier: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Supplier/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		output := map[string]interface{}{
			"name":           data["name"],
			"supplier_name":  data["supplier_name"],
			"supplier_group": data["supplier_group"],
			"supplier_type":  data["supplier_type"],
			"country":        data["country"],
			"disabled":       data["disabled"],
		}

		// Remove nil/empty values
		for k, v := range output {
			if v == nil || v == "" || v == float64(0) {
				delete(output, k)
			}
		}

		jsonOut, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonOut))
	}
	return nil
}

func (c *Client) supplierCreate(name string, opts supplierOptions) error {
	fmt.Printf("%sCreating supplier: %s%s\n", Blue, name, Reset)

	body := map[string]interface{}{
		"supplier_name": name,
	}

	if opts.group != "" {
		body["supplier_group"] = opts.group
		fmt.Printf("  Group: %s\n", opts.group)
	}

	if opts.country != "" {
		body["country"] = opts.country
		fmt.Printf("  Country: %s\n", opts.country)
	}

	result, err := c.Request("POST", "Supplier", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("%s✓ Supplier created: %s%s\n", Green, data["name"], Reset)
	}

	return nil
}

func (c *Client) supplierDelete(name string) error {
	fmt.Printf("%sDeleting supplier: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	_, err := c.Request("DELETE", "Supplier/"+encoded, nil)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Supplier deleted: %s%s\n", Green, name, Reset)
	return nil
}
