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
			testROM: "instr_timing/instr_timing.gb",
		},
		// TODO: sound tests
		// TODO: interrupt timing tests
		/*
			TODO: The timing emulation is simplified, making all changes in a single
			cycle and then waiting for the remainder of the cycles. In reality, the
			read/write operation(s) should happen on the last cycle (if only one), or
			last two cycles (if both).
			{
			  testROM: "mem_timing/individual/01-read_timing.gb",
			},
			{
			  testROM: "mem_timing/individual/02-write_timing.gb",
			},
			{
			  testROM: "mem_timing/individual/03-modify_timing.gb",
			},
		*/
		{
			testROM: "cpu_instrs/individual/01-special.gb",
		},
		{
			testROM: "cpu_instrs/individual/02-interrupts.gb",
		},
		{
			testROM: "cpu_instrs/individual/03-op sp,hl.gb",
		},
		{
			testROM: "cpu_instrs/individual/04-op r,imm.gb",
		},
		{
			testROM: "cpu_instrs/individual/05-op rp.gb",
		},
		{
			testROM: "cpu_instrs/individual/06-ld r,r.gb",
		},
		{
			testROM: "cpu_instrs/individual/07-jr,jp,call,ret,rst.gb",
		},
		{
			testROM: "cpu_instrs/individual/08-misc instrs.gb",
		},
		{
			testROM: "cpu_instrs/individual/09-op r,r.gb",
		},
		{
			testROM: "cpu_instrs/individual/10-bit ops.gb",
		},
		{
			testROM: "cpu_instrs/individual/11-op a,(hl).gb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.testROM, func(t *testing.T) {
			testPath := fmt.Sprintf("testdata/roms/blargg/%s", tt.testROM)

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
