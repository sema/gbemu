package emulator

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOffsetAddress(t *testing.T) {
	type args struct {
		base   uint16
		offset int8
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
