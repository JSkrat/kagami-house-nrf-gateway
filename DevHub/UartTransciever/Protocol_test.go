package UartTransciever

import (
	"reflect"
	"testing"
)

func assertPanic(t *testing.T, f func(), message string) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("no panic when it was expected: " + message)
		}
	}()
	f()
}

func Test_stuffPacket(t *testing.T) {
	type args struct {
		data packet
	}
	tests := []struct {
		name    string
		args    args
		wantRet packet
	}{
		{
			name:    "empty packet",
			args:    args{packet{}},
			wantRet: []byte{0xC0},
		},
		{
			name:    "no escape symbols",
			args:    args{packet{0x00, 0x01, 0x02, 0xFF}},
			wantRet: []byte{0xC0, 0x00, 0x01, 0x02, 0xFF},
		},
		{
			name:    "with escape symbols",
			args:    args{packet{0xC0, 0xDB, 0x00, 0x01, 0x02, 0xFF, 0xC0}},
			wantRet: []byte{0xC0, 0xDB, 0xDC, 0xDB, 0xDD, 0x00, 0x01, 0x02, 0xFF, 0xDB, 0xDC},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRet := stuffPacket(tt.args.data); !reflect.DeepEqual(gotRet, tt.wantRet) {
				t.Errorf("stuffPacket() = %v, want %v", gotRet, tt.wantRet)
			}
		})
	}
}

func Test_unstuffPacket(t *testing.T) {
	type args struct {
		data packet
	}
	tests := []struct {
		name    string
		args    args
		wantRet packet
	}{
		{
			name:    "empty packet",
			args:    args{packet{0xC0}},
			wantRet: packet{},
		},
		{
			name:    "no escape symbols",
			args:    args{packet{0xC0, 0x11, 0x22, 0x33}},
			wantRet: packet{0x11, 0x22, 0x33},
		},
		{
			name:    "with escape symbols",
			args:    args{packet{0xC0, 0xDB, 0xDC, 0x11, 0x22, 0x33, 0xDB, 0xDD}},
			wantRet: packet{0xC0, 0x11, 0x22, 0x33, 0xDB},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotRet := unstuffPacket(tt.args.data); !reflect.DeepEqual(gotRet, tt.wantRet) {
				t.Errorf("unstuffPacket() = %v, want %v", gotRet, tt.wantRet)
			}
		})
	}
	// panic tests
	assertPanic(t, func() { unstuffPacket(packet{}) }, "empty packet")
	assertPanic(t, func() { unstuffPacket(packet{0xDB, 0xC0}) }, "packet with no 0xC0 at start")
	assertPanic(t, func() { unstuffPacket(packet{0xC0, 0xC0}) }, "packet with extra 0xC0 in it")
	assertPanic(t, func() { unstuffPacket(packet{0xC0, 0xDB}) }, "packet with incomplete escape sequence")
	assertPanic(t, func() { unstuffPacket(packet{0xC0, 0x00, 0xDB, 0xDC, 0x00, 0xDB}) }, "packet with incomplete escape sequence")
	assertPanic(t, func() { unstuffPacket(packet{0xC0, 0xDB, 0x00}) }, "packet with incorrect escape sequence")
}
