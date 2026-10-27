package protocol

import (
	"errors"
	"reflect"
	"testing"
)

func TestParseAndEncode(t *testing.T) {
	tests := []struct {
		name string
		wire string
		want Value
	}{
		{"simple string", "+OK\r\n", Value{Type: SimpleString, Data: "OK"}},
		{"simple error", "-ERR unknown command\r\n", Value{Type: SimpleError, Data: "ERR unknown command"}},
		{"integer", ":1000\r\n", Value{Type: Integer, Data: int64(1000)}},
		{"negative integer", ":-42\r\n", Value{Type: Integer, Data: int64(-42)}},
		{"bulk string", "$5\r\nhello\r\n", Value{Type: BulkString, Data: "hello"}},
		{"empty bulk string", "$0\r\n\r\n", Value{Type: BulkString, Data: ""}},
		{"bulk string containing CRLF", "$7\r\nfoo\r\nba\r\n", Value{Type: BulkString, Data: "foo\r\nba"}},
		{"null bulk string", "$-1\r\n", Value{Type: BulkString}},
		{"empty array", "*0\r\n", Value{Type: Array, Data: []Value{}}},
		{"null array", "*-1\r\n", Value{Type: Array}},
		{
			"command array",
			"*2\r\n$4\r\nECHO\r\n$3\r\nhey\r\n",
			Value{Type: Array, Data: []Value{
				{Type: BulkString, Data: "ECHO"},
				{Type: BulkString, Data: "hey"},
			}},
		},
		{
			"nested mixed array",
			"*3\r\n:1\r\n*2\r\n+a\r\n$-1\r\n-ERR x\r\n",
			Value{Type: Array, Data: []Value{
				{Type: Integer, Data: int64(1)},
				{Type: Array, Data: []Value{
					{Type: SimpleString, Data: "a"},
					{Type: BulkString},
				}},
				{Type: SimpleError, Data: "ERR x"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, n, err := Decode([]byte(tt.wire))
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}
			if n != len(tt.wire) {
				t.Errorf("consumed %d bytes, want %d", n, len(tt.wire))
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse = %+v, want %+v", got, tt.want)
			}
			if wire := string(tt.want.Encode()); wire != tt.wire {
				t.Errorf("Marshal = %q, want %q", wire, tt.wire)
			}
		})
	}
}

func TestParseConsumesOnlyFirstValue(t *testing.T) {
	data := []byte("+OK\r\n:1\r\n")

	_, n, err := Decode(data)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}

	got, _, err := Decode(data[n:])
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := (Value{Type: Integer, Data: int64(1)}); !reflect.DeepEqual(got, want) {
		t.Errorf("second value = %+v, want %+v", got, want)
	}
}

func TestParseIncomplete(t *testing.T) {
	inputs := []string{
		"",
		"+OK",
		"+OK\r",
		"$5\r\nhel",
		"$5\r\nhello",
		"*2\r\n$4\r\nECHO\r\n",
	}

	for _, in := range inputs {
		if _, _, err := Decode([]byte(in)); !errors.Is(err, ErrIncomplete) {
			t.Errorf("Parse(%q) error = %v, want ErrIncomplete", in, err)
		}
	}
}

func TestParseInvalid(t *testing.T) {
	inputs := []string{
		"?what\r\n",
		":abc\r\n",
		"$abc\r\n",
		"$-2\r\n",
		"$3\r\nhelloo\r\n",
		"*-2\r\n",
		"*1\r\n?\r\n",
	}

	for _, in := range inputs {
		_, _, err := Decode([]byte(in))
		if err == nil || errors.Is(err, ErrIncomplete) {
			t.Errorf("Parse(%q) error = %v, want a protocol error", in, err)
		}
	}
}
