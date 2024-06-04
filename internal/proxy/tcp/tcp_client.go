package tcp

import (
	"fmt"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type TcpClient struct {
	writeHttpCh chan *common.Data

	newConnWithUuidFunc newConnWithUuid
	closeConnFunc       closeConn

	wg      sync.WaitGroup
	stopped chan struct{}
	logger  logrus.FieldLogger
}

func NewTcpClient(logger logrus.FieldLogger, newConnWithUuidFunc newConnWithUuid, closeConnFunc closeConn) (*TcpClient, error) {
	return &TcpClient{
		writeHttpCh:         make(chan *common.Data),
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

// TODO 异常处理：对于所有error都需要告知对面代理这边连接建立失败了
func (tCli *TcpClient) handleConnectRequest(data *common.Data) {
	connUuid := data.Uuid
	// 1. 与目标节点建立TCP连接
	ip, port, err := net.SplitHostPort(string(data.Content))
	if err != nil {
		tCli.logger.Errorf("[Client] failed to parse network address, err: %s", err.Error())
		return
	}
	conn, cerr := tCli.Dial(ip, port)
	if cerr != nil {
		tCli.logger.Errorf("[Client] failed to dail %s:%s, err: %s", ip, port, cerr.Error())
		return
	}
	if nerr := tCli.newConnWithUuidFunc(connUuid, conn); nerr != nil {
		tCli.logger.Error(nerr)
		return
	}
	tCli.logger.Infof("[Client] Launch Outbound connection at %s:%s, uuid: %s", ip, port, connUuid)
	tCli.wg.Add(1)
	go tCli.readloop(conn, connUuid)

	// 2. 给对面代理发送ACK
	tCli.writeHttpCh <- &common.Data{
		Typ:     TcpProxyTypeConnectACK,
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
				Typ:     TcpProxyTypeForward,
				Uuid:    connUuid,
				Content: buffer[:n],
			}
		}
	}
}
