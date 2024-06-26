package tcp

import (
	"fmt"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/sirupsen/logrus"
	"net"
	"strconv"
	"sync"
)

type TcpClient struct {
	localP2PPort int
	writeHttpCh  chan *common.Data

	newConnWithUuidFunc newConnWithUuid
	closeConnFunc       closeConn

	wg      sync.WaitGroup
	stopped chan struct{}
	logger  logrus.FieldLogger
}

func NewTcpClient(localP2PPort int, logger logrus.FieldLogger, newConnWithUuidFunc newConnWithUuid, closeConnFunc closeConn) (*TcpClient, error) {
	return &TcpClient{
		localP2PPort:        localP2PPort,
		writeHttpCh:         make(chan *common.Data, 1),
		stopped:             make(chan struct{}),
		logger:              logger,
		newConnWithUuidFunc: newConnWithUuidFunc,
		closeConnFunc:       closeConnFunc,
	}, nil
}

func (tCli *TcpClient) Stop() {
	close(tCli.stopped)
	tCli.wg.Wait()
}

func (tCli *TcpClient) WriteHttpCh() chan *common.Data {
	return tCli.writeHttpCh
}

func (tCli *TcpClient) Dial(ip, port string) (net.Conn, error) {
	// 连接到服务器
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", ip, port))
	if err != nil {
		tCli.logger.Errorf("[Client] Error connecting: %s", err.Error())
		return nil, err
	}

	return conn, nil
}

func (tCli *TcpClient) handleConnectRequest(data *common.Data) {
	connUuid := data.Uuid
	// 1. 与目标节点建立TCP连接
	conn, cerr := tCli.Dial("127.0.0.1", strconv.Itoa(tCli.localP2PPort))
	if cerr != nil {
		tCli.logger.Errorf("[Client] failed to dail 127.0.0.1:%v, err: %s", tCli.localP2PPort, cerr.Error())
		return
	}
	if nerr := tCli.newConnWithUuidFunc(connUuid, conn); nerr != nil {
		tCli.logger.Error(nerr)
		return
	}
	tCli.logger.Infof("[Client] Launch Outbound connection at 127.0.0.1:%v, uuid: %s", tCli.localP2PPort, connUuid)
	tCli.wg.Add(1)
	go tCli.readloop(conn, connUuid)

	// 2. 给对面代理发送ACK
	tCli.writeHttpCh <- &common.Data{
		Typ:     common.TcpProxyTypeConnectACK,
		Uuid:    connUuid,
		Content: []byte{1},
	}
}

func (tCli *TcpClient) readloop(conn net.Conn, connUuid string) {
	defer func() {
		tCli.closeConnFunc(connUuid)
		tCli.logger.Infof("[Client] Close connection uuid: %s", connUuid)
		tCli.wg.Done()
	}()

	for {
		select {
		case <-tCli.stopped:
			return
		default:
			buffer := make([]byte, 1024*10)
			n, err := conn.Read(buffer)
			if err != nil {
				tCli.logger.Warningf("[Client] read tcp error: %s", err.Error())
				return
			}

			tCli.writeHttpCh <- &common.Data{
				Typ:     common.TcpProxyTypeForward,
				Uuid:    connUuid,
				Content: buffer[:n],
			}
		}
	}
}
