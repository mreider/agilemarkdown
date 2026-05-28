package commands

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v3"
)

// Running `am create-item Foo` outside a backlog directory used to
// print the error and exit 0, so wrappers that got cwd wrong (e.g.
// the VS Code extension before the path-prefix fix) saw a "success"
// but no file. Lock the loud failure behavior in here.
func TestCreateItemErrorsOutsideBacklogDirectory(t *testing.T) {
	dir := t.TempDir()
	prev, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prev) })

	app := &cli.Command{Commands: []*cli.Command{CreateItemCommand}}
	stdout, stderr := captureStdio(t)
	defer stdout.restore()
	defer stderr.restore()

	err = app.Run(context.Background(), []string{"am", "create-item", "Foo"})
	if err == nil {
		t.Fatalf("expected an error from create-item outside a backlog dir; got nil")
	}
	if !strings.Contains(err.Error(), "backlog folder") {
		t.Fatalf("error %q did not mention 'backlog folder'", err)
	}
	// No phantom Foo.md or _icebox.md should have been written.
	for _, name := range []string{"Foo.md", "_icebox.md", "_priority.md"} {
		p := filepath.Join(dir, name)
		if _, statErr := os.Stat(p); !os.IsNotExist(statErr) {
			t.Fatalf("create-item should not have written %q (stat err: %v)", p, statErr)
		}
	}
}

// Helper: capture+restore stdout/stderr so the cli library's prints
// don't leak into the test output.
type stdioCapture struct {
	target  **os.File
	saved   *os.File
	w       *os.File
	buf     bytes.Buffer
	doneCh  chan struct{}
	closed  bool
}

func (s *stdioCapture) restore() {
	if s.closed {
		return
	}
	s.closed = true
	_ = s.w.Close()
	<-s.doneCh
	*s.target = s.saved
}

func captureStdio(t *testing.T) (*stdioCapture, *stdioCapture) {
	t.Helper()
	cap1 := captureOne(t, &os.Stdout)
	cap2 := captureOne(t, &os.Stderr)
	return cap1, cap2
}

func captureOne(t *testing.T, target **os.File) *stdioCapture {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	c := &stdioCapture{target: target, saved: *target, w: w, doneCh: make(chan struct{})}
	*target = w
	go func() {
		_, _ = io.Copy(&c.buf, r)
		close(c.doneCh)
	}()
	return c
}
