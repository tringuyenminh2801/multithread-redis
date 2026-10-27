package protocol

import (
	"errors"
	"fmt"
	"strconv"

	"multithreadredis/internal/algo"
)

const CRLF = "\r\n"

var crlfMatcher = algo.NewKMP([]byte("\r\n"))

type Type byte

const (
	SimpleString Type = '+'
	SimpleError  Type = '-'
	Integer      Type = ':'
	BulkString   Type = '$'
	Array        Type = '*'
)

// ErrIncomplete means data ends before the value does; read more bytes and parse again.
// It can also implies that the data ends before we get CRLF for the last value read.
var ErrIncomplete = errors.New("resp: incomplete data")

type Command struct {
	Name string
	Args []string
}

// Decode decodes the first RESP value in data and returns it along with the
// number of bytes it occupied, so callers can continue with data[n:].
func Decode(data []byte) (Value, int, error) {
	if len(data) == 0 {
		return Value{}, 0, ErrIncomplete
	}

	t := Type(data[0])
	switch t {
	case SimpleString, SimpleError, Integer, BulkString, Array:
	default:
		// Reject before waiting for a CRLF that may never come.
		return Value{}, 0, fmt.Errorf("RESP: unknown type byte %q", data[0])
	}

	line, n, err := readLine(data[1:])

	if err != nil {
		return Value{}, 0, err
	}

	n++

	switch t {
	case SimpleString, SimpleError:
		return Value{Type: t, Data: string(line)}, n, nil

	case Integer:
		return decodeInteger(line, n)

	case BulkString:
		return decodeBulkString(data, line, n)

	case Array:
		return decodeArray(data, line, n)

	default:
		return Value{}, 0, fmt.Errorf("RESP: unknown type byte %q", data[0])
	}
}

// Array: line holds the element count, and the elements start at data[n:].
func decodeArray(data, line []byte, n int) (Value, int, error) {
	count, err := strconv.Atoi(string(line))
	if err != nil || count < -1 {
		return Value{}, 0, fmt.Errorf("RESP: invalid array length %q", line)
	}
	if count == -1 {
		return Value{Type: Array}, n, nil
	}

	// Every element takes at least 3 bytes, which bounds the allocation
	// by the input size rather than by a client-supplied count.
	elems := make([]Value, 0, min(count, (len(data)-n)/3))
	for i := 0; i < count; i++ {
		elem, m, err := Decode(data[n:])
		if err != nil {
			return Value{}, 0, err
		}
		elems = append(elems, elem)
		n += m
	}
	return Value{Type: Array, Data: elems}, n, nil
}

// Bulk String: line holds the length, and the body starts at data[n:].
func decodeBulkString(data, line []byte, n int) (Value, int, error) {
	length, err := strconv.Atoi(string(line))

	if err != nil || length < -1 {
		return Value{}, 0, fmt.Errorf("RESP: invalid bulk string length %q", line)
	}

	if length == -1 {
		return Value{Type: BulkString}, n, nil
	}

	end := n + length
	if len(data) < end+len(CRLF) {
		return Value{}, 0, ErrIncomplete
	}

	if string(data[end:end+len(CRLF)]) != CRLF {
		return Value{}, 0, errors.New("RESP: bulk string not terminated by CRLF")
	}

	return Value{Type: BulkString, Data: string(data[n:end])}, end + len(CRLF), nil
}

// Integer
func decodeInteger(data []byte, n int) (Value, int, error) {
	i, err := strconv.ParseInt(string(data), 10, 64)

	if err != nil {
		return Value{}, 0, fmt.Errorf("resp: invalid integer %q", data)
	}

	return Value{Type: Integer, Data: i}, n, nil
}

// readLine returns all the bytes before encountering the first CRLF and the number of bytes
// consumed including the CRLF.
// Return:
// - Data in byte array before first CRLF
// - How many bytes read, including the CRLF
// - In case we dont get to the CRLF, then expected to return ErrIncomplete
// Time: O(n) with Knuth-Morris-Pratt
// Space: O(1) - the failure table is built once for CRLF
func readLine(data []byte) ([]byte, int, error) {
	i := crlfMatcher.Index(data)
	if i == -1 {
		return nil, -1, ErrIncomplete
	}

	return data[:i], i + len(CRLF), nil
}

// Value is a single RESP value. The dynamic type of Data depends on Type:
// string for SimpleString, SimpleError and BulkString, int64 for Integer,
// []Value for Array. A nil Data is a null bulk string ($-1) or a null array (*-1).
type Value struct {
	Type Type
	Data any
}

// Encode encodes v into its RESP wire format. It panics if Data does not
// hold the Go type that v.Type requires.
func (v Value) Encode() []byte {
	return v.appendTo(nil)
}

func (v Value) appendTo(buf []byte) []byte {
	buf = append(buf, byte(v.Type))

	switch v.Type {
	case SimpleString, SimpleError:
		buf = append(buf, v.Data.(string)...)

	case Integer:
		buf = strconv.AppendInt(buf, v.Data.(int64), 10)

	case BulkString:
		if v.Data == nil {
			buf = append(buf, "-1"...)
			break
		}
		s := v.Data.(string)
		buf = strconv.AppendInt(buf, int64(len(s)), 10)
		buf = append(buf, CRLF...)
		buf = append(buf, s...)

	case Array:
		if v.Data == nil {
			buf = append(buf, "-1"...)
			break
		}
		elems := v.Data.([]Value)
		buf = strconv.AppendInt(buf, int64(len(elems)), 10)
		buf = append(buf, CRLF...)
		for _, elem := range elems {
			buf = elem.appendTo(buf)
		}
		return buf
	}

	return append(buf, CRLF...)
}
