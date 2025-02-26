/*
Default is a helper tool to print the default values to stdout.
Currently, it can print

- Daemon Configuration File
- Command Lists
- Command Templates
*/
package main

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/telekom-mms/oc-daemon/internal/cmdtmpl"
	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// command line arguments.
const (
	DaemonConfig     = "daemon-config"
	CommandLists     = "command-lists"
	CommandTemplates = "command-templates"
)

// printUsage prints usage.
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n"+
		"\tdefault %s\n"+
		"\tdefault %s\n"+
		"\tdefault %s\n",
		DaemonConfig, CommandLists, CommandTemplates,
	)
}

func main() {
	// make sure command line argument is present
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "command line argument required\n")
		printUsage()
		return
	}

	switch os.Args[1] {

	case DaemonConfig:
		// get default config
		c := daemoncfg.NewConfig()

		// convert to json
		b, err := json.MarshalIndent(c, "", "    ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return
		}

		// print to stdout
		fmt.Fprintf(os.Stdout, "%s\n", b)

	case CommandLists:
		// sort function
		sf := func(a, b *cmdtmpl.CommandList) int {
			return cmp.Compare(a.Name, b.Name)
		}

		// get command lists as sorted slice
		cl := slices.SortedFunc(maps.Values(cmdtmpl.CommandLists), sf)

		// convert to json
		b, err := json.MarshalIndent(cl, "", "    ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s\n", err)
			return
		}

		// print to stdout
		fmt.Fprintf(os.Stdout, "%s\n", b)

	case CommandTemplates:
		// print to stdout
		fmt.Fprintf(os.Stdout, "%s\n", cmdtmpl.DefaultTemplate)

	default:
		// unknown, print error message to stderr
		fmt.Fprintf(os.Stderr, "%s unknown\n", os.Args[1])
		printUsage()
	}
}
