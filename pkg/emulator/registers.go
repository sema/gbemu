package emulator

import (
	"encoding/binary"
	"fmt"
)

type register8 uint
type register16 uint
type flag uint

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
	registerAF register16 = 0
	registerBC            = 2
	registerDE            = 4
	registerHL            = 6
	registerSP            = 8
)

const (
	flagZ flag = 0 // Zero
	flagN      = 1 // Subtract
	flagH      = 2 // HalfCarry
	flagC      = 3 // Carry
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

var flagNames = map[flag]string{
	flagZ: "Z",
	flagN: "N",
	flagH: "H",
	flagC: "C",
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

func (f flag) String() string {
	name, ok := flagNames[f]
	if !ok {
		panic(fmt.Sprintf("unable to determine name of flag (%d)", f))
	}

	return name
}

type registers struct {
	// data contains the common registers A-E, H, L at predefined offsets (see registerX constants)
	//
	// The 8 bit registers may also be referenced in pairs as the 16 bit registers AF, BC, DE, and HL
	// (see registerXY constants). In this mode, the 8 bit registers are ordered using little-endian
	// (lowest order byte first).
	//
	// Notice the AF register is special, as the "F" is used for flags, and is otherwise not directly
	// addressable in 8bit mode.
	//
	// Structure:
	// 16bit Hi   Lo   Comment
	// AF    A    -    Lower bits used for flags
	// BC    B    C
	// DE    D    E
	// HL    H    L
	// SP    -    -    Stack pointer. Can't be addressed in 8bit
	data []byte
}

func newRegisters() *registers {
	return &registers{
		data: make([]byte, 10),
	}
}

func (r *registers) Read16(register register16) uint16 {
	return toAddress(r.data[register : register+2])
}

func (r *registers) Write16(register register16, v uint16) {
	binary.LittleEndian.PutUint16(r.data[register:register+2], v)
}

func (r *registers) Read1(flag flag) bool {
	return r.data[0]&(1<<flag) > 0
}

func (r *registers) Write1(flag flag, v bool) {
	if v {
		// Example [flags] ORed 00100000 -> sets 3rd bit to 1
		r.data[0] |= (1 << flag)
	} else {
		// Example [flags] ANDed 11011111 (negated)  -> forces 3rd bit to 0
		r.data[0] &= ^(1 << flag)
	}
}
