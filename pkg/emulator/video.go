package emulator

type videoRegister uint16

const (
	offsetRegisters uint16 = 0xFF40
	offsetVRAM             = 0x8000
)

const (
	// Bit 7 - LCD Display Enable             (0=Off, 1=On)
	// Bit 6 - Window Tile Map Display Select (0=9800-9BFF, 1=9C00-9FFF)
	// Bit 5 - Window Display Enable          (0=Off, 1=On)
	// Bit 4 - BG & Window Tile Data Select   (0=8800-97FF, 1=8000-8FFF)
	// Bit 3 - BG Tile Map Display Select     (0=9800-9BFF, 1=9C00-9FFF)
	// Bit 2 - OBJ (Sprite) Size              (0=8x8, 1=8x16)
	// Bit 1 - OBJ (Sprite) Display Enable    (0=Off, 1=On)
	// Bit 0 - BG/Window Display/Priority     (0=Off, 1=On)
	registerFF40 videoRegister = 0xFF40

	// Bit 6 - LYC=LY Coincidence Interrupt (1=Enable) (Read/Write)
	// Bit 5 - Mode 2 OAM Interrupt         (1=Enable) (Read/Write)
	// Bit 4 - Mode 1 V-Blank Interrupt     (1=Enable) (Read/Write)
	// Bit 3 - Mode 0 H-Blank Interrupt     (1=Enable) (Read/Write)
	// Bit 2 - Coincidence Flag  (0:LYC<>LY, 1:LYC=LY) (Read Only)
	// Bit 1-0 - Mode Flag       (Mode 0-3, see below) (Read Only)
	//           0: During H-Blank
	//           1: During V-Blank
	//           2: During Searching OAM
	//           3: During Transferring Data to LCD Driver
	registerFF41 = 0xFF41

	// LY - LCDC Y-Coordinate (Read)
	registerFF44 = 0xFF44
)

type videoFlag struct {
	register  videoRegister
	bitOffset uint8
}

var (
	flagVideoEnabled = videoFlag{register: 0xFF40, bitOffset: 7}
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

	nextCycle uint
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
		case registerFF41:
			// lowest 3 bits are read-only
			current := s.registers[address-offsetRegisters]
			s.registers[address-offsetRegisters] = copyBits(v, current, 0, 1, 2)
		case registerFF44:
			// do nothing - address is read-only
		default:
			s.registers[address-offsetRegisters] = v
		}
		return
	}

	// TODO block writes in specific modes
	s.vram[address-offsetVRAM] = v
}

// Cycle progresses the video rendering (i.e. PPU)
//
// The exact process used by the GB is not fully understood and some details, such
// as the duration of phase 3 vs phase 2 require very detailed modelling of the
// underlying hardware. We can most likely get away with emulating a simplified
// version of the PPU.
//
// - The PPU renders 60 frames/s.
// - Each frame has 144 lines (+ 10 unrendered VBLANK lines)
// - Each line contains ?? pixels, and is rendered in 456 cycles.
//
// For normal lines, the PPU transitions between modes 2, 3, and 0 (HBLANK). For the
// last 10 lines the PPU is in mode 1 (VBLANK).
//
// Mode  Action        Cycles  Memory Available
// 2	   Scanning OAM	 80      VRAM, CGB palettes
// 3	   Write pixels	 168
// 0	   HBLANK      	 208     VRAM, CGB palettes, OAM
// 1	   VBLANK        456     VRAM, CGB palettes, OAM
//
func (s *videoController) Cycle() {
	if !s.readFlag(flagVideoEnabled) {
		return // do nothing if diabled
	}

	line := uint8(s.nextCycle / 456)
	dot := s.nextCycle % 456
	s.nextCycle = (s.nextCycle + 1) % (456 * 154)

	var mode uint8

	switch {
	case line >= 144: // VBLANK
		mode = 1
	case dot < 80: // Scanning OAM
		mode = 2
	case dot < 80+168: // Write pixels
		mode = 3
	default: // HBLANK
		mode = 0
	}

	s.writeRegister(registerFF44, line)

	// Set mode in 0xFF41 (lower two bits)
	s.writeRegister(registerFF41, copyBits(s.readRegister(registerFF41), mode, 0, 1))
}

func (s *videoController) readFlag(f videoFlag) bool {
	return readBitN(s.readRegister(f.register), f.bitOffset)
}

func (s *videoController) readRegister(r videoRegister) byte {
	return s.registers[uint16(r)-offsetRegisters]
}

func (s *videoController) writeRegister(r videoRegister, v byte) {
	s.registers[uint16(r)-offsetRegisters] = v
}

func (s *videoController) isRegisterAddress(address uint16) bool {
	return address >= offsetRegisters
}

func (s *videoController) String() string {
	return "VIDEO"
}
