package erp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CmdAttr handles attribute commands
func (c *Client) CmdAttr(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli attr <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create-text, create-numeric, create-list, add-values, delete")
		return nil
	}

	switch args[0] {
	case "list":
		return c.attrList()
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli attr get <name>")
		}
		return c.attrGet(args[1])
	case "create-text":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli attr create-text <name>")
		}
		return c.attrCreateText(args[1])
	case "create-numeric":
		if len(args) < 5 {
			return fmt.Errorf("usage: erp-cli attr create-numeric <name> <from> <to> <increment>")
		}
		return c.attrCreateNumeric(args[1], args[2], args[3], args[4])
	case "create-list":
		if len(args) < 3 {
			return fmt.Errorf("usage: erp-cli attr create-list <name> <value:abbr> [value:abbr...]")
		}
		return c.attrCreateList(args[1], args[2:])
	case "add-values":
		if len(args) < 3 {
			return fmt.Errorf("usage: erp-cli attr add-values <name> <value:abbr> [value:abbr...]")
		}
		return c.attrAddValues(args[1], args[2:])
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli attr delete <name>")
		}
		return c.attrDelete(args[1])
	default:
		return fmt.Errorf("unknown attr subcommand: %s", args[0])
	}
}

func (c *Client) attrList() error {
	fmt.Printf("%sFetching item attributes...%s\n", Blue, Reset)

	result, err := c.Request("GET", "Item%20Attribute?limit_page_length=0", nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				fmt.Println(m["name"])
			}
		}
	}
	return nil
}

func (c *Client) attrGet(name string) error {
	fmt.Printf("%sFetching attribute: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Item%20Attribute/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		output := map[string]interface{}{
			"name":    data["attribute_name"],
			"numeric": data["numeric_values"],
		}

		if values, ok := data["item_attribute_values"].([]interface{}); ok {
			var vals []map[string]string
			for _, v := range values {
				if vm, ok := v.(map[string]interface{}); ok {
					vals = append(vals, map[string]string{
						"value": fmt.Sprintf("%v", vm["attribute_value"]),
						"abbr":  fmt.Sprintf("%v", vm["abbr"]),
					})
				}
			}
			output["values"] = vals
		}

		jsonOut, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonOut))
	}
	return nil
}

func (c *Client) attrCreateText(name string) error {
	fmt.Printf("%sCreating text attribute: %s%s\n", Blue, name, Reset)

	body := map[string]interface{}{
		"attribute_name": name,
	}

	_, err := c.Request("POST", "Item%20Attribute", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Attribute created: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) attrCreateNumeric(name, from, to, increment string) error {
	fmt.Printf("%sCreating numeric attribute: %s (%s-%s, step %s)%s\n", Blue, name, from, to, increment, Reset)

	body := map[string]interface{}{
		"attribute_name": name,
		"numeric_values": 1,
		"from_range":     from,
		"to_range":       to,
		"increment":      increment,
	}

	_, err := c.Request("POST", "Item%20Attribute", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Numeric attribute created: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) attrCreateList(name string, values []string) error {
	fmt.Printf("%sCreating list attribute: %s%s\n", Blue, name, Reset)

	var attrValues []map[string]string
	for _, v := range values {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format '%s'. Use 'value:abbreviation'", v)
		}
		attrValues = append(attrValues, map[string]string{
			"attribute_value": parts[0],
			"abbr":            parts[1],
		})
	}

	body := map[string]interface{}{
		"attribute_name":        name,
		"item_attribute_values": attrValues,
	}

	_, err := c.Request("POST", "Item%20Attribute", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ List attribute created: %s%s\n", Green, name, Reset)
	fmt.Printf("  Values: %s\n", strings.Join(values, ", "))
	return nil
}

func (c *Client) attrAddValues(name string, values []string) error {
	fmt.Printf("%sAdding values to attribute: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	result, err := c.Request("GET", "Item%20Attribute/"+encoded, nil)
	if err != nil {
		return err
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("attribute not found")
	}

	var existingValues []map[string]interface{}
	if ev, ok := data["item_attribute_values"].([]interface{}); ok {
		for _, v := range ev {
			if vm, ok := v.(map[string]interface{}); ok {
				existingValues = append(existingValues, vm)
			}
		}
	}

	for _, v := range values {
		parts := strings.SplitN(v, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format '%s'. Use 'value:abbreviation'", v)
		}
		existingValues = append(existingValues, map[string]interface{}{
			"attribute_value": parts[0],
			"abbr":            parts[1],
		})
	}

	body := map[string]interface{}{
		"item_attribute_values": existingValues,
	}

	_, err = c.Request("PUT", "Item%20Attribute/"+encoded, body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Values added to: %s%s\n", Green, name, Reset)
	fmt.Printf("  New values: %s\n", strings.Join(values, ", "))
	return nil
}

func (c *Client) attrDelete(name string) error {
	fmt.Printf("%sDeleting attribute: %s%s\n", Blue, name, Reset)

	encoded := url.PathEscape(name)
	_, err := c.Request("DELETE", "Item%20Attribute/"+encoded, nil)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Attribute deleted: %s%s\n", Green, name, Reset)
	return nil
}
