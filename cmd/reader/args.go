package main

import (
	"path/filepath"
	"strings"
)

// command is the parsed CLI intent.
type command struct {
	Mode string // tui | open | import | stream | list | version
	Arg  string
}

// parseArgs interprets os.Args[1:].
func parseArgs(args []string) command {
	if len(args) == 0 {
		return command{Mode: "tui"}
	}
	switch args[0] {
	case "version", "--version", "-v":
		return command{Mode: "version"}
	case "list":
		return command{Mode: "list"}
	case "stream":
		if len(args) > 1 {
			return command{Mode: "stream", Arg: args[1]}
		}
		return command{Mode: "stream"}
	case "import":
		if len(args) > 1 {
			return command{Mode: "import", Arg: args[1]}
		}
		return command{Mode: "import"}
	}
	// Otherwise treat the first arg as an epub path to open.
	return command{Mode: "open", Arg: args[0]}
}

// resolveDataDir picks the data directory: env override wins, else a "data"
// folder next to the executable.
func resolveDataDir(exePath, envOverride string) string {
	if strings.TrimSpace(envOverride) != "" {
		return envOverride
	}
	return filepath.Join(filepath.Dir(exePath), "data")
}
