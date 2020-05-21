package emulator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSerialCycleTriggersInterruptWhenByteIsTransferred(t *testing.T) {
	serial := newSerialController()
	serial.Write8(0xFF02, 0x81) // 01000001 - set transfer start flag and set master mode

	for i := 0; i < 1000; i++ {
		require.False(t, serial.Interrupt.ReadAndClear())
		serial.Cycle()
	}

	require.True(t, serial.Interrupt.ReadAndClear())
	require.Equal(t, uint8(0xFF), serial.Read8(0xFF01))

	transferStarted := readBitN(serial.Read8(0xFF02), 7)
	require.False(t, transferStarted)
}
