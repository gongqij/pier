package http

import (
	"fmt"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/meshplus/pier/internal/proxy/config"
	"github.com/meshplus/pier/pkg/rediscli"
	"github.com/sirupsen/logrus"
	msp "github.com/ultramesh/flato-msp"
	mspcert "github.com/ultramesh/flato-msp-cert"
	"github.com/ultramesh/flato-msp-cert/plugin"
	"golang.org/x/net/http2"
	"math"
	"net"
	"net/http"
	"sync"
)

const maxRequestId = math.MaxUint16

type Http struct {
	send chan *common.Data // 向 tcp 层发送数据
	recv chan *common.Data // 接收来自 tcp 层的数据

	httpSrv   *http.Server
	httpCli   *http.Client
	redisCli  rediscli.Wrapper
	requestID uint16 // 范围 1 ~ maxRequestId
	nodes     []*node

	conf *config.ProxyConfig
	log  logrus.FieldLogger

	wg      sync.WaitGroup
	stopped chan struct{}
}

func New(redisCli rediscli.Wrapper, send chan *common.Data, recv chan *common.Data, conf *config.ProxyConfig, log logrus.FieldLogger) (*Http, error) {
	handler, err := newHttpServerHandler(send, log, conf)
	if err != nil {
		return nil, err
	}
	mux := &http.ServeMux{}
	mux.Handle("/", handler)

	nodes := make([]*node, len(conf.HTTPRemoteAddress))
	for i, v := range conf.HTTPRemoteAddress {
		nodes[i] = newNode(v, conf.SecurityTLSEnable, i)
	}

	return &Http{
		httpSrv: &http.Server{
			Handler:     mux,
			ReadTimeout: conf.HTTPRequestTimeout,
		},
		httpCli:   newClient(conf, log),
		redisCli:  redisCli,
		requestID: 1,
		nodes:     nodes,
		send:      send,
		recv:      recv,
		conf:      conf,
		log:       log,
	}, nil
}

func (h *Http) Start() error {
	err := h.startHttpServer()
	if err != nil {
		h.log.Errorf("start http server error: %s", err.Error())
		return err
	}
	h.wg.Add(1)
	go h.readTcpLoop()
	return nil
}

func (h *Http) Stop() error {
	_ = h.httpSrv.Close()
	close(h.stopped)
	h.wg.Wait()
	h.log.Info("successfully stop http layer!")
	return nil
}

func (h *Http) startHttpServer() error {
	var (
		listener net.Listener
		lerr     error
	)

	if h.conf.SecurityTLSEnable {
		tls := mspcert.GetTLS(plugin.GetSoftwareEngine(msp.NoneKeyStore))
		tls, nerr := tls.NewServer(h.conf.SecurityTLSCAPath, h.conf.SecurityTLSCertPath, h.conf.SecurityTLSPrivPath, h.conf.HTTP2Enable)
		if nerr != nil {
			h.log.Errorf("create tls server error: %s", nerr.Error())
			return nerr
		}
		// start http listener with secure connection and http/2
		if listener, lerr = tls.Listen("tcp", fmt.Sprintf(":%d", h.conf.HTTPPort)); lerr != nil {
			h.log.Errorf("failed to listen tls service, Err: %v", lerr)
			return lerr
		}
		if h.conf.HTTP2Enable {
			// http2, https
			h.log.Infof("starting http/2 service at port %v ... , secure connection is enabled.", h.conf.HTTPPort)
			_ = http2.ConfigureServer(h.httpSrv, &http2.Server{})
		} else {
			// http1.1, https
			h.log.Infof("starting http/1.1 service at port %v ... , secure connection is enabled.", h.conf.HTTPPort)
		}
	} else {
		listener, lerr = net.Listen("tcp", fmt.Sprintf(":%d", h.conf.HTTPPort))
	}

	if lerr != nil {
		return lerr
	}
	go h.httpSrv.Serve(listener)

	return nil
}

func (h *Http) readTcpLoop() {
	defer h.wg.Done()

	for {
		select {
		case <-h.stopped:
			return
		case data := <-h.recv:
			// receive data from tcp layer
			h.sendHttpRequest(data)
		}
	}
}
