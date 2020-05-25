package emulator

import (
	"fmt"
	"io/ioutil"
	"log"
)

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
	} else if len(data) < bytes32k {
		return fmt.Errorf("invalid ROM size: expected ROM to contain at least %d bytes but contained %d bytes", bytes32k, len(data))
	}

	r.data = data

	// Support memory bank controller protocols 0 and 1
	mbcProtocol := r.data[0x0147]
	if mbcProtocol > 1 {
		return fmt.Errorf("unsupported MBC %d", mbcProtocol)
	}

	log.Printf("Loaded %d bytes from ROM", len(data))
	return nil
}
