package emulator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPendingInterruptsFromSourcesTriggerInterruptFlagChanges(t *testing.T) {
	source := newInterruptSource()
	source.Set()

	interrupt := newInterruptController()
	interrupt.registerSource(1, source)

	require.Equal(t, uint8(0), interrupt.Read8(0xFF0F))
	interrupt.CheckSourcesForInterrupts()
	require.Equal(t, uint8(2), interrupt.Read8(0xFF0F))
}
