package emulator

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
)

type cpu struct {
	Memory         *memory
	Registers      *registers
	ProgramCounter uint16
	PowerOn        bool
}

func newCPU(memory *memory, registers *registers) *cpu {
	return &cpu{
		Memory:         memory,
		Registers:      registers,
		ProgramCounter: 0x0100,
		PowerOn:        true,
	}
}

func (c *cpu) cycle() {
	opcode := c.Memory.Data[c.ProgramCounter]
	inst := instructions[opcode]
	if opcode == 0xCB {
		// 0xCB is a prefix for a 2-byte opcode. Lookup the 2nd byte.
		opcode = c.Memory.Data[c.ProgramCounter+1]
		inst = cbInstructions[opcode]
	}
	c.ProgramCounter += inst.Size

	log.Printf("Execute %#04x %-30s %s", c.ProgramCounter-inst.Size, inst.String(), c.reprOperandValues(inst))

	// TODO remove when we support everything
	if inst.Todo != "" {
		notImplemented("Unsupported instruction [%s] %s called: %s", inst.Opcode, inst.Mnemonic, inst.Todo)
	}

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
		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, false)
		// TODO calculate correctly
		//lowerHalfInOverflowPosition := v&0b00001111 == 0
		//c.Registers.Write1(flagH, lowerHalfInOverflowPosition)
	case "DEC8":
		// DEC8 $OP; $OP--
		v := c.read8(inst.Operands[0]) - 1
		c.write8(inst.Operands[0], v)
		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, true)
		// TODO calculate correctly
		//lowerHalfInOverflowPosition := v&0b00001111 == 0 // TODO this is almost certainly incorrect
		//c.Registers.Write1(flagH, lowerHalfInOverflowPosition)
	case "JP":
		// JP $TO [$CONDITION]; PC=$TO
		jump := true
		if len(inst.Operands) > 1 {
			jump = c.isFlagSet(inst.Operands[1])
		}

		if jump {
			assertOperandType(inst.Operands[0], operandA16, operandReg16)
			addr := c.read16(inst.Operands[0])
			c.ProgramCounter = addr
		}
	case "JR":
		// JR $OFFSET [$CONDITION]; PC=PC+-$OFFSET
		jump := true
		if len(inst.Operands) > 1 {
			assertOperandType(inst.Operands[1], operandFlag)
			jump = c.isFlagSet(inst.Operands[1])
		}

		if jump {
			assertOperandType(inst.Operands[0], operandR8)
			offset := c.read8signed(inst.Operands[0])
			c.ProgramCounter = offsetAddress(c.ProgramCounter, offset)
		}
	case "XOR":
		// XOR $A $X; $A=$A^$X
		assertOperandType(inst.Operands[0], operandReg8)
		assertOperandType(inst.Operands[1], operandReg8, operandReg16Ptr, operandD8)

		v := c.read8(inst.Operands[0]) ^ c.read8(inst.Operands[1])
		c.write8(inst.Operands[0], v)

		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, false)
		c.Registers.Write1(flagH, false)
		c.Registers.Write1(flagC, false)
	case "AND":
		// AND $A $X; $A=$A&$X
		assertOperandType(inst.Operands[0], operandReg8)
		assertOperandType(inst.Operands[1], operandReg8, operandReg16Ptr, operandD8)

		v := c.read8(inst.Operands[0]) & c.read8(inst.Operands[1])
		c.write8(inst.Operands[0], v)

		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, false)
		c.Registers.Write1(flagH, true)
		c.Registers.Write1(flagC, false)
	case "OR":
		// OR $A $X; $A=$A|$X
		assertOperandType(inst.Operands[0], operandReg8)
		assertOperandType(inst.Operands[1], operandReg8, operandReg16Ptr, operandD8)

		v := c.read8(inst.Operands[0]) | c.read8(inst.Operands[1])
		c.write8(inst.Operands[0], v)

		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, false)
		c.Registers.Write1(flagH, false)
		c.Registers.Write1(flagC, false)
	case "BIT":
		// BIT n X: z=true if the n'th bit in X is unset
		assertOperandType(inst.Operands[0], operandConst8)
		assertOperandType(inst.Operands[1], operandReg8, operandReg16Ptr)
		v := ReadBitN(c.read8(inst.Operands[1]), inst.Operands[0].RefConst8)

		c.Registers.Write1(flagZ, v == false)
		c.Registers.Write1(flagN, false)
		c.Registers.Write1(flagH, true)
	case "STOP":
		// STOP; stop running
		log.Println("POWER OFF")
		c.PowerOn = false
	default:
		notImplemented("instruction not implemented yet")
	}

	// Some instructions automatically increment/decrement values after they complete
	for _, op := range inst.Operands {
		if op.IncrementReg16 || op.DecrementReg16 {
			assertOperandType(op, operandReg16, operandReg16Ptr)
			address := c.Registers.Read16(op.RefRegister16)
			if op.IncrementReg16 {
				address++
			} else {
				address--
			}
			c.Registers.Write16(op.RefRegister16, address)
		}
	}
}

func (c *cpu) read16(op operand) uint16 {
	switch op.Type {
	case operandD16:
		// TODO little endian conversion here may be wrong
		return c.Memory.Read16(c.ProgramCounter - 2)
	case operandA16:
		return c.Memory.Read16(c.ProgramCounter - 2)
	case operandReg16:
		return c.Registers.Read16(op.RefRegister16)
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 16bit value", op.Type.String())
		return 0
	}
}

func (c *cpu) write16(op operand, v uint16) {
	switch op.Type {
	case operandReg16:
		c.Registers.Write16(op.RefRegister16, v)
	default:
		log.Panicf("unexpected operand (%s) encountered while writing 16bit value", op.Type.String())
	}
}

func (c *cpu) read8(op operand) byte {
	switch op.Type {
	case operandD8:
		return c.Memory.Data[c.ProgramCounter-1]
	case operandReg8:
		return c.Registers.Data[op.RefRegister8]
	case operandReg16Ptr:
		address := c.Registers.Read16(op.RefRegister16)
		return c.Memory.Data[address]
	case operandReg8Ptr:
		offset := c.Registers.Data[op.RefRegister8]
		return c.Memory.Data[0xFF00+uint16(offset)]
	case operandA8Ptr:
		offset := c.Memory.Data[c.ProgramCounter-1]
		return c.Memory.Data[0xFF00+uint16(offset)]
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 8bit value", op.Type.String())
		return 0
	}
}

func (c *cpu) read8signed(op operand) int8 {
	switch op.Type {
	case operandR8:
		return int8(c.Memory.Data[c.ProgramCounter-1])
	default:
		log.Panicf("unexpected operand (%s) encountered while reading signed 8bit value", op.Type.String())
		return 0
	}
}

func (c *cpu) write8(op operand, v byte) {
	switch op.Type {
	case operandReg8:
		c.Registers.Data[op.RefRegister8] = v
	case operandReg16Ptr:
		data := c.Registers.Data[op.RefRegister16 : op.RefRegister16+2]
		address := toAddress(data)
		c.Memory.Data[address] = v
	case operandReg8Ptr:
		offset := c.Registers.Data[op.RefRegister8]
		c.Memory.Data[0xFF00+uint16(offset)] = v
	case operandA8Ptr:
		offset := c.Memory.Data[c.ProgramCounter-1]
		c.Memory.Data[0xFF00+uint16(offset)] = v
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
		case operandD8, operandReg8, operandReg8Ptr, operandReg16Ptr, operandA8Ptr:
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
	condition := c.Registers.Read1(op.RefFlag)
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
