package matrix

// Protocol magic bytes for Framework LED matrix communication.
const (
	MagicByte1 = 0x32
	MagicByte2 = 0xAC
)

// Command IDs for LED matrix operations.
const (
	CmdBrightness = 0x00
	CmdPattern    = 0x01
	CmdAnimate    = 0x04
	CmdDrawBW     = 0x06
	CmdStageCol   = 0x07
	CmdFlushCols  = 0x08
	CmdVersion    = 0x20
)

// Pattern types for LED matrix display modes.
const (
	PatternPercentage = 0x00
	PatternGradient   = 0x01
	PatternZigZag     = 0x04
	PatternFullBright = 0x05
)

// Command represents a LED matrix command with ID and parameters.
type Command struct {
	Params []byte
	ID     byte
}

// NewCommand creates a new command with the specified ID and parameters.
func NewCommand(id byte, params ...byte) Command {
	return Command{
		ID:     id,
		Params: params,
	}
}

// ToBytes converts the command to its byte representation for transmission.
func (c Command) ToBytes() []byte {
	result := []byte{MagicByte1, MagicByte2, c.ID}
	result = append(result, c.Params...)

	return result
}

// BrightnessCommand creates a command to set LED matrix brightness (0-255).
func BrightnessCommand(level byte) Command {
	return NewCommand(CmdBrightness, level)
}

// PatternCommand creates a command to display a specific pattern with parameters.
func PatternCommand(pattern byte, params ...byte) Command {
	args := []byte{pattern}
	args = append(args, params...)

	return NewCommand(CmdPattern, args...)
}

// PercentageCommand creates a command to display a percentage bar (0-100).
func PercentageCommand(percent byte) Command {
	return PatternCommand(PatternPercentage, percent)
}

// GradientCommand creates a command to display a gradient pattern.
func GradientCommand() Command {
	return PatternCommand(PatternGradient)
}

// ZigZagCommand creates a command to display a zigzag pattern.
func ZigZagCommand() Command {
	return PatternCommand(PatternZigZag)
}

// FullBrightCommand creates a command to illuminate all LEDs at maximum brightness.
func FullBrightCommand() Command {
	return PatternCommand(PatternFullBright)
}

// AnimateCommand creates a command to enable or disable animation effects.
func AnimateCommand(enable bool) Command {
	var param byte
	if enable {
		param = 1
	}

	return NewCommand(CmdAnimate, param)
}

// DrawBWCommand creates a command to draw a black and white bitmap.
func DrawBWCommand(pixels [39]byte) Command {
	return NewCommand(CmdDrawBW, pixels[:]...)
}

// StageColCommand creates a command to stage a column of pixels for display.
func StageColCommand(col byte, pixels [34]byte) Command {
	params := []byte{col}
	params = append(params, pixels[:]...)

	return NewCommand(CmdStageCol, params...)
}

// FlushColsCommand creates a command to flush all staged columns to the display.
func FlushColsCommand() Command {
	return NewCommand(CmdFlushCols)
}

// VersionCommand creates a command to request the firmware version.
func VersionCommand() Command {
	return NewCommand(CmdVersion)
}
