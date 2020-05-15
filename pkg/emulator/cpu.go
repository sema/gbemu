package emulator

import (
	"encoding/binary"
	"fmt"
	"log"
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

func (c *cpu) Cycle() {
	opcode := c.Memory.Read8(c.ProgramCounter)
	inst := instructions[opcode]
	if opcode == 0xCB {
		// 0xCB is a prefix for a 2-byte opcode. Lookup the 2nd byte.
		opcode = c.Memory.Read8(c.ProgramCounter + 1)
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
		assertOperandType(inst.Operands[0], operandReg8, operandReg16Ptr)
		v := c.read8(inst.Operands[0]) + 1
		c.write8(inst.Operands[0], v)
		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, false)
		// TODO calculate correctly
		//lowerHalfInOverflowPosition := v&0b00001111 == 0
		//c.Registers.Write1(flagH, lowerHalfInOverflowPosition)
	case "INC16":
		// INC16 $OP; $OP++
		assertOperandType(inst.Operands[0], operandReg16)
		v := c.read16(inst.Operands[0]) + 1
		c.write16(inst.Operands[0], v)
	case "DEC8":
		// DEC8 $OP; $OP--
		assertOperandType(inst.Operands[0], operandReg8, operandReg16Ptr)
		v, _, halfborrow := subtract(c.read8(inst.Operands[0]), 1)
		c.write8(inst.Operands[0], v)

		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, true)
		c.Registers.Write1(flagH, halfborrow)
	case "DEC16":
		// DEC16 $OP; $OP--
		assertOperandType(inst.Operands[0], operandReg16)
		v := c.read16(inst.Operands[0]) - 1
		c.write16(inst.Operands[0], v)
	case "ADD8":
		// ADD8 A $V; A=A+$V
		assertOperandType(inst.Operands[0], operandReg8)
		assertOperandType(inst.Operands[1], operandReg8, operandD8, operandReg16Ptr)
		v, carry, halfcarry := add(c.read8(inst.Operands[0]), c.read8(inst.Operands[1]))
		c.write8(inst.Operands[0], v)
		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, false)
		c.Registers.Write1(flagH, halfcarry)
		c.Registers.Write1(flagC, carry)
	case "SUB":
		// SUB A $V; A=A-$V
		assertOperandType(inst.Operands[0], operandReg8)
		assertOperandType(inst.Operands[1], operandReg8, operandD8, operandReg16Ptr)
		v, carry, halfcarry := subtract(c.read8(inst.Operands[0]), c.read8(inst.Operands[1]))
		c.write8(inst.Operands[0], v)
		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, true)
		c.Registers.Write1(flagH, halfcarry)
		c.Registers.Write1(flagC, carry)
	case "CP":
		// CP A $V; A-$V - don't store result but set flags based on calculation
		assertOperandType(inst.Operands[0], operandReg8)
		assertOperandType(inst.Operands[1], operandReg8, operandD8, operandReg16Ptr)
		v, carry, halfcarry := subtract(c.read8(inst.Operands[0]), c.read8(inst.Operands[1]))
		c.Registers.Write1(flagZ, v == 0)
		c.Registers.Write1(flagN, true)
		c.Registers.Write1(flagH, halfcarry)
		c.Registers.Write1(flagC, carry)
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
	case "CALL":
		// CALL $TARGET [$CONDITION]; PC=$TARGET if $CONDITION is true. Old PC is added to stack.
		jump := true
		if len(inst.Operands) > 1 {
			assertOperandType(inst.Operands[1], operandFlag)
			jump = c.isFlagSet(inst.Operands[1])
		}

		if jump {
			assertOperandType(inst.Operands[0], operandA16)
			c.stackPush(c.ProgramCounter)
			c.ProgramCounter = c.read16(inst.Operands[0])
		}
	case "PUSH":
		// PUSH RR; SP=SP-2, register RR pushed to stack
		assertOperandType(inst.Operands[0], operandReg16)
		v := c.read16(inst.Operands[0])
		c.stackPush(v)
	case "POP":
		// POP RR; SP=SP+2, register RR restored from stack
		assertOperandType(inst.Operands[0], operandReg16)
		v := c.stackPop()
		c.write16(inst.Operands[0], v)
	case "RET":
		// RET [$CONDITION]; restore PC from stack if $CONDITION is true
		ret := true
		if len(inst.Operands) > 0 {
			assertOperandType(inst.Operands[0], operandFlag)
			ret = c.isFlagSet(inst.Operands[0])
		}

		if ret {
			c.ProgramCounter = c.stackPop()
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
	case "RL", "RLA", "RLC", "RLCA", "RR", "RRA", "RRCA", "SLA", "SRA", "SRL":
		// RL   R; rotate bits left          C <- [7 <- 0] <- C
		// RLC  R; rotate bits left          C <- [7 <- 0] <- [7]
		// RR-- R; variants rotate right
		// R--A R; RR/RL/RRC/RLC with different flagZ semantic
		// SLA  R; shift bits left           C <- [7 <- 0] <- 0
		// SRA  R; arithmetic right shift  [7] -> [7 -> 1] -> C
		// SRL  R; logical right shift       0 -> [7 -> 1] -> C

		assertOperandType(inst.Operands[0], operandReg8, operandReg16Ptr)

		v := c.read8(inst.Operands[0])
		var carry bool

		switch inst.Mnemonic {
		case "RL", "RLA":
			in := c.Registers.Read1(flagC)
			v, carry = shiftByteLeft(v, in)
		case "RR":
			in := c.Registers.Read1(flagC)
			v, carry = shiftByteRight(v, in)
		case "RLC", "RLCA":
			in := readBitN(v, 7)
			v, carry = shiftByteLeft(v, in)
		case "RRC":
			in := readBitN(v, 0)
			v, carry = shiftByteRight(v, in)
		case "SLA":
			v, carry = shiftByteLeft(v, false)
		case "SRA":
			in := readBitN(v, 7)
			v, carry = shiftByteRight(v, in)
		case "SRL":
			v, carry = shiftByteRight(v, false)
		default:
			log.Panicf("unhandled shift and rotate instruction (%s)", inst.Mnemonic)
		}

		c.write8(inst.Operands[0], v)

		if inst.Mnemonic == "RLA" || inst.Mnemonic == "RLCA" {
			c.Registers.Write1(flagZ, false)
		} else {
			c.Registers.Write1(flagZ, v == 0)
		}
		c.Registers.Write1(flagN, false)
		c.Registers.Write1(flagH, false)
		c.Registers.Write1(flagC, carry)
	case "BIT":
		// BIT n X: z=true if the n'th bit in X is unset
		assertOperandType(inst.Operands[0], operandConst8)
		assertOperandType(inst.Operands[1], operandReg8, operandReg16Ptr)
		v := readBitN(c.read8(inst.Operands[1]), inst.Operands[0].RefConst8)

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
	case operandA8:
		offset := c.Memory.Read8(c.ProgramCounter - 1)
		return 0xFF00 + uint16(offset)
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
		return c.Memory.Read8(c.ProgramCounter - 1)
	case operandReg8:
		return c.Registers.Data[op.RefRegister8]
	case operandReg16Ptr:
		address := c.Registers.Read16(op.RefRegister16)
		return c.Memory.Read8(address)
	case operandReg8Ptr:
		offset := c.Registers.Data[op.RefRegister8]
		return c.Memory.Read8(0xFF00 + uint16(offset))
	case operandA8Ptr:
		offset := c.Memory.Read8(c.ProgramCounter - 1)
		return c.Memory.Read8(0xFF00 + uint16(offset))
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 8bit value", op.Type.String())
		return 0
	}
}

