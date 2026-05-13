package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// GamePhase representa a fase atual de uma partida.
type GamePhase int

// Fases possíveis de uma partida, usando iota para gerar os valores 0, 1, 2, 3.
const (
	PhaseWaiting   GamePhase = iota // 0 — aguardando o segundo jogador entrar
	PhasePlacement                  // 1 — jogadores posicionando navios
	PhaseBattle                     // 2 — batalha em andamento
	PhaseOver                       // 3 — partida encerrada
)

// Room representa uma sala de jogo com dois jogadores.
type Room struct {
	id      string
	players [2]*Client // ponteiros para os dois clientes; nil se o slot estiver vazio
	boards  [2]*Board  // tabuleiro de cada jogador
	phase   GamePhase
	turn    int        // índice do jogador cujo turno é agora (0 ou 1)
	ready   [2]bool    // indica se cada jogador já posicionou seus navios
	closed  bool       // true quando a sala foi encerrada (desconexão, etc.)
	mu      sync.Mutex // protege todos os campos acima contra acesso concorrente
}

// IsFull retorna true se os dois slots de jogador estão preenchidos.
func (r *Room) IsFull() bool {
	return r.players[0] != nil && r.players[1] != nil
}

// Hub centraliza todas as salas ativas do servidor.
type Hub struct {
	rooms map[string]*Room
	// RWMutex permite múltiplas leituras simultâneas (RLock/RUnlock),
	// mas garante exclusividade em escritas (Lock/Unlock).
	// Ideal aqui porque buscas de sala são muito mais frequentes que criações/deleções.
	mu sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[string]*Room)}
}

// NewRoom cria uma nova sala com ID aleatório de 8 caracteres hexadecimais.
func (h *Hub) NewRoom() *Room {
	b := make([]byte, 4)
	rand.Read(b) // gera 4 bytes aleatórios
	room := &Room{id: hex.EncodeToString(b)} // converte para string hex (ex: "a1b2c3d4")
	h.mu.Lock()
	h.rooms[room.id] = room
	h.mu.Unlock()
	return room
}

// GetRoom busca uma sala pelo ID. Usa RLock pois só lê o map.
// Retorna o ponteiro da sala e um bool indicando se foi encontrada.
func (h *Hub) GetRoom(id string) (*Room, bool) {
	h.mu.RLock()
	r, ok := h.rooms[id]
	h.mu.RUnlock()
	return r, ok
}

// DeleteRoom remove uma sala do hub. Usa Lock pois modifica o map.
func (h *Hub) DeleteRoom(id string) {
	h.mu.Lock()
	delete(h.rooms, id)
	h.mu.Unlock()
}
