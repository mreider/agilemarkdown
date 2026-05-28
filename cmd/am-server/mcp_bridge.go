package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// mcpBridge wraps a single `am mcp` child process. Every laboratory
// visitor shares the same child (decision #5 in LABORATORY.md), the
// bridge multiplexes per-request via the JSON-RPC `id` so concurrent
// callers don't crosstalk.
//
// The bridge speaks JSON-RPC over the child's stdio:
//
//   client → stdin:  {"jsonrpc":"2.0","id":N,"method":"tools/call","params":{...}}
//   stdout → client: {"jsonrpc":"2.0","id":N,"result":{...}}
//
// Initialization (the JSON-RPC initialize handshake the MCP spec
// requires) happens once when the bridge starts.
type mcpBridge struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  *bufio.Reader
	writeMu sync.Mutex

	nextID  atomic.Int64
	waitMu  sync.Mutex
	waiters map[int64]chan jsonrpcResponse
	done    chan struct{}
}

type jsonrpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      int64          `json:"id"`
	Method  string         `json:"method"`
	Params  any            `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newMcpBridge(root string) (*mcpBridge, error) {
	cmd := exec.Command("am", "mcp")
	cmd.Dir = root
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	// Surface stderr; the child uses it for diagnostics.
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start am mcp: %w", err)
	}

	b := &mcpBridge{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  bufio.NewReader(stdout),
		waiters: make(map[int64]chan jsonrpcResponse),
		done:    make(chan struct{}),
	}
	go b.readLoop()

	// JSON-RPC initialize handshake. The MCP server's stdio transport
	// rejects tool calls until it sees `initialize`.
	if _, err := b.rawCall(context.Background(), "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "am-laboratory", "version": "0.1"},
	}); err != nil {
		b.Close()
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}
	// Some servers expect a `notifications/initialized` follow-up.
	_ = b.notify("notifications/initialized", nil)
	return b, nil
}

func (b *mcpBridge) Close() {
	close(b.done)
	_ = b.stdin.Close()
	if b.cmd != nil && b.cmd.Process != nil {
		_ = b.cmd.Process.Kill()
		_ = b.cmd.Wait()
	}
}

// readLoop pulls JSON-RPC responses one line at a time and routes each
// to the waiting caller by id. Notifications (no id) are dropped.
func (b *mcpBridge) readLoop() {
	for {
		line, err := b.stdout.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				// Surface the error to any pending waiter so callers
				// don't hang on a dead bridge.
				b.waitMu.Lock()
				for id, ch := range b.waiters {
					select {
					case ch <- jsonrpcResponse{ID: id, Error: &jsonrpcError{Code: -1, Message: err.Error()}}:
					default:
					}
				}
				b.waitMu.Unlock()
			}
			return
		}
		if len(line) == 0 {
			continue
		}
		var resp jsonrpcResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			continue
		}
		if resp.ID == 0 {
			continue
		}
		b.waitMu.Lock()
		ch, ok := b.waiters[resp.ID]
		if ok {
			delete(b.waiters, resp.ID)
		}
		b.waitMu.Unlock()
		if ok {
			ch <- resp
		}
	}
}

// rawCall sends one JSON-RPC request and waits for the matching
// response. Caller owns the request ID; we mint a fresh one here.
func (b *mcpBridge) rawCall(ctx context.Context, method string, params any) (jsonrpcResponse, error) {
	id := b.nextID.Add(1)
	req := jsonrpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	payload, err := json.Marshal(req)
	if err != nil {
		return jsonrpcResponse{}, err
	}
	payload = append(payload, '\n')

	ch := make(chan jsonrpcResponse, 1)
	b.waitMu.Lock()
	b.waiters[id] = ch
	b.waitMu.Unlock()
	defer func() {
		b.waitMu.Lock()
		delete(b.waiters, id)
		b.waitMu.Unlock()
	}()

	b.writeMu.Lock()
	_, err = b.stdin.Write(payload)
	b.writeMu.Unlock()
	if err != nil {
		return jsonrpcResponse{}, err
	}

	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		// Don't hang forever if the child stalls.
		deadline = time.Now().Add(20 * time.Second)
	}
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	select {
	case resp := <-ch:
		return resp, nil
	case <-timer.C:
		return jsonrpcResponse{}, fmt.Errorf("mcp call %s timeout", method)
	case <-ctx.Done():
		return jsonrpcResponse{}, ctx.Err()
	}
}

// notify sends a JSON-RPC notification (no response expected).
func (b *mcpBridge) notify(method string, params any) error {
	req := map[string]any{"jsonrpc": "2.0", "method": method}
	if params != nil {
		req["params"] = params
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	b.writeMu.Lock()
	_, err = b.stdin.Write(payload)
	b.writeMu.Unlock()
	return err
}

// CallTool packages a tools/call request, returns the raw JSON-RPC
// envelope (both directions) so the UI's wire view can render it.
type mcpCallTrace struct {
	Request  json.RawMessage `json:"request"`
	Response json.RawMessage `json:"response"`
}

func (b *mcpBridge) CallTool(ctx context.Context, name string, args any) (mcpCallTrace, error) {
	id := b.nextID.Add(1)
	req := jsonrpcRequest{
		JSONRPC: "2.0", ID: id, Method: "tools/call",
		Params: map[string]any{"name": name, "arguments": args},
	}
	reqBytes, _ := json.Marshal(req)

	// Manually drive the call so we can capture the raw JSON for the
	// trace; the rawCall path drops the literal bytes once unmarshaled.
	ch := make(chan jsonrpcResponse, 1)
	b.waitMu.Lock()
	b.waiters[id] = ch
	b.waitMu.Unlock()
	defer func() {
		b.waitMu.Lock()
		delete(b.waiters, id)
		b.waitMu.Unlock()
	}()

	b.writeMu.Lock()
	_, err := b.stdin.Write(append(reqBytes, '\n'))
	b.writeMu.Unlock()
	if err != nil {
		return mcpCallTrace{}, err
	}

	deadline := time.Now().Add(20 * time.Second)
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()

	select {
	case resp := <-ch:
		respBytes, _ := json.Marshal(resp)
		return mcpCallTrace{Request: reqBytes, Response: respBytes}, nil
	case <-timer.C:
		return mcpCallTrace{}, fmt.Errorf("mcp call %s timeout", name)
	case <-ctx.Done():
		return mcpCallTrace{}, ctx.Err()
	}
}

// ListTools returns the registered tools so the laboratory's MCP
// console can render its dropdown + form fields.
func (b *mcpBridge) ListTools(ctx context.Context) (json.RawMessage, error) {
	resp, err := b.rawCall(ctx, "tools/list", map[string]any{})
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("tools/list: %s", resp.Error.Message)
	}
	return resp.Result, nil
}
