package tcp

import (
	"fmt"
	"github.com/google/uuid"
	"net"
	"sync"
)

type connMgr struct {
	rwMux   sync.RWMutex
	connMap map[string]net.Conn

	notifyCloseCh chan string // conn uuid
}

func NewConnMgr() *connMgr {
	return &connMgr{
		connMap:       make(map[string]net.Conn),
		notifyCloseCh: make(chan string),
	}
}

func (cmgr *connMgr) newConn(conn net.Conn) string {
	cid := uuid.New()
	cmgr.rwMux.Lock()
	defer cmgr.rwMux.Unlock()

	cmgr.connMap[cid.String()] = conn

	return cid.String()
}

func (cmgr *connMgr) newConnWithUuid(connUuid string, conn net.Conn) error {
	cmgr.rwMux.Lock()
	defer cmgr.rwMux.Unlock()

	_, ok := cmgr.connMap[connUuid]
	if ok {
		return fmt.Errorf("failed to set conn because conn with uuid %s has existed", connUuid)
	}
	cmgr.connMap[connUuid] = conn
	return nil
}

func (cmgr *connMgr) getConn(connUuid string) net.Conn {
	cmgr.rwMux.RLock()
	defer cmgr.rwMux.RUnlock()
	return cmgr.connMap[connUuid]
}

func (cmgr *connMgr) closeAllConn() {
	cmgr.rwMux.Lock()
	defer cmgr.rwMux.Unlock()

	for _, conn := range cmgr.connMap {
		_ = conn.Close()
	}
	cmgr.connMap = make(map[string]net.Conn)
}

func (cmgr *connMgr) closeConn(connUuid string) {
	cmgr.rwMux.Lock()
	defer cmgr.rwMux.Unlock()

	conn, ok := cmgr.connMap[connUuid]
	if ok {
		_ = conn.Close()
		delete(cmgr.connMap, connUuid)
		cmgr.notifyCloseCh <- connUuid
	}
}

func (cmgr *connMgr) getNotifyCloseCh() chan string {
	return cmgr.notifyCloseCh
}
