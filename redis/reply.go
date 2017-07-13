package redis

import (
	"io"
	"strconv"
)

type ReplyWriter io.WriterTo

type ErrorReply struct {
	code    string
	message string
}

func NewErrorReply(message string) *ErrorReply {
	return &ErrorReply{code: "ERROR", message: message}
}

func (this *ErrorReply) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte("-" + this.code + " " + this.message + "\r\n"))
	return int64(n), err
}

func (this *ErrorReply) Error() string {
	return "-" + this.code + " " + this.message + "\r\n"
}

type StatusReply struct {
	code string
}

func (r *StatusReply) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte("+" + r.code + "\r\n"))
	return int64(n), err
}

type IntegerReply struct {
	number int
}

func (r *IntegerReply) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write([]byte(":" + strconv.Itoa(r.number) + "\r\n"))
	return int64(n), err
}

type BulkReply struct {
	value []byte
}

func (r *BulkReply) WriteTo(w io.Writer) (int64, error) {
	return writeBytes(r.value, w)
}

type MultiBulkReply struct {
	values []interface{}
}

func (r *MultiBulkReply) WriteTo(w io.Writer) (int64, error) {
	return writeMultiBytes(r.values, w)
}
