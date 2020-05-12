package emulator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAndUnloadBootROM(t *testing.T) {
	memory := newMemory()

	// the whiteout.gb ROM contains only 0x01s for the entire ROM (32kb)
	err := memory.LoadROM("testdata/roms/whiteout.gb")
	require.NoError(t, err)

	// boot-whiteout.gb contains only 0x02s for the entire ROM (256bytes)
	err = memory.LoadBootROM("testdata/roms/boot-whiteout.gb")
	require.NoError(t, err)

	require.Equal(t, uint8(0x02), memory.Data[255], "expected 256th bit to contain Boot ROM data")
	require.Equal(t, uint8(0x01), memory.Data[256], "expected 257th bit to contain ROM data")
	require.True(t, memory.IsBootROMLoaded)

	memory.UnloadBootROM()

	require.Equal(t, uint8(0x01), memory.Data[255], "expected 256th bit to be restored to ROM data")
	require.False(t, memory.IsBootROMLoaded)
}
