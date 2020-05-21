package emulator

type interruptSource struct {
	pending bool
}

func newInterruptSource() *interruptSource {
	return &interruptSource{}
}

func (i *interruptSource) ReadAndClear() bool {
	result := i.pending
	i.pending = false
	return result
}

func (i *interruptSource) Set() {
	i.pending = true
}

type interruptRegister uint16

const (
	// Interrupt Flag (read/write)
	// Bit 0: V-Blank  Interrupt Request (INT 40h)  (1=Request)
	// Bit 1: LCD STAT Interrupt Request (INT 48h)  (1=Request)
	// Bit 2: Timer    Interrupt Request (INT 50h)  (1=Request)
	// Bit 3: Serial   Interrupt Request (INT 58h)  (1=Request)
	// Bit 4: Joypad   Interrupt Request (INT 60h)  (1=Request)
	registerFF0F interruptRegister = 0xFF0F

	// Interrupt Enable (read/write)
	//
	// Bit 0: V-Blank  Interrupt Enable  (INT 40h)  (1=Enable)
	// Bit 1: LCD STAT Interrupt Enable  (INT 48h)  (1=Enable)
	// Bit 2: Timer    Interrupt Enable  (INT 50h)  (1=Enable)
	// Bit 3: Serial   Interrupt Enable  (INT 58h)  (1=Enable)
	// Bit 4: Joypad   Interrupt Enable  (INT 60h)  (1=Enable)
	registerFFFF = 0xFFFF
)

// interruptController encapsulates the interrupt logic
type interruptController struct {
	interruptFlag    byte
	interruptEnabled byte

	interruptSources []*interruptSource
}

func newInterruptController() *interruptController {
	return &interruptController{
		interruptSources: make([]*interruptSource, 5),
	}
}

// Read8 is exposed in the address space, and may be read by the program
func (i *interruptController) Read8(address uint16) byte {
	switch address {
	case 0xFF0F:
		return i.interruptFlag
	case 0xFFFF:
		return i.interruptEnabled
	}

	notImplemented("read of unimplemented INTERRUPT register at %#4x", address)
	return byte(0)
}

// Write8 is exposed in the address space, and may be written to by the program
func (i *interruptController) Write8(address uint16, v byte) {
	switch address {
	case 0xFF0F:
		i.interruptFlag = v
	case 0xFFFF:
		i.interruptEnabled = v
	default:
		notImplemented("write of unimplemented INTERRUPT register at %#4x", address)
	}
}

func (i *interruptController) registerSource(offset uint8, source *interruptSource) {
	i.interruptSources[offset] = source
}

// CheckSourcesForInterrupts checks all registered sources of interrupts and sets the interrupt flag
// if any source has a pending interrupt
func (i *interruptController) CheckSourcesForInterrupts() {
	for offset, source := range i.interruptSources {
		if source == nil {
			continue
		}

		if source.ReadAndClear() {
			i.interruptFlag = writeBitN(i.interruptFlag, uint8(offset), true)
		}
	}
}

func (i *interruptController) String() string {
	return "INTERRUPT"
}
