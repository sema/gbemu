package emulator

const (
	offsetRegisters uint16 = 0xFF40
	offsetVRAM             = 0x8000
)

// videoController handles everything video/graphics/PPU related
type videoController struct {
	// registers contains control and status registers mapped to 0xFF40 - 0xFF6B
	registers []byte

	// vram contains Video RAM mapped to 0x8000 - 0x9FFF
	//
	// 8000 - 87FF  Block 0
	// 8080 - 8FFF  Block 1
	// 9000 - 97FF  Block 2
	// Info: https://gbdev.io/pandocs/#vram-tile-data
	vram []byte
}

func newVideoController() *videoController {
	return &videoController{
		registers: make([]byte, 0xFF49-0xFF40+1),
		vram:      make([]byte, 0x9FFF-0x8000+1),
	}
}

// Read8 is exposed in the address space, and may be read by the program
func (s *videoController) Read8(address uint16) byte {
	if s.isRegisterAddress(address) {
		return s.registers[address-offsetRegisters]
	}

	return s.vram[address-offsetVRAM]
}

// Write8 is exposed in the address space, and may be written to by the program
func (s *videoController) Write8(address uint16, v byte) {
	if s.isRegisterAddress(address) {
		switch address {
		case 0xFF44:
			// do nothing - address is read-only
		default:
			s.registers[address-offsetRegisters] = v
		}
		return
	}

	s.vram[address-offsetVRAM] = v
}

func (s *videoController) isRegisterAddress(address uint16) bool {
	return address >= offsetRegisters
}

func (s *videoController) String() string {
	return "VIDEO"
}
