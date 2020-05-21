package emulator

import (
	"encoding/json"
	"io/ioutil"
	"time"
)

// Emulator emulates a game Game Boy (DMG-01) machine
type Emulator struct {
	Video     *videoController
	Timer     *timerController
	Serial    *serialController
	Interrupt *interruptController
	Memory    *memory
	CPU       *cpu
	FrameChan chan Frame
}

// New returns an instance of Emulator
func New() *Emulator {
	timer := newTimerController()
	video := newVideoController()
	interrupt := newInterruptController()
	serial := newSerialController()
	memory := newMemory(video, timer, interrupt, serial)
	registers := newRegisters()
	cpu := newCPU(memory, registers)

	interrupt.registerSource(0, nil) // VBLANK
	interrupt.registerSource(1, nil) // LCD stat
	interrupt.registerSource(2, timer.Interrupt)
	interrupt.registerSource(3, serial.Interrupt)
	interrupt.registerSource(4, nil) // Joypad

	return &Emulator{
		CPU:       cpu,
		Memory:    memory,
		Video:     video,
		Timer:     timer,
		Serial:    serial,
		Interrupt: interrupt,
		FrameChan: make(chan Frame),
	}
}

// Run runs the ROM in the emulator, and returns when the emulator halts
func (e *Emulator) Run(path string, bootPath string) error {
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

	frameSync := time.NewTicker(time.Second / 60)
	cpuIdleCycles := 0
	for e.CPU.PowerOn {
		if cpuIdleCycles > 0 {
			cpuIdleCycles--
		} else {
			cpuIdleCycles = e.CPU.Cycle() - 1
		}

		e.Video.Cycle()
		e.Timer.Cycle()
		e.Serial.Cycle()

		e.Interrupt.CheckSourcesForInterrupts()

		if e.Video.FrameReady {
			// Cap rendering to 60 fps
			<-frameSync.C

			e.FrameChan <- e.Video.Frame
		}
	}

	return nil
}

func (e *Emulator) snapshot(path string) error {
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(path, data, 0644)
}
