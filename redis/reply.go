package redis

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"
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

type ChannelWriter struct {
	Name          string
	ChannelName   string
	FirstReply    []interface{}
	ReplyChannel  chan []interface{}
	ClientRequest *Request
}

func NewChannelWriter(name string, channelName string) *ChannelWriter {
	return &ChannelWriter{
		Name:        name,
		ChannelName: channelName,
		FirstReply: []interface{}{
			"subscribe",
			channelName,
			1,
		},
		ReplyChannel: make(chan []interface{}),
	}
}

func (c *ChannelWriter) PublishMessage(message []byte) error {
	select {
	case c.ReplyChannel <- []interface{}{
		"message",
		c.ChannelName,
		message,
	}:
	default:
		return errors.New(fmt.Sprintf("send on closed channel client %s", c.ClientRequest.Host))
	}

	return nil
}

func (c *ChannelWriter) WriteTo(w io.Writer) (int64, error) {
	totalBytes, err := writeMultiBytes(c.FirstReply, w)
	if err != nil {
		return totalBytes, err
	}

	ticker := time.NewTicker(15 * time.Second)

	defer func() {
		ticker.Stop()
		close(c.ReplyChannel)
	}()

	for {
		select {
		case <-ticker.C:
			wroteBytes, err := writeMultiBytes([]interface{}{"message", c.ChannelName, "PING"}, w)
			totalBytes += wroteBytes
			if err != nil {
				Logger.Printf("write replay to %s failed %s", c.ClientRequest.Host, err)
				return totalBytes, err
			}
			Logger.Printf("send ping message to subscribe client %s", c.ClientRequest.Host)
		case <-c.ClientRequest.Channel:
			Logger.Printf("client %s closed", c.ClientRequest.Host)
			return totalBytes, err
		case reply := <-c.ReplyChannel:
			Logger.Printf("read %s", reply)
			if reply == nil {
				return totalBytes, nil
			} else {
				wroteBytes, err := writeMultiBytes(reply, w)
				totalBytes += wroteBytes
				if err != nil {
					Logger.Printf("write replay to %s failed %s", c.ClientRequest.Host, err)
					return totalBytes, err
				}
			}
		}
	}

	return totalBytes, nil
}

type MultiChannelWriter struct {
	ChannelWriters []*ChannelWriter
}

func NewMultiChannelWriter(num int) *MultiChannelWriter {
	return &MultiChannelWriter{
		ChannelWriters: make([]*ChannelWriter, 0, num),
	}
}

func (c *MultiChannelWriter) WriteTo(w io.Writer) (n int64, err error) {
	chans := make(chan struct{}, len(c.ChannelWriters))
	for _, elem := range c.ChannelWriters {
		go func(elem io.WriterTo) {
			defer func() { chans <- struct{}{} }()
			if n2, err2 := elem.WriteTo(w); err2 != nil {
				n += n2
				err = err2
				return
			} else {
				n += n2
			}
		}(elem)
	}
	for i := 0; i < len(c.ChannelWriters); i++ {
		<-chans
	}
	return n, err
}
