package matrix

const (
	MagicByte1 = 0x32
	MagicByte2 = 0xAC
)

const (
	CmdBrightness = 0x00
	CmdPattern    = 0x01
	CmdAnimate    = 0x04
	CmdDrawBW     = 0x06
	CmdStageCol   = 0x07
	CmdFlushCols  = 0x08
	CmdVersion    = 0x20
)

const (
	PatternPercentage = 0x00
	PatternGradient   = 0x01
	PatternZigZag     = 0x04
	PatternFullBright = 0x05
)

type Command struct {
	Params []byte
	ID     byte
}

func NewCommand(id byte, params ...byte) Command {
	return Command{
		ID:     id,
		Params: params,
	}
}

func (c Command) ToBytes() []byte {
	result := []byte{MagicByte1, MagicByte2, c.ID}
	result = append(result, c.Params...)

	return result
}

func BrightnessCommand(level byte) Command {
	return NewCommand(CmdBrightness, level)
}

func PatternCommand(pattern byte, params ...byte) Command {
	args := []byte{pattern}
	args = append(args, params...)

	return NewCommand(CmdPattern, args...)
}

func PercentageCommand(percent byte) Command {
	return PatternCommand(PatternPercentage, percent)
}

func GradientCommand() Command {
	return PatternCommand(PatternGradient)
}

func ZigZagCommand() Command {
	return PatternCommand(PatternZigZag)
}

func FullBrightCommand() Command {
	return PatternCommand(PatternFullBright)
}

func AnimateCommand(enable bool) Command {
	var param byte
	if enable {
		param = 1
	}

	return NewCommand(CmdAnimate, param)
}

func DrawBWCommand(pixels [39]byte) Command {
	return NewCommand(CmdDrawBW, pixels[:]...)
}

func StageColCommand(col byte, pixels [34]byte) Command {
	params := []byte{col}
	params = append(params, pixels[:]...)

	return NewCommand(CmdStageCol, params...)
}

func FlushColsCommand() Command {
	return NewCommand(CmdFlushCols)
}

func VersionCommand() Command {
	return NewCommand(CmdVersion)
}
