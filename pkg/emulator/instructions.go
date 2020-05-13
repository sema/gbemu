package emulator

import (
	"fmt"
	"strings"
)

type instruction struct {
	Opcode   string
	Mnemonic string
	// Size of instruction in bytes (1 byte opcode + operands)
	Size     uint16
	Cycles   []int
	Operands []operand
	Flags    flags

	// TODO flags instruction as unsupported temporarily as we expand codegen
	Todo string
}

type operand struct {
	Name string
	Type operandType

	Ref           string
	RefRegister8  register8
	RefRegister16 register16
	RefFlag       flag
	RefFlagNegate bool
	RefConst8     uint8

	IncrementReg16 bool
	DecrementReg16 bool
}

type flags struct {
	Z string
	N string
	H string
	C string
}

type operandType int

// Operands for instructions
//
// TODO complete docs as I gain insight into the types
//
// The FooPtr variants of the operands are similar to their Foo counterpart,
// with the value of Foo interpreted as a pointer into the memory space. Any
// reads/writes to this operand are done on the dereferenced pointer.
const (
	// operandD8 is a 8bit value immediately following the opcode (i.e. PC+1)
	operandD8 operandType = iota

	// operandD16 is a 16bit value immediately following the opcode (i.e. PC+1 and PC+2)
	operandD16

	// operandA8 is a 8bit offset to the 0xFF00 address, resulting in the address 0xFF00+A8
	operandA8

	// operandA8Ptr is the pointer variant of operandA8
	operandA8Ptr

	// operandA16 ??
	operandA16

	// operandA16Ptr is the pointer variant of operandA16
	operandA16Ptr

	// operandR8 is a signed 8bit value.
	operandR8

	// operandFlag is a CPU flag (see cpu.go).
	operandFlag

	// operandReg8 is a 8bit register (see cpu.go).
	// The exact register for an operand of this type is stored in RefRegister8.
	operandReg8

	// operandReg8Ptr is the pointer variant of operandReg8. In this mode, the value
	// of operandReg8 is an offset to 0xFF00 when calculating the target memory address.
	operandReg8Ptr

	// operandReg16 is a 16bit register (see cpu.go).
	// The exact register for an operand of this type is stored in RefRegister16.
	operandReg16

	// operandReg16Ptr is the pointer varint of operandReg16
	operandReg16Ptr

	// operandConst8 is a static 8bit value associated with the opcode.
	operandConst8
)

var operandTypeNames = map[operandType]string{
	operandD8:       "d8",
	operandD16:      "d16",
	operandA8:       "a8",
	operandA8Ptr:    "a8ptr",
	operandA16:      "a16",
	operandA16Ptr:   "a16ptr",
	operandR8:       "r8",
	operandFlag:     "flag",
	operandReg8:     "reg8",
	operandReg8Ptr:  "reg8ptr",
	operandReg16:    "reg16",
	operandReg16Ptr: "reg16ptr",
	operandConst8:   "const8",
}

func (o operandType) String() string {
	name, ok := operandTypeNames[o]
	if !ok {
		panic(fmt.Sprintf("unable to determine name of operand (%d)", o))
	}

	return name
}

func (inst instruction) String() string {
	var operandStrs []string
	for _, op := range inst.Operands {
		operandStrs = append(operandStrs, fmt.Sprintf("%-5s", op.Name))
	}

	return fmt.Sprintf("[%5s] %-6s %s", inst.Opcode, inst.Mnemonic, strings.Join(operandStrs, " "))
}
