package emulator

const (
	// Joypad select & state (read/write)
	//
	// Bit 7 - Not used
	// Bit 6 - Not used
	// Bit 5 - P15 Select Button Keys      (0=Select)
	// Bit 4 - P14 Select Direction Keys   (0=Select)
	// Bit 3 - P13 Input Down  or Start    (0=Pressed) (Read Only)
	// Bit 2 - P12 Input Up    or Select   (0=Pressed) (Read Only)
	// Bit 1 - P11 Input Left  or Button B (0=Pressed) (Read Only)
	// Bit 0 - P10 Input Right or Button A (0=Pressed) (Read Only)
	registerFF00 uint16 = 0xFF00
)

// joypadController handles joypad state and interrupts
type joypadController struct {
	// Bit 3 - Down
	// Bit 2 - Up
	// Bit 1 - Left
	// Bit 0 - Right
	inputArrows byte

	// Bit 3 - Start
	// Bit 2 - Select
	// Bit 1 - Button B
	// Bit 0 - Button A
	inputButton byte

	register byte

	// Interrupt is true if the joypad wants to trigger the INT 60 interrupt
	// TODO: trigger interrupts when we accept input
	Interrupt *interruptSource
}

func newJoypadController() *joypadController {
	return &joypadController{
		Interrupt: newInterruptSource(),
	}
}

// Read8 is exposed in the address space, and may be read by the program
func (j *joypadController) Read8(address uint16) byte {
	switch address {
	case 0xFF00:
		buttonSelected := readBitN(j.register, 5)
		arrowSelected := readBitN(j.register, 4)

		out := j.register
		if buttonSelected {
			out = out | j.inputButton
		}
		if arrowSelected {
			out = out | j.inputArrows
		}

		return out
	}

	notImplemented("read of unimplemented JOYPAD register at %#4x", address)
	return byte(0)
}

// Write8 is exposed in the address space, and may be written to by the program
func (j *joypadController) Write8(address uint16, v byte) {
	switch address {
	case 0xFF00:
		j.register = v & 0xF0 // lower 4 bits are readonly
	default:
		notImplemented("write of unimplemented JOYPAD register at %#4x", address)
	}
}

func (j *joypadController) String() string {
	return "JOYPAD"
}
