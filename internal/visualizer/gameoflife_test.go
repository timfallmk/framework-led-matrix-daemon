package visualizer

import (
	"math/rand"
	"testing"
)

// newTestGoL returns a zeroed GameOfLife with a fixed RNG — seed() is not called.
func newTestGoL() *GameOfLife {
	return &GameOfLife{
		rng: rand.New(rand.NewSource(42)),
	}
}

// addStillBlock places a 2x2 still-life block at (col, row) to keep the alive
// count >= 4 so the alive<4 reseed threshold doesn't interfere with rule tests.
// The block must be far enough from the test cells that they don't interact.
func addStillBlock(g *GameOfLife, col, row int) {
	g.grid[col][row] = true
	g.grid[col][row+1] = true
	g.grid[col+1][row] = true
	g.grid[col+1][row+1] = true
}

// TestGameOfLife_Neighbors tests the neighbour-counting function including
// the toroidal wrap that keeps patterns flowing on the small grid.
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
	// Cells that are neighbours of (0,0) only via toroidal wrap.
	g.grid[golWidth-1][golHeight-1] = true // (-1,-1) wrapped
	g.grid[0][golHeight-1] = true          // (0,-1) wrapped
	g.grid[golWidth-1][0] = true           // (-1,0) wrapped

	if n := g.neighbors(0, 0); n != 3 {
		t.Errorf("neighbors(0,0) toroidal = %d, want 3", n)
	}
}

func TestGameOfLife_Neighbors_SelfExcluded(t *testing.T) {
	g := newTestGoL()
	g.grid[4][16] = true // only the cell itself

	if n := g.neighbors(4, 16); n != 0 {
		t.Errorf("neighbors(4,16) counted self: got %d, want 0", n)
	}
}

// TestGameOfLife_Tick_Birth verifies that a dead cell with exactly 3 live
// neighbours is born in the next generation.
func TestGameOfLife_Tick_Birth(t *testing.T) {
	g := newTestGoL()
	addStillBlock(g, 7, 30) // background: keeps alive>=4, far from test area
	// Horizontal blinker at row 15; (4,16) is dead with 3 neighbours below it.
	g.grid[3][15] = true
	g.grid[4][15] = true
	g.grid[5][15] = true

	g.Tick()

	if !g.grid[4][16] {
		t.Error("Tick(): dead cell with 3 neighbours should be born")
	}
}

// TestGameOfLife_Tick_SurvivalWith2 verifies that a live cell with exactly 2
// live neighbours survives.
func TestGameOfLife_Tick_SurvivalWith2(t *testing.T) {
	g := newTestGoL()
	addStillBlock(g, 7, 30)
	// Horizontal blinker; centre cell (4,16) has exactly 2 live neighbours.
	g.grid[3][16] = true
	g.grid[4][16] = true
	g.grid[5][16] = true

	g.Tick()

	if !g.grid[4][16] {
		t.Error("Tick(): live cell with 2 neighbours should survive")
	}
}

// TestGameOfLife_Tick_DeathByIsolation verifies that a live cell with 0
// neighbours dies.
func TestGameOfLife_Tick_DeathByIsolation(t *testing.T) {
	g := newTestGoL()
	addStillBlock(g, 7, 30) // keeps alive>=4 so no population-reseed fires
	g.grid[4][16] = true    // isolated — 0 neighbours

	g.Tick()

	if g.grid[4][16] {
		t.Error("Tick(): isolated live cell should die (0 neighbours)")
	}
}

// TestGameOfLife_Tick_DeathByOvercrowding verifies that a live cell with more
// than 3 live neighbours dies.
func TestGameOfLife_Tick_DeathByOvercrowding(t *testing.T) {
	g := newTestGoL()
	addStillBlock(g, 7, 30)
	g.grid[4][16] = true // centre: 4 neighbours → dies
	g.grid[3][15] = true
	g.grid[4][15] = true
	g.grid[5][15] = true
	g.grid[3][16] = true

	g.Tick()

	if g.grid[4][16] {
		t.Error("Tick(): live cell with 4 neighbours should die (overcrowding)")
	}
}

// TestGameOfLife_Tick_StillLife_HashStagnation verifies that the hash-based
// stagnation detector reseeds a 2x2 still-life block (oscillators that maintain
// a constant alive-count would be missed by the old alive==lastAlive check).
func TestGameOfLife_Tick_StillLife_HashStagnation(t *testing.T) {
	g := newTestGoL()
	// 2x2 block is a still life: same grid hash every generation.
	g.grid[3][16] = true
	g.grid[3][17] = true
	g.grid[4][16] = true
	g.grid[4][17] = true

	// Tick 1: hash stored for the first time; staleCount stays 0.
	g.Tick()
	if g.staleCount != 0 {
		t.Errorf("tick 1: staleCount = %d, want 0 (first seen hash)", g.staleCount)
	}

	// Ticks 2-4: same hash detected each time; staleCount climbs to 3.
	for tick := 2; tick <= 4; tick++ {
		g.Tick()
		want := tick - 1
		if g.staleCount != want {
			t.Errorf("tick %d: staleCount = %d, want %d", tick, g.staleCount, want)
		}
	}

	// Tick 5: staleCount reaches 4 → reseed fires → gen and staleCount reset.
	g.Tick()
	if g.gen != 0 {
		t.Errorf("after still-life reseed: gen = %d, want 0", g.gen)
	}
	if g.staleCount != 0 {
		t.Errorf("after still-life reseed: staleCount = %d, want 0", g.staleCount)
	}
}

// TestGameOfLife_Tick_ReseedOnLowPopulation verifies that fewer than 4 live
// cells triggers an immediate reseed.
func TestGameOfLife_Tick_ReseedOnLowPopulation(t *testing.T) {
	g := newTestGoL()
	// Three mutually non-adjacent isolated cells (no neighbours → all die).
	// (0,0) and (8,33) would be toroidal neighbours, so use (8,20) instead.
	g.grid[0][0] = true
	g.grid[4][16] = true
	g.grid[8][20] = true

	g.Tick()

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

// TestGameOfLife_Seed_ResetsState verifies that seed() clears generation,
// stale counter, and hash history.
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

// TestGameOfLife_Frame_MatchesGrid verifies that Frame() maps live cells to 255
// and dead cells to 0.
func TestGameOfLife_Frame_MatchesGrid(t *testing.T) {
	g := newTestGoL()
	g.grid[0][0] = true
	g.grid[4][16] = true
	g.grid[8][33] = true

	f := g.Frame()

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
			t.Errorf("Frame()[%d][%d] = %d, want %d", tc.col, tc.row, f[tc.col][tc.row], tc.want)
		}
	}
}
