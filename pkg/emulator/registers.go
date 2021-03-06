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
	flagZ flag = 7 // Zero
	flagN      = 6 // Subtract
	flagH      = 5 // HalfCarry
	flagC      = 4 // Carry
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
	// Data contains the common registers A-E, H, L at predefined offsets (see registerX constants)
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
	//
	// The stack pointer is usually initialized to point to 0xFFFE (second to last address in the
	// memory space, just before the interrupt registers at 0xFFFF), and grows "down" towards
	// lower addresses.
	Data []byte
}

func newRegisters() *registers {
	return &registers{
		Data: make([]byte, 10),
	}
}

func (r *registers) Read16(register register16) uint16 {
	return toAddress(r.Data[register : register+2])
}

func (r *registers) Write16(register register16, v uint16) {
	if register == registerAF {
		// Force lower 4 bits of the flags register to always be zero
		//
		// Unable to find this specified in the spec, but this semantics
		// is explicitly tested in Blargg's test ROMs.
		v = v & 0xFFF0
	}

	binary.LittleEndian.PutUint16(r.Data[register:register+2], v)
}

func (r *registers) Read1(flag flag) bool {
	return readBitN(r.Data[0], uint8(flag))
}

func (r *registers) Write1(flag flag, v bool) {
	r.Data[0] = writeBitN(r.Data[0], uint8(flag), v)
}
