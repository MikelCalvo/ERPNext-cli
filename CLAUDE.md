# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Important Guidelines

### Repository Context
- This is a **public repository** - all code and commits are visible to everyone
- Maintain a **clean git history** - commits should be meaningful and well-formatted
- **Never push without asking** - always ask the user before running `git push`

### Git Workflow
- Version release commits use format: `vX.Y.Z: Title`
- Regular commits use standard format: `Description of change`
- Decide when to bump versions based on feature significance:
  - Patch (1.0.X): Bug fixes, minor tweaks
  - Minor (1.X.0): New features, new commands
  - Major (X.0.0): Breaking changes

### Self-Improvement
- **Update this CLAUDE.md** as you learn new things about the codebase
- Document patterns, gotchas, and architectural decisions
- Keep the file concise but comprehensive

### Code Quality
- Avoid generating "slop" - no generic, low-quality, or filler code
- Be precise and intentional with every change
- Follow existing patterns in the codebase

### Feature Parity
- When adding new CLI commands, also add them to the TUI
- Keep TUI and CLI functionality synchronized

## Build Commands

```bash
# Build the CLI binary
go build -o erp-cli ./cmd/erp-cli

# Run without building
go run ./cmd/erp-cli [command]
```

## Architecture Overview

This is a Go CLI application for managing ERPNext instances, featuring both command-line and interactive TUI modes.

### Core Structure

- **Entry point**: `cmd/erp-cli/main.go` - Command routing and dispatcher
- **Core logic**: `internal/erp/` - All business logic and API interactions

### Key Files in `internal/erp/`

| File | Purpose |
|------|---------|
| `client.go` | Config loading, HTTP client, connection detection, currency |
| `attr.go` | Item attribute CRUD operations |
| `item.go` | Items, templates, groups, brands management |
| `variant.go` | Variant creation and listing |
| `stock.go` | Warehouse and stock operations (CLI) |
| `serial.go` | Serial number management (CLI) |
| `import.go` | CSV import/export functionality |
| `supplier.go` | Supplier management (CLI) |
| `purchase.go` | Purchase Orders and Purchase Invoices (CLI) |
| `customer.go` | Customer management (CLI) |
| `sales.go` | Quotations, Sales Orders, Sales Invoices (CLI) |
| `delivery.go` | Delivery Notes (CLI) |
| `receipt.go` | Purchase Receipts (CLI) |
| `payment.go` | Payment Entries (CLI) - receive/pay invoices |
| `report.go` | Dashboard and reports (CLI) |

### TUI Files in `internal/erp/`

| File | Purpose |
|------|---------|
| `tui.go` | Core TUI: Model, Views enum, menu, navigation, Update/View |
| `tui_dashboard.go` | Dashboard view with metrics display |
| `tui_stock.go` | Warehouses, Stock operations, Serial Numbers |
| `tui_purchasing.go` | Suppliers, Purchase Orders, Purchase Invoices, Purchase Receipts |
| `tui_sales.go` | Customers, Quotations, Sales Orders, Sales Invoices, Delivery Notes, Payments |
| `tui_inventory.go` | CRUD for Attributes, Groups, Brands, Warehouses, Variants |
| `tui_forms.go` | Reusable form components, confirmations, list footer, helpers |

### Command Pattern

All commands are methods on the `*Client` struct. The main dispatcher routes based on `os.Args[1]`:
- No args or `tui` → Launches interactive TUI
- Other commands → Load config, detect connection, execute handler

### Connection Detection

The client auto-detects connectivity mode:
1. Tries VPN URL first (2s timeout)
2. Falls back to internet URL
3. Sets `Client.Mode` ("vpn" or "internet") and `Client.ActiveURL`

### Currency System

The client auto-detects the company's default currency:
- `GetCurrency()` fetches from Company doctype's `default_currency` field
- Cached in `Client.Currency` to avoid repeated API calls
- `FormatCurrency(amount)` formats with the correct symbol
- Fallback to USD if detection fails
- Symbol map in `client.go` covers 27+ common currencies

**Important**: Use `c.FormatCurrency()` for ALL monetary values in output.

### API Integration

- Authentication: `Authorization: token API_KEY:API_SECRET`
- Endpoint pattern: `/api/resource/{DocType}`
- URL encoding: spaces become `%20` (e.g., `Purchase%20Order`)
- Filters use JSON array format, URL-encoded
- Optional nginx cookie support for reverse proxy setups

### TUI Implementation

Uses Charm's BubbleTea framework:
- `Model` struct holds all state (view, lists, inputs, data, spinner, breadcrumbs, notifications)
- Views: main menu, lists, details, forms, confirmations
- Async data loading via custom message types (`dataLoadedMsg`, `itemDetailMsg`, etc.)
- Navigation: Esc to go back, q to quit from main menu
- Forms: Tab to navigate fields, Enter to submit, Esc to cancel
- Key shortcuts: n=new, d=delete, r=refresh/receive, t=transfer, i=issue/invoice, s=submit, x=cancel, o=sort order (lists)/create SO (quotations), q=from quotation, p=create payment, v=create variant (templates)

**v1.7.0 TUI Features:**
- Animated spinner (dots) while loading data
- Breadcrumb navigation trail (Main > Section > Item)
- Auto-dismissing success notifications (3s)
- Colored status badges (Draft/Submitted/Paid/Unpaid/Cancelled)
- Dashboard: 5 sections (Stock, Sales, Purchases, Payments, System)

**v1.7.0 TUI Features (continued):**
- **Quick Actions**: 'i' in PO detail creates PI, 'v' in template detail creates variant
- **List sorting**: 'o' key cycles through Date↓, Date↑, Name, Total↓ (indicator in title)
- **List footer**: Shows total items, total amount, and status counts (draft/unpaid/pending)
- **CRUD for master data**: Create Attributes (text/numeric/select), Groups, Brands, Warehouses
- ListItem extended with `amount` and `status` fields for aggregations

**TUI Main Menu** (6 categories with submenus):
1. **Dashboard** - Executive summary with KPIs (direct view)
2. **Inventory** → Items, Templates, Groups, Brands, Attributes
3. **Stock** → Warehouses, Stock Levels, Serial Numbers
4. **Sales** → Customers, Quotations, Sales Orders, Sales Invoices, Delivery Notes
5. **Purchasing** → Suppliers, Purchase Orders, Purchase Invoices, Purchase Receipts
6. **Payments** → All Payments (receive/pay invoices)

### Reports Module

Dashboard fetches data in parallel using goroutines:
- `fetchStockMetrics()`, `fetchPurchaseMetrics()`, `fetchSystemMetrics()`, `fetchSalesMetrics()`, `fetchPaymentMetrics()`
- Uses `sync.WaitGroup` and `sync.Mutex` for coordination
- Pre-fetches currency before rendering

## Configuration

Config file: `.erp-config` (shell-style key=value)

Searched in order:
1. Current directory
2. Parent directory
3. Executable directory
4. Executable parent directory

Required fields: `ERP_URL`, `ERP_API_KEY`, `ERP_API_SECRET`

## Version Management

Version constant is in `internal/erp/tui.go`:
```go
const (
    Version = "1.7.0"
    Author  = "Mikel Calvo"
    Year    = "2025"
)
```
