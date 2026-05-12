package main

import "fmt"

const BoardSize = 10

type CellState uint8

const (
	CellEmpty CellState = iota
	CellShip
	CellHit
	CellMiss
)

type PlacedShip struct {
	Name       string
	Size       int
	X, Y       int
	Horizontal bool
	Hits       int
}

func (s *PlacedShip) Contains(x, y int) bool {
	for i := 0; i < s.Size; i++ {
		cx, cy := s.X, s.Y
		if s.Horizontal {
			cx += i
		} else {
			cy += i
		}
		if cx == x && cy == y {
			return true
		}
	}
	return false
}

func (s *PlacedShip) Sunk() bool { return s.Hits >= s.Size }

type Board struct {
	cells [BoardSize][BoardSize]CellState
	ships []PlacedShip
}

var shipSizes = map[string]int{
	"Carrier": 5, "Battleship": 4, "Cruiser": 3, "Submarine": 3, "Destroyer": 2,
}

func (b *Board) Place(ship PlacedShip) error {
	endX, endY := ship.X, ship.Y
	if ship.Horizontal {
		endX += ship.Size - 1
	} else {
		endY += ship.Size - 1
	}
	if ship.X < 0 || ship.Y < 0 || endX >= BoardSize || endY >= BoardSize {
		return fmt.Errorf("ship out of bounds")
	}
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

type ShotResult struct {
	Hit       bool
	Sunk      bool
	Ship      string
	SunkCells [][2]int
	Winner    bool
}

func (b *Board) Shoot(x, y int) (ShotResult, error) {
	if x < 0 || x >= BoardSize || y < 0 || y >= BoardSize {
		return ShotResult{}, fmt.Errorf("coordinate out of bounds")
	}
	switch b.cells[y][x] {
	case CellHit, CellMiss:
		return ShotResult{}, fmt.Errorf("already shot here")
	case CellEmpty:
		b.cells[y][x] = CellMiss
		return ShotResult{Hit: false}, nil
	}
	b.cells[y][x] = CellHit
	for idx := range b.ships {
		if !b.ships[idx].Contains(x, y) {
			continue
		}
		b.ships[idx].Hits++
		sunk := b.ships[idx].Sunk()
		var sunkCells [][2]int
		if sunk {
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
