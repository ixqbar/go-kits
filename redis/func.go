package redis

import (
	"errors"
	"fmt"
	"io"
	"strconv"
)

func malformed(expected string, got string) error {
	return fmt.Errorf("Mailformed request:'%s does not match %s\\r\\n'", got, expected)
}

func malformedLength(expected int, got int) error {
	return fmt.Errorf("Mailformed request: argument length '%d does not match %d\\r\\n'", got, expected)
}

func malformedMissingCRLF() error {
	return fmt.Errorf("Mailformed request: line should end with \\r\\n")
}

func SeqMapReply(h *SeqMap) (*MultiBulkReply, error) {
	values := make([]interface{}, h.Len()*2)
	i := 0
	for _, key := range h.Keys {
		values[i] = []byte(key)
		values[i+1] = h.Data[key]
		i += 2
	}

	return &MultiBulkReply{values: values}, nil
}

func hashValueReply(h map[string][]byte) (*MultiBulkReply, error) {
	m := make(map[string]interface{})
	for k, v := range h {
		m[k] = v
	}

	return MultiBulkFromMap(m), nil
}

func MultiBulkFromMap(m map[string]interface{}) *MultiBulkReply {
	values := make([]interface{}, len(m)*2)
	i := 0
	for key, val := range m {
		values[i] = []byte(key)
		values[i+1] = val
		i += 2
	}

	return &MultiBulkReply{values: values}
}

func writeBytes(value interface{}, w io.Writer) (int64, error) {
	if value == nil {
		n, err := w.Write([]byte("$-1\r\n"))
		return int64(n), err
	}

	switch v := value.(type) {
	case string:
		if len(v) == 0 {
			n, err := w.Write([]byte("$-1\r\n"))
			return int64(n), err
		}
		wrote, err := w.Write([]byte("$" + strconv.Itoa(len(v)) + "\r\n"))
		if err != nil {
			return int64(wrote), err
		}
		wroteBytes, err := w.Write([]byte(v))
		if err != nil {
			return int64(wrote + wroteBytes), err
		}
		wroteCrLf, err := w.Write([]byte("\r\n"))
		return int64(wrote + wroteBytes + wroteCrLf), err
	case []byte:
		if len(v) == 0 {
			n, err := w.Write([]byte("$-1\r\n"))
			return int64(n), err
		}
		wrote, err := w.Write([]byte("$" + strconv.Itoa(len(v)) + "\r\n"))
		if err != nil {
			return int64(wrote), err
		}
		wroteBytes, err := w.Write(v)
		if err != nil {
			return int64(wrote + wroteBytes), err
		}
		wroteCrLf, err := w.Write([]byte("\r\n"))
		return int64(wrote + wroteBytes + wroteCrLf), err
	case int:
		wrote, err := w.Write([]byte(":" + strconv.Itoa(v) + "\r\n"))
		if err != nil {
			return int64(wrote), err
		}
		return int64(wrote), err
	}

	return 0, errors.New("Invalid type sent to writeBytes")
}

func writeMultiBytes(values []interface{}, w io.Writer) (int64, error) {
	if values == nil {
		return 0, errors.New("Nil in multi bulk replies are not ok")
	}

	wrote, err := w.Write([]byte("*" + strconv.Itoa(len(values)) + "\r\n"))
	if err != nil {
		return int64(wrote), err
	}

	wrote64 := int64(wrote)
	for _, v := range values {
		wroteBytes, err := writeBytes(v, w)
		if err != nil {
			return wrote64 + wroteBytes, err
		}
		wrote64 += wroteBytes
	}

	return wrote64, err
}
