package erp

import (
	"fmt"
	"net/url"
	"strings"
)

// CmdVariant handles variant commands
func (c *Client) CmdVariant(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli variant <subcommand> [args...]")
		fmt.Println("Subcommands: create, list")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli variant list PSU-ATX")
		fmt.Println("  erp-cli variant create PSU-ATX PSU-EVGA-500-80G \"Brand=EVGA\" \"Wattage (W)=500\"")
		return nil
	}

	switch args[0] {
	case "create":
		if len(args) < 4 {
			return fmt.Errorf("usage: erp-cli variant create <template> <code> <attr1=val1> [attr2=val2...]")
		}
		return c.variantCreate(args[1], args[2], args[3:])
	case "list":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli variant list <template>")
		}
		return c.variantList(args[1])
	default:
		return fmt.Errorf("unknown variant subcommand: %s", args[0])
	}
}

func (c *Client) variantCreate(template, code string, attrPairs []string) error {
	fmt.Printf("%sCreating variant from template: %s%s\n", Blue, template, Reset)
	fmt.Printf("  Variant code: %s\n", code)

	encoded := url.PathEscape(template)
	result, err := c.Request("GET", "Item/"+encoded, nil)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	data, ok := result["data"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("template not found: %s", template)
	}

	hasVariants, _ := data["has_variants"].(float64)
	if hasVariants != 1 {
		return fmt.Errorf("%s is not a template (has_variants=0)", template)
	}

	itemGroup, _ := data["item_group"].(string)
	stockUom, _ := data["stock_uom"].(string)
	if stockUom == "" {
		stockUom = "Unit"
	}

	templateAttrs := make(map[string]bool)
	if attrs, ok := data["attributes"].([]interface{}); ok {
		for _, a := range attrs {
			if am, ok := a.(map[string]interface{}); ok {
				if attrName, ok := am["attribute"].(string); ok {
					templateAttrs[attrName] = true
				}
			}
		}
	}

	variantAttrs := make([]map[string]string, 0)
	providedAttrs := make(map[string]string)

	for _, pair := range attrPairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid attribute format '%s'. Use 'Attribute=Value'", pair)
		}
		attrName := strings.TrimSpace(parts[0])
		attrValue := strings.TrimSpace(parts[1])

		if !templateAttrs[attrName] {
			return fmt.Errorf("attribute '%s' is not defined in template '%s'", attrName, template)
		}

		providedAttrs[attrName] = attrValue
		variantAttrs = append(variantAttrs, map[string]string{
			"attribute":       attrName,
			"attribute_value": attrValue,
		})
	}

	for attr := range templateAttrs {
		if _, ok := providedAttrs[attr]; !ok {
			return fmt.Errorf("missing required attribute: %s", attr)
		}
	}

	variantName := template
	for _, pair := range attrPairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			variantName += " " + parts[1]
		}
	}

	body := map[string]interface{}{
		"item_code":     code,
		"item_name":     variantName,
		"variant_of":    template,
		"item_group":    itemGroup,
		"is_stock_item": 1,
		"stock_uom":     stockUom,
		"attributes":    variantAttrs,
	}

	_, err = c.Request("POST", "Item", body)
	if err != nil {
		return err
	}

	fmt.Printf("%s✓ Variant created: %s%s\n", Green, code, Reset)
	fmt.Printf("  Name: %s\n", variantName)
	fmt.Printf("  Group: %s\n", itemGroup)
	fmt.Printf("  Attributes:\n")
	for attr, val := range providedAttrs {
		fmt.Printf("    • %s = %s\n", attr, val)
	}

	return nil
}

func (c *Client) variantList(template string) error {
	fmt.Printf("%sFetching variants of: %s%s\n", Blue, template, Reset)

	filter := fmt.Sprintf(`[["variant_of","=","%s"]]`, template)
	encodedFilter := url.QueryEscape(filter)

	result, err := c.Request("GET", "Item?limit_page_length=0&filters="+encodedFilter, nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			fmt.Printf("%sNo variants found for template: %s%s\n", Yellow, template, Reset)
			return nil
		}

		fmt.Printf("\n%sVariants (%d):%s\n", Cyan, len(data), Reset)
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				code := m["name"]
				name := m["item_name"]
				fmt.Printf("  • %s%s%s - %s\n", Green, code, Reset, name)
			}
		}
	}

	return nil
}
