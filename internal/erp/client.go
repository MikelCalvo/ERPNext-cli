package erp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// configPaths returns the list of paths to search for config file
func configPaths() []string {
	return []string{
		".erp-config",
		"../.erp-config",
		filepath.Join(filepath.Dir(os.Args[0]), ".erp-config"),
		filepath.Join(filepath.Dir(os.Args[0]), "..", ".erp-config"),
	}
}

// ConfigExists checks if the config file exists in any of the search paths
func ConfigExists() bool {
	for _, path := range configPaths() {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// decodeJSON decodes JSON from a reader into a map
func decodeJSON(r io.Reader, v interface{}) error {
	return json.NewDecoder(r).Decode(v)
}

// Colors for terminal output
const (
	Red    = "\033[0;31m"
	Green  = "\033[0;32m"
	Yellow = "\033[1;33m"
	Blue   = "\033[0;34m"
	Cyan   = "\033[0;36m"
	Reset  = "\033[0m"
)

// Config holds the CLI configuration
type Config struct {
	ERPVPN          string
	ERPURL          string
	APIKey          string
	APISecret       string
	NginxCookie     string
	NginxCookieName string // Cookie name for reverse proxy auth (default: "auth_cookie")
	Company         string // Company name for stock operations (auto-detected if empty)
	Brand           string // CLI branding shown in TUI (default: "ERPNext CLI")
}

// CurrencyInfo holds currency details
type CurrencyInfo struct {
	Code   string // e.g., "EUR", "USD"
	Symbol string // e.g., "€", "$"
}

// Client handles API requests
type Client struct {
	Config     *Config
	HTTPClient *http.Client
	ActiveURL  string
	Mode       string // "vpn" or "internet"
	Currency   *CurrencyInfo
}

// LoadConfig reads the .erp-config file
func LoadConfig() (*Config, error) {
	// Find config file in various locations
	var configPath string
	for _, p := range configPaths() {
		if _, err := os.Stat(p); err == nil {
			configPath = p
			break
		}
	}

	if configPath == "" {
		return nil, fmt.Errorf("config file not found. Copy .erp-config.example to .erp-config")
	}

	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("cannot open config: %w", err)
	}
	defer file.Close()

	config := &Config{
		NginxCookieName: "auth_cookie",
		Brand:           "ERPNext CLI",
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

		switch key {
		case "ERP_VPN":
			config.ERPVPN = value
		case "ERP_URL":
			config.ERPURL = value
		case "ERP_API_KEY":
			config.APIKey = value
		case "ERP_API_SECRET":
			config.APISecret = value
		case "NGINX_COOKIE":
			config.NginxCookie = value
		case "NGINX_COOKIE_NAME":
			if value != "" {
				config.NginxCookieName = value
			}
		case "ERP_COMPANY":
			config.Company = value
		case "ERP_BRAND":
			if value != "" {
				config.Brand = value
			}
		}
	}

	if config.ERPURL == "" || config.APIKey == "" || config.APISecret == "" {
		return nil, fmt.Errorf("missing required config: ERP_URL, ERP_API_KEY, ERP_API_SECRET")
	}

	return config, nil
}

// NewClient creates a new API client
func NewClient(config *Config) *Client {
	return &Client{
		Config: config,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// DetectConnection tries VPN first, falls back to internet
func (c *Client) DetectConnection() {
	if c.Config.ERPVPN != "" {
		req, _ := http.NewRequest("GET", c.Config.ERPVPN+"/api/method/frappe.auth.get_logged_user", nil)
		req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", c.Config.APIKey, c.Config.APISecret))

		client := &http.Client{Timeout: 2 * time.Second}
		resp, err := client.Do(req)
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			c.Mode = "vpn"
			c.ActiveURL = c.Config.ERPVPN
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	c.Mode = "internet"
	c.ActiveURL = c.Config.ERPURL
}

// Request makes an API request
func (c *Client) Request(method, endpoint string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	fullURL := fmt.Sprintf("%s/api/resource/%s", c.ActiveURL, endpoint)
	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", c.Config.APIKey, c.Config.APISecret))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	if c.Mode == "internet" && c.Config.NginxCookie != "" {
		req.AddCookie(&http.Cookie{Name: c.Config.NginxCookieName, Value: c.Config.NginxCookie})
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %s", string(respBody))
	}

	if exc, ok := result["exception"]; ok {
		return nil, fmt.Errorf("API error: %v", exc)
	}

	return result, nil
}

// CmdPing tests the connection
func (c *Client) CmdPing() error {
	fmt.Printf("%sTesting connection to ERP...%s\n", Blue, Reset)

	c.DetectConnection()

	fullURL := fmt.Sprintf("%s/api/method/frappe.auth.get_logged_user", c.ActiveURL)
	req, _ := http.NewRequest("GET", fullURL, nil)
	req.Header.Set("Authorization", fmt.Sprintf("token %s:%s", c.Config.APIKey, c.Config.APISecret))

	if c.Mode == "internet" && c.Config.NginxCookie != "" {
		req.AddCookie(&http.Cookie{Name: c.Config.NginxCookieName, Value: c.Config.NginxCookie})
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	if msg, ok := result["message"].(string); ok && msg != "" {
		fmt.Printf("%s✓ Connection successful%s\n", Green, Reset)
		fmt.Printf("  Authenticated as: %s%s%s\n", Yellow, msg, Reset)
		if c.Mode == "vpn" {
			fmt.Printf("  Mode: %sVPN direct%s (%s)\n", Cyan, Reset, c.ActiveURL)
		} else {
			fmt.Printf("  Mode: %sInternet%s (%s)\n", Yellow, Reset, c.ActiveURL)
		}
		return nil
	}

	return fmt.Errorf("authentication failed: %s", string(body))
}

// Common currency symbols map
var currencySymbols = map[string]string{
	"USD": "$",
	"EUR": "€",
	"GBP": "£",
	"JPY": "¥",
	"CNY": "¥",
	"INR": "₹",
	"AUD": "A$",
	"CAD": "C$",
	"CHF": "CHF",
	"MXN": "$",
	"BRL": "R$",
	"KRW": "₩",
	"RUB": "₽",
	"TRY": "₺",
	"ZAR": "R",
	"SEK": "kr",
	"NOK": "kr",
	"DKK": "kr",
	"PLN": "zł",
	"THB": "฿",
	"SGD": "S$",
	"HKD": "HK$",
	"NZD": "NZ$",
	"CLP": "$",
	"COP": "$",
	"ARS": "$",
	"PEN": "S/",
}

// GetCurrency gets the default currency from the company
func (c *Client) GetCurrency() (*CurrencyInfo, error) {
	// Return cached currency if available
	if c.Currency != nil {
		return c.Currency, nil
	}

	// Get company name first
	company, err := c.GetCompany()
	if err != nil {
		// Fallback to USD if we can't determine the company
		c.Currency = &CurrencyInfo{Code: "USD", Symbol: "$"}
		return c.Currency, nil
	}

	// Fetch company details to get default_currency
	result, err := c.Request("GET", "Company/"+company+"?fields=[\"default_currency\"]", nil)
	if err != nil {
		c.Currency = &CurrencyInfo{Code: "USD", Symbol: "$"}
		return c.Currency, nil
	}

	currencyCode := "USD"
	if data, ok := result["data"].(map[string]interface{}); ok {
		if currency, ok := data["default_currency"].(string); ok && currency != "" {
			currencyCode = currency
		}
	}

	// Get symbol from map or use code as fallback
	symbol := currencyCode
	if s, ok := currencySymbols[currencyCode]; ok {
		symbol = s
	}

	c.Currency = &CurrencyInfo{
		Code:   currencyCode,
		Symbol: symbol,
	}

	return c.Currency, nil
}

// FormatCurrency formats an amount with the currency symbol
func (c *Client) FormatCurrency(amount float64) string {
	currency, _ := c.GetCurrency()
	if currency == nil {
		return fmt.Sprintf("$%.2f", amount)
	}
	return fmt.Sprintf("%s%.2f", currency.Symbol, amount)
}

// CmdConfig shows current configuration
func (c *Client) CmdConfig() error {
	fmt.Printf("%sCurrent configuration:%s\n", Blue, Reset)
	if c.Config.ERPVPN != "" {
		fmt.Printf("  VPN URL: %s\n", c.Config.ERPVPN)
	} else {
		fmt.Printf("  VPN URL: %snot configured%s\n", Yellow, Reset)
	}
	fmt.Printf("  Internet URL: %s\n", c.Config.ERPURL)
	fmt.Printf("  API Key: %s...\n", c.Config.APIKey[:8])
	fmt.Printf("  API Secret: ****\n")

	if c.Config.NginxCookie != "" {
		fmt.Printf("  Nginx Cookie: configured\n")
	} else {
		fmt.Printf("  Nginx Cookie: %snot configured%s (needed for internet mode)\n", Yellow, Reset)
	}

	if c.Config.Company != "" {
		fmt.Printf("  Company: %s\n", c.Config.Company)
	}

	fmt.Println()
	c.DetectConnection()
	if c.Mode == "vpn" {
		fmt.Printf("  Active mode: %sVPN direct%s\n", Cyan, Reset)
	} else {
		fmt.Printf("  Active mode: %sInternet%s\n", Yellow, Reset)
	}
	fmt.Printf("  Active URL: %s\n", c.ActiveURL)

	return nil
}
