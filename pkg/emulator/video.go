package emulator

import (
	"fmt"
	"strings"
)

// Frame represent a drawn frame on the LCD screen
//
// The frame has 144 rows (outer array) and 160 columns (inner array)
type Frame [][]Shade

type videoRegister uint16

const (
	offsetRegisters uint16 = 0xFF40
	offsetVRAM             = 0x8000
	offsetOAM              = 0xFE00
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

	// Screen Y (Read/Write)
	registerFF42 = 0xFF42

	// Screen X (Read/Write)
	registerFF43 = 0xFF43

	// LY - LCDC Y-Coordinate (Read)
	registerFF44 = 0xFF44

	// Line compare - change status field and trigger interrupt if line is equal
	// to the value of this register
	registerFF45 = 0xFF45

	// Maps BG/Window color # -> shade (see shade type) (Read/Write)
	// Bit 7-6 - Shade for Color Number 3
	// Bit 5-4 - Shade for Color Number 2
	// Bit 3-2 - Shade for Color Number 1
	// Bit 1-0 - Shade for Color Number 0
	registerFF47 = 0xFF47

	// Maps Sprite color # -> shade (see shade type) (Platte 0) (Read/Write)
	// Bit 7-6 - Shade for Color Number 3
	// Bit 5-4 - Shade for Color Number 2
	// Bit 3-2 - Shade for Color Number 1
	// Bit 1-0 - Unused as Color Number 0 is interpreted as transparrent
	registerFF48 = 0xFF48

	// Same as registerFF48, but for Platte 1
	registerFF49 = 0xFF49

	// Window Y position (Read/Write)
	registerFF4A = 0xFF4A

	// Window X position (Read/Write)
	registerFF4B = 0xFF4B
)

// shadePriority is used to determine which of two (or more) overlapping shades
// should be shown on the LCD
//
// The shadePriority constants are ordered by their priority, so sp1 > sp2 means
// that sp1 should be shown over sp2.
type shadePriority uint8

const (
	shadePriorityHidden shadePriority = iota
	shadePriorityBackgroundZero
	shadePrioritySpriteLow
	shadePriorityBackgroundOther
	shadePrioritySpriteHigh
)

type videoFlag struct {
	register  videoRegister
	bitOffset uint8
}

// Shade is the shade of gray shown in a cell on the LCD screen
type Shade uint8

const (
	white Shade = iota
	grayLight
	grayDark
	black
	transparrent = 255
)

var (
	flagVideoEnabled           = videoFlag{register: 0xFF40, bitOffset: 7}
	flagBGWindowTileDataSelect = videoFlag{register: 0xFF40, bitOffset: 4}
	flagBGTileMapSelect        = videoFlag{register: 0xFF40, bitOffset: 3}
	flagSpriteSize             = videoFlag{register: 0xFF40, bitOffset: 2}
	flagSpriteDisplay          = videoFlag{register: 0xFF40, bitOffset: 1}
	flagBGWindowDisplay        = videoFlag{register: 0xFF40, bitOffset: 0}
)

