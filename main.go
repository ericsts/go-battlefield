package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	conn     *websocket.Conn
	playerID int
	mu       sync.Mutex
}

func (c *Client) Send(v interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.WriteJSON(v)
}

type InMsg struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

var hub = NewHub()

func wsHandler(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimPrefix(r.URL.Path, "/ws/")
	room, ok := hub.GetRoom(roomID)
	if !ok {
		http.Error(w, "room not found", http.StatusNotFound)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	room.mu.Lock()
	if room.closed || room.IsFull() {
		room.mu.Unlock()
		conn.WriteJSON(map[string]string{"type": "error", "message": "room unavailable"})
		conn.Close()
		return
	}

	client := &Client{conn: conn}
	slot := -1
	for i, p := range room.players {
		if p == nil {
			room.players[i] = client
			client.playerID = i
			slot = i
			break
		}
	}
	full := room.IsFull()
	if full && room.phase == PhaseWaiting {
		room.phase = PhasePlacement
		room.boards[0] = &Board{}
		room.boards[1] = &Board{}
	}
	room.mu.Unlock()

	if slot == -1 {
		conn.Close()
		return
	}

	client.Send(map[string]interface{}{
		"type":      "joined",
		"player_id": client.playerID,
		"room_id":   roomID,
		"waiting":   !full,
	})

	if full {
		broadcast(room, map[string]interface{}{
			"type":  "phase_change",
			"phase": "placement",
		})
	}

	defer func() {
		conn.Close()
		room.mu.Lock()
		room.players[slot] = nil
		wasWaiting := room.phase == PhaseWaiting
		room.closed = true
		opp := room.players[1-slot]
		room.mu.Unlock()

		hub.DeleteRoom(roomID)
		if !wasWaiting && opp != nil {
			opp.Send(map[string]string{"type": "opponent_disconnected"})
		}
	}()

	for {
		var msg InMsg
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}
		handleMsg(room, client, msg)
	}
}

func handleMsg(room *Room, c *Client, msg InMsg) {
	switch msg.Type {

	case "place_ships":
		var data struct {
			Ships []struct {
				Name       string `json:"name"`
				X          int    `json:"x"`
				Y          int    `json:"y"`
				Horizontal bool   `json:"horizontal"`
			} `json:"ships"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil || len(data.Ships) != 5 {
			c.Send(map[string]string{"type": "error", "message": "invalid request"})
			return
		}

		room.mu.Lock()
		if room.phase != PhasePlacement || room.ready[c.playerID] {
			room.mu.Unlock()
			return
		}

		required := map[string]bool{
			"Carrier": true, "Battleship": true, "Cruiser": true,
			"Submarine": true, "Destroyer": true,
		}
		board := &Board{}
		valid := true
		for _, s := range data.Ships {
			size, exists := shipSizes[s.Name]
			if !exists || !required[s.Name] {
				valid = false
				break
			}
			delete(required, s.Name)
			if err := board.Place(PlacedShip{Name: s.Name, Size: size, X: s.X, Y: s.Y, Horizontal: s.Horizontal}); err != nil {
				valid = false
				break
			}
		}
		if !valid || len(required) > 0 {
			room.mu.Unlock()
			c.Send(map[string]string{"type": "error", "message": "invalid ship placement"})
			return
		}

		room.boards[c.playerID] = board
		room.ready[c.playerID] = true
		bothReady := room.ready[0] && room.ready[1]
		if bothReady {
			room.phase = PhaseBattle
			room.turn = 0
		}
		room.mu.Unlock()

		if bothReady {
			for i, p := range room.players {
				if p != nil {
					p.Send(map[string]interface{}{"type": "game_start", "your_turn": i == 0})
				}
			}
		} else {
			c.Send(map[string]string{"type": "waiting_for_opponent"})
		}

	case "shoot":
		var data struct {
			X int `json:"x"`
			Y int `json:"y"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return
		}

		room.mu.Lock()
		if room.phase != PhaseBattle || room.turn != c.playerID {
			room.mu.Unlock()
			c.Send(map[string]string{"type": "error", "message": "not your turn"})
			return
		}
		res, err := room.boards[1-c.playerID].Shoot(data.X, data.Y)
		if err != nil {
			room.mu.Unlock()
			c.Send(map[string]string{"type": "error", "message": err.Error()})
			return
		}
		if res.Winner {
			room.phase = PhaseOver
		} else if !res.Hit {
			room.turn = 1 - room.turn
		}
		opp := room.players[1-c.playerID]
		myNextTurn := !res.Winner && res.Hit
		oppNextTurn := !res.Winner && !res.Hit
		room.mu.Unlock()

		c.Send(map[string]interface{}{
			"type":       "shot_result",
			"x":          data.X,
			"y":          data.Y,
			"hit":        res.Hit,
			"sunk":       res.Sunk,
			"ship":       res.Ship,
			"sunk_cells": res.SunkCells,
			"winner":     res.Winner,
			"your_turn":  myNextTurn,
		})
		if opp != nil {
			opp.Send(map[string]interface{}{
				"type":       "enemy_shot",
				"x":          data.X,
				"y":          data.Y,
				"hit":        res.Hit,
				"sunk":       res.Sunk,
				"ship":       res.Ship,
				"sunk_cells": res.SunkCells,
				"winner":     res.Winner,
				"your_turn":  oppNextTurn,
			})
		}
	}
}

func broadcast(room *Room, v interface{}) {
	for _, p := range room.players {
		if p != nil {
			p.Send(v)
		}
	}
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	http.HandleFunc("/create", func(w http.ResponseWriter, r *http.Request) {
		room := hub.NewRoom()
		http.Redirect(w, r, "/game/"+room.id, http.StatusSeeOther)
	})

	http.HandleFunc("/ws/", wsHandler)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/index.html")
	})

	log.Println("Battleship running → http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
