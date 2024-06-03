package rediscli

import "errors"

type Wrapper interface {
	HeldLock() bool
}

var ErrClosed = errors.New("wrapper closed")
