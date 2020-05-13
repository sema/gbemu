package emulator

// offsetAddress adjusts a base address by a signed offset
//
// Beware the operation may over/under-flow the base address.
func offsetAddress(base uint16, offset int8) uint16 {
	if offset > 0 {
		return base + uint16(offset)
	}
	offsetAbsolute := -offset
	return base - uint16(offsetAbsolute)
}

func ReadBitN(v byte, offset uint8) bool {
	return v&(1<<offset) > 0
}
