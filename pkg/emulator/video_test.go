package emulator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

	progressCycles(video, 456*154)
	require.Equal(t, uint8(0), video.Read8(registerFF44))
}

func progressCycles(v *videoController, cycles uint) {
	for i := uint(0); i < cycles; i++ {
		v.Cycle()
	}
}
