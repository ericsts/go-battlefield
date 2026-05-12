package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

type GamePhase int

const (
	PhaseWaiting   GamePhase = iota
	PhasePlacement
	PhaseBattle
	PhaseOver
)

type Room struct {
	id      string
	players [2]*Client
	boards  [2]*Board
	phase   GamePhase
	turn    int
	ready   [2]bool
	closed  bool
	mu      sync.Mutex
}

func (r *Room) IsFull() bool {
	return r.players[0] != nil && r.players[1] != nil
}

type Hub struct {
	rooms map[string]*Room
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]*Room)}
}

func (h *Hub) NewRoom() *Room {
	b := make([]byte, 4)
	rand.Read(b)
	room := &Room{id: hex.EncodeToString(b)}
	h.mu.Lock()
	h.rooms[room.id] = room
	h.mu.Unlock()
	return room
}

func (h *Hub) GetRoom(id string) (*Room, bool) {
	h.mu.RLock()
	r, ok := h.rooms[id]
	h.mu.RUnlock()
	return r, ok
}

func (h *Hub) DeleteRoom(id string) {
	h.mu.Lock()
	delete(h.rooms, id)
	h.mu.Unlock()
}
