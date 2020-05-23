package ptr

// UInt16 converts a uint16 value to a ptr to an uint16 value
func UInt16(v uint16) *uint16 {
	return &v
}
