package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/jeonghanlee/agent-relay/internal/version"
)

func main() {
	versionFlag := flag.Bool("version", false, "print version information and exit")
	vFlag := flag.Bool("v", false, "print version information and exit")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: agent-relay [flags] <command> [args]\n\n")
		fmt.Fprintf(os.Stderr, "Commands (Available in M5):\n")
		fmt.Fprintf(os.Stderr, "  daemon      Run the relay daemon\n")
		fmt.Fprintf(os.Stderr, "  ctl         Execute control commands\n")
		fmt.Fprintf(os.Stderr, "  completion  Generate shell completion scripts\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *versionFlag || *vFlag {
		fmt.Printf("agent-relay %s (commit: %s, built: %s)\n",
			version.Version, version.GitCommit, version.BuildDate)
		os.Exit(0)
	}

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	subcmd := flag.Arg(0)
	switch subcmd {
	case "daemon", "ctl", "completion":
		fmt.Fprintf(os.Stderr, "agent-relay: command %q will be wired in milestone M5\n", subcmd)
		os.Exit(1)
	default:
		fmt.Fprintf(os.Stderr, "agent-relay: unknown command %q\n", subcmd)
		flag.Usage()
		os.Exit(1)
	}
}
