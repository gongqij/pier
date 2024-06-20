package tcp

import (
	"fmt"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/sirupsen/logrus"
	"net"
	"sync"
)

type TcpServer struct {
	port     int64 // 监听的端口
	listener net.Listener

	writeHttpCh   chan *common.Data
	newConnFunc   newConn
	closeConnFunc closeConn

	stopped chan struct{}
	wg      sync.WaitGroup
	logger  logrus.FieldLogger
}

func NewTcpServer(port int64, logger logrus.FieldLogger, newConnFunc newConn, closeConnFunc closeConn) (*TcpServer, error) {
	return &TcpServer{
		port:          port,
		writeHttpCh:   make(chan *common.Data, 1),
		stopped:       make(chan struct{}),
		logger:        logger,
		newConnFunc:   newConnFunc,
		closeConnFunc: closeConnFunc,
	}, nil
}

func (srv *TcpServer) WriteHttpCh() chan *common.Data {
	return srv.writeHttpCh
}

func (srv *TcpServer) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%v", srv.port))
	if err != nil {
		srv.logger.Errorf("[Server] Error listening: %s", err.Error())
		return err
	}
	srv.listener = listener
	srv.logger.Infof("[Server] TCP Listening on port %v", srv.port)

	srv.wg.Add(1)
	go srv.acceptConn()

	return nil
}

func (srv *TcpServer) Stop() error {
	srv.logger.Info("[Server] start to stop TCP server...")
	close(srv.stopped)
	if srv.listener != nil {
		if err := srv.listener.Close(); err != nil {
			srv.logger.Errorf("listener stop error: %s, should release port manually", err.Error())
		}
	}
	srv.wg.Wait()
	srv.logger.Info("[Server] successfully stop TCP server!")
	return nil
}

func (srv *TcpServer) acceptConn() {
	defer srv.wg.Done()

	for {
		select {
		case <-srv.stopped:
			return
		default:
			// 等待客户端连接
			conn, err := srv.listener.Accept()
			if err != nil {
				srv.logger.Warningf("[Server] Error accepting: %s", err.Error())
				return
			}

			connUuid := srv.newConnFunc(conn)

			srv.logger.Infof("[Server] Received Inbound connection, uuid: %s", connUuid)

			// 3. 向目标代理发送消息让其去连接目标节点(通过http发送)，消息里的地址信息由http模块赋值
			data := &common.Data{
				Typ:  common.TcpProxyTypeConnect,
				Uuid: connUuid,
			}
			srv.writeHttpCh <- data

			// 处理客户端连接
			srv.wg.Add(1)
			go srv.readloop(conn, connUuid)
		}
	}
}

func (srv *TcpServer) readloop(conn net.Conn, connUuid string) {
	defer func() {
		srv.closeConnFunc(connUuid)
		srv.logger.Infof("[Server] Close connection uuid: %s", connUuid)
		srv.wg.Done()
	}()

	for {
		select {
		case <-srv.stopped:
			return
		default:
			buffer := make([]byte, 1024*10)
			n, err := conn.Read(buffer)
			if err != nil {
				srv.logger.Warningf("[Server] read tcp error: %s", err.Error())
				return
			}

			srv.writeHttpCh <- &common.Data{
				Typ:     common.TcpProxyTypeForward,
				Uuid:    connUuid,
				Content: buffer[:n],
			}
		}
	}
}
