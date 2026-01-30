package erp

import (
	"encoding/json"
	"fmt"
	"net/url"
)

// Customer represents an ERPNext Customer
type Customer struct {
	Name          string `json:"name,omitempty"`
	CustomerName  string `json:"customer_name"`
	CustomerGroup string `json:"customer_group,omitempty"`
	Territory     string `json:"territory,omitempty"`
	CustomerType  string `json:"customer_type,omitempty"`
	Disabled      int    `json:"disabled,omitempty"`
}

// CmdCustomer handles customer commands
func (c *Client) CmdCustomer(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli customer <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create, delete")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli customer list")
		fmt.Println("  erp-cli customer get \"Acme Corp\"")
		fmt.Println("  erp-cli customer create \"New Customer\" --group=\"Commercial\" --territory=\"Spain\"")
		fmt.Println("  erp-cli customer delete \"Old Customer\"")
		return nil
	}

	switch args[0] {
	case "list":
		return c.customerList()
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli customer get <name>")
		}
		return c.customerGet(args[1])
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli customer create <name> [--group=X] [--territory=X]")
		}
		opts := parseCustomerOptions(args[2:])
		return c.customerCreate(args[1], opts)
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli customer delete <name>")
		}
		return c.customerDelete(args[1])
	default:
		return fmt.Errorf("unknown customer subcommand: %s", args[0])
	}
}

type customerOptions struct {
	group     string
	territory string
}

func parseCustomerOptions(args []string) customerOptions {
	opts := customerOptions{}
	for _, arg := range args {
		if len(arg) > 8 && arg[:8] == "--group=" {
			opts.group = arg[8:]
		}
		if len(arg) > 12 && arg[:12] == "--territory=" {
			opts.territory = arg[12:]
		}
	}
	return opts
}

func (c *Client) customerList() error {
	fmt.Printf("%sFetching customers...%s\n", Blue, Reset)

	result, err := c.Request("GET", "Customer?limit_page_length=0&fields=[\"name\",\"customer_name\",\"customer_group\",\"territory\",\"disabled\"]", nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			fmt.Printf("%sNo customers found%s\n", Yellow, Reset)
			return nil
		}

		fmt.Printf("\n%sCustomers (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				customerName := m["customer_name"]
				group := m["customer_group"]
				disabled, _ := m["disabled"].(float64)

				status := ""
				if disabled == 1 {
					status = fmt.Sprintf(" %s[disabled]%s", Red, Reset)
				}

				if customerName != nil && customerName != name {
					fmt.Printf("  %s (%s)%s", name, customerName, status)
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

func (c *Client) customerGet(name string) error {
	fmt.Printf("%sFetching customer: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Customer/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		output := map[string]interface{}{
			"name":           data["name"],
			"customer_name":  data["customer_name"],
			"customer_group": data["customer_group"],
			"customer_type":  data["customer_type"],
			"territory":      data["territory"],
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

func (c *Client) customerCreate(name string, opts customerOptions) error {
	fmt.Printf("%sCreating customer: %s%s\n", Blue, name, Reset)

	body := map[string]interface{}{
		"customer_name": name,
	}

	if opts.group != "" {
		body["customer_group"] = opts.group
		fmt.Printf("  Group: %s\n", opts.group)
	}

	if opts.territory != "" {
		body["territory"] = opts.territory
		fmt.Printf("  Territory: %s\n", opts.territory)
	}

	result, err := c.Request("POST", "Customer", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		fmt.Printf("%s✓ Customer created: %s%s\n", Green, data["name"], Reset)
	}

	return nil
}

func (c *Client) customerDelete(name string) error {
	fmt.Printf("%sDeleting customer: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	_, err := c.Request("DELETE", "Customer/"+encoded, nil)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Customer deleted: %s%s\n", Green, name, Reset)
	return nil
}
