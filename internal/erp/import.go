package erp

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// CmdExport handles export commands
func (c *Client) CmdExport(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli export <type> -o <file>")
		fmt.Println("Types: items, templates, attributes, variants")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli export items -o items.csv")
		fmt.Println("  erp-cli export templates -o templates.csv")
		fmt.Println("  erp-cli export attributes -o attrs.csv")
		fmt.Println("  erp-cli export variants PSU-ATX -o psu-variants.csv")
		return nil
	}

	outputFile := ""
	for i, arg := range args {
		if arg == "-o" && i+1 < len(args) {
			outputFile = args[i+1]
		}
	}

	if outputFile == "" {
		return fmt.Errorf("output file required. Use -o <file>")
	}

	switch args[0] {
	case "items":
		return c.exportItems(outputFile, false)
	case "templates":
		return c.exportItems(outputFile, true)
	case "attributes":
		return c.exportAttributes(outputFile)
	case "variants":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli export variants <template> -o <file>")
		}
		return c.exportVariants(args[1], outputFile)
	default:
		return fmt.Errorf("unknown export type: %s", args[0])
	}
}

// CmdImport handles import commands
func (c *Client) CmdImport(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli import <type> -f <file> [--dry-run]")
		fmt.Println("Types: items, variants")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli import items -f items.csv")
		fmt.Println("  erp-cli import variants -f variants.csv --dry-run")
		return nil
	}

	inputFile := ""
	dryRun := false

	for i, arg := range args {
		if arg == "-f" && i+1 < len(args) {
			inputFile = args[i+1]
		}
		if arg == "--dry-run" {
			dryRun = true
		}
	}

	if inputFile == "" {
		return fmt.Errorf("input file required. Use -f <file>")
	}

	switch args[0] {
	case "items":
		return c.importItems(inputFile, dryRun)
	case "variants":
		return c.importVariants(inputFile, dryRun)
	default:
		return fmt.Errorf("unknown import type: %s", args[0])
	}
}

func (c *Client) exportItems(outputFile string, templatesOnly bool) error {
	itemType := "items"
	if templatesOnly {
		itemType = "templates"
	}
	fmt.Printf("%sExporting %s...%s\n", Blue, itemType, Reset)

	endpoint := "Item?limit_page_length=0&fields=[\"item_code\",\"item_name\",\"item_group\",\"stock_uom\",\"has_variants\",\"variant_of\"]"
	if templatesOnly {
		endpoint += "&filters=%5B%5B%22has_variants%22%2C%22%3D%22%2C1%5D%5D"
	}

	result, err := c.Request("GET", endpoint, nil)
	if err != nil {
		return err
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"item_code", "item_name", "item_group", "stock_uom", "has_variants", "variant_of"}
	writer.Write(header)

	count := 0
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				row := make([]string, len(header))
				for i, col := range header {
					if val, ok := m[col]; ok && val != nil {
						row[i] = fmt.Sprintf("%v", val)
					}
				}
				writer.Write(row)
				count++
			}
		}
	}

	fmt.Printf("%s✓ Exported %d %s to %s%s\n", Green, count, itemType, outputFile, Reset)
	return nil
}

func (c *Client) exportAttributes(outputFile string) error {
	fmt.Printf("%sExporting attributes...%s\n", Blue, Reset)

	result, err := c.Request("GET", "Item%20Attribute?limit_page_length=0", nil)
	if err != nil {
		return err
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"attribute_name", "numeric_values", "from_range", "to_range", "increment", "values"}
	writer.Write(header)

	count := 0
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := fmt.Sprintf("%v", m["name"])

				encoded := url.PathEscape(name)
				detail, err := c.Request("GET", "Item%20Attribute/"+encoded, nil)
				if err != nil {
					continue
				}

				if attrData, ok := detail["data"].(map[string]interface{}); ok {
					row := []string{
						fmt.Sprintf("%v", attrData["attribute_name"]),
						fmt.Sprintf("%v", attrData["numeric_values"]),
						fmt.Sprintf("%v", attrData["from_range"]),
						fmt.Sprintf("%v", attrData["to_range"]),
						fmt.Sprintf("%v", attrData["increment"]),
					}

					var values []string
					if attrValues, ok := attrData["item_attribute_values"].([]interface{}); ok {
						for _, v := range attrValues {
							if vm, ok := v.(map[string]interface{}); ok {
								values = append(values, fmt.Sprintf("%v:%v", vm["attribute_value"], vm["abbr"]))
							}
						}
					}
					row = append(row, strings.Join(values, "|"))

					writer.Write(row)
					count++
				}
			}
		}
	}

	fmt.Printf("%s✓ Exported %d attributes to %s%s\n", Green, count, outputFile, Reset)
	return nil
}

