package emulator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmulatorBlarggSuite(t *testing.T) {
	tests := []struct {
		testROM string
	}{
		{
			testROM: "03-op sp,hl.gb",
		},
		{
			testROM: "09-op r,r.gb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testROM, func(t *testing.T) {
			testPath := fmt.Sprintf("testdata/roms/blargg/cpu_instrs/individual/%s", tt.testROM)

			output := strings.Builder{}
			serialDataCallback := func(data uint8) {
				output.WriteByte(data)
			}

			ctx := context.Background()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			e := New(
				WithSpeedUncapped(),
				WithSerialDataCallback(serialDataCallback))

			// Detect if the Blargg test has completed
			//
			// The test will enter an infinite loop when done (failed or succeeded)
			// by calling JR -2.
			lastObservedPC := uint16(0)
			e.CPU.instructionCallback = func(mnemonic string, pc uint16) {
				if pc == lastObservedPC {
					cancel() // Loop detected, indicates the Blargg test is done
				}
				lastObservedPC = pc
			}

			go func() {
				for {
					select {
					case <-e.FrameChan:
					case <-ctx.Done():
						return // exit
					}
				}
			}()

			e.Run(ctx, testPath, "")

			require.Contains(t, output.String(), "Passed")

		})
	}
}
