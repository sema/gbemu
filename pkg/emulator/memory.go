package emulator

import (
	"encoding/binary"
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

type memory struct {
	// Data contains the current addressable memory (ROM, RAM, I/O)
	//
	// See https://gbdev.io/pandocs/#memory-map for details on the layout.
	Data []byte

	// ShadowData contains a temporary copy of Data e.g. when the Boot ROM is loaded.
	ShadowData []byte

	// IsBootROMLoaded is true if the Boot ROM is currently loaded
	IsBootROMLoaded bool
}

func newMemory() *memory {
	return &memory{
		Data:       make([]byte, bytes64k),
		ShadowData: make([]byte, 256),
	}
}

// Source https://gbdev.io/pandocs/#the-cartridge-header
func (m *memory) LoadROM(path string) error {
	log.Printf("loading ROM at %s", path)

	rom, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	} else if len(rom) != bytes32k {
		return fmt.Errorf("invalid ROM size: expected ROM to contain %d bytes but contained %d bytes", bytes32k, len(rom))
	}

	copy(m.Data[0:bytes32k], rom[:])

	log.Printf("Loaded %d bytes from ROM", len(rom))
	return nil
}

// LoadBootROM loads the Boot ROM (256bytes) at the beginning of the memory space
//
// The Boot ROM should be unloaded again when the PC reaches 0x0100. Do so by calling
// UnloadBootROM.
func (m *memory) LoadBootROM(path string) error {
	log.Printf("loading Boot ROM at %s", path)

	rom, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	} else if len(rom) != 256 {
		return fmt.Errorf("invalid ROM size: expected Boot ROM to contain %d bytes but contained %d bytes", 256, len(rom))
	}

	m.IsBootROMLoaded = true
	copy(m.ShadowData[0:256], m.Data[0:256])
	copy(m.Data[0:256], rom[:])

	log.Printf("Loaded %d bytes from Boot ROM", len(rom))
	return nil
}

func (m *memory) UnloadBootROM() {
	log.Println("Unloaded Boot ROM")
	m.IsBootROMLoaded = false
	copy(m.Data[0:256], m.ShadowData[0:256])
}

func (m *memory) Read16(address uint16) uint16 {
	return binary.LittleEndian.Uint16(m.Data[address : address+2])
}
