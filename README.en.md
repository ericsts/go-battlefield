# go-battleship

> [🇧🇷 Leia em Português](README.md)

Real-time multiplayer Battleship game — Go backend with WebSockets and a pure HTML/CSS/JS frontend, no frameworks. Create a room, share the link with a friend, and sink the enemy fleet.

🎮 **Demo:** [battleship.ericsantos.eu](https://battleship.ericsantos.eu)

## Technologies

| Technology | Purpose |
|---|---|
| **Go** (stdlib) | HTTP server and game logic |
| **gorilla/websocket** | real-time communication |
| **HTML / CSS / JS** | frameworkless frontend |

## How it works

```
Player A opens /
  └─► clicks "Create New Game"
  └─► gets a unique URL (e.g. /game/a3f9c2b1)
  └─► shares the link with Player B

Player B opens the link
  └─► both enter the placement phase

Placement phase (simultaneous)
  └─► each player positions 5 ships on their board
  └─► clicks "Ready!" when done
  └─► game starts once both confirm

Battle phase (alternating turns)
  └─► click the enemy board to fire
  └─► hit = fire again
  └─► miss = opponent's turn
  └─► sink the entire enemy fleet to win
```

## Fleet

| Ship | Size |
|---|---|
| Carrier | 5 |
| Battleship | 4 |
| Cruiser | 3 |
| Submarine | 3 |
| Destroyer | 2 |

## Architecture

```
main.go       # HTTP server, WebSocket upgrade, message routing
hub.go        # room management (create, lookup, delete)
game.go       # board logic: ship placement, shots, sinking, win condition
static/
  index.html  # HTML structure — 6 screens (lobby, waiting, placement, battle…)
  style.css   # dark naval theme
  game.js     # client state machine + WebSocket + board rendering
```

### WebSocket Protocol

**Client → Server:**

| Type | Payload | Description |
|---|---|---|
| `place_ships` | `{ ships: [{name, x, y, horizontal}] }` | send placement of all 5 ships |
| `shoot` | `{ x, y }` | fire at coordinate (x, y) on the enemy board |

**Server → Client:**

| Type | Description |
|---|---|
| `joined` | room entry confirmed (`player_id`, `waiting`) |
| `phase_change` | phase transition (`placement`) |
| `waiting_for_opponent` | you are ready, waiting for opponent |
| `game_start` | battle started (`your_turn: true/false`) |
| `shot_result` | result of your shot (`hit`, `sunk`, `sunk_cells`, `winner`, `your_turn`) |
| `enemy_shot` | opponent fired at you (same fields) |
| `opponent_disconnected` | opponent left the game |

### Server state machine

```
PhaseWaiting    → waiting for 2nd player to connect
PhasePlacement  → both players place ships (simultaneous)
PhaseBattle     → alternating turns
PhaseOver       → someone sank the entire enemy fleet
```

## Running locally

```bash
git clone <repo>
cd go-battleship
go mod tidy
go run .
```

Open `http://localhost:8080`, click **Create New Game**, and send the link to a friend.

## How to play

1. **Select** a ship from the sidebar list
2. **Hover** over the board to preview the placement
3. Press **R** (or click "Rotate") to toggle horizontal/vertical orientation
4. **Click** the desired position to place the ship
5. Use **🎲 Random** for automatic placement
6. Click an already-placed ship to **pick it back up**
7. Once all ships are placed, click **Ready!**
8. In battle, click the enemy board to **fire**
   - Hit → fire again
   - Miss → opponent's turn
