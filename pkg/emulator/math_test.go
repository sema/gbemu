package emulator

import (
	"testing"
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
			if got := ReadBitN(tt.args.v, tt.args.offset); got != tt.want {
				t.Errorf("ReadBitN() = %v, want %v", got, tt.want)
			}
		})
	}
}
