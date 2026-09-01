// UIAI Engine — unified AI + Vision binary for WPUIAI.
//
// Replaces: Bun AI Server (3007), PHP Screenshot API (3006),
// Vision Daemon (3011), Browserless (3005).
//
// One binary. Every capability. Zero runtime dependencies.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/server"
)

var (
	version   = "2.0.3"
	buildTime = "unknown"
)

func main() {
	configPath := flag.String("config", "", "Path to config.yaml")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("uiai-engine %s (built %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Find config file
	cfgPath := *configPath
	if cfgPath == "" {
		candidates := []string{
			"config.yaml",
			"/etc/uiai/engine.yaml",
			"/home/wpuiai/uiai-engine/config.yaml",
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				cfgPath = c
				break
			}
		}
	}
	if cfgPath == "" {
		log.Fatal("No config file found. Use -config=path/to/config.yaml")
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("uiai-engine %s starting (config: %s)", version, cfgPath)

	// Ensure data directory exists
	if err := os.MkdirAll(cfg.Storage.DataDir, 0750); err != nil {
		log.Fatalf("Failed to create data dir %s: %v", cfg.Storage.DataDir, err)
	}

	eng := server.NewWithBrowserProfiles(cfg, cfgPath)
	if err := eng.Run(); err != nil {
		log.Fatalf("Engine stopped: %v", err)
	}
}
