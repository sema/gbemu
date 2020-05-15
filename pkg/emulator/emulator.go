package emulator

import (
	"encoding/json"
	"io/ioutil"
)

type emulator struct {
	Video  *videoController
	Memory *memory
	CPU    *cpu
}

func New() emulator {
	video := newVideoController()
	memory := newMemory(video)
	registers := newRegisters()
	cpu := newCPU(memory, registers)
	return emulator{
		CPU:    cpu,
		Memory: memory,
		Video:  video,
	}
}

func (e *emulator) Run(path string, bootPath string) error {
	if err := e.Memory.LoadROM(path); err != nil {
		return err
	}

	if bootPath != "" {
		// Load and run the boot ROM (optional) - this will display the
		// iconic loading screen when starting the emulator.
		e.Memory.LoadBootROM(bootPath)
		e.CPU.ProgramCounter = 0 // execute the boot rom
	} else {
		// TODO set registers if we skip
		e.CPU.ProgramCounter = 0x0100 // skip past boot rom and run ROM directly
	}

	cpuIdleCycles := 0
	for e.CPU.PowerOn {
		if cpuIdleCycles > 0 {
			cpuIdleCycles--
		} else {
			cpuIdleCycles = e.CPU.Cycle() - 1
		}

		e.Video.Cycle()
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
