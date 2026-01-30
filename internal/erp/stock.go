package erp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

// CmdWarehouse handles warehouse commands
func (c *Client) CmdWarehouse(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli warehouse <subcommand> [args...]")
		fmt.Println("Subcommands: list")
		return nil
	}

	switch args[0] {
	case "list":
		return c.warehouseList()
	default:
		return fmt.Errorf("unknown warehouse subcommand: %s", args[0])
	}
}

func (c *Client) warehouseList() error {
	fmt.Printf("%sFetching warehouses...%s\n", Blue, Reset)

	result, err := c.Request("GET", "Warehouse?limit_page_length=0&fields=[\"name\",\"warehouse_name\",\"is_group\",\"parent_warehouse\"]", nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				name := m["name"]
				isGroup := m["is_group"]
				parent := m["parent_warehouse"]

				prefix := "  "
				if isGroup == float64(1) {
					prefix = "📁"
				} else {
					prefix = "  📦"
				}

				if parent != nil && parent != "" {
					fmt.Printf("%s %s (parent: %s)\n", prefix, name, parent)
				} else {
					fmt.Printf("%s %s\n", prefix, name)
				}
			}
		}
	}
	return nil
}

// CmdStock handles stock commands
func (c *Client) CmdStock(args []string) error {
	if len(args) == 0 {
		fmt.Println("Usage: erp-cli stock <subcommand> [args...]")
		fmt.Println("Subcommands: get, receive, transfer, issue")
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  erp-cli stock get CPU-I7-12700K")
		fmt.Println("  erp-cli stock get CPU-I7-12700K \"Stores\"")
		fmt.Println("  erp-cli stock receive CPU-I7-12700K 10 \"Stores\" --rate=450")
		fmt.Println("  erp-cli stock transfer CPU-I7-12700K 5 \"Stores\" \"Dispatch\"")
		fmt.Println("  erp-cli stock issue CPU-I7-12700K 2 \"Stores\"")
		return nil
	}

	switch args[0] {
	case "get":
		if len(args) < 2 {
			return fmt.Errorf("usage: erp-cli stock get <item_code> [warehouse]")
		}
		warehouse := ""
		if len(args) > 2 {
			warehouse = args[2]
		}
		return c.stockGet(args[1], warehouse)
	case "receive":
		if len(args) < 4 {
			return fmt.Errorf("usage: erp-cli stock receive <item_code> <qty> <warehouse> [--rate=X]")
		}
		rate := 0.0
		for _, arg := range args[4:] {
			if len(arg) > 7 && arg[:7] == "--rate=" {
				rate, _ = strconv.ParseFloat(arg[7:], 64)
			}
		}
		qty, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return fmt.Errorf("invalid quantity: %s", args[2])
		}
		return c.stockReceive(args[1], qty, args[3], rate)
	case "transfer":
		if len(args) < 5 {
			return fmt.Errorf("usage: erp-cli stock transfer <item_code> <qty> <from_warehouse> <to_warehouse>")
		}
		qty, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return fmt.Errorf("invalid quantity: %s", args[2])
		}
		return c.stockTransfer(args[1], qty, args[3], args[4])
	case "issue":
		if len(args) < 4 {
			return fmt.Errorf("usage: erp-cli stock issue <item_code> <qty> <warehouse>")
		}
		qty, err := strconv.ParseFloat(args[2], 64)
		if err != nil {
			return fmt.Errorf("invalid quantity: %s", args[2])
		}
		return c.stockIssue(args[1], qty, args[3])
	default:
		return fmt.Errorf("unknown stock subcommand: %s", args[0])
	}
}

func (c *Client) stockGet(itemCode, warehouse string) error {
	fmt.Printf("%sFetching stock for: %s%s\n", Blue, itemCode, Reset)

	filter := fmt.Sprintf(`[["item_code","=","%s"]]`, itemCode)
	if warehouse != "" {
		filter = fmt.Sprintf(`[["item_code","=","%s"],["warehouse","=","%s"]]`, itemCode, warehouse)
	}
	encodedFilter := url.QueryEscape(filter)

	result, err := c.Request("GET", "Bin?filters="+encodedFilter+"&fields=[\"warehouse\",\"actual_qty\",\"reserved_qty\",\"ordered_qty\"]", nil)
	if err != nil {
		return err
	}

	if data, ok := result["data"].([]interface{}); ok {
		if len(data) == 0 {
			fmt.Printf("%sNo stock found for: %s%s\n", Yellow, itemCode, Reset)
			return nil
		}

		fmt.Printf("\n%sStock for %s:%s\n", Cyan, itemCode, Reset)
		totalQty := 0.0
		for _, item := range data {
			if m, ok := item.(map[string]interface{}); ok {
				wh := m["warehouse"]
				actualQty, _ := m["actual_qty"].(float64)
				reservedQty, _ := m["reserved_qty"].(float64)
				orderedQty, _ := m["ordered_qty"].(float64)

				totalQty += actualQty

				fmt.Printf("  📦 %s\n", wh)
				fmt.Printf("     Actual: %s%.0f%s", Green, actualQty, Reset)
				if reservedQty > 0 {
					fmt.Printf(" | Reserved: %s%.0f%s", Yellow, reservedQty, Reset)
				}
				if orderedQty > 0 {
					fmt.Printf(" | Ordered: %s%.0f%s", Blue, orderedQty, Reset)
				}
				fmt.Println()
			}
		}

		if warehouse == "" && len(data) > 1 {
			fmt.Printf("\n  %sTotal: %.0f%s\n", Cyan, totalQty, Reset)
		}
	}

	return nil
}

