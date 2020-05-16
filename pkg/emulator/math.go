package emulator

// subtract v2 from v1 (v1-v2)
//
// Borrow is true if the computation caused an underflow (v2 > v1)
// Borrow is true if the subtract borrowed from bit 3 (i.e. underflow in the lower 4 bits)
func subtract(v1, v2 uint8) (result uint8, borrow bool, halfborrow bool) {
	result = v1 - v2
	borrow = v1 < v2
	halfborrow = v1>>4 != result>>4 || borrow // check if upper 4 bits are unchanged
	return
}

// add v1 and v2 (v1+v2)
//
// Overflow is true if the computation caused an overflow
// Half overflow is true if the addition overflows from bit 3 (i.e. underflow in the lower 4 bits)
func add(v1, v2 uint8) (result uint8, overflow bool, halfoverflow bool) {
	result = v1 + v2
	overflow = v1 > (0xFF - v2)
	halfoverflow = v1>>4 != result>>4 || overflow // check if upper 4 bits are unchanged
	return
}

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

func readBitN(b byte, offset uint8) bool {
	return b&(1<<offset) > 0
}

func writeBitN(b byte, offset uint8, v bool) byte {
	if v {
		// Example [flags] ORed 00100000 -> sets 3rd bit to 1
		return b | (1 << offset)
	}

	// Example [flags] ANDed 11011111 (negated)  -> forces 3rd bit to 0
	return b & ^(1 << offset)
}

func copyBits(to byte, from byte, offsets ...uint8) byte {
	for _, offset := range offsets {
		to = writeBitN(to, offset, readBitN(from, offset))
	}

	return to
}

// shiftByteLeft shifts all bits to the left, adding a new bit to the right and returning the left most bit
//
// out <- [7 <- 0] <- in
func shiftByteLeft(v byte, in bool) (vout byte, out bool) {
	out = readBitN(v, 7)
	vout = v << 1
	vout = writeBitN(vout, 0, in)
	return vout, out
}

// shiftByteRight shifts all bits to the right, adding a new bit to the left and returning the right most bit
//
// in -> [7 -> 0] -> out
func shiftByteRight(v byte, in bool) (vout byte, out bool) {
	out = readBitN(v, 0)
	vout = v >> 1
	vout = writeBitN(vout, 7, in)
	return vout, out
}

// swapByte swaps the upper 4 bits and lower 4 bits
//
// For exaxmple, 00001111 -> 11110000
func swapByte(v byte) byte {
	upper := v << 4
	lower := v >> 4
	return upper | lower
}
