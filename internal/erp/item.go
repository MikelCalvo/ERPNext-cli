package erp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CmdItem handles item commands
func (c *Client) CmdItem(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli item <subcommand> [args...]")
		fmt.Println("Subcommands: list, get, create, add-attr, set, delete")
		fmt.Println()
		fmt.Println("Set options:")
		fmt.Println("  item set <code> serial=on|off       Enable/disable serial numbers")
		fmt.Println("  item set <code> batch=on|off        Enable/disable batch numbers")
		fmt.Println("  item set <code> serial-series=XXX   Set serial number series (e.g., SN-.#####)")
		return nil
	}

	switch args[0] {
	case "list":
		templatesOnly := len(args) > 1 && args[1] == "--templates"
		return c.itemList(templatesOnly)
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli item get <code>")
		}
		return c.itemGet(args[1])
	case "create":
		if len(args) < 4 {
			return fmt.Errorf("usage: erp-cli item create <code> <name> <group>")
		}
		return c.itemCreate(args[1], args[2], args[3])
	case "add-attr":
		if len(args) < 3 {
			return fmt.Errorf("usage: erp-cli item add-attr <code> <attr1> [attr2...]")
		}
		return c.itemAddAttr(args[1], args[2:])
	case "set":
		if len(args) < 3 {
			return fmt.Errorf("usage: erp-cli item set <code> <property=value>")
		}
		return c.itemSet(args[1], args[2:])
	case "delete":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli item delete <code>")
		}
		return c.itemDelete(args[1])
	default:
		return fmt.Errorf("unknown item subcommand: %s", args[0])
	}
}

func (c *Client) itemList(templatesOnly bool) error {
	fmt.Printf("%sFetching items...%s\n", Blue, Reset)

	endpoint := "Item?limit_page_length=0"
	if templatesOnly {
		endpoint = "Item?limit_page_length=0&filters=%5B%5B%22has_variants%22%2C%22%3D%22%2C1%5D%5D"
		fmt.Printf("%sTemplates only:%s\n", Yellow, Reset)
	}

	result, err := c.Request("GET", endpoint, nil)
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

func (c *Client) itemGet(code string) error {
	fmt.Printf("%sFetching item: %s%s\n", Blue, code, Reset)

	encoded := url.PathEscape(code)
	result, err := c.Request("GET", "Item/"+encoded, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		var attrs []string
		if attributes, ok := data["attributes"].([]interface{}); ok {
			for _, a := range attributes {
				if am, ok := a.(map[string]interface{}); ok {
					attrs = append(attrs, fmt.Sprintf("%v", am["attribute"]))
				}
			}
		}

		hasSerial := "no"
		if hs, ok := data["has_serial_no"].(float64); ok && hs == 1 {
			hasSerial = "yes"
		}
		hasBatch := "no"
		if hb, ok := data["has_batch_no"].(float64); ok && hb == 1 {
			hasBatch = "yes"
		}

		output := map[string]interface{}{
			"item_code":        data["item_code"],
			"item_name":        data["item_name"],
			"item_group":       data["item_group"],
			"stock_uom":        data["stock_uom"],
			"has_variants":     data["has_variants"],
			"variant_of":       data["variant_of"],
			"has_serial_no":    hasSerial,
			"has_batch_no":     hasBatch,
			"serial_no_series": data["serial_no_series"],
			"attributes":       attrs,
		}

		for k, v := range output {
			if k == "has_serial_no" || k == "has_batch_no" {
				continue
			}
			if v == nil || v == "" || v == float64(0) {
				delete(output, k)
			}
		}

		jsonOut, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonOut))
	}
	return nil
}

