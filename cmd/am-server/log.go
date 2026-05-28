package main

import (
	"encoding/json"
	"sync"
	"time"
)

// logEvent is one line in the CommandLog feed. The Verb field is
// the literal `am` invocation you could type in a terminal to do the
// same thing, pedagogical bridge between the UI and the CLI.
type logEvent struct {
	TS      string `json:"ts"`
	Verb    string `json:"verb"`
	OK      bool   `json:"ok"`
	Summary string `json:"summary,omitempty"`
}

type logBus struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func newLogBus() *logBus {
	return &logBus{subs: make(map[chan []byte]struct{})}
}

func (b *logBus) subscribe() chan []byte {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *logBus) unsubscribe(ch chan []byte) {
	b.mu.Lock()
	delete(b.subs, ch)
	b.mu.Unlock()
	close(ch)
}

func (b *logBus) emit(verb string, ok bool, summary string) {
	ev := logEvent{
		TS:      time.Now().UTC().Format(time.RFC3339),
		Verb:    verb,
		OK:      ok,
		Summary: summary,
	}
	payload, err := json.Marshal(ev)
	if err != nil {
		return
	}
	b.mu.Lock()
	subs := make([]chan []byte, 0, len(b.subs))
	for c := range b.subs {
		subs = append(subs, c)
	}
	b.mu.Unlock()
	for _, c := range subs {
		select {
		case c <- payload:
		default:
		}
	}
}