// videoController handles everything video/graphics/PPU related
type videoController struct {
	// registers contains control and status registers mapped to 0xFF40 - 0xFF6B
	registers []byte

	// vram contains Video RAM mapped to 0x8000 - 0x9FFF
	//
	// 1) Tile Data Table, split into 3 blocks, at 0x8000-0x97FF
	// 8000 - 87FF  Block 0  Sprite, BG/Window (8000 mode)
	// 8800 - 8FFF  Block 1  Sprite, BG/Window (all modes)
	// 9000 - 97FF  Block 2  BG/Window (8800 mode)
	//
	// - Each tile is 8x8 pixles
	// - Each tile is 16 bytes, where every 2 bytes represent a line
	// - For each byte pair, the first byte represent the lower bits
	//   of the pixels' color number, and the second byte represent
	//   the higher bits.
	//
	//   Bytes:
	//   Lower    Higher
	//   1010101Y 0101010X
	//
	//   Pixels
	//   b01 b10 b01 b10 b01 b10 b01 bYX
	//
	// Addressing modes:
	// 8000: 0x8000 as the base pointer, and the tile number in the
	//   background tile map is unsigned, such that it can refer to
	//   tiles in block 0 and 1.
	// 8800: 0x9000 as the base pointer, and the tile number in the
	//   background tile map is signed, such that it can refer to
	//   tiles in block 1 and 2.
	//
	// Sprites always use 8000 mode, and BG/Window can use either
	// depending on a bit in 0xFF40.
	//
	// 2) Background Tile Maps
	// 9800 - 9BFF  Background tiles
	// 9C00 - 9FFF  Window tiles
	//
	// Each range defines a 32x32 grid, with 32 consecutive bytes
	// defining each line. Each byte references a tile from the
	// Tile Data Table using the addressing mode described above.
	vram           []byte
	vramAccessible bool

	// oam contains the Sprite attribute table at 0xFE00 - 0xFE9F
	//
	// The Sprite attribute table contains up to 40 entries of 4 bytes
	//
	// Byte 0 - Y position (minus 16, 0 = hidden)
	// Byte 1 - X position (minus 8, 0 = hidden)
	// Byte 2 - Tile/pattern number (references the tile data table in VRAM)
	// Byte 3 - Attributes/flags
	//
	// Flags:
	// Bit7   OBJ-to-BG Priority (0=OBJ Above BG, 1=OBJ Behind BG color 1-3) (Used for both BG and Window. BG color 0 is always behind OBJ)
	// Bit6   Y flip          (0=Normal, 1=Vertically mirrored)
	// Bit5   X flip          (0=Normal, 1=Horizontally mirrored)
	// Bit4   Palette number  (0=OBP0, 1=OBP1)
	oam           []byte
	oamAccessible bool

	nextCycle uint

	// scanline data (snapshot at the start of a line)
	screenY uint8
	screenX uint8
	windowY uint8
	windowX uint8

	Frame Frame // row -> col -> color

	// True once every frame has been calculated, such that it can be flushed
	// to screen.
	FrameReady bool

	// lastLineCompare stores the previous cycles result for line comparison, such
	// that we can trigger interrupts only on changes to this value
	lastLineCompare bool

	InterruptVBlank     *interruptSource // INT 40
	InterruptLCDCStatus *interruptSource // INT 48
}

func newVideoController() *videoController {
	v := &videoController{
		registers:           make([]byte, 0xFF4B-0xFF40+1),
		vram:                make([]byte, 0x9FFF-0x8000+1),
		oam:                 make([]byte, 0xFE9F-0xFE00+1),
		vramAccessible:      true,
		oamAccessible:       true,
		InterruptLCDCStatus: newInterruptSource(),
		InterruptVBlank:     newInterruptSource(),
	}
	v.clearFrame()

	return v
}

func (s *videoController) clearFrame() {
	frame := make([][]Shade, 144)
	for row := 0; row < 144; row++ {
		frame[row] = make([]Shade, 160)
	}

	s.Frame = frame
}

