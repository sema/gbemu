package emulator

import "testing"

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
