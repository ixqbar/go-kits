package redis

import (
	"fmt"
	"net"
	"reflect"
	"strings"
	"errors"
	"time"
	"bytes"
)

type HandlerFn func(r *Request, c *Client) (ReplyWriter, error)
type CheckerFn func(request *Request) (reflect.Value, ReplyWriter)

type Server struct {
	proto   string
	addr    string
	methods map[string]HandlerFn
	socket  interface{}
	cm      *ConnectionManager
}

func NewServer(addr string, handler Handler) (*Server, error) {
	srv := &Server{
		proto   : "",
		addr    : addr,
		methods : make(map[string]HandlerFn),
		socket  : nil,
		cm      : NewConnectionManager(),
	}

	rh := reflect.TypeOf(handler)
	for i := 0; i < rh.NumMethod(); i++ {
		method := rh.Method(i)

		if handler.CheckShield(method.Name) {
			continue
		}

		handlerFn, err := srv.createHandlerFn(handler, &method)
		if err != nil {
			return nil, err
		}

		srv.methods[strings.ToLower(method.Name)] = handlerFn
	}

	return srv, nil
}

func (srv *Server) Start() error {
	if strings.Contains(srv.addr, ":") {
		addr, err := net.ResolveTCPAddr("tcp", srv.addr)
		if err != nil {
			return fmt.Errorf("fail to resolve addr: %v", err)
		}

		sock, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return fmt.Errorf("fail to listen tcp: %v", err)
		}

		srv.socket = sock
		srv.proto = "tcp"
	} else {
		addr, err := net.ResolveUnixAddr("unix", srv.addr)
		if err != nil {
			return fmt.Errorf("fail to resolve addr: %v", err)
		}

		sock, err := net.ListenUnix("unix", addr)
		if err != nil {
			return fmt.Errorf("fail to listen tcp: %v", err)
		}

		srv.socket = sock
		srv.proto = "unix"
	}

	return srv.acceptLoop()
}

func (srv *Server) acceptLoop() error {
	defer func() {
		if (srv.proto == "tcp") {
			srv.socket.(*net.TCPListener).Close()
		} else {
			srv.socket.(*net.UnixListener).Close()
		}
	}()

	for {
		conn, err := func() (net.Conn, error) {
			if (srv.proto == "tcp") {
				return srv.socket.(*net.TCPListener).Accept()
			} else {
				return srv.socket.(*net.UnixListener).Accept()
			}
		}()

		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				break;
			}
			continue
		}

		go func() {
			srv.cm.Add(1)
			srv.handleConn(conn)
			srv.cm.Done()
		}()
	}

	return nil
}

func (srv *Server) Stop(timeout uint) error {
	if (srv.proto == "tcp") {
		srv.socket.(*net.TCPListener).SetDeadline(time.Now())
	} else {
		srv.socket.(*net.UnixListener).SetDeadline(time.Now())
	}

	tt := time.NewTimer(time.Second * time.Duration(timeout))
	wait := make(chan struct{})
	go func() {
		srv.cm.Wait()
		wait <- struct{}{}
	}()

	select {
	case <-tt.C:
		return ErrStopServerTimeout
	case <-wait:
		return nil
	}
}

func (srv *Server) handleConn(conn net.Conn) (err error) {
	clientChan := make(chan struct{})

	defer func() {
		conn.Close()
		close(clientChan)
	}()

	var clientAddr string

	switch co := conn.(type) {
	case *net.UnixConn:
		f, err := conn.(*net.UnixConn).File()
		if err != nil {
			return err
		}
		clientAddr = f.Name()
	default:
		clientAddr = co.RemoteAddr().String()
	}

	client := &Client{
		Conn:conn,
		DB:0,
		ConnectedTime:time.Now(),
	}

	for {
		request, err := parseRequest(conn)
		if err != nil {
			reply := NewErrorReply(err.Error())

			if _, err = reply.WriteTo(conn); err != nil {
				return err
			}

			return err
		}

		request.Host = clientAddr
		reply, err := srv.apply(request, client)
		if err != nil {
			reply = NewErrorReply(err.Error())
		}

		if _, err = reply.WriteTo(conn); err != nil {
			return err
		}
	}

	return nil
}