func (c *Client) stockReceive(itemCode string, qty float64, warehouse string, rate float64) error {
	fmt.Printf("%sReceiving stock...%s\n", Blue, Reset)
	fmt.Printf("  Item: %s\n", itemCode)
	fmt.Printf("  Quantity: %.0f\n", qty)
	fmt.Printf("  Warehouse: %s\n", warehouse)

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	item := map[string]interface{}{
		"item_code":   itemCode,
		"qty":         qty,
		"t_warehouse": warehouse,
	}

	if rate > 0 {
		item["basic_rate"] = rate
		fmt.Printf("  Rate: %s\n", c.FormatCurrency(rate))
	}

	body := map[string]interface{}{
		"stock_entry_type": "Material Receipt",
		"company":          company,
		"items":            []interface{}{item},
	}

	result, err := c.Request("POST", "Stock%20Entry", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		entryName := data["name"]
		fmt.Printf("%s✓ Stock Entry created: %s%s\n", Green, entryName, Reset)

		if err := c.submitStockEntry(fmt.Sprintf("%v", entryName)); err != nil {
			fmt.Printf("%sWarning: Entry created but not submitted: %s%s\n", Yellow, err, Reset)
			fmt.Println("  You may need to submit it manually in the ERP")
		} else {
			fmt.Printf("%s✓ Stock Entry submitted%s\n", Green, Reset)
		}
	}

	return nil
}

func (c *Client) stockTransfer(itemCode string, qty float64, fromWarehouse, toWarehouse string) error {
	fmt.Printf("%sTransferring stock...%s\n", Blue, Reset)
	fmt.Printf("  Item: %s\n", itemCode)
	fmt.Printf("  Quantity: %.0f\n", qty)
	fmt.Printf("  From: %s\n", fromWarehouse)
	fmt.Printf("  To: %s\n", toWarehouse)

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"stock_entry_type": "Material Transfer",
		"company":          company,
		"items": []interface{}{
			map[string]interface{}{
				"item_code":   itemCode,
				"qty":         qty,
				"s_warehouse": fromWarehouse,
				"t_warehouse": toWarehouse,
			},
		},
	}

	result, err := c.Request("POST", "Stock%20Entry", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		entryName := data["name"]
		fmt.Printf("%s✓ Stock Entry created: %s%s\n", Green, entryName, Reset)

		if err := c.submitStockEntry(fmt.Sprintf("%v", entryName)); err != nil {
			fmt.Printf("%sWarning: Entry created but not submitted: %s%s\n", Yellow, err, Reset)
		} else {
			fmt.Printf("%s✓ Stock Entry submitted%s\n", Green, Reset)
		}
	}

	return nil
}

func (c *Client) stockIssue(itemCode string, qty float64, warehouse string) error {
	fmt.Printf("%sIssuing stock...%s\n", Blue, Reset)
	fmt.Printf("  Item: %s\n", itemCode)
	fmt.Printf("  Quantity: %.0f\n", qty)
	fmt.Printf("  Warehouse: %s\n", warehouse)

	company, err := c.GetCompany()
	if err != nil {
		return err
	}

	body := map[string]interface{}{
		"stock_entry_type": "Material Issue",
		"company":          company,
		"items": []interface{}{
			map[string]interface{}{
				"item_code":   itemCode,
				"qty":         qty,
				"s_warehouse": warehouse,
			},
		},
	}

	result, err := c.Request("POST", "Stock%20Entry", body)
	if err != nil {
		return err
	}

	if data, ok := result["data"].(map[string]interface{}); ok {
		entryName := data["name"]
		fmt.Printf("%s✓ Stock Entry created: %s%s\n", Green, entryName, Reset)

		if err := c.submitStockEntry(fmt.Sprintf("%v", entryName)); err != nil {
			fmt.Printf("%sWarning: Entry created but not submitted: %s%s\n", Yellow, err, Reset)
		} else {
			fmt.Printf("%s✓ Stock Entry submitted%s\n", Green, Reset)
		}
	}

	return nil
}

// GetCompany gets the company name from config or API
func (c *Client) GetCompany() (string, error) {
	if c.Config.Company != "" {
		return c.Config.Company, nil
	}

	result, err := c.Request("GET", "Company?limit_page_length=1", nil)
	if err != nil {
		return "", err
	}

	if data, ok := result["data"].([]interface{}); ok && len(data) > 0 {
		if m, ok := data[0].(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok {
				return name, nil
			}
		}
	}

	return "", fmt.Errorf("no company found. Set ERP_COMPANY in config")
}

func (c *Client) submitStockEntry(name string) error {
	fullURL := fmt.Sprintf("%s/api/method/frappe.client.submit", c.ActiveURL)

	body := map[string]interface{}{
		"doc": map[string]interface{}{
			"doctype": "Stock Entry",
			"name":    name,
		},
	}

	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequest("POST", fullURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", c.Config.APIKey, c.Config.APISecret))
	req.Header.Set("Content-Type", "application/json")

	if c.Mode == "internet" && c.Config.NginxCookie != "" {
		req.AddCookie(&http.Cookie{Name: c.Config.NginxCookieName, Value: c.Config.NginxCookie})
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("failed to parse response: %s", string(respBody))
	}

	if exc, ok := result["exception"]; ok {
		return fmt.Errorf("submit failed: %v", exc)
	}

	return nil
}
