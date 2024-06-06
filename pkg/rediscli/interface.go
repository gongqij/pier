package rediscli

import "errors"

type Wrapper interface {
	SendLock() bool
	SendUnlock() bool
}

type MockWrapperImpl struct{}

func (m *MockWrapperImpl) SendLock() bool {
	return true
}

func (m *MockWrapperImpl) SendUnlock() bool {
	return true
}

var ErrClosed = errors.New("wrapper closed")
