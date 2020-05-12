package emulator

import (
	"encoding/json"
	"io/ioutil"
)

type emulator struct {
	Memory *memory
	CPU    *cpu
}

func New() emulator {
	memory := newMemory()
	registers := newRegisters()
	cpu := newCPU(memory, registers)
	return emulator{
		CPU:    cpu,
		Memory: memory,
	}
}

func (e *emulator) Run(path string) error {
	if err := e.Memory.LoadROM(path); err != nil {
		return err
	}

	for e.CPU.PowerOn {
		e.CPU.cycle()
	}

	return nil
}

func (e *emulator) Snapshot(path string) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}
