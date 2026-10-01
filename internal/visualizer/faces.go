package visualizer

import "time"

const (
	faceWidth      = 9  // physical matrix columns
	faceTileHeight = 10 // rows per face tile
)

// facePattern describes one compact facial expression fitting in a 9×10 tile.
// rows[r] is a 9-character string; '#'=255, 'o'=160 (soft), '.'=0.
type facePattern struct {
	rows    [faceTileHeight]string
	holdFor time.Duration
}

// toTile converts the row-major string pattern to a column-major [9][10]byte tile.
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

// Each face is a 9-wide × 10-tall pixel tile.
// Layout within the tile:
//   rows 0-1  : top margin / brows
//   rows 2-3  : eyes
//   row  4    : gap
//   rows 5-6  : mouth
//   rows 7-9  : bottom margin
var facePatterns = []facePattern{
	// 0 — Kawaii Happy (^‿^)
	{holdFor: 6 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".........", // 1
		".##...##.", // 2 eyes
		".##...##.", // 3
		".........", // 4
		"..#...#..", // 5 smile corners (high)
		"...###...", // 6 smile centre (low) → arc up = smile
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 1 — Kawaii UwU (ᵕ̈)
	{holdFor: 5 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".........", // 1
		".#.#.#.#.", // 2 U-eyes open tops
		".###.###.", // 3 U-eyes solid bottoms
		".........", // 4
		"...#.#...", // 5 ω mouth humps
		"..#...#..", // 6 ω mouth outer dips
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 2 — Kawaii Sleepy (-ᴗ-)
	{holdFor: 10 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".........", // 1
		".........", // 2
		".##...##.", // 3 flat single-row half-closed eyes
		".........", // 4
		".........", // 5
		"...###...", // 6 flat mouth
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 3 — Kawaii Surprised (O_O)
	{holdFor: 4 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		"..#...#..", // 1 hollow O eyes top
		".#.#.#.#.", // 2 hollow O eyes sides
		"..#...#..", // 3 hollow O eyes bottom
		".........", // 4
		"....#....", // 5 small O mouth top
		"...#.#...", // 6 small O mouth sides
		"....#....", // 7 small O mouth bottom
		".........", // 8
		".........", // 9
	}},

	// 4 — Anime Neutral (-_-)
	{holdFor: 5 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".........", // 1
		".........", // 2
		".#######.", // 3 single long flat eyes
		".........", // 4
		".........", // 5
		"...###...", // 6 flat mouth
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 5 — Anime Sad (T_T) with single tear
	{holdFor: 7 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".........", // 1
		".##...##.", // 2 eyes
		".##...##.", // 3
		".#.......", // 4 tear drop (left)
		"...###...", // 5 frown top (centre high)
		"..#...#..", // 6 frown corners (low) → arc down = frown
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 6 — Punk Smirk (one eye wink, mouth raised on right)
	{holdFor: 5 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".........", // 1
		".##......", // 2 left eye top only
		".##...##.", // 3 left eye bottom + right wink (single row)
		".........", // 4
		".....##..", // 5 smirk right side raised
		"....###..", // 6 smirk body (leans right)
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 7 — Punk Mean (ò_ó) — angry brows, squint, tight frown
	{holdFor: 5 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".#.....#.", // 1 angry brow outer ends
		"..#...#..", // 2 angry brow inner (diagonal slant toward centre)
		".##...##.", // 3 squinting eyes
		".........", // 4
		"...###...", // 5 tight frown top (centre high)
		"..#...#..", // 6 tight frown corners (low) → frown
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 8 — Don't Care (._.)
	{holdFor: 8 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".........", // 1
		".........", // 2
		"...#.#...", // 3 tiny dot eyes
		".........", // 4
		".........", // 5
		"...###...", // 6 flat mouth
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 9 — Dead Inside (x_x)
	{holdFor: 7 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		".#.#.#.#.", // 1 X eyes upper diagonals
		"..#...#..", // 2 X eyes centres
		".#.#.#.#.", // 3 X eyes lower diagonals
		".........", // 4
		"..#####..", // 5 flat mouth
		".........", // 6
		".........", // 7
		".........", // 8
		".........", // 9
	}},

	// 10 — Genki / Energetic (^u^)
	{holdFor: 4 * time.Second, rows: [faceTileHeight]string{
		".........", // 0
		"..#...#..", // 1 ^ eye peaks
		".#.#.#.#.", // 2 ^ eye wings
		"o.......o", // 3 blush cheeks
		".........", // 4
		"..#...#..", // 5 big smile corners
		"...###...", // 6 smile centre
		".#.....#.", // 7 wide outer smile
		".........", // 8
		".........", // 9
	}},
}

// slotOffsets defines where each of the three face slots starts on the 34-row matrix.
// Layout: [0-9] face, [10-11] dark gap, [12-21] face, [22-23] dark gap, [24-33] face.
var slotOffsets = [3]int{0, 12, 24}

// FaceAnimator runs three independently cycling face slots on the secondary matrix.
type FaceAnimator struct {
	tiles    [][faceWidth][faceTileHeight]byte // precomputed per-face tiles
	patterns []facePattern
	slots    [3]int        // which face pattern each slot currently shows
	switched [3]time.Time  // when each slot last changed faces
}

// NewFaceAnimator precomputes all face tiles and staggers the three slots so they
// start on different expressions and cycle out of phase.
func NewFaceAnimator() *FaceAnimator {
	fa := &FaceAnimator{
		patterns: facePatterns,
		tiles:    make([][faceWidth][faceTileHeight]byte, len(facePatterns)),
	}
	for i := range facePatterns {
		fa.tiles[i] = facePatterns[i].toTile()
	}

	n := len(facePatterns)
	fa.slots = [3]int{0, n / 3, 2 * n / 3}

	now := time.Now()
	fa.switched = [3]time.Time{now, now, now}

	return fa
}

// Tick advances each slot independently when its current face's hold duration expires.
func (fa *FaceAnimator) Tick() {
	now := time.Now()
	for i := range fa.slots {
		if now.Sub(fa.switched[i]) >= fa.patterns[fa.slots[i]].holdFor {
			fa.slots[i] = (fa.slots[i] + 1) % len(fa.patterns)
			fa.switched[i] = now
		}
	}
}

// Frame composes the three active face tiles into a single [9][34]byte matrix frame,
// with 2-row dark separators between slots.
func (fa *FaceAnimator) Frame() [faceWidth][34]byte {
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