func (c *cpu) read8signed(op operand) int8 {
	switch op.Type {
	case operandR8:
		return int8(c.Memory.Read8(c.ProgramCounter - 1))
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
		c.Memory.Write8(address, v)
	case operandReg8Ptr:
		offset := c.Registers.Data[op.RefRegister8]
		c.Memory.Write8(0xFF00+uint16(offset), v)
	case operandA8Ptr:
		offset := c.Memory.Read8(c.ProgramCounter - 1)
		c.Memory.Write8(0xFF00+uint16(offset), v)
	case operandA16Ptr:
		address := c.Memory.Read16(c.ProgramCounter - 2)
		c.Memory.Write8(address, v)
	default:
		log.Panicf("unexpected operand (%s) encountered while writing 8bit value", op.Type.String())
	}
}

func (c *cpu) reprOperandValues(inst instruction) string {
	var operands []operand
	for _, op := range inst.Operands {
		switch op.Type {
		case operandA8Ptr, operandA16Ptr, operandReg8Ptr, operandReg16Ptr:
			// Also print the non-pointer variant of the operand for easier debugging
			ptr := operand(op)
			ptr.Name = ptr.Name[1 : len(ptr.Name)-1]
			switch op.Type {
			case operandA8Ptr:
				ptr.Type = operandA8
			case operandA16Ptr:
				ptr.Type = operandA16
			case operandReg8Ptr:
				ptr.Type = operandReg8
			case operandReg16Ptr:
				ptr.Type = operandReg16
			}
			operands = append(operands, ptr)
			operands = append(operands, op)
		default:
			operands = append(operands, op)
		}
	}

	var builder strings.Builder
	for _, op := range operands {
		v := c.reprOperandValue(op)
		fmt.Fprintf(&builder, "%-5s= %6s  ", op.Name, v)
	}

	return builder.String()
}

func (c *cpu) reprOperandValue(op operand) (v string) {
	defer func() {
		// Handle invalid memory lookups
		if r := recover(); r != nil {
			v = "ERR"
		}
	}()

	switch op.Type {
	case operandA16, operandD16, operandReg16, operandA8:
		v = fmt.Sprintf("%#04x", c.read16(op))
	case operandD8, operandReg8, operandReg8Ptr, operandReg16Ptr, operandA8Ptr, operandA16Ptr:
		v = fmt.Sprintf("%#02x", c.read8(op))
	case operandFlag:
		v = fmt.Sprintf("%t", c.isFlagSet(op))
	case operandR8:
		v = fmt.Sprintf("%d", c.read8signed(op))
	default:
		v = "?"
	}

	return
}

func (c *cpu) isFlagSet(op operand) bool {
	assertOperandType(op, operandFlag)
	condition := c.Registers.Read1(op.RefFlag)
	if op.RefFlagNegate {
		condition = !condition
	}
	return condition
}

// stackPush pushes a 16bit value onto the stck
//
// The value is represented by two bytes at SP-1 and SP-2.
// The stack pointer is left pointing at the higher-order
// byte of the value at SP-2.
func (c *cpu) stackPush(v uint16) {
	sp := c.Registers.Read16(registerSP)
	c.Registers.Write16(registerSP, sp-2)
	c.Memory.Write16(sp-2, v)
}

// stackPop pops a 16bit value from the stack
//
// The value is represented by two bytes at SP and SP+1).
// The stack pointer is left pointing at the next value (SP+2).
func (c *cpu) stackPop() uint16 {
	sp := c.Registers.Read16(registerSP)
	c.Registers.Write16(registerSP, sp+2)
	return c.Memory.Read16(sp)
}

func notImplemented(msg string, args ...interface{}) {
	log.Panicf(msg, args...)
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
