package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/browserprofile"
)

func main() {
	configPath := flag.String("config", "config.yaml", "engine YAML configuration path")
	profileName := flag.String("profile", "", "browser profile name")
	mode := flag.String("mode", "", "browser mode: detect|no_detect|operator|research|auto")
	targetURL := flag.String("url", "about:blank", "URL to open")
	resolveOnly := flag.Bool("resolve-only", false, "resolve and print profile without launching")
	timeout := flag.Duration("timeout", 45*time.Second, "launch and navigation timeout")
	flag.Parse()

	registry, err := browserprofile.LoadFile(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	resolved, selection, err := registry.Select(*targetURL, *profileName, browserprofile.Mode(*mode))
	if err != nil {
		log.Fatal(err)
	}

	if *resolveOnly {
		writeJSON(map[string]any{"selection": selection, "profile": resolved})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	runtime, err := browserprofile.Launch(ctx, resolved)
	if err != nil {
		log.Fatal(err)
	}
	defer runtime.Close()

	page, err := runtime.OpenPage(ctx, *targetURL)
	if err != nil {
		log.Fatal(err)
	}
	defer page.Close()

	title := ""
	if result, evalErr := page.Eval(`() => document.title`); evalErr == nil {
		title = result.Value.Str()
	}
	writeJSON(map[string]any{
		"selection": selection,
		"profile": map[string]any{
			"id": resolved.ID,
			"mode": resolved.Mode,
			"engine": resolved.Engine,
			"digest": resolved.Digest,
		},
		"runtime": map[string]any{
			"pid": runtime.PID,
			"url": *targetURL,
			"title": title,
		},
	})
}

func writeJSON(value any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
