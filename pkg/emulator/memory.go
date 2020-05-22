package emulator

import (
	"fmt"
	"io/ioutil"
	"log"
)

const (
	bytes08k = 0x2000
	bytes16k = bytes08k * 2
	bytes32k = bytes16k * 2
	bytes64k = bytes32k * 2
)

type memoryPage interface {
	Read8(address uint16) byte
	Write8(address uint16, v byte)
	String() string
}

type rom struct {
	data []byte
}

func newROM() *rom {
	return &rom{
		data: make([]byte, bytes32k),
	}
}

func (r *rom) Read8(address uint16) byte {
	// as the ROM is placed at the beginning of the address space we don't need to offset the input address
	return r.data[address]
}

func (r *rom) Write8(address uint16, v byte) {
	// TODO write only allowed for MBC
	notImplemented("writes to MBC not implemented")
}

func (r *rom) String() string {
	return "ROM"
}

func (r *rom) LoadROM(path string) error {
	log.Printf("loading ROM at %s", path)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	} else if len(data) != bytes32k {
		return fmt.Errorf("invalid ROM size: expected ROM to contain %d bytes but contained %d bytes", bytes32k, len(data))
	}

	r.data = data

	log.Printf("Loaded %d bytes from ROM", len(data))
	return nil
}

type bootROM struct {
	data []byte
}

func newBootROM() *bootROM {
	return &bootROM{
		data: make([]byte, 256),
	}
}

func (b *bootROM) Read8(address uint16) byte {
	// as the ROM is placed at the beginning of the address space we don't need to offset the input address
	return b.data[address]
}

func (b *bootROM) Write8(address uint16, v byte) {
	// BootROM is read-only
	// TODO decide proper semantics when/if writes like these occur
	notImplemented("writes to MBC not implemented")
}

func (b *bootROM) LoadBootROM(path string) error {
	log.Printf("loading Boot ROM at %s", path)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	} else if len(data) != 256 {
		return fmt.Errorf("invalid ROM size: expected Boot ROM to contain %d bytes but contained %d bytes", 256, len(data))
	}

	b.data = data

	log.Printf("Loaded %d bytes from Boot ROM", len(data))
	return nil
}

func (b *bootROM) String() string {
	return "Boot ROM"
}

type ram struct {
	data   []byte
	offset uint16
	name   string
}

func newRAM(name string, size uint16, offset uint16) *ram {
	return &ram{
		data:   make([]byte, size),
		offset: offset,
		name:   name,
	}
}

func (r *ram) Read8(address uint16) byte {
	return r.data[address-r.offset]
}

func (r *ram) Write8(address uint16, v byte) {
	r.data[address-r.offset] = v
}

func (r *ram) String() string {
	return r.name
}

//https://gbdev.io/pandocs/#ff26-nr52-sound-on-off
// ffPage represents the last page in the address space (0xFF00-0xFFFF), contiaining various IO registers and HRAM
//
// The page dispatches to other more specialized memoryPages based on the accessed address. See `memory` for
// a high-level overview of the structure of 0xFFXX or `newFFPage` for details.
type ffPage struct {
	entries []memoryPage

	timer *timerController
}

func newFFPage(video *videoController, timer *timerController, interrupt *interruptController, serial *serialController) *ffPage {
	hram := newRAM("HRAM", 0xFE-0x7F, 0xFF80)
	sound := newSoundController()

	layout := []struct {
		Controller memoryPage
		End        uint8
	}{
		{End: 0x00, Controller: nil},
		{End: 0x02, Controller: serial},
		{End: 0x03, Controller: nil},
		{End: 0x07, Controller: timer},
		{End: 0x0E, Controller: nil},
		{End: 0x0F, Controller: interrupt},
		{End: 0x3F, Controller: sound},
		{End: 0x4B, Controller: video},
		{End: 0x7F, Controller: nil},
		{End: 0xFE, Controller: hram},
		{End: 0xFF, Controller: interrupt},
	}

	entries := make([]memoryPage, 256)
	next := uint8(0x00)
	for _, entry := range layout {
		for i := uint16(next); i <= uint16(entry.End); i++ {
			entries[i] = entry.Controller
		}
		next = entry.End + 1
	}

	return &ffPage{
		entries: entries,
		timer:   timer,
	}
}

func (f *ffPage) Read8(address uint16) byte {
	entry := f.entries[address-0xFF00]
	if entry == nil {
		notImplemented("memory operations at address %#04x not implemented", address)
	}

	return entry.Read8(address)
}

func (f *ffPage) Write8(address uint16, v byte) {
	entry := f.entries[address-0xFF00]
	if entry == nil {
		notImplemented("memory operations at address %#04x not implemented", address)
	}

	entry.Write8(address, v)
}

