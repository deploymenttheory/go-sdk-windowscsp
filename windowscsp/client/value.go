package client

import (
	"encoding/base64"
	"fmt"
	"strconv"
)

// Format is an OMA-DM DFFormat as it appears in DDF v2.
type Format string

const (
	FormatB64   Format = "b64"
	FormatBin   Format = "bin"
	FormatBool  Format = "bool"
	FormatChr   Format = "chr"
	FormatInt   Format = "int"
	FormatNode  Format = "node"
	FormatNull  Format = "null"
	FormatXML   Format = "xml"
	FormatDate  Format = "date"
	FormatTime  Format = "time"
	FormatFloat Format = "float"
)

// Value is an OMA-DM node value: a wire-encoded payload tagged with its
// DFFormat. Binary payloads (b64/bin) are carried base64-encoded, matching
// SyncML transport encoding.
type Value struct {
	Format Format
	Data   string
}

// Int builds an int-format Value.
func Int(v int64) Value { return Value{Format: FormatInt, Data: strconv.FormatInt(v, 10)} }

// Bool builds a bool-format Value.
func Bool(v bool) Value { return Value{Format: FormatBool, Data: strconv.FormatBool(v)} }

// Float builds a float-format Value.
func Float(v float64) Value {
	return Value{Format: FormatFloat, Data: strconv.FormatFloat(v, 'g', -1, 64)}
}

// Chr builds a chr (string) Value.
func Chr(v string) Value { return Value{Format: FormatChr, Data: v} }

// XML builds an xml-format Value.
func XML(v string) Value { return Value{Format: FormatXML, Data: v} }

// Date builds a date-format Value (ISO 8601 date string).
func Date(v string) Value { return Value{Format: FormatDate, Data: v} }

// Time builds a time-format Value (ISO 8601 time string).
func Time(v string) Value { return Value{Format: FormatTime, Data: v} }

// B64 builds a b64-format Value from raw bytes.
func B64(v []byte) Value {
	return Value{Format: FormatB64, Data: base64.StdEncoding.EncodeToString(v)}
}

// Bin builds a bin-format Value from raw bytes (base64-encoded for
// transport, as SyncML does).
func Bin(v []byte) Value {
	return Value{Format: FormatBin, Data: base64.StdEncoding.EncodeToString(v)}
}

// Null builds an empty null-format Value, the payload for Exec-only nodes.
func Null() Value { return Value{Format: FormatNull} }

// Node builds a node-format Value, the payload for creating interior
// (container) nodes with Add.
func Node() Value { return Value{Format: FormatNode} }

// Int decodes an int-format payload.
func (v Value) Int() (int64, error) {
	n, err := strconv.ParseInt(v.Data, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("csp value %q is not an int: %w", v.Data, err)
	}
	return n, nil
}

// Bool decodes a bool-format payload ("true"/"false", case-insensitive).
func (v Value) Bool() (bool, error) {
	b, err := strconv.ParseBool(v.Data)
	if err != nil {
		return false, fmt.Errorf("csp value %q is not a bool: %w", v.Data, err)
	}
	return b, nil
}

// Float decodes a float-format payload.
func (v Value) Float() (float64, error) {
	f, err := strconv.ParseFloat(v.Data, 64)
	if err != nil {
		return 0, fmt.Errorf("csp value %q is not a float: %w", v.Data, err)
	}
	return f, nil
}

// Str returns the payload as a string (chr, xml, date and time formats).
func (v Value) Str() string { return v.Data }

// Bytes decodes a base64-encoded binary payload (b64 and bin formats).
func (v Value) Bytes() ([]byte, error) {
	b, err := base64.StdEncoding.DecodeString(v.Data)
	if err != nil {
		return nil, fmt.Errorf("csp value is not valid base64: %w", err)
	}
	return b, nil
}
