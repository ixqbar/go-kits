package redis

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"net"
)

func parseRequest(conn net.Conn) (*Request, error) {
	r := bufio.NewReader(conn)

	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}

	var argsCount int

	if line[0] == '*' {
		if _, err := fmt.Sscanf(line, "*%d\r\n", &argsCount); err != nil {
			return nil, malformed("*<numberOfArguments>", line)
		}

		firstArg, err := readArgument(r)
		if err != nil {
			return nil, err
		}

		args := make([][]byte, argsCount-1)
		for i := 0; i < argsCount-1; i += 1 {
			if args[i], err = readArgument(r); err != nil {
				return nil, err
			}
		}

		return &Request{
			Name : strings.ToLower(string(firstArg)),
			Args : args,
			Conn : conn,
		}, nil
	}

	fields := strings.Split(strings.Trim(line, "\r\n"), " ")

	var args [][]byte
	if len(fields) > 1 {
		for _, arg := range fields[1:] {
			args = append(args, []byte(arg))
		}
	}

	return &Request{
		Name : strings.ToLower(string(fields[0])),
		Args : args,
		Conn : conn,
	}, nil

}

func readArgument(r *bufio.Reader) ([]byte, error) {

	line, err := r.ReadString('\n')
	if err != nil {
		return nil, malformed("$<argumentLength>", line)
	}
	var argSize int
	if _, err := fmt.Sscanf(line, "$%d\r\n", &argSize); err != nil {
		return nil, malformed("$<argumentSize>", line)
	}

	data, err := ioutil.ReadAll(io.LimitReader(r, int64(argSize)))
	if err != nil {
		return nil, err
	}

	if len(data) != argSize {
		return nil, malformedLength(argSize, len(data))
	}

	if b, err := r.ReadByte(); err != nil || b != '\r' {
		return nil, malformedMissingCRLF()
	}

	if b, err := r.ReadByte(); err != nil || b != '\n' {
		return nil, malformedMissingCRLF()
	}

	return data, nil
}
