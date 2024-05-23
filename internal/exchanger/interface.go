package exchanger

import "github.com/meshplus/pier/internal/peermgr"

type IExchanger interface {
	// Start starts the service of exchanger
	Start() error

	// Stop stops the service of exchanger
	Stop() error

	// RenewPeerManager will set new peerManager instance to destAdapter
	RenewPeerManager(pm peermgr.PeerManager)
}
