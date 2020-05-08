package emulator

import (
	"encoding/binary"
	"fmt"
)

type instImpl func(args []byte)

func (vm *vm) opNOP(args []byte) {}

type instruction struct {
	mnemonic string
	impl     instImpl
	args     uint16 // # bytes
}

type loadable16 interface {
	String() string
	Load16(vm *vm, args []byte) uint16
	TakesArgs() uint16
}

type storable16 interface {
	String() string
	Store16(vm *vm, v uint16)
}

type operandRegister16 struct {
	r register16
}

func newOperandRegister16(r register16) operandRegister16 {
	return operandRegister16{
		r: r,
	}
}

func (o operandRegister16) String() string {
	return o.r.String()
}

func (o operandRegister16) Store16(vm *vm, v uint16) {
	binary.LittleEndian.PutUint16(vm.registers.data[o.r:o.r+2], v)
}

func (o operandRegister16) Load16(vm *vm, args []byte) uint16 {
	return binary.LittleEndian.Uint16(vm.registers.data[o.r : o.r+2])
}

func (o operandRegister16) TakesArgs() uint16 {
	return 0
}

type operandData16 struct{}

func newOperandData16() operandData16 {
	return operandData16{}
}

func (o operandData16) String() string {
	return "d16"
}

func (o operandData16) Load16(vm *vm, args []byte) uint16 {
	return binary.LittleEndian.Uint16(args)
}

func (o operandData16) TakesArgs() uint16 {
	return 2
}

func makeLoad16(vm *vm, to storable16, from loadable16) instruction {
	return instruction{
		mnemonic: fmt.Sprintf("LD %s=%s", to.String(), from.String()),
		args:     from.TakesArgs(),
		impl: func(args []byte) {
			v := from.Load16(vm, args)
			to.Store16(vm, v)
		},
	}
}

type loadable8 interface {
	String() string
	Load8(vm *vm, args []byte) byte
	TakesArgs() uint16
}

type storable8 interface {
	String() string
	Store8(vm *vm, v byte)
}

type operandRegister8 struct {
	r register8
}

func newOperandRegister8(r register8) operandRegister8 {
	return operandRegister8{
		r: r,
	}
}

func (o operandRegister8) String() string {
	return o.r.String()
}

func (o operandRegister8) Store8(vm *vm, v byte) {
	vm.registers.data[o.r] = v
}

func (o operandRegister8) Load8(vm *vm, args []byte) byte {
	return vm.registers.data[o.r]
}

func (o operandRegister8) TakesArgs() uint16 {
	return 0
}

type operandData8 struct{}

func newOperandData8() operandData8 {
	return operandData8{}
}

func (o operandData8) String() string {
	return "d8"
}

func (o operandData8) Load8(vm *vm, args []byte) byte {
	return args[0]
}

func (o operandData8) TakesArgs() uint16 {
	return 1
}

func makeLoad8(vm *vm, to storable8, from loadable8) instruction {
	return instruction{
		mnemonic: fmt.Sprintf("LD %s=%s", to.String(), from.String()),
		args:     from.TakesArgs(),
		impl: func(args []byte) {
			v := from.Load8(vm, args)
			to.Store8(vm, v)
		},
	}
}
