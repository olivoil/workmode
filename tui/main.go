package main

import (
	"fmt"
	"os"

	"github.com/olivoil/workmode/tui/internal/app"
)

func main() {
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v", "version":
			fmt.Printf("%s tui %s\n", app.AppName, app.AppVersion)
			return
		case "--help", "-h", "help":
			fmt.Printf("%s tui %s\n\n", app.AppName, app.AppVersion)
			fmt.Println("Interactive terminal UI for workmode.")
			fmt.Println("\nUsage: workmode-tui")
			return
		}
	}

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
