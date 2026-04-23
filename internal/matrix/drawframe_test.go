package matrix

import (
	"fmt"
	"testing"
)

// newMDMWithMock builds a MultiDisplayManager backed by a single MockPort-wired Client.
func newMDMWithMock(name string) (*MultiDisplayManager, *MockPort) {
	mockPort := NewMockPort()
	client := &Client{port: mockPort}
	mc := &MultiClient{
		clients: map[string]*Client{name: client},
		config:  make(map[string]*SingleMatrixConfig),
	}
	return NewMultiDisplayManager(mc, "split"), mockPort
}

func TestDrawFrame_AllColumnsStaged(t *testing.T) {
	mdm, mockPort := newMDMWithMock("primary")

	var frame [9][34]byte
	for col := 0; col < 9; col++ {
		for row := 0; row < 34; row++ {
			frame[col][row] = byte(col*10 + row%10)
		}
	}

	if err := mdm.DrawFrame("primary", frame); err != nil {
		t.Fatalf("DrawFrame() unexpected error: %v", err)
	}

	// Each StageColumn write: [MagicByte1, MagicByte2, CmdStageCol, col, 34 pixels] = 38 bytes
	// FlushColumns write:     [MagicByte1, MagicByte2, CmdFlushCols]                =  3 bytes
	const wantLen = 9*38 + 3
	if got := len(mockPort.writeData); got != wantLen {
		t.Errorf("DrawFrame() wrote %d bytes, want %d", got, wantLen)
	}

	// Verify each column's magic header and column-index byte.
	for col := 0; col < 9; col++ {
		base := col * 38
		if mockPort.writeData[base] != MagicByte1 {
			t.Errorf("col %d byte[0] = %#x, want MagicByte1 %#x", col, mockPort.writeData[base], MagicByte1)
		}
		if mockPort.writeData[base+1] != MagicByte2 {
			t.Errorf("col %d byte[1] = %#x, want MagicByte2 %#x", col, mockPort.writeData[base+1], MagicByte2)
		}
		if mockPort.writeData[base+2] != CmdStageCol {
			t.Errorf("col %d byte[2] = %#x, want CmdStageCol %#x", col, mockPort.writeData[base+2], CmdStageCol)
		}
		if mockPort.writeData[base+3] != byte(col) {
			t.Errorf("col %d column-index byte = %d, want %d", col, mockPort.writeData[base+3], col)
		}
	}
}

func TestDrawFrame_FlushColumnsCalledOnce(t *testing.T) {
	mdm, mockPort := newMDMWithMock("primary")

	if err := mdm.DrawFrame("primary", [9][34]byte{}); err != nil {
		t.Fatalf("DrawFrame() unexpected error: %v", err)
	}

	// FlushColumns is a 3-byte command; it should appear exactly once, at the end.
	flushBase := 9 * 38
	if len(mockPort.writeData) < flushBase+3 {
		t.Fatalf("not enough bytes written: got %d", len(mockPort.writeData))
	}
	if mockPort.writeData[flushBase+2] != CmdFlushCols {
		t.Errorf("FlushColumns byte = %#x, want CmdFlushCols %#x", mockPort.writeData[flushBase+2], CmdFlushCols)
	}
}

func TestDrawFrame_PixelDataCorrect(t *testing.T) {
	mdm, mockPort := newMDMWithMock("primary")

	var frame [9][34]byte
	frame[3][7] = 0xAB

	if err := mdm.DrawFrame("primary", frame); err != nil {
		t.Fatalf("DrawFrame() unexpected error: %v", err)
	}

	// Col 3 packet starts at byte 3*38. Pixel bytes start at offset +4 (after header+col).
	col3PixelBase := 3*38 + 4
	if got := mockPort.writeData[col3PixelBase+7]; got != 0xAB {
		t.Errorf("pixel[col=3][row=7] = %#x, want 0xAB", got)
	}
}

func TestDrawFrame_UnknownMatrix(t *testing.T) {
	mdm, _ := newMDMWithMock("primary")

	err := mdm.DrawFrame("secondary", [9][34]byte{})
	if err == nil {
		t.Error("DrawFrame() with unknown matrix: expected error, got nil")
	}
}

func TestDrawFrame_StageColumnErrorPropagated(t *testing.T) {
	mdm, mockPort := newMDMWithMock("primary")
	mockPort.writeError = fmt.Errorf("port write failure")

	err := mdm.DrawFrame("primary", [9][34]byte{})
	if err == nil {
		t.Error("DrawFrame() should propagate StageColumn port error")
	}
}
