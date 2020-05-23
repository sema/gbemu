package emulator

import (
	"testing"

	"github.com/sema/gbemu/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func TestIsFlagSet(t *testing.T) {
	type args struct {
		op operand
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "return true if flag is set",
			args: args{
				op: operand{
					Name:          "Z",
					Type:          operandFlag,
					RefFlag:       flagZ,
					RefFlagNegate: false,
				},
			},
			want: true,
		},
		{
			name: "return false if flag is set but operand is negated",
			args: args{
				op: operand{
					Name:          "NZ",
					Type:          operandFlag,
					RefFlag:       flagZ,
					RefFlagNegate: true,
				},
			},
			want: false,
		},
		{
			name: "return false if flag is unset",
			args: args{
				op: operand{
					Name:          "C",
					Type:          operandFlag,
					RefFlag:       flagC,
					RefFlagNegate: false,
				},
			},
			want: false,
		},
		{
			name: "return true if flag is unset but operand is negated",
			args: args{
				op: operand{
					Name:          "NC",
					Type:          operandFlag,
					RefFlag:       flagC,
					RefFlagNegate: true,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registers := newRegisters()
			registers.Write1(flagZ, true)
			registers.Write1(flagC, false)

			c := &cpu{
				Registers: registers,
			}

			if got := c.isFlagSet(tt.args.op); got != tt.want {
				t.Errorf("cpu.isFlagSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStackPushPopReturnsSameValue(t *testing.T) {
	cpu := testCPU()

	cpu.Registers.Write16(registerSP, 0xFFFE) // Initialize SP

	cpu.stackPush(0x1005)
	require.Equal(t, uint16(0x1005), cpu.stackPop())
}

func TestInstructions(t *testing.T) {
	type iao struct {
		inst instruction
		data []uint8
	}

	run := func(inst uint16, data ...uint8) iao {
		return iao{
			inst: instructions[inst],
			data: data,
		}
	}

	tests := []struct {
		name         string
		instructions []iao
		regSP        uint16
		wantRegHL    *uint16
	}{
		{
			name: "0xF8 LD HL SP+r8 with r8=1 increments SP and stores it to HL",
			instructions: []iao{
				run(0xF8, 0x01),
			},
			regSP:     0xFFFE,
			wantRegHL: ptr.UInt16(0xFFFF),
		},
		{
			name: "0xF8 LD HL SP+r8 with r8=-1 decrements SP and stores it to HL",
			instructions: []iao{
				run(0xF8, 0xFF),
			},
			regSP:     0xFFFE,
			wantRegHL: ptr.UInt16(0xFFFD),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu := testCPU()
			cpu.Registers.Write16(registerSP, tt.regSP)

			for _, inst := range tt.instructions {
				// Emulate instruction placed at 0xCF00, with optional data after it
				cpu.ProgramCounter = 0xCF01
				for _, d := range inst.data {
					cpu.Memory.Write8(cpu.ProgramCounter, d)
					cpu.ProgramCounter++
				}

				cpu.execute(inst.inst)
			}

			if tt.wantRegHL != nil {
				require.Equal(t, *tt.wantRegHL, cpu.Registers.Read16(registerHL))
			}

		})
	}
}

func testCPU() *cpu {
	video := newVideoController()
	timer := newTimerController()
	serial := newSerialController()
	interrupt := newInterruptController()
	registers := newRegisters()
	memory := newMemory(video, timer, interrupt, serial)
	return newCPU(memory, registers, options{})
}
