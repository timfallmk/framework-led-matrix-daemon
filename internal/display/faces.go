package display

import (
	"time"

	"github.com/timfallmk/framework-led-matrix-daemon/internal/stats"
)

func init() {
	Register("faces", func() MatrixDisplay { return newFacesDisplay() })
}

const (
	faceWidth      = 9  // physical matrix columns
	faceTileHeight = 10 // rows per face tile
)

// facePattern is one compact 9×10 facial expression.
// rows[r] is a 9-character string: '#'=255, 'o'=160 (soft glow), '.'=0 (off).
type facePattern struct {
	rows    [faceTileHeight]string
	holdFor time.Duration
}

func (fp *facePattern) toTile() [faceWidth][faceTileHeight]byte {
	var tile [faceWidth][faceTileHeight]byte
	for row, s := range fp.rows {
		for col := 0; col < faceWidth && col < len(s); col++ {
			switch s[col] {
			case '#':
				tile[col][row] = 255
			case 'o':
				tile[col][row] = 160
			}
		}
	}
	return tile
}

// Face tile layout (rows within the 10-row tile):
//   0-1  top margin / brows
//   2-3  eyes
//   4    gap
//   5-6  mouth
//   7-9  bottom margin
var allFaces = []facePattern{
	// 0 — Kawaii Happy (^‿^)
	{holdFor: 6 * time.Second, rows: [faceTileHeight]string{
		".........", ".........",
		".##...##.", ".##...##.", // eyes
		".........",
		"..#...#..", "...###...", // smile (corners high, centre low → arc up)
		".........", ".........", ".........",
	}},
	// 1 — Kawaii UwU (ᵕ̈)
	{holdFor: 5 * time.Second, rows: [faceTileHeight]string{
		".........", ".........",
		".#.#.#.#.", ".###.###.", // U-shaped eyes
		".........",
		"...#.#...", "..#...#..", // ω mouth
		".........", ".........", ".........",
	}},
	// 2 — Kawaii Sleepy (-ᴗ-)
	{holdFor: 10 * time.Second, rows: [faceTileHeight]string{
		".........", ".........", ".........",
		".##...##.", // single-row half-closed eyes
		".........", ".........",
		"...###...", // flat mouth
		".........", ".........", ".........",
	}},
	// 3 — Kawaii Surprised (O_O)
	{holdFor: 4 * time.Second, rows: [faceTileHeight]string{
		".........",
		"..#...#..", ".#.#.#.#.", "..#...#..", // hollow circle eyes
		".........",
		"....#....", "...#.#...", "....#....", // small O mouth
		".........", ".........",
	}},
	// 4 — Anime Neutral (-_-)
	{holdFor: 5 * time.Second, rows: [faceTileHeight]string{
		".........", ".........", ".........",
		".#######.", // long flat eye line
		".........", ".........",
		"...###...", // flat mouth
		".........", ".........", ".........",
	}},
	// 5 — Anime Sad (T_T)
	{holdFor: 7 * time.Second, rows: [faceTileHeight]string{
		".........", ".........",
		".##...##.", ".##...##.", // eyes
		".#.......",             // tear
		"...###...", "..#...#..", // frown (centre high, corners low → arc down)
		".........", ".........", ".........",
	}},
	// 6 — Punk Smirk (wink + mouth raised right)
	{holdFor: 5 * time.Second, rows: [faceTileHeight]string{
		".........", ".........",
		".##......", ".##...##.", // left full eye, right wink (single row)
		".........",
		".....##..", "....###..", // smirk leaning right
		".........", ".........", ".........",
	}},
	// 7 — Punk Mean (ò_ó)
	{holdFor: 5 * time.Second, rows: [faceTileHeight]string{
		".........",
		".#.....#.", "..#...#..", // angry brows slanting inward
		".##...##.", // squinting eyes
		".........",
		"...###...", "..#...#..", // tight frown
		".........", ".........", ".........",
	}},
	// 8 — Don't Care (._.)
	{holdFor: 8 * time.Second, rows: [faceTileHeight]string{
		".........", ".........", ".........",
		"...#.#...", // tiny dot eyes
		".........", ".........",
		"...###...", // flat mouth
		".........", ".........", ".........",
	}},
	// 9 — Dead Inside (x_x)
	{holdFor: 7 * time.Second, rows: [faceTileHeight]string{
		".........",
		".#.#.#.#.", "..#...#..", ".#.#.#.#.", // X eyes
		".........",
		"..#####..", // flat mouth
		".........", ".........", ".........", ".........",
	}},
	// 10 — Genki / Energetic (^u^)
	{holdFor: 4 * time.Second, rows: [faceTileHeight]string{
		".........",
		"..#...#..", ".#.#.#.#.", // ^ eyes
		"o.......o", // blush
		".........",
		"..#...#..", "...###...", ".#.....#.", // wide smile
		".........", ".........",
	}},
}

// slotOffsets places three face tiles on the 34-row matrix with 2-row dark gaps.
// Layout: [0-9] face · [10-11] gap · [12-21] face · [22-23] gap · [24-33] face
var slotOffsets = [3]int{0, 12, 24}

// faceAnimator runs three independently cycling face slots.
type faceAnimator struct {
	tiles    [][faceWidth][faceTileHeight]byte
	patterns []facePattern
	slots    [3]int
	switched [3]time.Time
}

func newFaceAnimator() *faceAnimator {
	fa := &faceAnimator{
		patterns: allFaces,
		tiles:    make([][faceWidth][faceTileHeight]byte, len(allFaces)),
	}
	for i := range allFaces {
		fa.tiles[i] = allFaces[i].toTile()
	}
	n := len(allFaces)
	fa.slots = [3]int{0, n / 3, 2 * n / 3}
	now := time.Now()
	fa.switched = [3]time.Time{now, now, now}
	return fa
}

func (fa *faceAnimator) tick() {
	now := time.Now()
	for i := range fa.slots {
		if now.Sub(fa.switched[i]) >= fa.patterns[fa.slots[i]].holdFor {
			fa.slots[i] = (fa.slots[i] + 1) % len(fa.patterns)
			fa.switched[i] = now
		}
	}
}

func (fa *faceAnimator) frame() [faceWidth][34]byte {
	var frame [faceWidth][34]byte
	for i, slot := range fa.slots {
		tile := fa.tiles[slot]
		offset := slotOffsets[i]
		for col := 0; col < faceWidth; col++ {
			for row := 0; row < faceTileHeight; row++ {
				frame[col][offset+row] = tile[col][row]
			}
		}
	}
	return frame
}

// FacesDisplay cycles three independent face expressions on the 34-row matrix.
type FacesDisplay struct {
	animator *faceAnimator
}

func newFacesDisplay() *FacesDisplay {
	return &FacesDisplay{animator: newFaceAnimator()}
}

// Render advances each face slot's timer and returns the composed frame. stats is unused.
func (d *FacesDisplay) Render(_ *stats.StatsSummary) [9][34]byte {
	d.animator.tick()
	return d.animator.frame()
}
