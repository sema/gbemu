package emulator

// videoController handles everything video/graphics related
type videoController struct {
	// data contains values from 0xFF40 - 0xFF6B
	data []byte
}

func newVideoController() *videoController {
	return &videoController{
		data: make([]byte, 0xFF49-0xFF40+1),
	}
}

// Read8 is exposed in the address space, and may be read by the program
func (s *videoController) Read8(address uint16) byte {
	return s.data[address-0xFF40]
}

// Write8 is exposed in the address space, and may be written to by the program
func (s *videoController) Write8(address uint16, v byte) {
	switch address {
	case 0xFF44:
		return // address is read-only
	default:
		s.data[address-0xFF40] = v
	}
}

func (s *videoController) String() string {
	return "VIDEO"
}
