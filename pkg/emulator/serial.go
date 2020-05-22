package emulator

type serialRegister uint16

const (
	offsetSerialRegisters uint16 = 0xFF01
)

const (
	// Serial transfer data (read/write)
	registerFF01 serialRegister = 0xFF01

	// Serial transfer control (read/write)
	//
	// Bit 7 - Transfer Start Flag (0=No transfer is in progress or requested, 1=Transfer in progress, or requested)
	// Bit 1 - Clock Speed (0=Normal, 1=Fast) ** CGB Mode Only **
	// Bit 0 - Shift Clock (0=External Clock, 1=Internal Clock)
	registerFF02 = 0xFF02
)

type SerialDataCallback func(data uint8)

// serialController handles data transfers over the serial port
//
// Currently, does not support connecting an external device, thus:
// a) A transfer will only happen if the device initiates it by setting bit 7 in 0xFF02
// b) The incoming byte will always be 0xFF
type serialController struct {
	// registers contains control and data registers mapped to 0xFF01 - 0xFF02
	registers []byte

	// transferTicks represent the current number of ticks spent on transferring the
	// current byte. Each transfer takes 1000 cycles.
	transferTicks int

	// Interrupt is true if the serial port wants to trigger the INT 58 interrupt
	Interrupt *interruptSource

	// Callback is called (if set) on every byte that is transferred over the
	// serial port.
	Callback SerialDataCallback
}

func newSerialController() *serialController {
	return &serialController{
		registers: make([]byte, 0xFF02-0xFF01+1),
		Interrupt: newInterruptSource(),
	}
}

// Read8 is exposed in the address space, and may be read by the program
func (s *serialController) Read8(address uint16) byte {
	switch address {
	case 0xFF01:
		return s.readRegister(registerFF01)
	case 0xFF02:
		return s.readRegister(registerFF02)
	}

	notImplemented("read of unimplemented SERIAL register at %#4x", address)
	return byte(0)
}

// Write8 is exposed in the address space, and may be written to by the program
func (s *serialController) Write8(address uint16, v byte) {
	switch address {
	case 0xFF01:
		s.writeRegister(registerFF01, v)
	case 0xFF02:
		s.writeRegister(registerFF02, v)
	default:
		notImplemented("write of unimplemented SERIAL register at %#4x", address)
	}
}

// Cycle transfers bytes on the serial port if requested
func (s *serialController) Cycle() {
	control := s.readRegister(0xFF02)
	isMaster := readBitN(control, 0)
	transferRequested := readBitN(control, 7)

	if !isMaster || !transferRequested {
		// - Do nothing if this device is not the master device, as there is no external device
		//   to communicate with
		// - Do nothing if a transfer has not been requested, as the local device (as master)
		//   should be initiating the transfer
		return
	}

	s.transferTicks++

	transferDone := s.transferTicks >= 1000
	if transferDone {
		if s.Callback != nil {
			s.Callback(s.readRegister(0xFF01))
		}

		s.transferTicks = 0
		s.writeRegister(0xFF01, 0xFF)
		s.writeRegister(0xFF02, writeBitN(control, 7, false))
		s.Interrupt.Set()
	}
}

func (s *serialController) readRegister(r serialRegister) byte {
	return s.registers[uint16(r)-offsetSerialRegisters]
}

func (s *serialController) writeRegister(r serialRegister, v byte) {
	s.registers[uint16(r)-offsetSerialRegisters] = v
}

func (s *serialController) String() string {
	return "SERIAL"
}
