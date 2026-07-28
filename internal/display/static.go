package display

import "github.com/timfallmk/framework-led-matrix-daemon/internal/stats"

func init() {
	Register("off", func() MatrixDisplay { return staticDisplay(blankFrame()) })
	Register("all-on", func() MatrixDisplay { return staticDisplay(solidFrame(255)) })
	Register("half-bright", func() MatrixDisplay { return staticDisplay(solidFrame(128)) })
	Register("checkerboard", func() MatrixDisplay { return staticDisplay(checkerboardFrame()) })
	Register("border", func() MatrixDisplay { return staticDisplay(borderFrame()) })
}

// staticDisplay is a MatrixDisplay that always returns the same precomputed frame.
type staticDisplay [9][34]byte

func (d staticDisplay) Render(_ *stats.StatsSummary) [9][34]byte {
	return [9][34]byte(d)
}

func blankFrame() [9][34]byte {
	return [9][34]byte{}
}

func solidFrame(brightness byte) [9][34]byte {
	var f [9][34]byte
	for col := range f {
		for row := range f[col] {
			f[col][row] = brightness
		}
	}
	return f
}

func checkerboardFrame() [9][34]byte {
	var f [9][34]byte
	for col := 0; col < 9; col++ {
		for row := 0; row < 34; row++ {
			if (col+row)%2 == 0 {
				f[col][row] = 255
			}
		}
	}
	return f
}

func borderFrame() [9][34]byte {
	var f [9][34]byte
	for col := 0; col < 9; col++ {
		for row := 0; row < 34; row++ {
			if col == 0 || col == 8 || row == 0 || row == 33 {
				f[col][row] = 255
			}
		}
	}
	return f
}
