package emulator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLookupShadeInPlatter(t *testing.T) {
	tests := []struct {
		name      string
		platter   byte
		colorNum  uint8
		wantShade Shade
	}{
		{
			name:      "lookup color 0 returns first shade",
			platter:   0x03, // 00000011
			colorNum:  0,
			wantShade: black,
		},
		{
			name:      "lookup color 1 returns second shade",
			platter:   0x0C, // 00001100
			colorNum:  1,
			wantShade: black,
		},
		{
			name:      "lookup color 2 returns third shade",
			platter:   0x30, // 00110000
			colorNum:  2,
			wantShade: black,
		},
		{
			name:      "lookup color 3 returns fourth shade",
			platter:   0xC0, // 11000000
			colorNum:  3,
			wantShade: black,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lookupShadeInPlatter(tt.platter, tt.colorNum)
			require.Equal(t, tt.wantShade, got)
		})
	}
}

func TestVideoYLineProgressesAsPPUCycles(t *testing.T) {
	video := newVideoController()

	video.Write8(uint16(registerFF40), 0x80) // Enable Video

	video.Cycle()
	require.Equal(t, uint8(0), video.Read8(registerFF44))

	progressCycles(video, 500)
	require.Equal(t, uint8(1), video.Read8(registerFF44))
}

func TestVideoYLineResetsBackToZeroAfterFullFrame(t *testing.T) {
	video := newVideoController()

	video.Write8(uint16(registerFF40), 0x80) // Enable Video

	progressCycles(video, 456*154+1)
	require.Equal(t, uint8(0), video.Read8(registerFF44)) // FF44 = Y-offset
}

func progressCycles(v *videoController, cycles uint) {
	for i := uint(0); i < cycles; i++ {
		v.Cycle()
	}
}