// Read8 is exposed in the address space, and may be read by the program
func (s *videoController) Read8(address uint16) byte {
	if s.isRegisterAddress(address) {
		return s.registers[address-offsetRegisters]
	}

	if s.isOAMAddress(address) {
		return s.oam[address-offsetOAM]
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
		case 0xFF46:
			notImplemented("OAM DMA transfers not implemented")
		default:
			s.registers[address-offsetRegisters] = v
		}
		return
	}

	if s.isOAMAddress(address) {
		if s.oamAccessible {
			s.oam[address-offsetOAM] = v
		}
		return
	}

	if s.vramAccessible {
		s.vram[address-offsetVRAM] = v
	}
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
// - Each line contains 160 pixels, and is rendered in 456 cycles.
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
		return // do nothing if disabled
	}

	line := s.nextCycle / 456
	dot := s.nextCycle % 456
	s.nextCycle = (s.nextCycle + 1) % (456 * 154)

	status := s.readRegister(registerFF41)

	interruptLineCompareEnabled := readBitN(status, 6)
	interruptMode2Enabled := readBitN(status, 5)
	interruptMode1Enabled := readBitN(status, 4)
	interruptMode0Enabled := readBitN(status, 3)

	lineCompare := s.readRegister(registerFF45)
	lineCompareEqual := uint(lineCompare) == line
	lineCompareChanged := lineCompareEqual != s.lastLineCompare

	if interruptLineCompareEnabled && lineCompareEqual && lineCompareChanged {
		s.InterruptLCDCStatus.Set()
	}

	s.FrameReady = false

	var mode uint8

	switch {
	case line >= 144: // VBLANK
		if line == 144 && dot == 0 {
			// Entered VBLANK, signal that we have a complete frame ready
			s.FrameReady = true
			s.InterruptVBlank.Set()
			if interruptMode1Enabled {
				s.InterruptLCDCStatus.Set()
			}
		}
		mode = 1
		s.vramAccessible = true
		s.oamAccessible = true
	case dot < 80: // Scanning OAM
		if dot == 0 {
			// Start of scanline
			s.screenY = s.readRegister(registerFF42)
			s.screenX = s.readRegister(registerFF43)
			s.windowY = s.readRegister(registerFF4A)
			s.windowX = s.readRegister(registerFF4B)
			if interruptMode2Enabled {
				s.InterruptLCDCStatus.Set()
			}
		}
		mode = 2
		s.vramAccessible = true
		s.oamAccessible = false
	case dot < 80+168: // Write pixels
		y := uint8(line)
		x := uint8(dot - 80)
		if x < 160 {
			s.Frame[y][x] = s.calculateShade(y, x)
		}

		mode = 3
		s.vramAccessible = false
		s.oamAccessible = false
	default: // HBLANK
		if dot == 80+168 {
			// Start of HBLANK
			if interruptMode0Enabled {
				s.InterruptLCDCStatus.Set()
			}
		}
		mode = 0
		s.vramAccessible = true
		s.oamAccessible = true
	}

	s.writeRegister(registerFF44, uint8(line))

	// Set mode in 0xFF41 (lower two bits)
	status = copyBits(status, mode, 0, 1)
	status = writeBitN(status, 2, lineCompareEqual)
	s.writeRegister(registerFF41, status)

	// TODO support window
}

// calculateShade determines the shade of color for given line, dot coordinate
//
// The GB display shows the contents of the screen (inner area shown below using "-").
// The screen shows a subset of a larger background. If the screen crosses the rightmost or
// lower boundary of the background then it wraps back around.
//
// The 0, 0 coordinate is in the upper left corner.
// ________
// |  --  |
// |  --  |
//  _______
//
// The shade is calculated by overlaying the background, window, and sprites,
// with various rules of priority, transparrency, etc.
func (s *videoController) calculateShade(line uint8, dot uint8) Shade {
	// Find absolute x, y coordinates in background for input dot, line,
	// affected by current position of the screen (view into background)
	backgroundX := (uint16(s.screenX) + uint16(dot)) % 256
	backgroundY := (uint16(s.screenY) + uint16(line)) % 256

	matchShade := white // fallback color if no other layers apply
	matchPriority := shadePriorityHidden

	spriteShade, spritePriority := s.calculateSpriteShade(uint16(line), uint16(dot))
	if spritePriority > matchPriority {
		matchShade = spriteShade
		matchPriority = spritePriority
	}

	bgShade, bgPriority := s.calculateBackgroundShade(backgroundY, backgroundX)
	if bgPriority > matchPriority {
		matchShade = bgShade
		matchPriority = bgPriority
	}

	return matchShade
}