func (c *Client) exportVariants(template, outputFile string) error {
	fmt.Printf("%sExporting variants of: %s%s\n", Blue, template, Reset)

	encoded := url.PathEscape(template)
	tplResult, err := c.Request("GET", "Item/"+encoded, nil)
	if err != nil {
		return err
	}

	var attrNames []string
	if data, ok := tplResult["data"].(map[string]interface{}); ok {
		if attrs, ok := data["attributes"].([]interface{}); ok {
			for _, a := range attrs {
				if am, ok := a.(map[string]interface{}); ok {
					if attrName, ok := am["attribute"].(string); ok {
						attrNames = append(attrNames, attrName)
					}
				}
			}
		}
	}

	filter := fmt.Sprintf(`[["variant_of","=","%s"]]`, template)
	encodedFilter := url.QueryEscape(filter)
	result, err := c.Request("GET", "Item?limit_page_length=0&filters="+encodedFilter, nil)
	if err != nil {
		return err
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	header := []string{"template", "item_code", "item_name"}
	header = append(header, attrNames...)
	writer.Write(header)

	count := 0
	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				variantCode := fmt.Sprintf("%v", m["name"])

				variantEncoded := url.PathEscape(variantCode)
				varResult, err := c.Request("GET", "Item/"+variantEncoded, nil)
				if err != nil {
					continue
				}

				if varData, ok := varResult["data"].(map[string]interface{}); ok {
					row := []string{
						template,
						fmt.Sprintf("%v", varData["item_code"]),
						fmt.Sprintf("%v", varData["item_name"]),
					}

					attrValues := make(map[string]string)
					if attrs, ok := varData["attributes"].([]interface{}); ok {
						for _, a := range attrs {
							if am, ok := a.(map[string]interface{}); ok {
								attrName := fmt.Sprintf("%v", am["attribute"])
								attrValue := fmt.Sprintf("%v", am["attribute_value"])
								attrValues[attrName] = attrValue
							}
						}
					}

					for _, attrName := range attrNames {
						row = append(row, attrValues[attrName])
					}

					writer.Write(row)
					count++
				}
			}
		}
	}

	fmt.Printf("%s✓ Exported %d variants to %s%s\n", Green, count, outputFile, Reset)
	return nil
}

func (c *Client) importItems(inputFile string, dryRun bool) error {
	if dryRun {
		fmt.Printf("%s[DRY RUN] Importing items from: %s%s\n", Yellow, inputFile, Reset)
	} else {
		fmt.Printf("%sImporting items from: %s%s\n", Blue, inputFile, Reset)
	}

	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty or has no data rows")
	}

	header := records[0]
	created := 0
	skipped := 0
	failed := 0

	for i, record := range records[1:] {
		if len(record) < 3 {
			fmt.Printf("  %sRow %d: skipped (insufficient columns)%s\n", Yellow, i+2, Reset)
			skipped++
			continue
		}

		item := make(map[string]interface{})
		for j, col := range header {
			if j < len(record) && record[j] != "" {
				if col == "has_variants" || col == "is_stock_item" {
					if record[j] == "1" || record[j] == "true" {
						item[col] = 1
					} else {
						item[col] = 0
					}
				} else {
					item[col] = record[j]
				}
			}
		}

		if item["item_code"] == nil || item["item_code"] == "" {
			fmt.Printf("  %sRow %d: skipped (no item_code)%s\n", Yellow, i+2, Reset)
			skipped++
			continue
		}

		if item["stock_uom"] == nil {
			item["stock_uom"] = "Unit"
		}
		if item["is_stock_item"] == nil {
			item["is_stock_item"] = 1
		}

		if dryRun {
			fmt.Printf("  [DRY RUN] Would create: %s\n", item["item_code"])
			created++
		} else {
			_, err := c.Request("POST", "Item", item)
			if err != nil {
				fmt.Printf("  %s✗ Failed: %s (%s)%s\n", Red, item["item_code"], err, Reset)
				failed++
			} else {
				fmt.Printf("  %s✓ Created: %s%s\n", Green, item["item_code"], Reset)
				created++
			}
		}
	}

	fmt.Printf("\n%sSummary: %d created, %d skipped, %d failed%s\n", Cyan, created, skipped, failed, Reset)
	return nil
}

