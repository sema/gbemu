package emulator

import (
	"fmt"
	"log"
	"os"
)

const (
	bytes08k = 0x2000
	bytes16k = bytes08k * 2
	bytes32k = bytes16k * 2
	bytes64k = bytes32k * 2
)

type memory struct {
	// data contains the current addressable memory (ROM, RAM, I/O)
	//
	// See https://gbdev.io/pandocs/#memory-map for details on the layout.
	data []byte
}

func newMemory() memory {
	return memory{
		data: make([]byte, bytes64k),
	}
}

// Source https://gbdev.io/pandocs/#the-cartridge-header
func (m *memory) LoadROM(path string) error {
	log.Printf("loading ROM at %s", path)

	file, err := os.Open(path)
	if err != nil {
		return err
	}

	l, err := file.Read(m.data)
	if err != nil {
		return err
	}
	if l != bytes32k {
		return fmt.Errorf("invalid ROM size: expected ROM to contain %d bytes but contained %d bytes", bytes32k, l)
	}

	log.Printf("Loaded %d bytes from ROM", l)
	return nil
}
