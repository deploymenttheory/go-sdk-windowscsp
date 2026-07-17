package client

import (
	"bytes"
	"testing"
)

func TestValueRoundTrips(t *testing.T) {
	if v, err := Int(42).Int(); err != nil || v != 42 {
		t.Errorf("Int round-trip = %v %v", v, err)
	}
	if v, err := Bool(true).Bool(); err != nil || !v {
		t.Errorf("Bool round-trip = %v %v", v, err)
	}
	if v, err := Float(1.5).Float(); err != nil || v != 1.5 {
		t.Errorf("Float round-trip = %v %v", v, err)
	}
	if v := Chr("hello").Str(); v != "hello" {
		t.Errorf("Chr round-trip = %q", v)
	}
	raw := []byte{0x00, 0x01, 0xFF}
	if v, err := B64(raw).Bytes(); err != nil || !bytes.Equal(v, raw) {
		t.Errorf("B64 round-trip = %v %v", v, err)
	}
	if v, err := Bin(raw).Bytes(); err != nil || !bytes.Equal(v, raw) {
		t.Errorf("Bin round-trip = %v %v", v, err)
	}
}

func TestValueDecodeErrors(t *testing.T) {
	if _, err := Chr("nope").Int(); err == nil {
		t.Error("Int() on non-int should fail")
	}
	if _, err := Chr("nope").Bool(); err == nil {
		t.Error("Bool() on non-bool should fail")
	}
	if _, err := (Value{Format: FormatB64, Data: "!!"}).Bytes(); err == nil {
		t.Error("Bytes() on invalid base64 should fail")
	}
}

func TestFormats(t *testing.T) {
	if Null().Format != FormatNull || Null().Data != "" {
		t.Errorf("Null() = %+v", Null())
	}
	if Node().Format != FormatNode {
		t.Errorf("Node() = %+v", Node())
	}
	if Bool(false).Data != "false" {
		t.Errorf("Bool(false) = %+v", Bool(false))
	}
}
