package emulator

import (
	"encoding/binary"
	"fmt"
	"log"
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

	opCodes := map[uint8]instruction{
		0x00: {
			mnemonic: "NOP",
			impl:     vm.opNOP,
		},
		0xC3: {
			mnemonic: "JP nnnn",
			args:     2,
			impl: func(args []byte) {
				vm.programCounter = toAddress(args)
			},
		},
		// 8bit load instructions
		0x06: makeLoad8(vm, newOperandRegister8(registerB), newOperandData8()),
		0x16: makeLoad8(vm, newOperandRegister8(registerD), newOperandData8()),
		0x26: makeLoad8(vm, newOperandRegister8(registerH), newOperandData8()),
		0x0e: makeLoad8(vm, newOperandRegister8(registerC), newOperandData8()),
		0x1e: makeLoad8(vm, newOperandRegister8(registerE), newOperandData8()),
		0x2e: makeLoad8(vm, newOperandRegister8(registerL), newOperandData8()),
		0x3e: makeLoad8(vm, newOperandRegister8(registerA), newOperandData8()),

		0x40: makeLoad8(vm, newOperandRegister8(registerB), newOperandRegister8(registerB)),
		0x41: makeLoad8(vm, newOperandRegister8(registerB), newOperandRegister8(registerC)),
		0x42: makeLoad8(vm, newOperandRegister8(registerB), newOperandRegister8(registerD)),
		0x43: makeLoad8(vm, newOperandRegister8(registerB), newOperandRegister8(registerE)),
		0x44: makeLoad8(vm, newOperandRegister8(registerB), newOperandRegister8(registerH)),
		0x45: makeLoad8(vm, newOperandRegister8(registerB), newOperandRegister8(registerL)),
		// 0x46
		0x47: makeLoad8(vm, newOperandRegister8(registerB), newOperandRegister8(registerA)),

		// 16bit load instructions
		0x01: makeLoad16(vm, newOperandRegister16(registerBC), newOperandData16()),
		0x11: makeLoad16(vm, newOperandRegister16(registerDE), newOperandData16()),
		0x21: makeLoad16(vm, newOperandRegister16(registerHL), newOperandData16()),
		0x31: makeLoad16(vm, newOperandRegister16(registerSP), newOperandData16()),
	}

	for i := 0; i < 20; i++ {
		origPC := vm.programCounter

		// lookup opcode
		opcode := vm.memory.data[vm.programCounter]
		vm.programCounter++

		op, ok := opCodes[opcode]
		if !ok {
			return fmt.Errorf("unimplemented opcode [%#02x] encountered", opcode)
		}

		args := vm.memory.data[vm.programCounter : vm.programCounter+op.args]
		vm.programCounter += op.args

		log.Printf("Execute %#04x [%#02x] %s (%# 02x)", origPC, opcode, op.mnemonic, args)
		op.impl(args)
	}

	return nil
}

func toAddress(bytes []byte) uint16 {
	return binary.LittleEndian.Uint16(bytes)
}
