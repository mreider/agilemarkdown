// Package main is the agilemarkdown laboratory server.
//
// `am serve --port 8080 --root /path/to/project` mounts:
//
//   - the existing React webview as a standalone web app at /
//   - a JSON HTTP surface mirroring every mcpserver.* wrapper at /api/*
//   - a WebSocket snapshot channel at /ws/snapshot, debounced by the
//     file watcher
//   - a CommandLog channel at /ws/log (every write emits an event)
//   - an MCP bridge at /ws/mcp (single am mcp child)
//   - an xterm.js terminal channel at /ws/pty
//
// Phase 1 implements: /api/snapshot, /api/set-status, /ws/snapshot, and
// the embedded static bundle. The rest layers on in later phases.
//
// The laboratory is single-user by design, no auth, one shared mcp
// child, fixture wiped + reseeded on every start unless --no-reset is
// passed. See LABORATORY.md for the broader architecture.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	port := flag.Int("port", 8080, "HTTP port to listen on")
	root := flag.String("root", "", "project root (workspace). Required.")
	noReset := flag.Bool("no-reset", false, "skip fixture wipe + reseed on start")
	fixtureScript := flag.String("fixture", "", "path to scripts/build-demo.sh; auto-detected if empty")
	flag.Parse()

	if *root == "" {
		fmt.Fprintln(os.Stderr, "usage: am serve --root /path/to/project [--port 8080] [--no-reset]")
		os.Exit(2)
	}
	abs, err := filepath.Abs(*root)
	if err != nil {
		log.Fatalf("resolve root: %v", err)
	}
	*root = abs

	if !*noReset {
		if err := resetFixture(*root, *fixtureScript); err != nil {
			log.Fatalf("reset fixture: %v", err)
		}
	}

	srv := newServer(*root)
	defer srv.Close()

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("am-server listening on %s, root=%s", addr, *root)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

// resetFixture wipes the workspace and re-runs scripts/build-demo.sh
// into it. Determinism by design, the lab boots into the same state
// every visit. Run with --no-reset to skip.
func resetFixture(root, scriptPath string) error {
	if scriptPath == "" {
		// Walk up from the binary's working directory to find the
		// repo's scripts/build-demo.sh. The lab is dev-time tooling;
		// when deployed to a droplet the operator points to it
		// explicitly with --fixture.
		cwd, _ := os.Getwd()
		try := filepath.Join(cwd, "scripts", "build-demo.sh")
		if _, err := os.Stat(try); err == nil {
			scriptPath = try
		}
	}
	if scriptPath == "" {
		return fmt.Errorf("fixture script not found; pass --fixture path/to/build-demo.sh")
	}
	log.Printf("resetting fixture at %s via %s", root, scriptPath)
	cmd := exec.CommandContext(context.Background(), "bash", scriptPath, root)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
