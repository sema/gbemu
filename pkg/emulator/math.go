package emulator

// subtract v2 from v1 (v1-v2)
//
// Borrow is true if the computation caused an underflow (v2 > v1)
// Borrow is true if the subtract in the lower 4 bits causes an underflow
func subtract(v1, v2 uint8) (result uint8, borrow bool, halfborrow bool) {
	result = v1 - v2
	borrow = v1 < v2
	halfborrow = (v1 & 0x0F) < (v2 & 0x0F)
	return
}

// add v1 and v2 (v1+v2)
//
// Overflow is true if the computation caused an overflow
// Half overflow is true if addition between the lower 4 bits overflows
func add(v1, v2 uint8) (result uint8, overflow bool, halfoverflow bool) {
	result = v1 + v2
	overflow = v1 > (0xFF - v2)
	halfoverflow = (v1 & 0x0F) > (0x0F - (v2 & 0x0F))
	return
}

// bcdConversion adjusts a value from an addition/subtraction operation as if the
// addition/subtraction was done between BCD (binary coded decimal) values
//
// A BCD value is between 0x00 and 0x99, and the upper and lower 4 bits
// operate as separate values. As such, if a value exceeds 0x99 it overflows
// to 0x00 rather than when it exceeds 0xFF.
func bcdConversion(v uint8, wasSubtraction bool, halfcarry bool, carry bool) (vOut uint8, carryOut bool) {
	carryOut = carry
	vOut = v

	if wasSubtraction {
		// Previous instruction was a subtraction
		if carry {
			vOut = vOut - 0x60 // adjust underflow to skip from 0xF- 0x9-
		}
		if halfcarry {
			vOut = vOut - 0x06 // adjust half-underflow to skip from 0x-F to 0x-9
		}
	} else {
		// Previous instruction was an addition
		if carry || v > 0x99 {
			vOut = v + 0x60
			carryOut = true
		}
		if halfcarry || (v&0x0f) > 0x09 {
			vOut = vOut + 0x06
		}
	}

	return
}

// add16 v1 and v2 (v1+v2)  (16bit)
//
// Overflow is true if the computation caused an overflow
// Half overflow is true if the addition overflows in the lower 12 bits
func add16(v1, v2 uint16) (result uint16, overflow bool, halfoverflow bool) {
	result = v1 + v2
	overflow = v1 > (0xFFFF - v2)
	halfoverflow = (v1 & 0x0FFF) > (0x0FFF - (v2 & 0x0FFF))
	return
}

// offsetAddress adjusts a base address by a signed offset
//
// Beware the operation may over/under-flow the base address.
func offsetAddress(base uint16, offset int16) uint16 {
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
