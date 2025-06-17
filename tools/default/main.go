/*
Default is a helper tool to print the default values to stdout.
Currently, it can print

- Daemon Configuration File
- Command Lists
- Command Templates
- Client Configuration File
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
	"github.com/telekom-mms/oc-daemon/pkg/client"
)

// command line arguments.
const (
	DaemonConfig     = "daemon-config"
	CommandLists     = "command-lists"
	CommandTemplates = "command-templates"
	ClientConfig     = "client-config"
)

// printUsage prints usage.
func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage:\n"+
		"\tdefault %s\n"+
		"\tdefault %s\n"+
		"\tdefault %s\n"+
		"\tdefault %s\n",
		DaemonConfig, CommandLists, CommandTemplates, ClientConfig,
	)
}

// printJSON prints a as JSON.
func printJSON(a any) {
	// convert to json
	b, err := json.MarshalIndent(a, "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return
	}

	// print to stdout
	_, _ = fmt.Fprintf(os.Stdout, "%s\n", b)
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
		// get and print default config
		c := daemoncfg.NewConfig()
		printJSON(c)

	case CommandLists:
		// sort function
		sf := func(a, b *cmdtmpl.CommandList) int {
			return cmp.Compare(a.Name, b.Name)
		}

		// get command lists as sorted slice
		cl := slices.SortedFunc(maps.Values(cmdtmpl.CommandLists), sf)

		// print command lists
		printJSON(cl)

	case CommandTemplates:
		// print to stdout
		_, _ = fmt.Fprintf(os.Stdout, "%s\n", cmdtmpl.DefaultTemplate)

	case ClientConfig:
		// get and print default config
		c := client.NewConfig()
		printJSON(c)

	default:
		// unknown, print error message to stderr
		fmt.Fprintf(os.Stderr, "%s unknown\n", os.Args[1])
		printUsage()
	}
}
