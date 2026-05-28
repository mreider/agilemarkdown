package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
)

func (s *server) handleWsSnapshot(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // lab is open; LABORATORY.md decision #1
	})
	if err != nil {
		log.Printf("ws accept: %v", err)
		return
	}
	defer c.CloseNow()

	ch := make(chan []byte, 8)
	s.subsMu.Lock()
	s.subs[ch] = struct{}{}
	s.subsMu.Unlock()
	defer func() {
		s.subsMu.Lock()
		delete(s.subs, ch)
		s.subsMu.Unlock()
		close(ch)
	}()

	// Initial snapshot so the client doesn't need a separate GET.
	if snap, err := buildSnapshot(s.root); err == nil {
		if payload, err := json.Marshal(snap); err == nil {
			if err := c.Write(r.Context(), websocket.MessageText, payload); err != nil {
				return
			}
		}
	}

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		// Read pump: drop everything; the snapshot channel is server-
		// push only. A closed connection cancels the ctx.
		for {
			if _, _, err := c.Read(ctx); err != nil {
				cancel()
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-ch:
			if !ok {
				return
			}
			writeCtx, cancelWrite := context.WithTimeout(ctx, 5*time.Second)
			err := c.Write(writeCtx, websocket.MessageText, payload)
			cancelWrite()
			if err != nil {
				return
			}
		}
	}
}

// handleWsLog streams CommandLog events. Every write route emits one
// structured event per call; the panel in the lab renders them as a
// fake terminal so the reader sees "click → equivalent CLI verb".
func (s *server) handleWsLog(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		log.Printf("ws/log accept: %v", err)
		return
	}
	defer c.CloseNow()

	ch := s.log.subscribe()
	defer s.log.unsubscribe(ch)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	go func() {
		for {
			if _, _, err := c.Read(ctx); err != nil {
				cancel()
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-ch:
			if !ok {
				return
			}
			writeCtx, cancelWrite := context.WithTimeout(ctx, 5*time.Second)
			err := c.Write(writeCtx, websocket.MessageText, payload)
			cancelWrite()
			if err != nil {
				return
			}
		}
	}
}
