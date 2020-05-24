package emulator

import (
	"context"
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
	options   options
}

type options struct {
	DebugLogging bool
	// Speed determines the speed of the emulation
	//
	// Currently only allows for switching between uncapped (as fast as possible)) and
	// realtime (as if using a real device). Can support speedup/slowmotion in the future.
	//
	// 0 = uncapped
	// 1 = realtime
	Speed float64
}

type optionFunc func(e *Emulator)

// WithDebugLogging enables debug-level logging in the emulator
//
// Doing so greatly slows down emulation.
func WithDebugLogging() optionFunc {
	return func(e *Emulator) {
		e.options.DebugLogging = true
	}
}

// WithSpeedUncapped causes the emulator to run as fast as it can
func WithSpeedUncapped() optionFunc {
	return func(e *Emulator) {
		e.options.Speed = 0
	}
}

// WithSerialDataCallback provides a func f that will be called on
// every byte transferred out on the serial port
func WithSerialDataCallback(f SerialDataCallback) optionFunc {
	return func(e *Emulator) {
		e.Serial.Callback = f
	}
}

// New returns an instance of Emulator
func New(opts ...optionFunc) *Emulator {
	options := options{
		Speed: 1,
	}

	timer := newTimerController()
	video := newVideoController()
	interrupt := newInterruptController()
	serial := newSerialController()
	joypad := newJoypadController()
	memory := newMemory(video, timer, interrupt, serial, joypad)
	registers := newRegisters()
	cpu := newCPU(memory, registers, options)

	interrupt.registerSource(0, nil) // VBLANK
	interrupt.registerSource(1, nil) // LCD stat
	interrupt.registerSource(2, timer.Interrupt)
	interrupt.registerSource(3, serial.Interrupt)
	interrupt.registerSource(4, joypad.Interrupt)

	e := &Emulator{
		CPU:       cpu,
		Memory:    memory,
		Video:     video,
		Timer:     timer,
		Serial:    serial,
		Interrupt: interrupt,
		FrameChan: make(chan Frame),
		options:   options,
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Run runs the ROM in the emulator, and returns when the emulator halts
func (e *Emulator) Run(ctx context.Context, path string, bootPath string) error {
	if err := e.Memory.LoadROM(path); err != nil {
		return err
	}

	if bootPath != "" {
		// Load and run the boot ROM (optional) - this will display the
		// iconic loading screen when starting the emulator.
		e.Memory.LoadBootROM(bootPath)
		e.CPU.ProgramCounter = 0 // execute the boot rom
	} else {
		e.CPU.ProgramCounter = 0x0100 // skip past boot rom and run ROM directly
		e.CPU.Registers.Write16(registerAF, 0x01B0)
		e.CPU.Registers.Write16(registerBC, 0x0013)
		e.CPU.Registers.Write16(registerDE, 0x00D8)
		e.CPU.Registers.Write16(registerHL, 0x014D)
		e.CPU.Registers.Write16(registerSP, 0xFFFE)

		e.Memory.Write8(0xFF05, 0)
		e.Memory.Write8(0xFF06, 0)
		e.Memory.Write8(0xFF07, 0)
		e.Memory.Write8(0xFF10, 0x80)
		e.Memory.Write8(0xFF11, 0xBF)
		e.Memory.Write8(0xFF12, 0xF3)
		e.Memory.Write8(0xFF14, 0xBF)
		e.Memory.Write8(0xFF16, 0x3F)
		e.Memory.Write8(0xFF17, 0)
		e.Memory.Write8(0xFF19, 0xBF)
		e.Memory.Write8(0xFF1A, 0x7F)
		e.Memory.Write8(0xFF1B, 0xFF)
		e.Memory.Write8(0xFF1C, 0x9F)
		e.Memory.Write8(0xFF1E, 0xBF)
		e.Memory.Write8(0xFF20, 0xFF)
		e.Memory.Write8(0xFF21, 0)
		e.Memory.Write8(0xFF22, 0)
		e.Memory.Write8(0xFF23, 0xBF)
		e.Memory.Write8(0xFF24, 0x77)
		e.Memory.Write8(0xFF25, 0xF3)
		e.Memory.Write8(0xFF26, 0xF1)
		e.Memory.Write8(0xFF40, 0x91)
		e.Memory.Write8(0xFF42, 0)
		e.Memory.Write8(0xFF45, 0)
		e.Memory.Write8(0xFF47, 0xFC)
		e.Memory.Write8(0xFF48, 0xFF)
		e.Memory.Write8(0xFF49, 0xFF)
		e.Memory.Write8(0xFF4A, 0)
		e.Memory.Write8(0xFF4B, 0)
		e.Memory.Write8(0xFFFF, 0)
	}

	frameSync := time.NewTicker(time.Second / 60)
	cpuIdleCycles := 0

	for e.CPU.PowerOn {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

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
			if e.options.Speed > 0 {
				// Cap rendering to 60 fps
				select {
				case <-frameSync.C:
				case <-ctx.Done():
					return nil
				}
			}

			select {
			case e.FrameChan <- e.Video.Frame:
			case <-ctx.Done():
				return nil
			}
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
