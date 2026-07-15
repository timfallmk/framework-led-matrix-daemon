package display

import (
	"testing"
	"time"
)

func TestFacePattern_ToTile_Encoding(t *testing.T) {
	fp := facePattern{
		holdFor: time.Second,
		rows: [faceTileHeight]string{
			"#........", // row 0: col 0 = full bright
			"o........", // row 1: col 0 = soft glow
			".........", // row 2: all off
			".........",
			".........",
			".........",
			".........",
			".........",
			".........",
			".........",
		},
	}
	tile := fp.toTile()

	if tile[0][0] != 255 {
		t.Errorf("toTile(): '#' at (col=0,row=0) = %d, want 255", tile[0][0])
	}
	if tile[0][1] != 160 {
		t.Errorf("toTile(): 'o' at (col=0,row=1) = %d, want 160", tile[0][1])
	}
	if tile[0][2] != 0 {
		t.Errorf("toTile(): '.' at (col=0,row=2) = %d, want 0", tile[0][2])
	}
	for col := 1; col < faceWidth; col++ {
		if tile[col][0] != 0 {
			t.Errorf("toTile(): col=%d row=0 = %d, want 0", col, tile[col][0])
		}
	}
}

func TestFaceAnimator_Frame_SlotOffsets(t *testing.T) {
	fa := newFaceAnimator()
	frame := fa.frame()

	gaps := [][2]int{{10, 11}, {22, 23}}
	for _, g := range gaps {
		for _, row := range g {
			for col := 0; col < faceWidth; col++ {
				if frame[col][row] != 0 {
					t.Errorf("frame(): gap row %d col %d = %d, want 0", row, col, frame[col][row])
				}
			}
		}
	}
}

func TestFaceAnimator_Frame_NoBoundsOverrun(t *testing.T) {
	fa := newFaceAnimator()

	for i := range fa.patterns {
		fa.slots[0] = i
		fa.slots[1] = i
		fa.slots[2] = i
		frame := fa.frame()

		_ = frame[faceWidth-1][33]
	}
}

func TestFaceAnimator_Frame_TileDataPlacedAtOffset(t *testing.T) {
	fa := newFaceAnimator()

	fa.slots = [3]int{0, 0, 0}
	tile := fa.tiles[0]
	frame := fa.frame()

	for slotIdx, offset := range slotOffsets {
		for col := 0; col < faceWidth; col++ {
			for row := 0; row < faceTileHeight; row++ {
				got := frame[col][offset+row]
				want := tile[col][row]
				if got != want {
					t.Errorf("slot %d: frame[%d][%d] = %d, want %d (tile[%d][%d])",
						slotIdx, col, offset+row, got, want, col, row)
				}
			}
		}
	}
}

func TestFaceAnimator_Tick_NoAdvanceBeforeHoldExpires(t *testing.T) {
	fa := newFaceAnimator()
	fa.slots = [3]int{0, 0, 0}
	now := time.Now()
	fa.switched = [3]time.Time{now, now, now}

	fa.tick()

	for i, s := range fa.slots {
		if s != 0 {
			t.Errorf("slot %d advanced before holdFor expired: got %d, want 0", i, s)
		}
	}
}

func TestFaceAnimator_Tick_AdvancesAfterHoldExpires(t *testing.T) {
	fa := newFaceAnimator()
	fa.slots = [3]int{0, 0, 0}
	past := time.Now().Add(-24 * time.Hour)
	fa.switched = [3]time.Time{past, past, past}

	fa.tick()

	for i, s := range fa.slots {
		if s != 1 {
			t.Errorf("slot %d did not advance after holdFor expired: got %d, want 1", i, s)
		}
	}
}

func TestFaceAnimator_Tick_CyclesWrapAround(t *testing.T) {
	fa := newFaceAnimator()
	last := len(fa.patterns) - 1
	fa.slots = [3]int{last, last, last}
	past := time.Now().Add(-24 * time.Hour)
	fa.switched = [3]time.Time{past, past, past}

	fa.tick()

	for i, s := range fa.slots {
		if s != 0 {
			t.Errorf("slot %d did not wrap from last to 0: got %d", i, s)
		}
	}
}
