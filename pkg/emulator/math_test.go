package emulator

import "testing"

func Test_offsetAddress(t *testing.T) {
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
