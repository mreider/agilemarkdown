package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"

	"github.com/coder/websocket"
	"github.com/creack/pty"
)

// handleWsPty hooks an xterm.js client up to a real bash PTY rooted
// at the project. Bytes flow both ways: client input → PTY stdin,
// PTY stdout → client. The laboratory is single-user (decision #1)
// so we don't sandbox the shell, the visitor has the same access
// `am serve` itself has.
func (s *server) handleWsPty(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("ws/pty accept: %v", err)
		return
	}
	defer c.CloseNow()

	cmd := exec.Command("bash", "-il")
	cmd.Dir = s.root
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"AM_LAB=1",
		"PS1=lab $ ",
	)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		log.Printf("pty start: %v", err)
		return
	}
	defer func() {
		_ = ptmx.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_, _ = cmd.Process.Wait()
	}()

	// Sensible default terminal size. xterm.js sends a resize message
	// (see below) once it lays out, so this is just for the first
	// few hundred milliseconds.
	_ = pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 100})

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// PTY → WebSocket.
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if err := c.Write(ctx, websocket.MessageBinary, buf[:n]); err != nil {
					cancel()
					return
				}
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("pty read: %v", err)
				}
				cancel()
				return
			}
		}
	}()

	// WebSocket → PTY. The client sends raw user input as Text frames
	// and resize control messages as JSON Text frames starting with
	// the magic prefix \x1b[lab:resize ... (kept simple to avoid an
	// extra channel). For phase 3 we only handle keystrokes; resize
	// can layer in later.
	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return
		}
		if _, err := ptmx.Write(data); err != nil {
			return
		}
	}
}