// calculateBackgroundShade determines the background by doing the following calculations
//
// The GB display shows the contents of the screen (inner area shown below using "-").
// The screen shows a subset of a larger background. If the screen crosses the rightmost or
// lower boundary of the background then it wraps back around.
//
// The 0, 0 coordinate is in the upper left corner.
// ________
// |  --  |
// |  --  |
//  _______
//
// - line, dot (coordinates in the display/screen) ->
// - absolute y, x background coordinate ->
// - background tile # + tile y, x coordinate (within tile) ->
// - shade
func (s *videoController) calculateBackgroundShade(backgroundY uint16, backgroundX uint16) (Shade, shadePriority) {
	if !s.readFlag(flagBGWindowDisplay) {
		return transparrent, shadePriorityHidden
	}

	// Find tile # in Background Tile Map. Every tile in the background tile map
	// represent a 8x8 pixel area.
	tileNumber := s.lookupTileNumber(backgroundY, backgroundX, s.readFlag(flagBGTileMapSelect))
	tileY := uint8(backgroundY % 8)
	tileX := uint8(backgroundX % 8)

	// lookup color number for x,y coordinate within tile (referenced by tile number)
	colorNum := s.lookupTile(tileY, tileX, tileNumber, s.readFlag(flagBGWindowTileDataSelect))

	shadePriority := shadePriorityBackgroundOther
	if colorNum == 0 {
		shadePriority = shadePriorityBackgroundZero
	}

	shadePlatter := s.readRegister(registerFF47)
	return s.lookupShadeInPlatter(shadePlatter, colorNum), shadePriority
}

// lookupShadeInPlatter returns the shade encoded for a colorNum in a platter
//
// A platter contains 4 shades, 2 bits each, with color 0 encoded using the
// lower 2 bits.
func (s *videoController) lookupShadeInPlatter(platter byte, colorNum uint8) Shade {
	return Shade((platter >> 2 * colorNum) & 0x03)
}

func (s *videoController) calculateSpriteShade(line uint16, dot uint16) (Shade, shadePriority) {
	if !s.readFlag(flagSpriteDisplay) {
		return transparrent, shadePriorityHidden
	}

	spriteWidth := 8
	spriteHeight := 8
	if s.readFlag(flagSpriteSize) { // 0=8x8 1=8x16
		spriteHeight = 16
	}

	spritesFoundOnLine := 0

	match := false
	var matchY, matchX int
	var matchTileNumber byte

	// Bit7   OBJ-to-BG Priority (0=OBJ Above BG, 1=OBJ Behind BG color 1-3) Used for both BG and Window. BG color 0 is always behind OBJ)
	// Bit6   Y flip          (0=Normal, 1=Vertically mirrored)
	// Bit5   X flip          (0=Normal, 1=Horizontally mirrored)
	// Bit4   Palette number  (0=OBP0, 1=OBP1)
	var matchAttributes byte

	// Search for the highest priority sprite with a pixel at line, dot
	//
	// Rules:
	// - At most 10 sprites may be evaluated that overlap with line
	// - Sprites are priorited by their x-coordinate (lower is better)
	// - Sprites with the same x-coordinate are priorited on their spriteIdx (lower is better)
	for spriteIdx := 0; spriteIdx < 40; spriteIdx++ {
		if spritesFoundOnLine >= 10 {
			continue // evaluate at most 10 sprites on the current line
		}

		offset := spriteIdx * 4        // each sprite is 4 bytes long
		y := int(s.oam[offset+0]) - 16 // y is offset by 16 such that 0 = hide sprite
		x := int(s.oam[offset+1]) - 8  // x is offset by 8 such that 0 = hide sprite
		tileNumber := s.oam[offset+2]
		attributes := s.oam[offset+3]

		if y <= int(line) && int(line) <= y+spriteHeight {
			spritesFoundOnLine++
			if x <= int(dot) && int(dot) <= x+spriteWidth {
				if match && matchX < x {
					continue // existing sprite has higher priority
				}

				match = true
				matchY = y
				matchX = x
				matchTileNumber = tileNumber
				matchAttributes = attributes
			}
		}
	}

	if !match {
		return transparrent, shadePriorityHidden
	}

	tileY := uint8(int(line) - matchY)
	tileX := uint8(int(dot) - matchX)

	if readBitN(matchAttributes, 6) { // y-flip
		tileY = uint8(spriteHeight) - tileY
	}
	if readBitN(matchAttributes, 5) { // x-flip
		tileX = uint8(spriteWidth) - tileX
	}

	if spriteHeight == 16 {
		// stacked tile mode, in this mode the upper tile has the lower bit in
		// tileNumber forced to 0, and the lower tile has the lower bit forced to 1
		if tileY <= 7 {
			matchTileNumber = matchTileNumber & 0xFE
		} else {
			matchTileNumber = matchTileNumber | 0x01
			tileY = tileY - 8
		}
	}

	colorNum := s.lookupTile(tileY, tileX, matchTileNumber, true)
	if colorNum == 0 {
		return transparrent, shadePriorityHidden
	}

	shadePriority := shadePrioritySpriteHigh
	if readBitN(matchAttributes, 7) { // sprite behind background colors 1-3
		shadePriority = shadePrioritySpriteLow
	}

	shadePlatter := s.readRegister(registerFF48) // platter 0
	if readBitN(matchAttributes, 4) {
		shadePlatter = s.readRegister(registerFF49) // platter 1
	}

	return s.lookupShadeInPlatter(shadePlatter, colorNum), shadePriority
}

