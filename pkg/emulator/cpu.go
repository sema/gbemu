package emulator

import "fmt"

type register8 uint
type register16 uint

const (
	registerA register8 = 1
	registerB           = 3
	registerC           = 2
	registerD           = 5
	registerE           = 4
	registerH           = 7
	registerL           = 6
)

const (
	registerBC register16 = 2
	registerDE            = 4
	registerHL            = 6
	registerSP            = 8
)

var register8Names = map[register8]string{
	registerA: "A",
	registerB: "B",
	registerC: "C",
	registerD: "D",
	registerE: "E",
	registerH: "H",
	registerL: "L",
}

var register16Names = map[register16]string{
	registerBC: "BC",
	registerDE: "DE",
	registerHL: "HL",
	registerSP: "SP",
}

func (r register8) String() string {
	name, ok := register8Names[r]
	if !ok {
		panic(fmt.Sprintf("unable to determine name of register (%d)", r))
	}

	return name
}

func (r register16) String() string {
	name, ok := register16Names[r]
	if !ok {
		panic(fmt.Sprintf("unable to determine name of register (%d)", r))
	}

	return name
}

type registers struct {
	// data contains the common registers A-E, H, L at predefined offsets (see registerX constants)
	//
	// The 8 bit registers may also be referenced in pairs as the 16 bit registers BC, DE, and HL
	// (see registerXY constants). In this mode, the 8 bit registers are ordered using little-endian
	// (lowest order byte first).
	//
	// Structure:
	// 16bit Hi   Lo   Comment
	// A     A    -    Lower bits used for flags
	// BC    B    C
	// DE    D    E
	// HL    H    L
	// SP    -    -    Stack pointer. Can't be addressed in 8bit
	data []byte

	flagZero      bool // zf
	flagAddSub    bool // n
	flagHalfCarry bool // h
	flagCarry     bool // cy

}

func newRegisters() registers {
	return registers{
		data: make([]byte, 10),
	}
}
