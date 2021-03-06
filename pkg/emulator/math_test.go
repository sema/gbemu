package emulator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOffsetAddress(t *testing.T) {
	type args struct {
		base   uint16
		offset int16
	}
	tests := []struct {
		name string
		args args
		want uint16
	}{
		{
			name: "can increment address",
			args: args{
				base:   100,
				offset: 10,
			},
			want: 110,
		},
		{
			name: "can decrement address",
			args: args{
				base:   100,
				offset: -10,
			},
			want: 90,
		},
		{
			name: "retains address if offset is zero",
			args: args{
				base:   100,
				offset: 0,
			},
			want: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := offsetAddress(tt.args.base, tt.args.offset); got != tt.want {
				t.Errorf("offsetAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReadBitN(t *testing.T) {
	type args struct {
		v      byte
		offset uint8
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "return true if bit is set",
			args: args{
				v:      2,
				offset: 1,
			},
			want: true,
		},
		{
			name: "return false if bit is unset",
			args: args{
				v:      2,
				offset: 3,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := readBitN(tt.args.v, tt.args.offset); got != tt.want {
				t.Errorf("ReadBitN() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteBitN(t *testing.T) {
	got := writeBitN(0x00, 1, true)
	require.Equal(t, uint8(0x02), got)
}

func TestShiftByteLeft(t *testing.T) {
	type args struct {
		v  byte
		in bool
	}
	tests := []struct {
		name     string
		args     args
		wantVout byte
		wantOut  bool
	}{
		{
			name: "Existing values are shifted left",
			args: args{
				v:  0x02, // 00000010
				in: false,
			},
			wantVout: 0x04, // 00000100
			wantOut:  false,
		},
		{
			name: "Right bit can be set to false",
			args: args{
				v:  0x02, // 00000010
				in: false,
			},
			wantVout: 0x04, // 00000100
			wantOut:  false,
		},
		{
			name: "Right bit can be set to true",
			args: args{
				v:  0x02, // 00000010
				in: true,
			},
			wantVout: 0x05, // 00000101
			wantOut:  false,
		},
		{
			name: "Left bit is shifted out and returned",
			args: args{
				v:  0x80, // 10000000
				in: false,
			},
			wantVout: 0x00, // 00000000
			wantOut:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVout, gotOut := shiftByteLeft(tt.args.v, tt.args.in)
			if gotVout != tt.wantVout {
				t.Errorf("shiftByteLeft() gotVout = %v, want %v", gotVout, tt.wantVout)
			}
			if gotOut != tt.wantOut {
				t.Errorf("shiftByteLeft() gotOut = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestShiftByteRight(t *testing.T) {
	type args struct {
		v  byte
		in bool
	}
	tests := []struct {
		name     string
		args     args
		wantVout byte
		wantOut  bool
	}{
		{
			name: "Existing values are shifted right",
			args: args{
				v:  0x02, // 00000010
				in: false,
			},
			wantVout: 0x01, // 00000001
			wantOut:  false,
		},
		{
			name: "Left bit can be set to false",
			args: args{
				v:  0x04, // 00000100
				in: false,
			},
			wantVout: 0x02, // 00000010
			wantOut:  false,
		},
		{
			name: "Left bit can be set to true",
			args: args{
				v:  0x04, // 00000100
				in: true,
			},
			wantVout: 0x82, // 10000010
			wantOut:  false,
		},
		{
			name: "Right bit is shifted out and returned",
			args: args{
				v:  0x01, // 00000001
				in: false,
			},
			wantVout: 0x00, // 00000000
			wantOut:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVout, gotOut := shiftByteRight(tt.args.v, tt.args.in)
			if gotVout != tt.wantVout {
				t.Errorf("shiftByteRight() gotVout = %v, want %v", gotVout, tt.wantVout)
			}
			if gotOut != tt.wantOut {
				t.Errorf("shiftByteRight() gotOut = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestSubtract(t *testing.T) {
	type args struct {
		v1 uint8
		v2 uint8
	}
	tests := []struct {
		name           string
		args           args
		wantResult     uint8
		wantBorrow     bool
		wantHalfborrow bool
	}{
		{
			name: "subtract without underflow returns subtracted number",
			args: args{
				v1: 4,
				v2: 1,
			},
			wantResult:     3,
			wantBorrow:     false,
			wantHalfborrow: false,
		},
		{
			name: "subtract with 4bit underflow returns halfborrow as true",
			args: args{
				v1: 16,
				v2: 1,
			},
			wantResult:     15,
			wantBorrow:     false,
			wantHalfborrow: true,
		},
		{
			name: "subtract with underflow returns borrow as true",
			args: args{
				v1: 1,
				v2: 255,
			},
			wantResult:     2,
			wantBorrow:     true,
			wantHalfborrow: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotBorrow, gotHalfborrow := subtract(tt.args.v1, tt.args.v2)
			if gotResult != tt.wantResult {
				t.Errorf("subtract() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if gotBorrow != tt.wantBorrow {
				t.Errorf("subtract() gotBorrow = %v, want %v", gotBorrow, tt.wantBorrow)
			}
			if gotHalfborrow != tt.wantHalfborrow {
				t.Errorf("subtract() gotHalfborrow = %v, want %v", gotHalfborrow, tt.wantHalfborrow)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	type args struct {
		v1 uint8
		v2 uint8
	}
	tests := []struct {
		name             string
		args             args
		wantResult       uint8
		wantOverflow     bool
		wantHalfoverflow bool
	}{
		{
			name: "add without overflow returns sum",
			args: args{
				v1: 4,
				v2: 1,
			},
			wantResult:       5,
			wantOverflow:     false,
			wantHalfoverflow: false,
		},
		{
			name: "add with 4bit overflow returns halfoverflow as true",
			args: args{
				v1: 15,
				v2: 1,
			},
			wantResult:       16,
			wantOverflow:     false,
			wantHalfoverflow: true,
		},
		{
			name: "add with overflow returns overflow as true",
			args: args{
				v1: 255,
				v2: 1,
			},
			wantResult:       0,
			wantOverflow:     true,
			wantHalfoverflow: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotBorrow, gotHalfborrow := add(tt.args.v1, tt.args.v2)
			if gotResult != tt.wantResult {
				t.Errorf("add() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			if gotBorrow != tt.wantOverflow {
				t.Errorf("add() gotOverflow = %v, want %v", gotBorrow, tt.wantOverflow)
			}
			if gotHalfborrow != tt.wantHalfoverflow {
				t.Errorf("add() gotHalfoverflow = %v, want %v", gotHalfborrow, tt.wantHalfoverflow)
			}
		})
	}
}

func TestCopyBits(t *testing.T) {
	type args struct {
		to      byte
		from    byte
		offsets []uint8
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		{
			name: "copy true bits sets bits to true",
			args: args{
				to:      0x00, // 00000000
				from:    0xFF, // 11111111
				offsets: []uint8{0, 2},
			},
			want: 0x05, // 00000101
		},
		{
			name: "copy false bits sets bits to false",
			args: args{
				to:      0xFF, // 11111111
				from:    0x00, // 00000000
				offsets: []uint8{0, 2},
			},
			want: 0xFA, // 11111010
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := copyBits(tt.args.to, tt.args.from, tt.args.offsets...); got != tt.want {
				t.Errorf("copyBits() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSwapByte(t *testing.T) {
	type args struct {
		v byte
	}
	tests := []struct {
		name string
		args args
		want byte
	}{
		{
			name: "moves upper 4 bits to lower 4 bits",
			args: args{
				v: 0xF0, // 11110000
			},
			want: 0x0F, // 00001111
		},
		{
			name: "moves lower 4 bits to upper 4 bits",
			args: args{
				v: 0x0F, // 00001111
			},
			want: 0xF0, // 11110000
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := swapByte(tt.args.v); got != tt.want {
				t.Errorf("swapByte() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_BCDConversion(t *testing.T) {
	tests := []struct {
		name         string
		v1           uint8
		v2           uint8
		op           string
		wantVOut     uint8
		wantCarryOut bool
	}{
		{
			name:         "overflow of major digit loops to 0x00",
			v1:           0x99,
			v2:           0x01,
			op:           "addition",
			wantVOut:     0x00,
			wantCarryOut: true,
		},
		{
			name:         "overflow of both digits loops to 0x10",
			v1:           0x99,
			v2:           0x11,
			op:           "addition",
			wantVOut:     0x10,
			wantCarryOut: true,
		},
		/*
			{
				name:         "overflow of minor digit loops to 0x(+1)0",
				v1:           0x89,
				v2:           0x01,
				op:           "addition",
				wantVOut:     0x90,
				wantCarryOut: false,
			},
			{
				name:         "underflow of major digit loops to 0x99",
				v1:           0x00,
				v2:           0x01,
				op:           "subtraction",
				wantVOut:     0x99,
				wantCarryOut: true,
			},
			{
				name:         "underflow of minor digit loops to 0x(-1)9",
				v1:           0x90,
				v2:           0x01,
				op:           "subtraction",
				wantVOut:     0x89,
				wantCarryOut: false,
			},
			{
				name:         "addition within max of minor digit adds as expected",
				v1:           0x55,
				v2:           0x04,
				op:           "addition",
				wantVOut:     0x59,
				wantCarryOut: false,
			},
			{
				name:         "addition within max of major digit adds as expected",
				v1:           0x55,
				v2:           0x10,
				op:           "addition",
				wantVOut:     0x65,
				wantCarryOut: false,
			},
			{
				name:         "subtraction within max of minor digit subtracts as expected",
				v1:           0x55,
				v2:           0x04,
				op:           "subtraction",
				wantVOut:     0x51,
				wantCarryOut: false,
			},
			{
				name:         "subtraction within max of major digit subtracts as expected",
				v1:           0x55,
				v2:           0x10,
				op:           "subtraction",
				wantVOut:     0x45,
				wantCarryOut: false,
			},
		*/
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			op := add
			wasSubtraction := false
			if tt.op == "subtraction" {
				op = subtract
				wasSubtraction = true
			}

			v, overflow, halfoverflow := op(tt.v1, tt.v2)

			gotVOut, gotCarryOut := bcdConversion(v, wasSubtraction, halfoverflow, overflow)
			if gotVOut != tt.wantVOut {
				t.Errorf("bcdConversion() gotVOut = %v, want %v", gotVOut, tt.wantVOut)
			}
			if gotCarryOut != tt.wantCarryOut {
				t.Errorf("bcdConversion() gotCarryOut = %v, want %v", gotCarryOut, tt.wantCarryOut)
			}
		})
	}
}
