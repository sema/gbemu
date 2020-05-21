package emulator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDividerIncrementsAfter256Cycles(t *testing.T) {
	timer := newTimerController()
	for i := 0; i < 256; i++ {
		timer.Cycle()
	}

	require.Equal(t, uint8(1), timer.Read8(0xFF04))
}

func TestTimerIncrementsAfter265CyclesInMode0(t *testing.T) {
	timer := newTimerController()
	timer.Write8(0xFF07, 0x04) // b00000100 - enable timer, mode 0
	for i := 0; i < 256; i++ {
		timer.Cycle()
		require.False(t, timer.Interrupt.ReadAndClear())
	}

	require.Equal(t, uint8(1), timer.Read8(0xFF05))
}

func TestTimerCanInterrupt(t *testing.T) {
	timer := newTimerController()

	timer.Write8(0xFF07, 0x05) // b00000101 - enable timer, mode 1
	timer.Write8(0xFF06, 0x20) // value of 0xFF05 after interrupt

	for i := 0; i < 4; i++ { // 4 cycles to increment timer
		for j := 0; j < 0xFF+1; j++ { // 256+1 rounds to trigger interrupt
			require.False(t, timer.Interrupt.ReadAndClear())
			timer.Cycle()
		}
	}

	require.True(t, timer.Interrupt.ReadAndClear())
	require.Equal(t, uint8(0x20), timer.Read8(0xFF05))
}
