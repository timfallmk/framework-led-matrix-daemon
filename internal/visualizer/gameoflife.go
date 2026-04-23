package visualizer

import (
	"hash/fnv"
	"math/rand"
	"time"
)

const (
	golWidth        = 9  // physical columns
	golHeight       = 34 // physical rows
	staleHistoryLen = 16 // detect oscillators up to period 16
)

// GameOfLife runs Conway's Game of Life on a 9×34 LED matrix grid.
type GameOfLife struct {
	grid         [golWidth][golHeight]bool
	recentHashes [staleHistoryLen]uint64
	rng          *rand.Rand
	staleCount   int
	gen          int
}

// NewGameOfLife creates and randomly seeds a new Game of Life instance.
func NewGameOfLife() *GameOfLife {
	g := &GameOfLife{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	g.seed()
	return g
}

func (g *GameOfLife) seed() {
	for col := range g.grid {
		for row := range g.grid[col] {
			g.grid[col][row] = g.rng.Float64() < 0.35
		}
	}
	g.gen = 0
	g.staleCount = 0
	g.recentHashes = [staleHistoryLen]uint64{}
}

// gridHash returns a 64-bit FNV-1a hash of the current grid state so that
// oscillators (blinkers, beehives, spaceships) are detected as stagnant rather
// than being confused with a stable alive-count.
func (g *GameOfLife) gridHash() uint64 {
	h := fnv.New64a()
	var buf [(golWidth*golHeight + 7) / 8]byte
	for col := 0; col < golWidth; col++ {
		for row := 0; row < golHeight; row++ {
			if g.grid[col][row] {
				idx := col*golHeight + row
				buf[idx/8] |= 1 << (uint(idx) % 8)
			}
		}
	}
	_, _ = h.Write(buf[:])
	return h.Sum64()
}

func (g *GameOfLife) neighbors(col, row int) int {
	n := 0
	for dc := -1; dc <= 1; dc++ {
		for dr := -1; dr <= 1; dr++ {
			if dc == 0 && dr == 0 {
				continue
			}
			// Toroidal wrap so patterns flow continuously on the small grid.
			c := (col + dc + golWidth) % golWidth
			r := (row + dr + golHeight) % golHeight
			if g.grid[c][r] {
				n++
			}
		}
	}
	return n
}

// Tick advances the simulation by one generation and re-seeds if it stagnates.
// Stagnation is detected by comparing the current grid hash against the last
// staleHistoryLen hashes, catching oscillators that the naive alive-count check
// would miss.
func (g *GameOfLife) Tick() {
	var next [golWidth][golHeight]bool
	alive := 0

	for col := 0; col < golWidth; col++ {
		for row := 0; row < golHeight; row++ {
			n := g.neighbors(col, row)
			switch {
			case g.grid[col][row] && (n == 2 || n == 3):
				next[col][row] = true
				alive++
			case !g.grid[col][row] && n == 3:
				next[col][row] = true
				alive++
			}
		}
	}

	g.grid = next
	g.gen++

	h := g.gridHash()
	stale := false
	for _, prev := range g.recentHashes {
		if prev == h {
			stale = true
			break
		}
	}
	if stale {
		g.staleCount++
	} else {
		g.staleCount = 0
	}
	g.recentHashes[g.gen%staleHistoryLen] = h

	if g.staleCount >= 4 || alive < 4 {
		g.seed()
	}
}

// Frame returns the current grid as a column-major [9][34]byte brightness array.
func (g *GameOfLife) Frame() [golWidth][golHeight]byte {
	var f [golWidth][golHeight]byte
	for col := 0; col < golWidth; col++ {
		for row := 0; row < golHeight; row++ {
			if g.grid[col][row] {
				f[col][row] = 255
			}
		}
	}
	return f
}
