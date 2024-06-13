package signal

type QuitMainSignal interface {
	ReleaseMain()
}

type MockQuitMainSignal struct{}

func (m *MockQuitMainSignal) ReleaseMain() {}
