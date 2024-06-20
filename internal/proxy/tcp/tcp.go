package tcp

import (
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/meshplus/pier/internal/proxy/config"
	"github.com/sirupsen/logrus"
	"net"
	"reflect"
	"sync"
)

type newConn func(conn net.Conn) string
type newConnWithUuid func(connUuid string, conn net.Conn) error
type closeConn func(connUuid string)

type Tcp struct {
	send chan *common.Data // 向 http 层发送数据
	recv chan *common.Data // 接收来自 http 层的数据

	connMgr *connMgr

	tSrv []*TcpServer
	tCli *TcpClient
	conf *config.ProxyConfig
	log  logrus.FieldLogger

	wg      sync.WaitGroup
	stopped chan struct{}
}

func New(send chan *common.Data, recv chan *common.Data, conf *config.ProxyConfig, log logrus.FieldLogger) (*Tcp, error) {
	connMgr := NewConnMgr()

	tcpmgr := &Tcp{
		send:    send,
		recv:    recv,
		conf:    conf,
		log:     log,
		connMgr: connMgr,
		stopped: make(chan struct{}),
		tSrv:    make([]*TcpServer, 0),
	}

	for listen, _ := range conf.ReverseProxys {
		tSrv, err := NewTcpServer(listen, log, connMgr.newConn, connMgr.closeConn)
		if err != nil {
			return nil, err
		}
		tcpmgr.tSrv = append(tcpmgr.tSrv, tSrv)
	}

	tCli, err := NewTcpClient(log, connMgr.newConnWithUuid, connMgr.closeConn)
	if err != nil {
		return nil, err
	}
	tcpmgr.tCli = tCli

	return tcpmgr, nil
}

func (t *Tcp) Start() error {
	for _, tsrv := range t.tSrv {
		if err := tsrv.Start(); err != nil {
			return err
		}
	}

	t.wg.Add(2)
	go t.readFromHttpLoop()
	go t.sendHttpLoop()
	return nil
}

func (t *Tcp) Stop() error {
	t.log.Info("start to stop tcp layer...")
	t.log.Info("stop all TCP connection...")
	t.connMgr.closeAllConn()
	t.tCli.Stop()
	for _, tsrv := range t.tSrv {
		if err := tsrv.Stop(); err != nil {
			t.log.Errorf("stop TcpServer error: %s", err.Error())
			continue
		}
	}

	close(t.stopped)
	t.wg.Wait()
	t.log.Info("successfully stop tcp layer!")
	return nil
}

// 监听http层的消息
func (t *Tcp) readFromHttpLoop() {
	defer t.wg.Done()

	for {
		select {
		case <-t.stopped:
			return
		case data := <-t.recv:
			t.handleDataFromHttp(data)
		}
	}
}

func (t *Tcp) sendHttpLoop() {
	defer t.wg.Done()

	cases := make([]reflect.SelectCase, len(t.tSrv)+3)
	// Add stopped channel to cases
	cases[0] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(t.stopped)}
	// Add connMgr.getNotifyCloseCh to cases
	cases[1] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(t.connMgr.getNotifyCloseCh())}
	// Add tCli.WriteHttpCh to cases
	cases[2] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(t.tCli.WriteHttpCh())}
	// Add all tSrv.WriteHttpCh to cases
	for i, srv := range t.tSrv {
		cases[i+3] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(srv.WriteHttpCh())}
	}

	for {
		chosen, recv, ok := reflect.Select(cases)
		if !ok {
			// Handle channel close
			t.log.Warningf("channel %v closed", chosen)
			if chosen == 0 {
				return
			}
			//todo 这里应该也return？
			continue
		}
		switch chosen {
		case 0:
			// stopped channel received
			return
		case 1:
			// connMgr.getNotifyCloseCh received
			t.send <- &common.Data{
				Typ:  common.TcpProxyTypeDisconnect,
				Uuid: recv.String(),
			}
		case 2:
			// tCli.WriteHttpCh received
			t.send <- recv.Interface().(*common.Data)
		default:
			// tSrv.WriteHttpCh received
			t.send <- recv.Interface().(*common.Data)
		}
	}
}

func (t *Tcp) handleDataFromHttp(data *common.Data) {
	switch data.Typ {
	case common.TcpProxyTypeConnect:
		t.tCli.handleConnectRequest(data)
	case common.TcpProxyTypeDisconnect:
		t.log.Infof("receive TcpProxyTypeDisconnect from remote conn uuid %s", data.Uuid)
		t.connMgr.closeConn(data.Uuid)
	case common.TcpProxyTypeConnectACK, common.TcpProxyTypeForward:
		if data.Typ == common.TcpProxyTypeConnectACK {
			t.log.Infof("[Server] receive connect ACK from remote conn uuid: %s", data.Uuid)
			break
		} else {
			t.log.Infof("receive forwarded data from remote conn uuid %s", data.Uuid)
		}
		conn := t.connMgr.getConn(data.Uuid)
		if conn == nil {
			t.log.Errorf("can't get conn with uuid %s", data.Uuid)
			break
		}

		if _, werr := conn.Write(data.Content); werr != nil {
			t.log.Errorf("[tcp] write error: %s", werr.Error())
			break
		}
	}
}
