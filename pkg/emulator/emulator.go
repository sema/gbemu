package emulator

type emulator struct {
	memory *memory
	cpu    *cpu
}

func New() emulator {
	memory := newMemory()
	registers := newRegisters()
	cpu := newCPU(memory, registers)
	return emulator{
		cpu:    cpu,
		memory: memory,
	}
}

func (e *emulator) Run(path string) error {
	if err := e.memory.LoadROM(path); err != nil {
		return err
	}

	for e.cpu.powerOn {
		e.cpu.cycle()
	}

	return nil
}
