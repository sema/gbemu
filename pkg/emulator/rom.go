package emulator

import (
	"fmt"
	"io/ioutil"
	"log"
)

const (
	romMBCProtocol uint16 = 0x0147

	romSize = 0x0148
	ramSize = 0x0149
)

type rom struct {
	// data contains the entire ROM data
	data []byte

	// bankROMLow contains the lower 5 bits of the ROM bank number
	bankROMLow byte

	// bankROMHighRAM containers either the two lower bits of the RAM bank, or bit
	// 5-6 of the ROM bank number, depending on bankRAMMode
	bankROMHighRAM byte

	// bankRAMMode selects if bankROMHighRAM is used for selecting the ROM bank
	// (false) or the RAM bank (true)
	bankRAMMode bool
}

func newROM() *rom {
	return &rom{
		data: make([]byte, bytes32k),
	}
}

// Read8 reads ROM data currently mapped into the address space
//
// TODO: Technically, RAM is also provided by the cartridge, and the MBC
// protocol determines if (a) ram is available (at A000-BFFF), and (b) how much
// is available.
//
// - 0x0000-0x3FFF    Bank 0        Mapped directly to the beginning of ROM data
// - 0x4000-0x7FFF    Bank 01-7F
func (r *rom) Read8(address uint16) byte {
	switch {
	case 0x0000 <= address && address <= 0x3FFF:
		// as the ROM is placed at the beginning of the address space we don't need to offset the input address
		return r.data[address]
	case 0x4000 <= address && address <= 0x7FFF:
		return r.data[0x4000*uint16(r.romBankNumber())+(address-0x4000)]
	}

	notImplemented("reads from ROM at address %x not implemented", address)
	return 0
}

// Write8 interacts with the Memory Bank Controller (MBC), e.g. to switch ROM or
// RAM banks
//
// 0x2000-0x3FFF  Set bankROMLow
// 0x4000-0x5FFF  Set bankROMHighRAM
// 0x6000-0x7FFF  Set bankRAMMode
func (r *rom) Write8(address uint16, v byte) {
	switch {
	case 0x2000 <= address && address <= 0x3FFF:
		r.bankROMLow = v & 0x1F // only write the lower 5 bits
	case 0x4000 <= address && address <= 0x5FFF:
		r.bankROMHighRAM = v & 0x03 // only write the lower 2 bits
	case 0x6000 <= address && address <= 0x7FFF:
		r.bankRAMMode = readBitN(v, 0)
	default:
		notImplemented("writes to MBC at address %x not implemented", address)
	}
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

func (r *rom) romBankNumber() uint8 {
	num := r.bankROMLow
	if num == 0 {
		// interpret bank 0 as bank 1
		// NOTE: bank 20, 40, and 60 are not usable due to this semantic
		num = 1
	}
	if !r.bankRAMMode {
		num = (r.bankROMHighRAM << 5) | num
	}

	return num
}
