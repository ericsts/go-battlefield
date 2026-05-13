package main

import (
	"testing"
)

// --- PlacedShip.Contains ---

func TestContains_HorizontalShip(t *testing.T) {
	ship := PlacedShip{Name: "Destroyer", Size: 2, X: 3, Y: 5, Horizontal: true}

	// Células que pertencem ao navio
	if !ship.Contains(3, 5) {
		t.Error("esperava que (3,5) pertencesse ao navio")
	}
	if !ship.Contains(4, 5) {
		t.Error("esperava que (4,5) pertencesse ao navio")
	}

	// Células que não pertencem
	if ship.Contains(5, 5) {
		t.Error("(5,5) não deveria pertencer ao navio")
	}
	if ship.Contains(3, 6) {
		t.Error("(3,6) não deveria pertencer ao navio (linha diferente)")
	}
}

func TestContains_VerticalShip(t *testing.T) {
	ship := PlacedShip{Name: "Cruiser", Size: 3, X: 2, Y: 1, Horizontal: false}

	if !ship.Contains(2, 1) {
		t.Error("esperava que (2,1) pertencesse ao navio")
	}
	if !ship.Contains(2, 3) {
		t.Error("esperava que (2,3) pertencesse ao navio")
	}
	if ship.Contains(2, 4) {
		t.Error("(2,4) não deveria pertencer ao navio")
	}
}

// --- PlacedShip.Sunk ---

func TestSunk(t *testing.T) {
	ship := PlacedShip{Size: 3, Hits: 0}

	if ship.Sunk() {
		t.Error("navio com 0 hits não deveria estar afundado")
	}

	ship.Hits = 2
	if ship.Sunk() {
		t.Error("navio com 2/3 hits não deveria estar afundado")
	}

	ship.Hits = 3
	if !ship.Sunk() {
		t.Error("navio com 3/3 hits deveria estar afundado")
	}
}

// --- Board.Place ---

func TestPlace_ValidHorizontal(t *testing.T) {
	b := &Board{}
	ship := PlacedShip{Name: "Destroyer", Size: 2, X: 0, Y: 0, Horizontal: true}
	if err := b.Place(ship); err != nil {
		t.Fatalf("não esperava erro ao posicionar navio válido: %v", err)
	}
	if b.cells[0][0] != CellShip || b.cells[0][1] != CellShip {
		t.Error("células do navio deveriam ser CellShip")
	}
}

func TestPlace_ValidVertical(t *testing.T) {
	b := &Board{}
	ship := PlacedShip{Name: "Cruiser", Size: 3, X: 5, Y: 5, Horizontal: false}
	if err := b.Place(ship); err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}
	if b.cells[5][5] != CellShip || b.cells[6][5] != CellShip || b.cells[7][5] != CellShip {
		t.Error("células verticais deveriam ser CellShip")
	}
}

func TestPlace_OutOfBounds(t *testing.T) {
	b := &Board{}
	// Navio começa na coluna 9 e tem tamanho 3 → sai do tabuleiro (9+2=11 >= 10)
	ship := PlacedShip{Name: "Cruiser", Size: 3, X: 9, Y: 0, Horizontal: true}
	if err := b.Place(ship); err == nil {
		t.Error("esperava erro para navio fora dos limites")
	}
}

func TestPlace_NegativeCoordinate(t *testing.T) {
	b := &Board{}
	ship := PlacedShip{Name: "Destroyer", Size: 2, X: -1, Y: 0, Horizontal: true}
	if err := b.Place(ship); err == nil {
		t.Error("esperava erro para coordenada negativa")
	}
}

func TestPlace_Overlap(t *testing.T) {
	b := &Board{}
	ship1 := PlacedShip{Name: "Destroyer", Size: 2, X: 0, Y: 0, Horizontal: true}
	ship2 := PlacedShip{Name: "Cruiser", Size: 3, X: 1, Y: 0, Horizontal: true} // sobrepõe em (1,0)
	b.Place(ship1)
	if err := b.Place(ship2); err == nil {
		t.Error("esperava erro por sobreposição de navios")
	}
}

func TestPlace_ExactlyAtEdge(t *testing.T) {
	b := &Board{}
	// Navio de tamanho 2 começando na coluna 8 → termina em 9 (válido)
	ship := PlacedShip{Name: "Destroyer", Size: 2, X: 8, Y: 0, Horizontal: true}
	if err := b.Place(ship); err != nil {
		t.Fatalf("não esperava erro na borda válida: %v", err)
	}
}

// --- Board.Shoot ---

