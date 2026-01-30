package main

import (
	"fmt"
	"os"

	"github.com/mikelcalvo/erpnext-cli/internal/erp"
)

func main() {
	// No arguments or "tui" command -> launch TUI
	if len(os.Args) < 2 || os.Args[1] == "tui" {
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

%sImport/Export:%s
  %sexport items -o <file>%s            Export items to CSV
  %sexport templates -o <file>%s        Export templates to CSV
  %sexport attributes -o <file>%s       Export attributes to CSV
  %sexport variants <tpl> -o <file>%s   Export variants to CSV
  %simport items -f <file> [--dry-run]%s Import items from CSV
  %simport variants -f <file> [--dry-run]%s Import variants from CSV

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
		erp.Yellow, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Green, erp.Reset, erp.Green, erp.Reset, erp.Green, erp.Reset,
		erp.Yellow, erp.Reset,
	)
}
