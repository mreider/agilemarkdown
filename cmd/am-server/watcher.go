package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// watcher debounces filesystem events on the project tree and emits a
// fresh snapshot to a callback. The 200ms debounce coalesces the burst
// of writes `am sync` does when it regenerates derived files.
type watcher struct {
	root  string
	emit  func([]byte)
	fsw   *fsnotify.Watcher
	stop  chan struct{}
	mu    sync.Mutex
	dirty bool
}

func newWatcher(root string, emit func([]byte)) (*watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &watcher{
		root: root,
		emit: emit,
		fsw:  fsw,
		stop: make(chan struct{}),
	}
	if err := w.addRecursive(root); err != nil {
		fsw.Close()
		return nil, err
	}
	go w.loop()
	return w, nil
}

func (w *watcher) addRecursive(dir string) error {
	// fsnotify on macOS/Linux watches a directory non-recursively;
	// walk and Add every subdir. Skip noisy paths (.git, node_modules).
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		name := info.Name()
		if strings.HasPrefix(name, ".git") || name == "node_modules" {
			return filepath.SkipDir
		}
		return w.fsw.Add(path)
	})
}

func (w *watcher) loop() {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			// Ignore the derived view files the snapshot itself
			// already encodes, otherwise the snapshot push triggers
			// an event that triggers another snapshot push.
			base := filepath.Base(ev.Name)
			if strings.HasPrefix(base, "_priority") || strings.HasPrefix(base, "_icebox") || strings.HasPrefix(base, ".") {
				continue
			}
			w.mu.Lock()
			w.dirty = true
			w.mu.Unlock()
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("watcher: %v", err)
		case <-tick.C:
			w.mu.Lock()
			dirty := w.dirty
			w.dirty = false
			w.mu.Unlock()
			if !dirty {
				continue
			}
			snap, err := buildSnapshot(w.root)
			if err != nil {
				log.Printf("watcher build: %v", err)
				continue
			}
			payload, err := json.Marshal(snap)
			if err != nil {
				continue
			}
			w.emit(payload)
		}
	}
}

func (w *watcher) Close() {
	close(w.stop)
	w.fsw.Close()
}
