package display

import (
	"hash/fnv"
	"math/rand"
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
)

func init() {
	Register("gameoflife", func() MatrixDisplay { return newGameOfLifeDisplay() })
}

const (
	golWidth        = 9  // physical columns
	golHeight       = 34 // physical rows
	staleHistoryLen = 16 // detect oscillators up to period 16
	golTickRate     = 350 * time.Millisecond
)

// gameOfLife is the raw simulation — a 9×34 toroidal Conway's Game of Life.
type gameOfLife struct {
	grid         [golWidth][golHeight]bool
	recentHashes [staleHistoryLen]uint64
	rng          *rand.Rand
	staleCount   int
	gen          int
}

func newGameOfLife() *gameOfLife {
	g := &gameOfLife{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	g.seed()
	return g
}

func (g *gameOfLife) seed() {
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
func (g *gameOfLife) gridHash() uint64 {
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

func (g *gameOfLife) neighbors(col, row int) int {
	n := 0
	for dc := -1; dc <= 1; dc++ {
		for dr := -1; dr <= 1; dr++ {
			if dc == 0 && dr == 0 {
				continue
			}
			c := (col + dc + golWidth) % golWidth
			r := (row + dr + golHeight) % golHeight
			if g.grid[c][r] {
				n++
			}
		}
	}
	return n
}

func (g *gameOfLife) tick() {
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

func (g *gameOfLife) frame() [golWidth][golHeight]byte {
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

// GameOfLifeDisplay wraps the simulation and manages its own tick rate so it
// advances at ~350 ms regardless of the daemon's update_rate setting.
type GameOfLifeDisplay struct {
	gol      *gameOfLife
	lastTick time.Time
}

func newGameOfLifeDisplay() *GameOfLifeDisplay {
	return &GameOfLifeDisplay{
		gol:      newGameOfLife(),
		lastTick: time.Now(),
	}
}

// Render advances the simulation if enough time has elapsed, then returns
// the current generation as a brightness frame. stats is unused.
func (d *GameOfLifeDisplay) Render(_ *stats.StatsSummary) [9][34]byte {
	if time.Since(d.lastTick) >= golTickRate {
		d.gol.tick()
		d.lastTick = time.Now()
	}
	return d.gol.frame()
}