func (c *Client) importVariants(inputFile string, dryRun bool) error {
	if dryRun {
		fmt.Printf("%s[DRY RUN] Importing variants from: %s%s\n", Yellow, inputFile, Reset)
	} else {
		fmt.Printf("%sImporting variants from: %s%s\n", Blue, inputFile, Reset)
	}

	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty or has no data rows")
	}

	header := records[0]

	templateIdx := -1
	codeIdx := -1
	nameIdx := -1
	attrIndices := make(map[int]string)

	for i, col := range header {
		switch col {
		case "template":
			templateIdx = i
		case "item_code":
			codeIdx = i
		case "item_name":
			nameIdx = i
		default:
			attrIndices[i] = col
		}
	}

	if templateIdx == -1 || codeIdx == -1 {
		return fmt.Errorf("CSV must have 'template' and 'item_code' columns")
	}

	created := 0
	skipped := 0
	failed := 0

	templateCache := make(map[string]map[string]interface{})

	for i, record := range records[1:] {
		if len(record) < 3 {
			fmt.Printf("  %sRow %d: skipped (insufficient columns)%s\n", Yellow, i+2, Reset)
			skipped++
			continue
		}

		template := record[templateIdx]
		code := record[codeIdx]
		name := ""
		if nameIdx >= 0 && nameIdx < len(record) {
			name = record[nameIdx]
		}

		var tplData map[string]interface{}
		if cached, ok := templateCache[template]; ok {
			tplData = cached
		} else {
			encoded := url.PathEscape(template)
			result, err := c.Request("GET", "Item/"+encoded, nil)
			if err != nil {
				fmt.Printf("  %sRow %d: skipped (template not found: %s)%s\n", Yellow, i+2, template, Reset)
				skipped++
				continue
			}
			if data, ok := result["data"].(map[string]interface{}); ok {
				templateCache[template] = data
				tplData = data
			}
		}

		itemGroup, _ := tplData["item_group"].(string)
		stockUom, _ := tplData["stock_uom"].(string)
		if stockUom == "" {
			stockUom = "Unit"
		}

		var attributes []map[string]string
		for idx, attrName := range attrIndices {
			if idx < len(record) && record[idx] != "" {
				attributes = append(attributes, map[string]string{
					"attribute":       attrName,
					"attribute_value": record[idx],
				})
			}
		}

		if name == "" {
			name = template
			for _, attr := range attributes {
				name += " " + attr["attribute_value"]
			}
		}

		if dryRun {
			fmt.Printf("  [DRY RUN] Would create: %s (%s)\n", code, name)
			attrStr, _ := json.Marshal(attributes)
			fmt.Printf("    Attributes: %s\n", attrStr)
			created++
		} else {
			body := map[string]interface{}{
				"item_code":     code,
				"item_name":     name,
				"variant_of":    template,
				"item_group":    itemGroup,
				"is_stock_item": 1,
				"stock_uom":     stockUom,
				"attributes":    attributes,
			}

			_, err := c.Request("POST", "Item", body)
			if err != nil {
				fmt.Printf("  %s✗ Failed: %s (%s)%s\n", Red, code, err, Reset)
				failed++
			} else {
				fmt.Printf("  %s✓ Created: %s%s\n", Green, code, Reset)
				created++
			}
		}
	}

	fmt.Printf("\n%sSummary: %d created, %d skipped, %d failed%s\n", Cyan, created, skipped, failed, Reset)
	return nil
}
