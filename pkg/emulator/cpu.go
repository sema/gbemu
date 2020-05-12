package emulator

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
)

type cpu struct {
	memory         *memory
	registers      *registers
	programCounter uint16
	powerOn        bool
}

func newCPU(memory *memory, registers *registers) *cpu {
	return &cpu{
		memory:         memory,
		registers:      registers,
		programCounter: 0x0100,
		powerOn:        true,
	}
}

func (c *cpu) cycle() {
	opcode := c.memory.data[c.programCounter]
	inst := instructions[opcode]

	log.Printf("Execute %#04x %-30s %s", c.programCounter, inst.String(), c.reprOperandValues(inst))

	// TODO remove when we support everything
	if inst.Todo != "" {
		notImplemented("Unsupported instruction [%s] %s called: %s", inst.Opcode, inst.Mnemonic, inst.Todo)
	}

	autoIncrementPC := true

	switch inst.Mnemonic {
	case "ILLEGAL":
		log.Panicf("Illegal instruction [%s] called", inst.Mnemonic)
	case "NOP":
		// Intentionally left blank
	case "LD8":
		// LD8 $TARGET $VALUE; $TARGET=$VALUE
		v := c.read8(inst.Operands[1])
		c.write8(inst.Operands[0], v)
	case "LD16":
		// LD16 $TARGET $VALUE; $TARGET=$VALUE
		v := c.read16(inst.Operands[1])
		c.write16(inst.Operands[0], v)
	case "INC8":
		// INC8 $OP; $OP++
		v := c.read8(inst.Operands[0]) + 1
		c.write8(inst.Operands[0], v)
		c.registers.Write1(flagZ, v == 0)
		c.registers.Write1(flagN, false)
		lowerHalfInOverflowPosition := v&0b00001111 == 0
		c.registers.Write1(flagH, lowerHalfInOverflowPosition)
	case "DEC8":
		// DEC8 $OP; $OP--
		v := c.read8(inst.Operands[0]) - 1
		c.write8(inst.Operands[0], v)
		c.registers.Write1(flagZ, v == 0)
		c.registers.Write1(flagN, true)
		lowerHalfInOverflowPosition := v&0b00001111 == 0 // TODO this is almost certainly incorrect
		c.registers.Write1(flagH, lowerHalfInOverflowPosition)
	case "JP":
		// JP $TO [$CONDITION]; PC=$TO
		jump := true
		if len(inst.Operands) > 1 {
			jump = c.isFlagSet(inst.Operands[1])
		}

		if jump {
			assertOperandType(inst.Operands[0], operandA16, operandReg16)
			addr := c.read16(inst.Operands[0])
			c.programCounter = addr
			autoIncrementPC = false
		}
	case "JR":
		// JR $OFFSET [$CONDITION]; PC=PC+-$OFFSET
		jump := true
		if len(inst.Operands) > 1 {
			jump = c.isFlagSet(inst.Operands[1])
		}

		if jump {
			assertOperandType(inst.Operands[0], operandR8)
			offset := c.read8signed(inst.Operands[0])
			c.programCounter = offsetAddress(c.programCounter, offset)
			autoIncrementPC = false
		}
	case "STOP":
		// STOP; stop running
		log.Println("POWER OFF")
		c.powerOn = false
	default:
		notImplemented("instruction not implemented yet")
	}

	// Some instructions automatically increment/decrement values after they complete
	for _, op := range inst.Operands {
		if op.IncrementReg16 || op.DecrementReg16 {
			assertOperandType(op, operandReg16, operandReg16Ptr)
			address := c.registers.Read16(op.RefRegister16)
			if op.IncrementReg16 {
				address++
			} else {
				address--
			}
			c.registers.Write16(op.RefRegister16, address)
		}
	}

	if autoIncrementPC {
		c.programCounter += inst.Size
	}
}

func (c *cpu) read16(op operand) uint16 {
	switch op.Type {
	case operandD16:
		// TODO little endian conversion here may be wrong
		return c.memory.Read16(c.programCounter + 1)
	case operandA16:
		return c.memory.Read16(c.programCounter + 1)
	case operandReg16:
		return c.registers.Read16(op.RefRegister16)
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 16bit value", op.Type.String())
		return 0
	}
}

func (c *cpu) write16(op operand, v uint16) {
	switch op.Type {
	case operandReg16:
		c.registers.Write16(op.RefRegister16, v)
	default:
		log.Panicf("unexpected operand (%s) encountered while writing 16bit value", op.Type.String())
	}
}

func (c *cpu) read8(op operand) byte {
	switch op.Type {
	case operandD8:
		return c.memory.data[c.programCounter+1]
	case operandReg8:
		return c.registers.data[op.RefRegister8]
	case operandReg16Ptr:
		address := c.registers.Read16(op.RefRegister16)
		return c.memory.data[address]
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 8bit value", op.Type.String())
		return 0
	}
}

func (c *cpu) read8signed(op operand) int8 {
	switch op.Type {
	case operandR8:
		return int8(c.memory.data[c.programCounter+1])
	default:
		log.Panicf("unexpected operand (%s) encountered while reading signed 8bit value", op.Type.String())
		return 0
	}
}

func (c *cpu) write8(op operand, v byte) {
	switch op.Type {
	case operandReg8:
		c.registers.data[op.RefRegister8] = v
	case operandReg16Ptr:
		data := c.registers.data[op.RefRegister16 : op.RefRegister16+2]
		address := toAddress(data)
		c.memory.data[address] = v
	default:
		log.Panicf("unexpected operand (%s) encountered while writing 8bit value", op.Type.String())
	}
}

func (c *cpu) reprOperandValues(inst instruction) string {
	var builder strings.Builder
	for _, op := range inst.Operands {
		var value string
		switch op.Type {
		case operandA16, operandD16, operandReg16:
			value = fmt.Sprintf("%#04x", c.read16(op))
		case operandD8, operandReg8, operandReg16Ptr:
			value = fmt.Sprintf("%#02x", c.read8(op))
		case operandFlag:
			value = fmt.Sprintf("%t", c.isFlagSet(op))
		case operandR8:
			value = fmt.Sprintf("%d", c.read8signed(op))
		}
		if value != "" {
			fmt.Fprintf(&builder, "%-5s= %6s  ", op.Name, value)
		}
	}

	return builder.String()
}

func (c *cpu) isFlagSet(op operand) bool {
	assertOperandType(op, operandFlag)
	condition := c.registers.Read1(op.RefFlag)
	if op.RefFlagNegate {
		condition = !condition
	}
	return condition
}

func notImplemented(msg string, args ...interface{}) {
	log.Printf(msg, args...)
	os.Exit(1)
}

func toAddress(bytes []byte) uint16 {
	return binary.LittleEndian.Uint16(bytes)
}

func assertOperandType(op operand, expected ...operandType) {
	for _, e := range expected {
		if op.Type == e {
			return
		}
	}

	log.Panicf("unexpected operand type (%s) of operand: expected one of type %s", op.Type.String(), expected)
}
