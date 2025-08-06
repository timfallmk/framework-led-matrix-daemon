package matrix

import (
	"reflect"
	"testing"
)

func TestNewCommand(t *testing.T) {
	tests := []struct {
		name     string
		id       byte
		params   []byte
		expected Command
	}{
		{
			name:     "command without parameters",
			id:       CmdVersion,
			params:   nil,
			expected: Command{ID: CmdVersion, Params: nil},
		},
		{
			name:     "command with single parameter",
			id:       CmdBrightness,
			params:   []byte{128},
			expected: Command{ID: CmdBrightness, Params: []byte{128}},
		},
		{
			name:     "command with multiple parameters",
			id:       CmdPattern,
			params:   []byte{PatternPercentage, 75},
			expected: Command{ID: CmdPattern, Params: []byte{PatternPercentage, 75}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NewCommand(tt.id, tt.params...)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NewCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCommandToBytes(t *testing.T) {
	tests := []struct {
		name     string
		command  Command
		expected []byte
	}{
		{
			name:     "version command",
			command:  Command{ID: CmdVersion, Params: nil},
			expected: []byte{MagicByte1, MagicByte2, CmdVersion},
		},
		{
			name:     "brightness command",
			command:  Command{ID: CmdBrightness, Params: []byte{128}},
			expected: []byte{MagicByte1, MagicByte2, CmdBrightness, 128},
		},
		{
			name:     "percentage pattern command",
			command:  Command{ID: CmdPattern, Params: []byte{PatternPercentage, 75}},
			expected: []byte{MagicByte1, MagicByte2, CmdPattern, PatternPercentage, 75},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.command.ToBytes()
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ToBytes() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBrightnessCommand(t *testing.T) {
	tests := []struct {
		name     string
		level    byte
		expected Command
	}{
		{"minimum brightness", 0, Command{ID: CmdBrightness, Params: []byte{0}}},
		{"medium brightness", 128, Command{ID: CmdBrightness, Params: []byte{128}}},
		{"maximum brightness", 255, Command{ID: CmdBrightness, Params: []byte{255}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BrightnessCommand(tt.level)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("BrightnessCommand(%d) = %v, want %v", tt.level, result, tt.expected)
			}
		})
	}
}

func TestPatternCommand(t *testing.T) {
	tests := []struct {
		name     string
		pattern  byte
		params   []byte
		expected Command
	}{
		{
			name:     "gradient pattern",
			pattern:  PatternGradient,
			params:   nil,
			expected: Command{ID: CmdPattern, Params: []byte{PatternGradient}},
		},
		{
			name:     "percentage pattern with value",
			pattern:  PatternPercentage,
			params:   []byte{50},
			expected: Command{ID: CmdPattern, Params: []byte{PatternPercentage, 50}},
		},
		{
			name:     "pattern with multiple parameters",
			pattern:  PatternPercentage,
			params:   []byte{75, 100},
			expected: Command{ID: CmdPattern, Params: []byte{PatternPercentage, 75, 100}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PatternCommand(tt.pattern, tt.params...)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("PatternCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestPercentageCommand(t *testing.T) {
	tests := []struct {
		name     string
		percent  byte
		expected Command
	}{
		{"0 percent", 0, Command{ID: CmdPattern, Params: []byte{PatternPercentage, 0}}},
		{"50 percent", 50, Command{ID: CmdPattern, Params: []byte{PatternPercentage, 50}}},
		{"100 percent", 100, Command{ID: CmdPattern, Params: []byte{PatternPercentage, 100}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PercentageCommand(tt.percent)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("PercentageCommand(%d) = %v, want %v", tt.percent, result, tt.expected)
			}
		})
	}
}

func TestGradientCommand(t *testing.T) {
	expected := Command{ID: CmdPattern, Params: []byte{PatternGradient}}
	result := GradientCommand()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("GradientCommand() = %v, want %v", result, expected)
	}
}

func TestZigZagCommand(t *testing.T) {
	expected := Command{ID: CmdPattern, Params: []byte{PatternZigZag}}
	result := ZigZagCommand()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ZigZagCommand() = %v, want %v", result, expected)
	}
}

func TestFullBrightCommand(t *testing.T) {
	expected := Command{ID: CmdPattern, Params: []byte{PatternFullBright}}
	result := FullBrightCommand()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("FullBrightCommand() = %v, want %v", result, expected)
	}
}

func TestAnimateCommand(t *testing.T) {
	tests := []struct {
		name     string
		enable   bool
		expected Command
	}{
		{"enable animation", true, Command{ID: CmdAnimate, Params: []byte{1}}},
		{"disable animation", false, Command{ID: CmdAnimate, Params: []byte{0}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AnimateCommand(tt.enable)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("AnimateCommand(%t) = %v, want %v", tt.enable, result, tt.expected)
			}
		})
	}
}

func TestDrawBWCommand(t *testing.T) {
	pixels := [39]byte{}
	for i := range pixels {
		pixels[i] = byte(i)
	}

	result := DrawBWCommand(pixels)

	if result.ID != CmdDrawBW {
		t.Errorf("DrawBWCommand() ID = %d, want %d", result.ID, CmdDrawBW)
	}

	if len(result.Params) != 39 {
		t.Errorf("DrawBWCommand() params length = %d, want 39", len(result.Params))
	}

	for i, param := range result.Params {
		if param != byte(i) {
			t.Errorf("DrawBWCommand() param[%d] = %d, want %d", i, param, i)
		}
	}
}

func TestStageColCommand(t *testing.T) {
	col := byte(5)
	pixels := [34]byte{}
	for i := range pixels {
		pixels[i] = byte(i + 10)
	}

	result := StageColCommand(col, pixels)

	if result.ID != CmdStageCol {
		t.Errorf("StageColCommand() ID = %d, want %d", result.ID, CmdStageCol)
	}

	if len(result.Params) != 35 { // 1 for column + 34 for pixels
		t.Errorf("StageColCommand() params length = %d, want 35", len(result.Params))
	}

	if result.Params[0] != col {
		t.Errorf("StageColCommand() column = %d, want %d", result.Params[0], col)
	}

	for i, param := range result.Params[1:] {
		expected := byte(i + 10)
		if param != expected {
			t.Errorf("StageColCommand() pixel[%d] = %d, want %d", i, param, expected)
		}
	}
}

func TestFlushColsCommand(t *testing.T) {
	expected := Command{ID: CmdFlushCols, Params: nil}
	result := FlushColsCommand()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("FlushColsCommand() = %v, want %v", result, expected)
	}
}

func TestVersionCommand(t *testing.T) {
	expected := Command{ID: CmdVersion, Params: nil}
	result := VersionCommand()

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("VersionCommand() = %v, want %v", result, expected)
	}
}

func TestMagicBytes(t *testing.T) {
	if MagicByte1 != 0x32 {
		t.Errorf("MagicByte1 = 0x%02X, want 0x32", MagicByte1)
	}

	if MagicByte2 != 0xAC {
		t.Errorf("MagicByte2 = 0x%02X, want 0xAC", MagicByte2)
	}
}

func TestCommandConstants(t *testing.T) {
	expectedCommands := map[string]byte{
		"CmdBrightness": 0x00,
		"CmdPattern":    0x01,
		"CmdAnimate":    0x04,
		"CmdDrawBW":     0x06,
		"CmdStageCol":   0x07,
		"CmdFlushCols":  0x08,
		"CmdVersion":    0x20,
	}

	actualCommands := map[string]byte{
		"CmdBrightness": CmdBrightness,
		"CmdPattern":    CmdPattern,
		"CmdAnimate":    CmdAnimate,
		"CmdDrawBW":     CmdDrawBW,
		"CmdStageCol":   CmdStageCol,
		"CmdFlushCols":  CmdFlushCols,
		"CmdVersion":    CmdVersion,
	}

	for name, expected := range expectedCommands {
		if actual := actualCommands[name]; actual != expected {
			t.Errorf("%s = 0x%02X, want 0x%02X", name, actual, expected)
		}
	}
}

func TestPatternConstants(t *testing.T) {
	expectedPatterns := map[string]byte{
		"PatternPercentage": 0x00,
		"PatternGradient":   0x01,
		"PatternZigZag":     0x04,
		"PatternFullBright": 0x05,
	}

	actualPatterns := map[string]byte{
		"PatternPercentage": PatternPercentage,
		"PatternGradient":   PatternGradient,
		"PatternZigZag":     PatternZigZag,
		"PatternFullBright": PatternFullBright,
	}

	for name, expected := range expectedPatterns {
		if actual := actualPatterns[name]; actual != expected {
			t.Errorf("%s = 0x%02X, want 0x%02X", name, actual, expected)
		}
	}
}

// Benchmark tests
func BenchmarkCommandToBytes(b *testing.B) {
	cmd := Command{ID: CmdPattern, Params: []byte{PatternPercentage, 50}}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cmd.ToBytes()
	}
}

func BenchmarkPercentageCommand(b *testing.B) {
	for i := 0; i < b.N; i++ {
		PercentageCommand(50)
	}
}

func BenchmarkDrawBWCommand(b *testing.B) {
	pixels := [39]byte{}
	for i := range pixels {
		pixels[i] = byte(i)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		DrawBWCommand(pixels)
	}
}
