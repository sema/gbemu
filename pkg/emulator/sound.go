package emulator

// soundController handles everything sound related
//
// TODO For now, only support on/off of sound - all other sound is disabled
// Registers, see https://gbdev.io/pandocs/#sound-controller
// FF10 - FF1E
// FF20 - FF26
// FF30 - FF3F
type soundController struct {
	powerOn bool
}

func newSoundController() *soundController {
	return &soundController{}
}

// Read8 is exposed in the address space, and may be read by the program
func (s *soundController) Read8(address uint16) byte {
	switch address {
	case 0xFF26: // Sound on/off (NR52)
		// Bit 7 - All sound on/off  (0: stop all sound circuits) (Read/Write)
		// Bit 3 - Sound 4 ON flag (Read Only)
		// Bit 2 - Sound 3 ON flag (Read Only)
		// Bit 1 - Sound 2 ON flag (Read Only)
		// Bit 0 - Sound 1 ON flag (Read Only)
		return writeBitN(byte(0), 7, s.powerOn)
	}

	notImplemented("read of unimplemented SOUND register at %#4x", address)
	return byte(0)
}

// Write8 is exposed in the address space, and may be written to by the program
func (s *soundController) Write8(address uint16, v byte) {
	switch address {
	case 0xFF26:
		// Bit 7 - All sound on/off  (0: stop all sound circuits) (Read/Write)
		s.powerOn = readBitN(v, 7)
	default:
		// Ignore all unimplemented writes on purpose
	}

}

func (s *soundController) String() string {
	return "SOUND"
}
