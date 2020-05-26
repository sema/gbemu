package emulator

import "log"

type timerRegister uint16

const (
	offsetTimerRegisters uint16 = 0xFF04
)

const (
	// Divider register (read/write)
	//
	// Writing any value to the register resets it to zero.
	registerFF04 timerRegister = 0xFF04

	// Timer Counter (read/write)
	//
	// Incremented by frequency set in FF07. When overflows (0xFF++) then reset
	// to FF06 and trigger an interrupt.
	registerFF05 = 0xFF05

	// Timer Modulo - value to write to FF05 when it overflows (read/write)
	registerFF06 = 0xFF06

	// Timer Control (read/write)
	//
	// Bits 1-0 control the frequency at which FF05 is incremented. Each
	// cycle will add one or more increments to an internal counter,
	// incrementing FF05 when we reach 256 increments.
	//
	// Bit  2   - Timer Enable
	// Bits 1-0 - Input Clock Select
	//            00: Cycle / 256 = 1 increment
	//            01: Cycle / 4   = 64 increments
	//            10: Cycle / 16  = 16 increments
	//            11: Cycle / 64  = 4 increments
	registerFF07 = 0xFF07
)

// timerController handles time counters and interrupts
type timerController struct {
	// registers contains control and status registers mapped to 0xFF04 - 0xFF07
	registers []byte

	// incrementalTimer counts increments towards increasing the timer counter (see registerFF07)
	incrementalTimer int

	// incrementDivider counts increments towards increasing the divider counter (see registerFF04)
	incrementalDivider int

	// Interrupt is true if the timer wants to trigger the INT 50 interrupt
	Interrupt *interruptSource
}

func newTimerController() *timerController {
	return &timerController{
		registers: make([]byte, 0xFF07-0xFF04+1),
		Interrupt: newInterruptSource(),
	}
}

// Read8 is exposed in the address space, and may be read by the program
func (t *timerController) Read8(address uint16) byte {
	switch address {
	case 0xFF04:
		return t.readRegister(registerFF04)
	case 0xFF05:
		return t.readRegister(registerFF05)
	case 0xFF06:
		return t.readRegister(registerFF06)
	case 0xFF07:
		return t.readRegister(registerFF07)
	}

	notImplemented("read of unimplemented TIMER register at %#4x", address)
	return byte(0)
}

// Write8 is exposed in the address space, and may be written to by the program
func (t *timerController) Write8(address uint16, v byte) {
	switch address {
	case 0xFF04:
		t.writeRegister(registerFF04, 0) // write 0 on any write
		t.incrementalDivider = 0
	case 0xFF05:
		t.writeRegister(registerFF05, v)
	case 0xFF06:
		t.writeRegister(registerFF06, v)
	case 0xFF07:
		t.writeRegister(registerFF07, v)
		t.incrementalTimer = 0
	default:
		notImplemented("write of unimplemented TIMER register at %#4x", address)
	}
}

// Cycle progresses internal counters, and may trigger interrupts
//
// TODO: timer emulation is not exact, as there are a number of complex
// edge cases not currently handled.
// See https://gbdev.io/pandocs/Timer_Obscure_Behaviour.html
func (t *timerController) Cycle() {
	t.incrementalDivider++
	if t.incrementalDivider >= 256 {
		t.incrementalDivider = 0
		t.writeRegister(registerFF04, t.readRegister(registerFF04)+1)
	}

	timerEnabled := readBitN(t.readRegister(registerFF07), 2)
	if timerEnabled {
		mode := t.readRegister(registerFF07) & 0x03 // read lower 2 bits only
		switch mode {
		case 0:
			t.incrementalTimer++
		case 1:
			t.incrementalTimer += 64
		case 2:
			t.incrementalTimer += 16
		case 3:
			t.incrementalTimer += 4
		default:
			log.Panicf("unexpected mode (%d) for 0xFF07 timer observed", mode)
		}

		if t.incrementalTimer >= 256 {
			t.incrementalTimer = 0
			t.writeRegister(registerFF05, t.readRegister(registerFF05)+1)

			interruptTriggered := t.readRegister(registerFF05) == 0
			if interruptTriggered {
				t.writeRegister(registerFF05, t.readRegister(registerFF06))
				t.Interrupt.Set()
			}
		}
	}
}

func (t *timerController) readRegister(r timerRegister) byte {
	return t.registers[uint16(r)-offsetTimerRegisters]
}

func (t *timerController) writeRegister(r timerRegister, v byte) {
	t.registers[uint16(r)-offsetTimerRegisters] = v
}

func (t *timerController) String() string {
	return "TIMER"
}
