package display

import (
	"math/rand"
	"testing"
)

func newTestGoL() *gameOfLife {
	return &gameOfLife{
		rng: rand.New(rand.NewSource(42)),
	}
}

func addStillBlock(g *gameOfLife, col, row int) {
	g.grid[col][row] = true
	g.grid[col][row+1] = true
	g.grid[col+1][row] = true
	g.grid[col+1][row+1] = true
}

func TestGameOfLife_Neighbors_Center(t *testing.T) {
	g := newTestGoL()
	g.grid[3][16] = true
	g.grid[4][15] = true
	g.grid[5][17] = true

	if n := g.neighbors(4, 16); n != 3 {
		t.Errorf("neighbors(4,16) = %d, want 3", n)
	}
}

func TestGameOfLife_Neighbors_ToroidalWrap(t *testing.T) {
	g := newTestGoL()
	g.grid[golWidth-1][golHeight-1] = true
	g.grid[0][golHeight-1] = true
	g.grid[golWidth-1][0] = true

	if n := g.neighbors(0, 0); n != 3 {
		t.Errorf("neighbors(0,0) toroidal = %d, want 3", n)
	}
}

func TestGameOfLife_Neighbors_SelfExcluded(t *testing.T) {
	g := newTestGoL()
	g.grid[4][16] = true

	if n := g.neighbors(4, 16); n != 0 {
		t.Errorf("neighbors(4,16) counted self: got %d, want 0", n)
	}
}

func TestGameOfLife_Tick_Birth(t *testing.T) {
	g := newTestGoL()
	addStillBlock(g, 7, 30)
	g.grid[3][15] = true
	g.grid[4][15] = true
	g.grid[5][15] = true

	g.tick()

	if !g.grid[4][16] {
		t.Error("tick(): dead cell with 3 neighbours should be born")
	}
}

func TestGameOfLife_Tick_SurvivalWith2(t *testing.T) {
	g := newTestGoL()
	addStillBlock(g, 7, 30)
	g.grid[3][16] = true
	g.grid[4][16] = true
	g.grid[5][16] = true

	g.tick()

	if !g.grid[4][16] {
		t.Error("tick(): live cell with 2 neighbours should survive")
	}
}

func TestGameOfLife_Tick_DeathByIsolation(t *testing.T) {
	g := newTestGoL()
	addStillBlock(g, 7, 30)
	g.grid[4][16] = true

	g.tick()

	if g.grid[4][16] {
		t.Error("tick(): isolated live cell should die (0 neighbours)")
	}
}

func TestGameOfLife_Tick_DeathByOvercrowding(t *testing.T) {
	g := newTestGoL()
	addStillBlock(g, 7, 30)
	g.grid[4][16] = true
	g.grid[3][15] = true
	g.grid[4][15] = true
	g.grid[5][15] = true
	g.grid[3][16] = true

	g.tick()

	if g.grid[4][16] {
		t.Error("tick(): live cell with 4 neighbours should die (overcrowding)")
	}
}

func TestGameOfLife_Tick_StillLife_HashStagnation(t *testing.T) {
	g := newTestGoL()
	g.grid[3][16] = true
	g.grid[3][17] = true
	g.grid[4][16] = true
	g.grid[4][17] = true

	g.tick()
	if g.staleCount != 0 {
		t.Errorf("tick 1: staleCount = %d, want 0 (first seen hash)", g.staleCount)
	}

	for tick := 2; tick <= 4; tick++ {
		g.tick()
		want := tick - 1
		if g.staleCount != want {
			t.Errorf("tick %d: staleCount = %d, want %d", tick, g.staleCount, want)
		}
	}

	g.tick()
	if g.gen != 0 {
		t.Errorf("after still-life reseed: gen = %d, want 0", g.gen)
	}
	if g.staleCount != 0 {
		t.Errorf("after still-life reseed: staleCount = %d, want 0", g.staleCount)
	}
}

func TestGameOfLife_Tick_ReseedOnLowPopulation(t *testing.T) {
	g := newTestGoL()
	g.grid[0][0] = true
	g.grid[4][16] = true
	g.grid[8][20] = true

	g.tick()

	alive := 0
	for col := range g.grid {
		for _, cell := range g.grid[col] {
			if cell {
				alive++
			}
		}
	}
	if alive < 4 {
		t.Errorf("after low-population reseed: alive = %d, want >= 4", alive)
	}
}

func TestGameOfLife_Seed_ResetsState(t *testing.T) {
	g := newTestGoL()
	g.gen = 99
	g.staleCount = 7
	g.recentHashes[0] = 12345
	g.recentHashes[5] = 67890

	g.seed()

	if g.gen != 0 {
		t.Errorf("seed(): gen = %d, want 0", g.gen)
	}
	if g.staleCount != 0 {
		t.Errorf("seed(): staleCount = %d, want 0", g.staleCount)
	}
	for i, h := range g.recentHashes {
		if h != 0 {
			t.Errorf("seed(): recentHashes[%d] = %d, want 0", i, h)
		}
	}
}

func TestGameOfLife_Frame_MatchesGrid(t *testing.T) {
	g := newTestGoL()
	g.grid[0][0] = true
	g.grid[4][16] = true
	g.grid[8][33] = true

	f := g.frame()

	for _, tc := range []struct {
		col, row int
		want     byte
	}{
		{0, 0, 255},
		{4, 16, 255},
		{8, 33, 255},
		{1, 1, 0},
	} {
		if f[tc.col][tc.row] != tc.want {
			t.Errorf("frame()[%d][%d] = %d, want %d", tc.col, tc.row, f[tc.col][tc.row], tc.want)
		}
	}
}
