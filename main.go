package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"strings"
	"syscall"

	"axctl/pkg/config"
	"axctl/pkg/ipc"
	"axctl/pkg/ipc/hyprland"
	"axctl/pkg/ipc/mango"
	"axctl/pkg/ipc/niri"
	"axctl/pkg/server"
)

var Version = "dev"

func main() {
	if len(os.Args) >= 2 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			printVersion()
			return
		}
	}

	if len(os.Args) < 2 {
		usage()
		return
	}

	customConfigPath := ""
	i := 1
	for i < len(os.Args) && os.Args[i] != "daemon" {
		if os.Args[i] == "-c" && i+1 < len(os.Args) {
			customConfigPath = os.Args[i+1]
			i += 2
		} else {
			break
		}
	}
	remainingArgs := os.Args[i:]
	if len(remainingArgs) == 0 {
		usage()
		return
	}
	switch remainingArgs[0] {
	case "daemon":
		runDaemon(customConfigPath)
	case "subscribe":
		runSubscribe()
	case "window", "workspace", "monitor", "layout", "config", "system", "darkmode":
		if len(remainingArgs) < 2 {
			usage()
			return
		}
		handleRPC(remainingArgs[0], remainingArgs[1:])
	default:
		usage()
	}
}

func printVersion() {
	version := Version
	if version == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok {
			if bi.Main.Version != "" && bi.Main.Version != "(devel)" {
				version = bi.Main.Version
			}
		}
	}

	fmt.Printf("axctl %s\n", version)
}
