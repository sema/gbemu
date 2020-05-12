package emulator

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strings"
)

type vm struct {
	registers      registers
	memory         memory
	programCounter uint16
}

func New() vm {
	return vm{
		registers:      newRegisters(),
		memory:         newMemory(),
		programCounter: 0x0100,
	}
}

func (vm *vm) Run(path string) error {
	if err := vm.memory.LoadROM(path); err != nil {
		return err
	}

	for i := 0; i < 20; i++ {
		opcode := vm.memory.data[vm.programCounter]
		instruction := instructions[opcode]
		vm.execute(instruction)
	}

	return nil
}

func (vm *vm) execute(inst instruction) {
	log.Printf("Execute %#04x %-30s %s", vm.programCounter, inst.String(), vm.reprOperandValues(inst))

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
		// LD8 $TARGET $VALUE
		v := vm.read8(inst.Operands[1])
		vm.write8(inst.Operands[0], v)
	case "LD16":
		// LD16 $TARGET $VALUE
		v := vm.read16(inst.Operands[1])
		vm.write16(inst.Operands[0], v)
	case "INC8":
		v := vm.read8(inst.Operands[0]) + 1
		vm.write8(inst.Operands[0], v)
		vm.registers.Write1(flagZero, v == 0)
		vm.registers.Write1(flagSubtract, false)
		vm.registers.Write1(flagHalfCarry, v&0b00001111 == 0)
	case "JP":
		// JP $TO [$CONDITION]
		if len(inst.Operands) > 1 {
			notImplemented("JP with condition not implemented yet")
		}
		addr := vm.read16(inst.Operands[0])
		vm.programCounter = addr
		autoIncrementPC = false
	default:
		notImplemented("instruction not implemented yet")
	}

	for _, op := range inst.Operands {
		if op.IncrementReg16 || op.DecrementReg16 {
			assertOperandType(op, operandReg16, operandReg16Ptr)
			address := vm.registers.Read16(op.RefRegister16)
			if op.IncrementReg16 {
				address++
			} else {
				address--
			}
			vm.registers.Write16(op.RefRegister16, address)
		}
	}

	if autoIncrementPC {
		vm.programCounter += inst.Size
	}
}

func (vm *vm) read16(op operand) uint16 {
	switch op.Type {
	case operandD16:
		// TODO little endian conversion here may be wrong
		return vm.memory.Read16(vm.programCounter + 1)
	case operandA16:
		return vm.memory.Read16(vm.programCounter + 1)
	case operandReg16:
		return vm.registers.Read16(op.RefRegister16)
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 16bit value", op.Type.String())
		return 0
	}
}

func (vm *vm) write16(op operand, v uint16) {
	switch op.Type {
	case operandReg16:
		vm.registers.Write16(op.RefRegister16, v)
	default:
		log.Panicf("unexpected operand (%s) encountered while writing 16bit value", op.Type.String())
	}
}

func (vm *vm) read8(op operand) byte {
	switch op.Type {
	case operandD8:
		// TODO offset
		return vm.memory.data[vm.programCounter+1]
	case operandReg8:
		return vm.registers.data[op.RefRegister8]
	case operandReg16Ptr:
		address := vm.registers.Read16(op.RefRegister16)
		return vm.memory.data[address]
	default:
		log.Panicf("unexpected operand (%s) encountered while reading 8bit value", op.Type.String())
		return 0
	}
}

func (vm *vm) write8(op operand, v byte) {
	switch op.Type {
	case operandReg8:
		vm.registers.data[op.RefRegister8] = v
	case operandReg16Ptr:
		data := vm.registers.data[op.RefRegister16 : op.RefRegister16+2]
		address := toAddress(data)
		vm.memory.data[address] = v
	default:
		log.Panicf("unexpected operand (%s) encountered while writing 8bit value", op.Type.String())
	}
}

func (vm *vm) reprOperandValues(inst instruction) string {
	var builder strings.Builder
	for _, op := range inst.Operands {
		var value string
		switch op.Type {
		case operandA16, operandD16, operandReg16:
			value = fmt.Sprintf("%#04x", vm.read16(op))
		case operandD8, operandReg8, operandReg16Ptr:
			value = fmt.Sprintf("%#02x", vm.read8(op))
		}
		if value != "" {
			fmt.Fprintf(&builder, "%-5s= %6s  ", op.Name, value)
		}
	}

	return builder.String()
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
