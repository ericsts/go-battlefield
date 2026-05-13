package main

import "fmt"

const BoardSize = 10

// CellState representa o estado de uma célula do tabuleiro.
// uint8 = inteiro sem sinal de 8 bits (0 a 255), suficiente para 4 estados.
type CellState uint8

// Constantes que definem os possíveis estados de uma célula.
// iota é um contador automático: começa em 0 e incrementa +1 a cada linha.
// O tipo CellState é declarado só na primeira linha; as demais herdam.
const (
	CellEmpty CellState = iota // 0 — célula vazia
	CellShip                   // 1 — navio posicionado aqui
	CellHit                    // 2 — navio atingido
	CellMiss                   // 3 — tiro na água
)

// PlacedShip representa um navio já posicionado no tabuleiro.
type PlacedShip struct {
	Name       string
	Size       int
	X, Y       int  // coordenada inicial (canto superior esquerdo)
	Horizontal bool // true = cresce para direita; false = cresce para baixo
	Hits       int  // quantas células do navio já foram atingidas
}

// Contains verifica se a coordenada (x, y) faz parte deste navio.
// Itera célula por célula calculando a posição real com base na orientação.
func (s *PlacedShip) Contains(x, y int) bool {
	for i := 0; i < s.Size; i++ {
		cx, cy := s.X, s.Y
		if s.Horizontal {
			cx += i // avança na horizontal
		} else {
			cy += i // avança na vertical
		}
		if cx == x && cy == y {
			return true
		}
	}
	return false
}

// Sunk retorna true se todas as células do navio foram atingidas.
func (s *PlacedShip) Sunk() bool { return s.Hits >= s.Size }

// Board é o tabuleiro de um jogador: uma grade 10x10 e a lista de navios.
// cells[y][x] — primeiro índice é linha (y), segundo é coluna (x).
type Board struct {
	cells [BoardSize][BoardSize]CellState
	ships []PlacedShip
}

// shipSizes mapeia o nome de cada navio ao seu tamanho em células.
var shipSizes = map[string]int{
	"Carrier": 5, "Battleship": 4, "Cruiser": 3, "Submarine": 3, "Destroyer": 2,
}

// Place posiciona um navio no tabuleiro.
// Valida limites do tabuleiro e sobreposição com navios existentes antes de confirmar.
func (b *Board) Place(ship PlacedShip) error {
	// Calcula a coordenada final do navio para checar se sai do tabuleiro.
	endX, endY := ship.X, ship.Y
	if ship.Horizontal {
		endX += ship.Size - 1
	} else {
		endY += ship.Size - 1
	}
	if ship.X < 0 || ship.Y < 0 || endX >= BoardSize || endY >= BoardSize {
		return fmt.Errorf("ship out of bounds")
	}
	// Primeira passagem: só valida, não modifica nada ainda.
	for i := 0; i < ship.Size; i++ {
		x, y := ship.X, ship.Y
		if ship.Horizontal {
			x += i
		} else {
			y += i
		}
		if b.cells[y][x] != CellEmpty {
			return fmt.Errorf("cell %d,%d already occupied", x, y)
		}
	}
	// Segunda passagem: tudo ok, agora marca as células e adiciona o navio.
	for i := 0; i < ship.Size; i++ {
		x, y := ship.X, ship.Y
		if ship.Horizontal {
			x += i
		} else {
			y += i
		}
		b.cells[y][x] = CellShip
	}
	b.ships = append(b.ships, ship)
	return nil
}

// ShotResult carrega todas as informações resultantes de um tiro.
type ShotResult struct {
	Hit       bool     // true se acertou um navio
	Sunk      bool     // true se o navio foi afundado
	Ship      string   // nome do navio atingido/afundado
	SunkCells [][2]int // coordenadas de todas as células do navio afundado
	Winner    bool     // true se este tiro afundou o último navio (vitória)
}

// Shoot processa um tiro na coordenada (x, y) do tabuleiro.
// Atualiza o estado da célula e dos navios, retornando o resultado.
func (b *Board) Shoot(x, y int) (ShotResult, error) {
	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		return ShotResult{}, fmt.Errorf("coordinate out of bounds")
	}
	// switch sem expressão equivale a switch true — avalia cada case como bool.
	switch b.cells[y][x] {
	case CellHit, CellMiss:
		return ShotResult{}, fmt.Errorf("already shot here")
	case CellEmpty:
		b.cells[y][x] = CellMiss
		return ShotResult{Hit: false}, nil
	}

	// Se chegou aqui, a célula tinha CellShip — tiro acertou.
	b.cells[y][x] = CellHit
	for idx := range b.ships {
		if !b.ships[idx].Contains(x, y) {
			continue
		}
		b.ships[idx].Hits++
		sunk := b.ships[idx].Sunk()
		var sunkCells [][2]int
		if sunk {
			// Coleta todas as coordenadas do navio afundado para enviar ao cliente.
			for i := 0; i < b.ships[idx].Size; i++ {
				cx, cy := b.ships[idx].X, b.ships[idx].Y
				if b.ships[idx].Horizontal {
					cx += i
				} else {
					cy += i
				}
				sunkCells = append(sunkCells, [2]int{cx, cy})
			}
		}
		winner := sunk && b.allSunk()
		return ShotResult{Hit: true, Sunk: sunk, Ship: b.ships[idx].Name, SunkCells: sunkCells, Winner: winner}, nil
	}
	return ShotResult{Hit: true}, nil
}

// allSunk retorna true se todos os navios do tabuleiro foram afundados.
// Retorna false se não há navios (jogo ainda não configurado).
func (b *Board) allSunk() bool {
	if len(b.ships) == 0 {
		return false
	}
	for i := range b.ships {
		if !b.ships[i].Sunk() {
			return false
		}
	}
	return true
}
