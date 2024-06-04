package rediscli

import "errors"

type Wrapper interface {
	SendLock() bool
	SendUnlock() bool
}

var ErrClosed = errors.New("wrapper closed")
