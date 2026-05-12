# go-battleship

> [🇺🇸 Read in English](README.en.md)

Jogo de batalha naval multiplayer em tempo real — backend em Go com WebSockets e frontend HTML/CSS/JS puro, sem frameworks. Crie uma sala, compartilhe o link com um amigo e afunde a frota inimiga.

🎮 **Demo:** [battleship.ericsantos.eu](https://battleship.ericsantos.eu)

## Tecnologias

| Tecnologia | Uso |
|---|---|
| **Go** (stdlib) | servidor HTTP e lógica do jogo |
| **gorilla/websocket** | comunicação em tempo real |
| **HTML / CSS / JS** | frontend sem frameworks |

## Como funciona

```
Jogador A abre /
  └─► clica "Create New Game"
  └─► recebe URL única (ex: /game/a3f9c2b1)
  └─► compartilha o link com Jogador B

Jogador B abre o link
  └─► ambos entram na fase de colocação

Fase de colocação (simultânea)
  └─► cada um posiciona 5 navios no tabuleiro
  └─► clica "Ready!" quando terminar
  └─► jogo começa assim que os dois confirmam

Fase de batalha (turnos alternados)
  └─► clique no tabuleiro inimigo para atirar
  └─► acerto = atira novamente
  └─► erro = vez do oponente
  └─► afunde toda a frota inimiga para vencer
```

## Frota

| Navio | Tamanho |
|---|---|
| Carrier | 5 |
| Battleship | 4 |
| Cruiser | 3 |
| Submarine | 3 |
| Destroyer | 2 |

## Arquitetura

```
main.go       # servidor HTTP, upgrade WebSocket, roteamento de mensagens
hub.go        # gerenciamento de salas (criação, busca, exclusão)
game.go       # lógica do tabuleiro: posicionamento, tiros, afundamento, vitória
static/
  index.html  # estrutura HTML — 6 telas (lobby, aguardando, posicionamento, batalha…)
  style.css   # tema naval dark
  game.js     # máquina de estados do cliente + WebSocket + renderização dos tabuleiros
```

### Protocolo WebSocket

**Cliente → Servidor:**

| Tipo | Payload | Descrição |
|---|---|---|
| `place_ships` | `{ ships: [{name, x, y, horizontal}] }` | envia posicionamento dos 5 navios |
| `shoot` | `{ x, y }` | dispara na coordenada (x, y) do tabuleiro inimigo |

**Servidor → Cliente:**

| Tipo | Descrição |
|---|---|
| `joined` | confirmação de entrada na sala (`player_id`, `waiting`) |
| `phase_change` | mudança de fase (`placement`) |
| `waiting_for_opponent` | você está pronto, aguardando o oponente |
| `game_start` | batalha iniciada (`your_turn: true/false`) |
| `shot_result` | resultado do seu tiro (`hit`, `sunk`, `sunk_cells`, `winner`, `your_turn`) |
| `enemy_shot` | o oponente atirou em você (mesmos campos) |
| `opponent_disconnected` | oponente saiu da partida |

### Fluxo de estado no servidor

```
PhaseWaiting    → aguarda o 2º jogador conectar
PhasePlacement  → ambos posicionam navios (simultâneo)
PhaseBattle     → turnos alternados de tiro
PhaseOver       → alguém afundou toda a frota inimiga
```

## Como rodar

```bash
git clone <repo>
cd go-battleship
go mod tidy
go run .
```

Acesse `http://localhost:8080`, clique em **Create New Game** e envie o link para um amigo.

## Como jogar

1. **Selecione** um navio na lista lateral
2. **Passe o mouse** pelo tabuleiro para ver o preview da posição
3. Pressione **R** (ou clique em "Rotate") para alternar horizontal/vertical
4. **Clique** na posição desejada para posicionar
5. Use **🎲 Random** para posicionamento automático
6. Clique em um navio já posicionado para **recolhê-lo**
7. Quando todos os navios estiverem posicionados, clique em **Ready!**
8. Na batalha, clique no tabuleiro inimigo para **atirar**
   - Acerto → atire novamente
   - Erro → vez do oponente