func (f *ffPage) String() string {
	return "0xFFXX"
}

type memory struct {
	// Data contains the current addressable memory (ROM(s), RAM(s), I/O)
	//
	// See https://gbdev.io/pandocs/#memory-map for details on the layout.
	//
	// The memory is split into pages (256 pages, higher-order byte), and
	// each page has 265 entries (lower order byte).
	//
	// 00-3F  16KB ROM bank 00
	// 40-7F  16KB ROM bank 01~NN (switchable via MB)
	// 80-9F   8KB VRAM
	// A0-BF   8KB External RAM (in cartridge, switchable)
	// CO-CF   4KB WRAM bank 0
	// D0-DF   4KB WRAM bank 1
	// E0-FD       ECHO RAM (mirrors C0-DD)
	// FE          OAM (Sprite attribute table)
	// FF          00-7F IO Registers
	// --          80-FE HRAM
	// --          FF    IE (Interrupts Enable register)
	pages []memoryPage

	rom     *rom
	bootROM *bootROM
	video   *videoController

	// IsBootROMLoaded is true if the Boot ROM is currently loaded
	IsBootROMLoaded bool
}

func newMemory(video *videoController, timer *timerController, interrupt *interruptController, serial *serialController) *memory {
	rom := newROM()
	bootROM := newBootROM()
	ffPage := newFFPage(video, timer, interrupt, serial)
	externalRAM := newRAM("EXTERNAL RAM", 0xC000-0xA000, 0xA000)
	wRAM0 := newRAM("WRAM[0]", 0xD000-0xC000, 0xC000)
	wRAM1 := newRAM("WRAM[1]", 0xE000-0xD000, 0xD000)

	layout := []struct {
		Controller memoryPage
		End        uint8
	}{
		{End: 0x7F, Controller: rom},
		{End: 0x9F, Controller: video}, // VRAM
		{End: 0xBF, Controller: externalRAM},
		{End: 0xCF, Controller: wRAM0},
		{End: 0xDF, Controller: wRAM1},
		{End: 0xFD, Controller: nil}, // ECHO RAM
		{End: 0xFE, Controller: nil}, // OAM
		{End: 0xFF, Controller: ffPage},
	}

	pages := make([]memoryPage, 265)
	next := uint8(0x00)
	for _, entry := range layout {
		for i := uint16(next); i <= uint16(entry.End); i++ {
			pages[i] = entry.Controller
		}
		next = entry.End + 1
	}

	return &memory{
		pages:   pages,
		rom:     rom,
		bootROM: bootROM,
		video:   video,
	}
}

func (m *memory) LoadROM(path string) error {
	return m.rom.LoadROM(path)
}

// LoadBootROM loads the Boot ROM (256bytes) at the beginning of the memory space
//
// The Boot ROM should be unloaded again when the PC reaches 0x0100. Do so by calling
// UnloadBootROM.
func (m *memory) LoadBootROM(path string) error {
	if err := m.bootROM.LoadBootROM(path); err != nil {
		return err
	}

	m.IsBootROMLoaded = true
	m.pages[0] = m.bootROM // expose boot ROM in the lowest page
	return nil
}

func (m *memory) UnloadBootROM() {
	log.Println("Unloaded Boot ROM")
	m.IsBootROMLoaded = false
	m.pages[0] = m.rom
}

func (m *memory) Read8(address uint16) byte {
	if address == 0xFF50 { // Boot ROM loaded register
		// TODO the current design makes it difficult to catch this signal in a page
		// controller assigned to this exact address. Should be improved.
		return 0
	}

	pageIdx := uint8(address >> 8)
	page := m.pages[pageIdx]
	if page == nil {
		notImplemented("memory operations at address %#04x not implemented", address)
	}

	return page.Read8(address)
}

func (m *memory) Write8(address uint16, v byte) {
	if address == 0xFF50 && v == 0x01 {
		// TODO the current design makes it difficult to catch this signal in a page
		// controller assigned to this exact address. Should be improved.
		m.UnloadBootROM()
		return
	}

	pageIdx := uint8(address >> 8)
	page := m.pages[pageIdx]
	if page == nil {
		notImplemented("memory operations at address %#04x not implemented", address)
	}

	page.Write8(address, v)
}

// Read16 reads a 16bit value from memory
//
// NOTE: uses little-endian
func (m *memory) Read16(address uint16) uint16 {
	byteLow := m.Read8(address)
	byteHigh := m.Read8(address + 1)
	return uint16(byteLow) | uint16(byteHigh)<<8
}

// Write16 writes a 16bit value to memory
//
// NOTE: uses little-endian
func (m *memory) Write16(address uint16, v uint16) {
	m.Write8(address, byte(v))
	m.Write8(address+1, byte(v>>8))
}
