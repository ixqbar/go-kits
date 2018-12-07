package redis

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"reflect"
	"strings"
	"time"
)

const (
	CLIENT_STATE_IS_CONNECTED    = 1
	CLIENT_STATE_IS_DISCONNECTED = 2
)

type HandlerFn func(r *Request, c *Client) (ReplyWriter, error)
type CheckerFn func(request *Request) (reflect.Value, ReplyWriter)

type Server struct {
	addr     string
	methods  map[string]HandlerFn
	listener *net.TCPListener
	cm       *ConnectionManager
	running  bool
	handler  Handler
}

func NewServer(addr string, handler Handler) (*Server, error) {
	srv := &Server{
		addr:     addr,
		methods:  make(map[string]HandlerFn),
		listener: nil,
		cm:       NewConnectionManager(),
		running:  false,
		handler:  handler,
	}

	rh := reflect.TypeOf(handler)
	for i := 0; i < rh.NumMethod(); i++ {
		method := rh.Method(i)

		if handler.CheckShield(method.Name) {
			continue
		}

		handlerFn, err := srv.createHandlerFn(handler, &method)
		if err != nil {
			Logger.Print(err)
			return nil, err
		}

		srv.methods[strings.ToLower(method.Name)] = handlerFn
	}

	return srv, nil
}

func (srv *Server) Start() error {
	addr, err := net.ResolveTCPAddr("tcp", srv.addr)
	if err != nil {
		return fmt.Errorf("fail to resolve addr: %v", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("fail to listen tcp: %v", err)
	}

	srv.listener = listener
	srv.running = true

	return srv.acceptLoop()
}

func (srv *Server) acceptLoop() error {
	defer srv.listener.Close()

	for {
		conn, err := srv.listener.Accept()

		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				break
			}
			continue
		}

		go func() {
			clientAddr := "unknown"

			switch co := conn.(type) {
			case *net.UnixConn:
				f, err := conn.(*net.UnixConn).File()
				if err == nil {
					clientAddr = f.Name()
				}
			default:
				clientAddr = co.RemoteAddr().String()
			}

			srv.cm.Add(1)
			err := srv.handleConn(conn, clientAddr)
			if err != nil {
				Logger.Printf("handle connection %s failed %s", clientAddr, err)
			}
			srv.cm.Done()
		}()
	}

	return nil
}

func (srv *Server) Stop(timeout uint) error {
	if srv.running == false {
		return nil
	}

	srv.listener.SetDeadline(time.Now())

	defer Logger.Printf("redis server stop at %s", srv.addr)

	wait := make(chan struct{})

	defer close(wait)

	go func() {
		srv.cm.Wait()
		wait <- struct{}{}
	}()

	select {
	case <-time.After(time.Second * time.Duration(timeout)):
		return ErrStopServerTimeout
	case <-wait:
		return nil
	}
}

func (srv *Server) handleConn(conn net.Conn, clientAddr string) (err error) {
	clientChannel := make(chan struct{})
	client := &Client{
		Conn:         conn,
		DB:           0,
		Time:         time.Now(),
		Host:         clientAddr,
		UseSubscribe: false,
	}

	Logger.Printf("client %s connected", clientAddr)
	go srv.handler.CallClientStateChangedFunc(client, CLIENT_STATE_IS_CONNECTED)

	defer func() {
		Logger.Printf("client %s closed", clientAddr)
		if client.UseSubscribe {
			client.Handler.ClearSubscribe(clientAddr)
		}
		srv.handler.CallClientStateChangedFunc(client, CLIENT_STATE_IS_DISCONNECTED)
		close(clientChannel)
		conn.Close()
	}()

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
		request.Channel = clientChannel
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
		Logger.Printf("not found handle method `%s`", r.Name)
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
		m := len(request.Args)

		if n >= 2 && f.Type().In(1).AssignableTo(reflect.TypeOf(client)) {
			input = append(input, reflect.ValueOf(client))
			n -= 2
		} else {
			n -= 1
		}

		if n < m {
			return ErrWrongArgsNumber, nil
		} else {
			for i := 0; i < n-m; i++ {
				request.Args = append(request.Args, nil)
			}
		}

		for _, checker := range checkers {
			value, reply := checker(request)
			if reply != nil {
				Logger.Printf("error at checker and response %v", reply)
				return reply, nil
			}

			input = append(input, value)
		}

		var monitorString string
		if len(request.Args) > 0 {
			monitorString = fmt.Sprintf("%s \"%s\" \"%s\"",
				request.Host,
				request.Name,
				bytes.Join(request.Args, []byte{'"', ' ', '"'}))
		} else {
			monitorString = fmt.Sprintf("%s \"%s\"", request.Host, request.Name)
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
			Logger.Printf("%s do command `%s` failed with `%v`", request.Host, request.Name, err)
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
	case []string:
		m := make([]interface{}, len(v), cap(v))
		for i, elem := range v {
			m[i] = elem
		}
		return &MultiBulkReply{values: m}, nil
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
	case *ChannelWriter:
		return v, nil
	case *MultiChannelWriter:
		for _, mcw := range v.ChannelWriters {
			mcw.ClientRequest = r
		}
		return v, nil
	default:
		return nil, fmt.Errorf("Unsupported type: %s (%T)", v, v)
	}
}