func (c *Client) itemCreate(code, name, group string) error {
	fmt.Printf("%sCreating item: %s%s\n", Blue, code, Reset)

	body := map[string]interface{}{
		"item_code":     code,
		"item_name":     name,
		"item_group":    group,
		"stock_uom":     "Unit",
		"is_stock_item": 1,
	}

	_, err := c.Request("POST", "Item", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Item created: %s%s\n", Green, code, Reset)
	return nil
}

func (c *Client) itemAddAttr(code string, attrs []string) error {
	fmt.Printf("%sAdding attributes to item: %s%s\n", Blue, code, Reset)

	encoded := url.PathEscape(code)
	result, err := c.Request("GET", "Item/"+encoded, nil)
	if err != nil {
		return err
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("item not found")
	}

	var existingAttrs []map[string]interface{}
	if ea, ok := data["attributes"].([]interface{}); ok {
		for _, a := range ea {
			if am, ok := a.(map[string]interface{}); ok {
				existingAttrs = append(existingAttrs, am)
			}
		}
	}

	for _, attr := range attrs {
		existingAttrs = append(existingAttrs, map[string]interface{}{
			"attribute": attr,
		})
	}

	body := map[string]interface{}{
		"attributes": existingAttrs,
	}

	_, err = c.Request("PUT", "Item/"+encoded, body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Attributes added to: %s%s\n", Green, code, Reset)
	fmt.Printf("  New attributes: %s\n", strings.Join(attrs, ", "))
	return nil
}

func (c *Client) itemDelete(code string) error {
	fmt.Printf("%sDeleting item: %s%s\n", Blue, code, Reset)

	encoded := url.PathEscape(code)
	_, err := c.Request("DELETE", "Item/"+encoded, nil)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Item deleted: %s%s\n", Green, code, Reset)
	return nil
}

func (c *Client) itemSet(code string, settings []string) error {
	fmt.Printf("%sUpdating item: %s%s\n", Blue, code, Reset)

	body := make(map[string]interface{})

	for _, setting := range settings {
		parts := strings.SplitN(setting, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid setting format '%s'. Use 'property=value'", setting)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "serial", "has_serial_no":
			if value == "on" || value == "1" || value == "true" {
				body["has_serial_no"] = 1
				fmt.Printf("  Serial Numbers: %senabled%s\n", Green, Reset)
			} else if value == "off" || value == "0" || value == "false" {
				body["has_serial_no"] = 0
				fmt.Printf("  Serial Numbers: %sdisabled%s\n", Yellow, Reset)
			} else {
				return fmt.Errorf("invalid value for serial: use 'on' or 'off'")
			}

		case "batch", "has_batch_no":
			if value == "on" || value == "1" || value == "true" {
				body["has_batch_no"] = 1
				fmt.Printf("  Batch Numbers: %senabled%s\n", Green, Reset)
			} else if value == "off" || value == "0" || value == "false" {
				body["has_batch_no"] = 0
				fmt.Printf("  Batch Numbers: %sdisabled%s\n", Yellow, Reset)
			} else {
				return fmt.Errorf("invalid value for batch: use 'on' or 'off'")
			}

		case "serial-series", "serial_no_series":
			body["serial_no_series"] = value
			fmt.Printf("  Serial Series: %s%s%s\n", Cyan, value, Reset)

		case "stock", "is_stock_item":
			if value == "on" || value == "1" || value == "true" {
				body["is_stock_item"] = 1
				fmt.Printf("  Stock Item: %senabled%s\n", Green, Reset)
			} else if value == "off" || value == "0" || value == "false" {
				body["is_stock_item"] = 0
				fmt.Printf("  Stock Item: %sdisabled%s\n", Yellow, Reset)
			} else {
				return fmt.Errorf("invalid value for stock: use 'on' or 'off'")
			}

		case "valuation", "valuation_method":
			body["valuation_method"] = value
			fmt.Printf("  Valuation Method: %s\n", value)

		case "warranty", "warranty_period":
			body["warranty_period"] = value
			fmt.Printf("  Warranty Period: %s days\n", value)

		default:
			body[key] = value
			fmt.Printf("  %s: %s\n", key, value)
		}
	}

	if len(body) == 0 {
		return fmt.Errorf("no valid settings provided")
	}

	encoded := url.PathEscape(code)
	_, err := c.Request("PUT", "Item/"+encoded, body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Item updated: %s%s\n", Green, code, Reset)
	return nil
}

// CmdTemplate handles template commands
func (c *Client) CmdTemplate(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli template <subcommand> [args...]")
		fmt.Println("Subcommands: create")
		return nil
	}

	switch args[0] {
	case "create":
		if len(args) < 5 {
			return fmt.Errorf("usage: erp-cli template create <code> <name> <group> <attr1> [attr2...]")
		}
		return c.templateCreate(args[1], args[2], args[3], args[4:])
	default:
		return fmt.Errorf("unknown template subcommand: %s", args[0])
	}
}

func (c *Client) templateCreate(code, name, group string, attrs []string) error {
	fmt.Printf("%sCreating template: %s%s\n", Blue, code, Reset)
	fmt.Printf("  Attributes: %s\n", strings.Join(attrs, ", "))

	var attrList []map[string]string
	for _, attr := range attrs {
		attrList = append(attrList, map[string]string{"attribute": attr})
	}

	body := map[string]interface{}{
		"item_code":     code,
		"item_name":     name,
		"item_group":    group,
		"stock_uom":     "Unit",
		"is_stock_item": 1,
		"has_variants":  1,
		"attributes":    attrList,
	}

	result, err := c.Request("POST", "Item", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Template created: %s%s\n", Green, code, Reset)

	if data, ok := result["data"].(map[string]interface{}); ok {
		output := map[string]interface{}{
			"item_code":    data["item_code"],
			"item_name":    data["item_name"],
			"has_variants": data["has_variants"],
			"attributes":   attrs,
		}
		jsonOut, _ := json.MarshalIndent(output, "", "  ")
		fmt.Println(string(jsonOut))
	}

	return nil
}

// CmdGroup handles group commands
func (c *Client) CmdGroup(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli group <subcommand> [args...]")
		fmt.Println("Subcommands: list, create")
		return nil
	}

	switch args[0] {
	case "list":
		return c.groupList()
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli group create <name> [parent]")
		}
		parent := "All Item Groups"
		if len(args) > 2 {
			parent = args[2]
		}
		return c.groupCreate(args[1], parent)
	default:
		return fmt.Errorf("unknown group subcommand: %s", args[0])
	}
}

func (c *Client) groupList() error {
	fmt.Printf("%sFetching item groups...%s\n", Blue, Reset)

	result, err := c.Request("GET", "Item%20Group?limit_page_length=0", nil)
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

func (c *Client) groupCreate(name, parent string) error {
	fmt.Printf("%sCreating item group: %s%s\n", Blue, name, Reset)

	body := map[string]interface{}{
		"item_group_name":   name,
		"parent_item_group": parent,
	}

	_, err := c.Request("POST", "Item%20Group", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Group created: %s%s\n", Green, name, Reset)
	fmt.Printf("  Parent: %s\n", parent)
	return nil
}

// CmdBrand handles brand commands
func (c *Client) CmdBrand(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli brand <subcommand> [args...]")
		fmt.Println("Subcommands: list, create, add-to-attr")
		return nil
	}

	switch args[0] {
	case "list":
		return c.brandList()
	case "create":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli brand create <name>")
		}
		return c.brandCreate(args[1])
	case "add-to-attr":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli brand add-to-attr <name>")
		}
		return c.brandAddToAttr(args[1])
	default:
		return fmt.Errorf("unknown brand subcommand: %s", args[0])
	}
}

func (c *Client) brandList() error {
	fmt.Printf("%sFetching brands...%s\n", Blue, Reset)

	result, err := c.Request("GET", "Brand?limit_page_length=0", nil)
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

func (c *Client) brandCreate(name string) error {
	fmt.Printf("%sCreating brand: %s%s\n", Blue, name, Reset)

	body := map[string]interface{}{
		"brand": name,
	}

	_, err := c.Request("POST", "Brand", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Brand created: %s%s\n", Green, name, Reset)
	return nil
}

func (c *Client) brandAddToAttr(name string) error {
	fmt.Printf("%sAdding brand to Brand attribute: %s%s\n", Blue, name, Reset)
	return c.attrAddValues("Brand", []string{name + ":" + name})
}
