package redis

import (
	"errors"
	"fmt"
	"reflect"
)

func createCheckers(autoHandler interface{}, f *reflect.Value) ([]CheckerFn, error) {
	checkers := []CheckerFn{}
	mtype := f.Type()

	start := 0
	n := mtype.NumIn()

	if n > 1 && mtype.In(1).AssignableTo(reflect.TypeOf(&Client{})) {
		start = 2
	} else if n > 0 && mtype.In(0).AssignableTo(reflect.TypeOf(autoHandler)) {
		start = 1
	}

	for i := start; i < mtype.NumIn(); i += 1 {
		switch mtype.In(i) {
		case reflect.TypeOf(""):
			checkers = append(checkers, stringChecker(i-start))
		case reflect.TypeOf([]string{}):
			checkers = append(checkers, stringSliceChecker(i-start))
		case reflect.TypeOf([]byte{}):
			checkers = append(checkers, byteChecker(i-start))
		case reflect.TypeOf([][]byte{}):
			checkers = append(checkers, byteSliceChecker(i-start))
		case reflect.TypeOf(map[string][]byte{}):
			if i != mtype.NumIn()-1 {
				return nil, errors.New("Map should be the last argument")
			}
			checkers = append(checkers, mapChecker(i-start))
		case reflect.TypeOf(1):
			checkers = append(checkers, intChecker(i-start))
		case reflect.TypeOf(&Client{}):
			//ignore
		default:
			return nil, fmt.Errorf("Argument %d: wrong type %s (%s)", i, mtype.In(i), mtype.Name())
		}
	}

	return checkers, nil
}

func stringChecker(index int) CheckerFn {
	return func(request *Request) (reflect.Value, ReplyWriter) {
		v, err := request.GetString(index)
		if err != nil {
			return reflect.ValueOf(""), err
		}
		return reflect.ValueOf(v), nil
	}
}

func stringSliceChecker(index int) CheckerFn {
	return func(request *Request) (reflect.Value, ReplyWriter) {
		if !request.HasArgument(index) {
			return reflect.ValueOf([]string{}), nil
		} else {
			v, err := request.GetStringSlice(index)
			if err != nil {
				return reflect.ValueOf([]string{}), err
			}
			return reflect.ValueOf(v), nil
		}
	}
}

func byteChecker(index int) CheckerFn {
	return func(request *Request) (reflect.Value, ReplyWriter) {
		err := request.ExpectArgument(index)
		if err != nil {
			return reflect.ValueOf([]byte{}), err
		}
		return reflect.ValueOf(request.Args[index]), nil
	}
}

func byteSliceChecker(index int) CheckerFn {
	return func(request *Request) (reflect.Value, ReplyWriter) {
		if !request.HasArgument(index) {
			return reflect.ValueOf([][]byte{}), nil
		} else {
			return reflect.ValueOf(request.Args[index:]), nil
		}
	}
}

func mapChecker(index int) CheckerFn {
	return func(request *Request) (reflect.Value, ReplyWriter) {
		m, err := request.GetMap(index)
		return reflect.ValueOf(m), err
	}
}

func intChecker(index int) CheckerFn {
	return func(request *Request) (reflect.Value, ReplyWriter) {
		m, err := request.GetInteger(index)
		return reflect.ValueOf(m), err
	}
}
