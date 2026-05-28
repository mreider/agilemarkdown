// capture-mcp is a small CLI that reads a tutorial plan and writes
// JSON-RPC traces to disk. One file per moment.
//
// Plan format (YAML):
//
//   - name: 01-priority-list
//     tool: priority_list
//     args: {backlog: stories}
//   - name: 02-set-status-started
//     tool: set_status
//     args: {path: stories/wait-by-the-mailbox.md, status: started}
//
// For each entry it POSTs to ${AM_LAB_URL}/api/mcp/<tool> with the
// args body, and writes the {request, response} JSON-RPC trace to
// <out>/<name>.json.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type plan struct {
	Name string                 `yaml:"name"`
	Tool string                 `yaml:"tool"`
	Args map[string]interface{} `yaml:"args"`
}

func main() {
	planPath := flag.String("plan", "", "path to the chapter's mcp.yaml plan")
	outDir := flag.String("out", "", "directory to write the .json traces")
	flag.Parse()

	if *planPath == "" || *outDir == "" {
		fmt.Fprintln(os.Stderr, "usage: capture-mcp -plan <file> -out <dir>")
		os.Exit(2)
	}

	url := os.Getenv("AM_LAB_URL")
	if url == "" {
		url = "http://localhost:8090"
	}

	raw, err := os.ReadFile(*planPath)
	if err != nil {
		die(err)
	}
	var entries []plan
	if err := yaml.Unmarshal(raw, &entries); err != nil {
		die(fmt.Errorf("parse %s: %w", *planPath, err))
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		die(err)
	}

	for _, e := range entries {
		body, err := json.Marshal(e.Args)
		if err != nil {
			die(err)
		}
		endpoint := fmt.Sprintf("%s/api/mcp/%s", url, e.Tool)
		resp, err := http.Post(endpoint, "application/json", bytes.NewReader(body))
		if err != nil {
			die(fmt.Errorf("POST %s: %w", endpoint, err))
		}
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			die(err)
		}
		if resp.StatusCode/100 != 2 {
			die(fmt.Errorf("%s returned %d: %s", endpoint, resp.StatusCode, respBody))
		}
		// Pretty-print: the laboratory returns {request, response}
		// where each value is itself a JSON envelope. Re-marshal so
		// the file is human-readable in a diff.
		var trace map[string]json.RawMessage
		if err := json.Unmarshal(respBody, &trace); err != nil {
			die(err)
		}
		pretty := map[string]interface{}{}
		for k, v := range trace {
			var inner interface{}
			if err := json.Unmarshal(v, &inner); err == nil {
				pretty[k] = inner
			} else {
				pretty[k] = string(v)
			}
		}
		out, _ := json.MarshalIndent(pretty, "", "  ")
		path := filepath.Join(*outDir, e.Name+".json")
		if err := os.WriteFile(path, out, 0o644); err != nil {
			die(err)
		}
		fmt.Printf("  wrote %s\n", path)
	}
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