func TestShoot_Miss(t *testing.T) {
	b := &Board{}
	res, err := b.Shoot(0, 0)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}
	if res.Hit {
		t.Error("tiro em água não deveria ser Hit")
	}
	if b.cells[0][0] != CellMiss {
		t.Error("célula deveria ser CellMiss após tiro na água")
	}
}

func TestShoot_Hit(t *testing.T) {
	b := &Board{}
	b.Place(PlacedShip{Name: "Destroyer", Size: 2, X: 3, Y: 3, Horizontal: true})

	res, err := b.Shoot(3, 3)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}
	if !res.Hit {
		t.Error("deveria ter acertado o navio")
	}
	if res.Sunk {
		t.Error("navio de tamanho 2 com 1 hit não deveria estar afundado")
	}
	if b.cells[3][3] != CellHit {
		t.Error("célula deveria ser CellHit")
	}
}

func TestShoot_Sunk(t *testing.T) {
	b := &Board{}
	b.Place(PlacedShip{Name: "Destroyer", Size: 2, X: 0, Y: 0, Horizontal: true})

	b.Shoot(0, 0) // primeiro hit
	res, err := b.Shoot(1, 0)
	if err != nil {
		t.Fatalf("não esperava erro: %v", err)
	}
	if !res.Hit || !res.Sunk {
		t.Error("segundo hit no Destroyer deveria afundá-lo")
	}
	if res.Ship != "Destroyer" {
		t.Errorf("nome do navio afundado errado: %q", res.Ship)
	}
	// SunkCells deve conter as duas coordenadas do navio
	if len(res.SunkCells) != 2 {
		t.Errorf("esperava 2 SunkCells, got %d", len(res.SunkCells))
	}
}

func TestShoot_Winner(t *testing.T) {
	b := &Board{}
	b.Place(PlacedShip{Name: "Destroyer", Size: 2, X: 0, Y: 0, Horizontal: true})

	b.Shoot(0, 0)
	res, _ := b.Shoot(1, 0)
	if !res.Winner {
		t.Error("afundar o único navio deveria declarar vencedor")
	}
}

func TestShoot_AlreadyShot(t *testing.T) {
	b := &Board{}
	b.Shoot(5, 5)
	_, err := b.Shoot(5, 5) // mesmo lugar
	if err == nil {
		t.Error("esperava erro ao atirar no mesmo lugar duas vezes")
	}
}

func TestShoot_OutOfBounds(t *testing.T) {
	b := &Board{}
	_, err := b.Shoot(10, 0)
	if err == nil {
		t.Error("esperava erro para coordenada fora do tabuleiro")
	}
	_, err = b.Shoot(0, -1)
	if err == nil {
		t.Error("esperava erro para coordenada negativa")
	}
}

// --- Board.allSunk ---

func TestAllSunk_NoShips(t *testing.T) {
	b := &Board{}
	if b.allSunk() {
		t.Error("tabuleiro sem navios não deveria retornar allSunk=true")
	}
}

func TestAllSunk_PartialSunk(t *testing.T) {
	b := &Board{}
	b.Place(PlacedShip{Name: "Destroyer", Size: 2, X: 0, Y: 0, Horizontal: true})
	b.Place(PlacedShip{Name: "Cruiser", Size: 3, X: 0, Y: 2, Horizontal: true})

	// Afunda só o Destroyer
	b.Shoot(0, 0)
	b.Shoot(1, 0)

	if b.allSunk() {
		t.Error("ainda há navios intactos — allSunk deveria ser false")
	}
}

func TestAllSunk_AllSunk(t *testing.T) {
	b := &Board{}
	b.Place(PlacedShip{Name: "Destroyer", Size: 2, X: 0, Y: 0, Horizontal: true})

	b.Shoot(0, 0)
	b.Shoot(1, 0)

	if !b.allSunk() {
		t.Error("todos os navios afundados — allSunk deveria ser true")
	}
}

// --- Constantes iota ---

func TestCellStateValues(t *testing.T) {
	// Garante que os valores de iota não mudaram acidentalmente
	// (mudança quebraria comunicação com o cliente JS)
	if CellEmpty != 0 {
		t.Errorf("CellEmpty deveria ser 0, got %d", CellEmpty)
	}
	if CellShip != 1 {
		t.Errorf("CellShip deveria ser 1, got %d", CellShip)
	}
	if CellHit != 2 {
		t.Errorf("CellHit deveria ser 2, got %d", CellHit)
	}
	if CellMiss != 3 {
		t.Errorf("CellMiss deveria ser 3, got %d", CellMiss)
	}
}
