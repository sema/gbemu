package emulator

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
)

type emulator struct {
	registers      registers
	memory         memory
	programCounter uint16
	powerOn        bool
}

func New() emulator {
	return emulator{
		registers:      newRegisters(),
		memory:         newMemory(),
		programCounter: 0x0100,
	}
}

func (e *emulator) Run(path string) error {
	if err := e.memory.LoadROM(path); err != nil {
		return err
	}

	e.powerOn = true
	for e.powerOn {
		opcode := e.memory.data[e.programCounter]
		instruction := instructions[opcode]
		e.execute(instruction)
	}

	return nil
}

func (e *emulator) execute(inst instruction) {
	log.Printf("Execute %#04x %-30s %s", e.programCounter, inst.String(), e.reprOperandValues(inst))

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
		v := e.read8(inst.Operands[1])
		e.write8(inst.Operands[0], v)
	case "LD16":
		// LD16 $TARGET $VALUE; $TARGET=$VALUE
		v := e.read16(inst.Operands[1])
		e.write16(inst.Operands[0], v)
	case "INC8":
		// INC8 $OP; $OP++
		v := e.read8(inst.Operands[0]) + 1
		e.write8(inst.Operands[0], v)
		e.registers.Write1(flagZ, v == 0)
		e.registers.Write1(flagN, false)
		lowerHalfInOverflowPosition := v&0b00001111 == 0
		e.registers.Write1(flagH, lowerHalfInOverflowPosition)
	case "DEC8":
		// DEC8 $OP; $OP--
		v := e.read8(inst.Operands[0]) - 1
		e.write8(inst.Operands[0], v)
		e.registers.Write1(flagZ, v == 0)
		e.registers.Write1(flagN, true)
		lowerHalfInOverflowPosition := v&0b00001111 == 0 // TODO this is almost certainly incorrect
		e.registers.Write1(flagH, lowerHalfInOverflowPosition)
	case "JP":
		// JP $TO [$CONDITION]; PC=$TO
		jump := true
		if len(inst.Operands) > 1 {
			jump = e.isFlagSet(inst.Operands[1])
		}

		if jump {
			assertOperandType(inst.Operands[0], operandA16, operandReg16)
			addr := e.read16(inst.Operands[0])
			e.programCounter = addr
			autoIncrementPC = false
		}
	case "JR":
		// JR $OFFSET [$CONDITION]; PC=PC+-$OFFSET
		jump := true
		if len(inst.Operands) > 1 {
			jump = e.isFlagSet(inst.Operands[1])
		}

		if jump {
			assertOperandType(inst.Operands[0], operandR8)
			offset := e.read8signed(inst.Operands[0])
			e.programCounter = offsetAddress(e.programCounter, offset)
			autoIncrementPC = false
		}
	case "STOP":
		// STOP; stop running
		log.Println("POWER OFF")
		e.powerOn = false
	default:
		notImplemented("instruction not implemented yet")
	}

	// Some instructions automatically increment/decrement values after they complete
	for _, op := range inst.Operands {
		if op.IncrementReg16 || op.DecrementReg16 {
			assertOperandType(op, operandReg16, operandReg16Ptr)
			address := e.registers.Read16(op.RefRegister16)
			if op.IncrementReg16 {
				address++
			} else {
				address--
			}
			e.registers.Write16(op.RefRegister16, address)
		}
	}

	if autoIncrementPC {
		e.programCounter += inst.Size
	}
}

func (e *emulator) read16(op operand) uint16 {
	switch op.Type {
	case operandD16:
		// TODO little endian conversion here may be wrong
		return e.memory.Read16(e.programCounter + 1)
	case operandA16:
		return e.memory.Read16(e.programCounter + 1)
	case operandReg16:
		return e.registers.Read16(op.RefRegister16)
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 16bit value", op.Type.String())
		return 0
	}
}

func (e *emulator) write16(op operand, v uint16) {
	switch op.Type {
	case operandReg16:
		e.registers.Write16(op.RefRegister16, v)
	default:
		log.Panicf("unexpected operand (%s) encountered while writing 16bit value", op.Type.String())
	}
}

func (e *emulator) read8(op operand) byte {
	switch op.Type {
	case operandD8:
		// TODO offset
		return e.memory.data[e.programCounter+1]
	case operandReg8:
		return e.registers.data[op.RefRegister8]
	case operandReg16Ptr:
		address := e.registers.Read16(op.RefRegister16)
		return e.memory.data[address]
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 8bit value", op.Type.String())
		return 0
	}
}

func (e *emulator) read8signed(op operand) int8 {
	switch op.Type {
	case operandR8:
		return int8(e.memory.data[e.programCounter+1])
	default:
		log.Panicf("unexpected operand (%s) encountered while reading signed 8bit value", op.Type.String())
		return 0
	}
}

func (e *emulator) write8(op operand, v byte) {
	switch op.Type {
	case operandReg8:
		e.registers.data[op.RefRegister8] = v
	case operandReg16Ptr:
		data := e.registers.data[op.RefRegister16 : op.RefRegister16+2]
		address := toAddress(data)
		e.memory.data[address] = v
	default:
		log.Panicf("unexpected operand (%s) encountered while writing 8bit value", op.Type.String())
	}
}

func (e *emulator) reprOperandValues(inst instruction) string {
	var builder strings.Builder
	for _, op := range inst.Operands {
		var value string
		switch op.Type {
		case operandA16, operandD16, operandReg16:
			value = fmt.Sprintf("%#04x", e.read16(op))
		case operandD8, operandReg8, operandReg16Ptr:
			value = fmt.Sprintf("%#02x", e.read8(op))
		case operandFlag:
			value = fmt.Sprintf("%t", e.isFlagSet(op))
		case operandR8:
			value = fmt.Sprintf("%d", e.read8signed(op))
		}
		if value != "" {
			fmt.Fprintf(&builder, "%-5s= %6s  ", op.Name, value)
		}
	}

	return builder.String()
}

func (e *emulator) isFlagSet(op operand) bool {
	assertOperandType(op, operandFlag)
	condition := e.registers.Read1(op.RefFlag)
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
