# ERPNext CLI

A fast, interactive command-line interface for managing ERPNext: items, attributes, templates, stock, purchasing, and more.

## Features

- **Interactive TUI** - Visual interface with keyboard navigation
- **Traditional CLI** - Scriptable commands for automation
- **Dual-mode connection** - Auto-detects VPN or Internet access
- **Complete item management** - Items, attributes, templates, variants
- **Stock operations** - Receive, transfer, issue with warehouse support
- **Serial numbers** - Individual product tracking
- **Purchasing workflow** - Suppliers, Purchase Orders, Purchase Invoices
- **Reports & Dashboard** - Executive summary and detailed reports
- **Batch operations** - CSV import/export

## Quick Start

### Build from source

```bash
git clone https://github.com/mikelcalvo/erpnext-cli.git
cd erpnext-cli
go build -o erp-cli ./cmd/erp-cli
```

### Configure

1. Copy the example config:
   ```bash
   cp .erp-config.example .erp-config
   ```

2. Generate API keys in ERPNext:
   - Go to **User Settings** > **API Access** > **Generate Keys**
   - Copy the API Key and API Secret

3. Edit `.erp-config` with your credentials

4. Test the connection:
   ```bash
   ./erp-cli ping
   ```

## Usage

### Interactive TUI

Launch without arguments to start the interactive interface:

```bash
./erp-cli
```

### CLI Commands

```bash
# Connection
erp-cli ping                    # Test connection
erp-cli config                  # Show configuration

# Attributes
erp-cli attr list               # List all attributes
erp-cli attr get "Brand"        # Get attribute details
erp-cli attr create-text "Model"
erp-cli attr create-numeric "Power (W)" 100 2000 50
erp-cli attr create-list "Color" "Red:R" "Blue:B"

# Items and Templates
erp-cli item list               # List all items
erp-cli item list --templates   # List only templates
erp-cli item get "ITEM-CODE"    # Get item details
erp-cli template create "CODE" "Name" "Group" "Attr1" "Attr2"

# Variants
erp-cli variant list "TEMPLATE"
erp-cli variant create "TEMPLATE" "VARIANT-CODE" "Attr1=Value1"

# Stock
erp-cli warehouse list
erp-cli stock get "ITEM" ["Warehouse"]
erp-cli stock receive "ITEM" 10 "Warehouse" --rate=100
erp-cli stock transfer "ITEM" 5 "From" "To"
erp-cli stock issue "ITEM" 2 "Warehouse"

# Serial Numbers
erp-cli serial create "SN-001" "ITEM"
erp-cli serial list "ITEM"
erp-cli serial create-batch "ITEM" "SN-" 1 100

# Suppliers
erp-cli supplier list
erp-cli supplier get "Intel Corporation"
erp-cli supplier create "New Supplier" --group="Services"
erp-cli supplier delete "Old Supplier"

# Purchase Orders
erp-cli po list
erp-cli po list --supplier="Intel" --status=Draft
erp-cli po get PUR-ORD-2025-00001
erp-cli po create "Intel Corporation"
erp-cli po add-item PUR-ORD-2025-00001 CPU-I7 10 --rate=450
erp-cli po submit PUR-ORD-2025-00001
erp-cli po cancel PUR-ORD-2025-00001

# Purchase Invoices
erp-cli pi list
erp-cli pi list --supplier="Intel"
erp-cli pi get ACC-PINV-2025-00001
erp-cli pi create-from-po PUR-ORD-2025-00001
erp-cli pi submit ACC-PINV-2025-00001
erp-cli pi cancel ACC-PINV-2025-00001

# Reports & Dashboard
erp-cli report                  # Executive dashboard
erp-cli report stock            # Detailed stock report
erp-cli report purchases        # Detailed purchasing report

# Import/Export
erp-cli export templates -o templates.csv
erp-cli export variants "TEMPLATE" -o variants.csv
erp-cli import variants -f variants.csv --dry-run
erp-cli import variants -f variants.csv
```

## Configuration

The CLI reads configuration from `.erp-config` file. It searches in:
1. Current directory
2. Parent directory
3. Executable directory

### Configuration Options

```bash
# Connection URLs
ERP_VPN="http://YOUR_VPN_IP:8000"      # Optional: Direct VPN access
ERP_URL="https://your-erp.example.com" # Required: Internet URL

# API Authentication (required)
ERP_API_KEY="your_api_key"
ERP_API_SECRET="your_api_secret"

# Reverse Proxy (if applicable)
NGINX_COOKIE=""                        # Auth cookie value
NGINX_COOKIE_NAME="auth_cookie"        # Cookie name

# Instance Configuration
ERP_COMPANY=""                         # Company name (auto-detected if empty)
ERP_BRAND="ERPNext CLI"                # CLI branding
```

## TUI Controls

| Key | Action |
|-----|--------|
| `↑/↓` | Navigate |
| `Enter` | Select / View details |
| `/` | Search |
| `d` | Delete selected |
| `r` | Refresh |
| `Esc` | Back |
| `q` | Quit |

## Requirements

- Go 1.21+ (for building)
- ERPNext instance with API access enabled
- API Key and Secret

## License

MIT License - see [LICENSE](LICENSE) file.
