package redis

import "errors"

var (
	ErrMethodNotSupported   = NewErrorReply("Method is not supported")
	ErrNotEnoughArgs        = NewErrorReply("Not enough arguments for the command")
	ErrWrongArgsNumber      = NewErrorReply("Wrong number of arguments")
	ErrExpectInteger        = NewErrorReply("Expected integer")
	ErrExpectPositivInteger = NewErrorReply("Expected positive integer")
	ErrExpectMorePair       = NewErrorReply("Expected at least one key val pair")
	ErrExpectEvenPair       = NewErrorReply("Got uneven number of key val pairs")
	ErrStopServerTimeout    = errors.New("stop server wait timeout")
)