// lookupTileNumber returns the tile # for a given absolute x, y
// background/window coordinate
//
// tileMapSelect determines the memory address for the tilemap to be used
// (0x9800 if false, 0x9C00 if true). Use the tilemap selection value in the
// 0xFF40 register associated with the background or window when used.
func (s *videoController) lookupTileNumber(y, x uint16, tileMapSelect bool) byte {
	tileMapAddress := uint16(0x9800)
	if tileMapSelect {
		tileMapAddress = 0x9C00
	}

	// each tile represent a 8x8 area, and there are 32 tiles in every line.
	offset := y/8*32 + x/8

	return s.readVRAM(tileMapAddress + offset)
}

// lookupTile returns the color number for an y, x coordinate within a tile
//
// tileDataSelect determines the tile data to use: 8800 addressing mode if false
// or 8000 addressing mode if true.
func (s *videoController) lookupTile(tileY, tileX uint8, tileNumber byte, tileDataSelect bool) uint8 {
	// 8800 addressing mode - tileNumber is signed
	tileAddress := offsetAddress(0x9000, 16*int16(int8(tileNumber)))
	if tileDataSelect {
		// 8000 addressing mode
		tileAddress = 0x8000 + 16*uint16(tileNumber)
	}

	rowAddress := offsetAddress(tileAddress, 2*int16(tileY)) // 2 bytes for every row
	lowerByte := s.readVRAM(rowAddress)
	higherByte := s.readVRAM(rowAddress + 1)

	// The leftmost pixel is represented by the rightmost (index-0) bit, thus the "7-"
	lowerBit := readBitN(lowerByte, 7-tileX)
	higherBit := readBitN(higherByte, 7-tileX)

	colorNum := uint8(0)
	colorNum = writeBitN(colorNum, 0, lowerBit)
	colorNum = writeBitN(colorNum, 1, higherBit)

	return colorNum
}

func (s *videoController) readVRAM(address uint16) byte {
	return s.vram[address-offsetVRAM]
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

func (s *videoController) isOAMAddress(address uint16) bool {
	return 0xFE00 <= address && address <= 0xFE9F
}

func (s *videoController) String() string {
	return "VIDEO"
}

// Render renders the frame as a string for debugging
func (f Frame) Render() string {
	sb := strings.Builder{}
	for _, row := range f {
		for _, shade := range row {
			sb.WriteString(fmt.Sprintf("%d", shade))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("==============================\n")

	return sb.String()
}
