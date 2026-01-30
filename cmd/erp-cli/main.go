package main

import (
	"fmt"
	"os"

	"github.com/mikelcalvo/erpnext-cli/internal/erp"
)

func main() {
	// No arguments or "tui" command -> launch TUI
	if len(os.Args) < 2 || os.Args[1] == "tui" {
		// Check if config exists, if not launch setup wizard
		if !erp.ConfigExists() {
			if err := erp.RunSetupTUI(); err != nil {
				fmt.Printf("%sError: %s%s\n", erp.Red, err, erp.Reset)
				os.Exit(1)
			}
			// After setup, check if config was created
			if !erp.ConfigExists() {
				// User cancelled setup
				os.Exit(0)
			}
		}

		config, err := erp.LoadConfig()
		if err != nil {
			fmt.Printf("%sError: %s%s\n", erp.Red, err, erp.Reset)
			os.Exit(1)
		}
		client := erp.NewClient(config)
		if err := erp.RunTUI(client); err != nil {
			fmt.Printf("%sError: %s%s\n", erp.Red, err, erp.Reset)
			os.Exit(1)
		}
		os.Exit(0)
	}

	cmd := os.Args[1]

	// Help doesn't need config
	if cmd == "help" || cmd == "-h" || cmd == "--help" {
		printUsage()
		os.Exit(0)
	}

	// Version
	if cmd == "version" || cmd == "-v" || cmd == "--version" {
		fmt.Printf("ERPNext CLI v%s\n", erp.Version)
		fmt.Printf("Created by %s in %s\n", erp.Author, erp.Year)
		os.Exit(0)
	}

	// Load config
	config, err := erp.LoadConfig()
	if err != nil {
		fmt.Printf("%sError: %s%s\n", erp.Red, err, erp.Reset)
		os.Exit(1)
	}

	// Create client
	client := erp.NewClient(config)

	// Detect connection mode (except for ping/config which do it themselves)
	if cmd != "ping" && cmd != "config" {
		client.DetectConnection()
	}

	// Route commands
	var cmdErr error
	switch cmd {
	case "ping":
		cmdErr = client.CmdPing()
	case "config":
		cmdErr = client.CmdConfig()
	case "attr", "attribute":
		cmdErr = client.CmdAttr(os.Args[2:])
	case "item":
		cmdErr = client.CmdItem(os.Args[2:])
	case "template":
		cmdErr = client.CmdTemplate(os.Args[2:])
	case "group":
		cmdErr = client.CmdGroup(os.Args[2:])
	case "brand":
		cmdErr = client.CmdBrand(os.Args[2:])
	case "variant":
		cmdErr = client.CmdVariant(os.Args[2:])
	case "warehouse":
		cmdErr = client.CmdWarehouse(os.Args[2:])
	case "stock":
		cmdErr = client.CmdStock(os.Args[2:])
	case "serial":
		cmdErr = client.CmdSerial(os.Args[2:])
	case "supplier":
		cmdErr = client.CmdSupplier(os.Args[2:])
	case "po":
		cmdErr = client.CmdPO(os.Args[2:])
	case "pi":
		cmdErr = client.CmdPI(os.Args[2:])
	case "customer":
		cmdErr = client.CmdCustomer(os.Args[2:])
	case "quotation":
		cmdErr = client.CmdQuotation(os.Args[2:])
	case "so":
		cmdErr = client.CmdSO(os.Args[2:])
	case "si":
		cmdErr = client.CmdSI(os.Args[2:])
	case "dn":
		cmdErr = client.CmdDN(os.Args[2:])
	case "pr":
		cmdErr = client.CmdPR(os.Args[2:])
	case "payment":
		cmdErr = client.CmdPayment(os.Args[2:])
	case "report", "dashboard":
		cmdErr = client.CmdReport(os.Args[2:])
	case "export":
		cmdErr = client.CmdExport(os.Args[2:])
	case "import":
		cmdErr = client.CmdImport(os.Args[2:])
	default:
		fmt.Printf("%sUnknown command: %s%s\n", erp.Red, cmd, erp.Reset)
		printUsage()
		os.Exit(1)
	}

	if cmdErr != nil {
		fmt.Printf("%sError: %s%s\n", erp.Red, cmdErr, erp.Reset)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`%sERPNext CLI%s - Created by Mikel Calvo in 2025

Usage: erp-cli <command> [subcommand] [args...]

%sCommands:%s

  %sping%s                              Test connection and authentication
  %sconfig%s                            Show current configuration
  %sversion%s                           Show version information

%sAttributes:%s
  %sattr list%s                         List all item attributes
  %sattr get <name>%s                   Get attribute details
  %sattr create-text <name>%s           Create text attribute
  %sattr create-numeric <name> <from> <to> <increment>%s
                                      Create numeric attribute with range
  %sattr create-list <name> <val:abbr> [val:abbr...]%s
                                      Create attribute with predefined values
  %sattr add-values <name> <val:abbr> [val:abbr...]%s
                                      Add values to existing list attribute
  %sattr delete <name>%s                Delete an attribute

%sItems:%s
  %sitem list [--templates]%s           List items (optionally only templates)
  %sitem get <code>%s                   Get item details
  %sitem create <code> <name> <group>%s Create simple item
  %sitem add-attr <code> <attr1> [...]%s Add attributes to item/template
  %sitem set <code> <prop=val>%s        Update item properties
  %sitem delete <code>%s                Delete an item

%sTemplates:%s
  %stemplate create <code> <name> <group> <attr1> [...]%s
                                      Create item template with attributes

%sVariants:%s
  %svariant list <template>%s           List all variants of a template
  %svariant create <template> <code> <attr=val> [...]%s
                                      Create a variant from a template

%sGroups & Brands:%s
  %sgroup list%s                        List item groups
  %sgroup create <name> [parent]%s      Create item group
  %sbrand list%s                        List brands
  %sbrand create <name>%s               Create a new brand
  %sbrand add-to-attr <name>%s          Create brand AND add to attribute

%sStock:%s
  %swarehouse list%s                    List all warehouses
  %sstock get <item> [warehouse]%s      Get current stock
  %sstock receive <item> <qty> <wh> [--rate=X]%s
                                      Receive stock (Material Receipt)
  %sstock transfer <item> <qty> <from> <to>%s
                                      Transfer stock between warehouses
  %sstock issue <item> <qty> <wh>%s     Issue stock (Material Issue)

%sSerial Numbers:%s
  %sserial create <sn> <item>%s         Create a serial number
  %sserial list <item>%s                List serial numbers for an item
  %sserial get <sn>%s                   Get serial number details
  %sserial create-batch <item> <prefix> <start> <count>%s
                                      Create multiple serial numbers

%sSuppliers:%s
  %ssupplier list%s                     List all suppliers
  %ssupplier get <name>%s               Get supplier details
  %ssupplier create <name>%s            Create a new supplier
  %ssupplier delete <name>%s            Delete a supplier

%sPurchase Orders:%s
  %spo list [--supplier=X] [--status=X]%s
                                      List purchase orders
  %spo get <name>%s                     Get PO details with items
  %spo create <supplier>%s              Create draft PO
  %spo add-item <po> <item> <qty> [--rate=X]%s
                                      Add item to PO
  %spo submit <name>%s                  Submit PO
  %spo cancel <name>%s                  Cancel PO

%sPurchase Invoices:%s
  %spi list [--supplier=X] [--status=X]%s
                                      List purchase invoices
  %spi get <name>%s                     Get invoice details
  %spi create-from-po <po_name>%s       Create invoice from PO
  %spi submit <name>%s                  Submit invoice
  %spi cancel <name>%s                  Cancel invoice

%sCustomers:%s
  %scustomer list%s                     List all customers
  %scustomer get <name>%s               Get customer details
  %scustomer create <name>%s            Create a new customer
  %scustomer delete <name>%s            Delete a customer

%sQuotations:%s
  %squotation list [--customer=X] [--status=X]%s
                                      List quotations
  %squotation get <name>%s              Get quotation details
  %squotation create <customer>%s       Create draft quotation
  %squotation add-item <name> <item> <qty> [--rate=X]%s
                                      Add item to quotation
  %squotation submit <name>%s           Submit quotation
  %squotation cancel <name>%s           Cancel quotation

%sSales Orders:%s
  %sso list [--customer=X] [--status=X]%s
                                      List sales orders
  %sso get <name>%s                     Get SO details with items
  %sso create <customer>%s              Create draft SO
  %sso create-from-quotation <name>%s   Create SO from quotation
  %sso add-item <so> <item> <qty> [--rate=X]%s
                                      Add item to SO
  %sso submit <name>%s                  Submit SO
  %sso cancel <name>%s                  Cancel SO

%sSales Invoices:%s
  %ssi list [--customer=X] [--status=X]%s
                                      List sales invoices
  %ssi get <name>%s                     Get invoice details
  %ssi create-from-so <so_name>%s       Create invoice from SO
  %ssi submit <name>%s                  Submit invoice
  %ssi cancel <name>%s                  Cancel invoice

%sDelivery Notes:%s
  %sdn list [--customer=X] [--status=X]%s
                                      List delivery notes
  %sdn get <name>%s                     Get delivery note details
  %sdn create-from-so <so_name>%s       Create delivery note from SO
  %sdn submit <name>%s                  Submit delivery note
  %sdn cancel <name>%s                  Cancel delivery note

%sPurchase Receipts:%s
  %spr list [--supplier=X] [--status=X]%s
                                      List purchase receipts
  %spr get <name>%s                     Get receipt details
  %spr create-from-po <po_name>%s       Create receipt from PO
  %spr submit <name>%s                  Submit receipt
  %spr cancel <name>%s                  Cancel receipt

%sPayments:%s
  %spayment list [--party=X] [--type=receive|pay] [--status=X]%s
                                      List payment entries
  %spayment get <name>%s                Get payment details
  %spayment receive <si_name> [--amount=X]%s
                                      Create payment from Sales Invoice
  %spayment pay <pi_name> [--amount=X]%s
                                      Create payment for Purchase Invoice
  %spayment submit <name>%s             Submit payment
  %spayment cancel <name>%s             Cancel payment

%sImport/Export:%s
  %sexport items -o <file>%s            Export items to CSV
  %sexport templates -o <file>%s        Export templates to CSV
  %sexport attributes -o <file>%s       Export attributes to CSV
  %sexport variants <tpl> -o <file>%s   Export variants to CSV
  %simport items -f <file> [--dry-run]%s Import items from CSV
  %simport variants -f <file> [--dry-run]%s Import variants from CSV

%sReports:%s
  %sreport%s                            Executive dashboard
  %sreport stock%s                      Detailed stock report
  %sreport purchases%s                  Detailed purchasing report

%sExamples:%s
  erp-cli ping
  erp-cli attr create-text "CPU Model"
  erp-cli template create "PSU-ATX" "ATX PSU" "Power" "Brand" "Wattage"
  erp-cli variant create "PSU-ATX" "PSU-EVGA-500" "Brand=EVGA" "Wattage=500"
  erp-cli stock receive "CPU-I7" 10 "Stores" --rate=450

`,
		erp.Blue, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Customers
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Quotations
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Sales Orders
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Sales Invoices
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Delivery Notes
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Purchase Receipts
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Payments
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Import/Export
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Reports
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		// Examples
		erp.Yellow, erp.Reset,
	)
}