func (srv *Server) apply(r *Request, c *Client) (ReplyWriter, error) {
	if srv == nil || srv.methods == nil {
		return ErrMethodNotSupported, nil
	}

	fn, exists := srv.methods[strings.ToLower(r.Name)]
	if !exists {
		return ErrMethodNotSupported, nil
	}

	return fn(r, c)
}

func (srv *Server) createHandlerFn(autoHandler interface{}, f *reflect.Method) (HandlerFn, error) {
	errorType := reflect.TypeOf(srv.createHandlerFn).Out(1)
	fType := f.Func.Type()
	checkers, err := createCheckers(autoHandler, &f.Func)
	if err != nil {
		return nil, err
	}

	if fType.NumOut() == 0 {
		return nil, errors.New("Not enough return values for method " + f.Name)
	}

	if fType.NumOut() > 2 {
		return nil, errors.New("Too many return values for method " + f.Name)
	}

	if t := fType.Out(fType.NumOut() - 1); t != errorType {
		return nil, fmt.Errorf("The last return value must be an error (not %s)", t)
	}

	return srv.handlerFn(autoHandler, &f.Func, checkers)
}

func (srv *Server) handlerFn(autoHandler interface{}, f *reflect.Value, checkers []CheckerFn) (HandlerFn, error) {
	return func(request *Request, client *Client) (ReplyWriter, error) {
		input := []reflect.Value{reflect.ValueOf(autoHandler)}

		n := f.Type().NumIn()
		if n - 1 != len(request.Args) {
			if n >= 2 && f.Type().In(1).AssignableTo(reflect.TypeOf(client)) {
				input = append(input, reflect.ValueOf(client))
			} else {
				return ErrWrongArgsNumber,nil
			}
		}

		for _, checker := range checkers {
			value, reply := checker(request)
			if reply != nil {
				return reply, nil
			}

			input = append(input, value)
		}

		var monitorString string
		if len(request.Args) > 0 {
			monitorString = fmt.Sprintf("%.6f [%s] \"%s\" \"%s\"",
				float64(time.Now().UTC().UnixNano())/1e9,
				request.Host,
				request.Name,
				bytes.Join(request.Args, []byte{'"', ' ', '"'}))
		} else {
			monitorString = fmt.Sprintf("%.6f [%s] \"%s\"",
				float64(time.Now().UTC().UnixNano())/1e9,
				request.Host,
				request.Name)
		}

		Logger.Printf("%s", monitorString)

		var result []reflect.Value

		if f.Type().NumIn() == 0 {
			input = []reflect.Value{}
		} else if f.Type().In(0).AssignableTo(reflect.TypeOf(autoHandler)) == false {
			input = input[1:]
		}

		if f.Type().IsVariadic() {
			result = f.CallSlice(input)
		} else {
			result = f.Call(input)
		}

		var ret interface{}
		if ierr := result[len(result)-1].Interface(); ierr != nil {
			err := ierr.(error)
			return NewErrorReply(err.Error()), nil
		}

		if len(result) > 1 {
			ret = result[0].Interface()
			return srv.createReply(request, ret)
		}

		return &StatusReply{code: "OK"}, nil
	}, nil
}

func (srv *Server) createReply(r *Request, val interface{}) (ReplyWriter, error) {
	switch v := val.(type) {
	case []interface{}:
		return &MultiBulkReply{values: v}, nil
	case string:
		return &BulkReply{value: []byte(v)}, nil
	case [][]byte:
		if v, ok := val.([]interface{}); ok {
			return &MultiBulkReply{values: v}, nil
		}
		m := make([]interface{}, len(v), cap(v))
		for i, elem := range v {
			m[i] = elem
		}
		return &MultiBulkReply{values: m}, nil
	case []byte:
		return &BulkReply{value: v}, nil
	case map[string][]byte:
		return hashValueReply(v)
	case map[string]interface{}:
		return MultiBulkFromMap(v), nil
	case int:
		return &IntegerReply{number: v}, nil
	case *StatusReply:
		return v, nil
	case *SeqMap:
		return SeqMapReply(v)
	default:
		return nil, fmt.Errorf("Unsupported type: %s (%T)", v, v)
	}
}
