package http

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	json "github.com/json-iterator/go"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/meshplus/pier/internal/proxy/config"
	"github.com/meshplus/pier/pkg/rediscli"
	"github.com/meshplus/pier/pkg/redisha/signal"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/http2"
	"math"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

const maxRequestId = math.MaxUint16

const uri = "/pier/proxy"

type Http struct {
	send chan *common.Data // 向 tcp 层发送数据
	recv chan *common.Data // 接收来自 tcp 层的数据

	httpSrv   *http.Server
	httpCli   *http.Client
	redisCli  rediscli.Wrapper
	quitMain  signal.QuitMainSignal
	requestID uint16 // 范围 1 ~ maxRequestId
	nodes     []*node

	conf *config.ProxyConfig
	log  logrus.FieldLogger

	wg      sync.WaitGroup
	stopped chan struct{}
}

func New(redisCli rediscli.Wrapper, quitMain signal.QuitMainSignal, send chan *common.Data, recv chan *common.Data, conf *config.ProxyConfig, log logrus.FieldLogger) (*Http, error) {
	handler, err := newHttpServerHandler(send, log, conf)
	if err != nil {
		return nil, err
	}
	mux := &http.ServeMux{}
	mux.Handle(uri, handler)

	nodes := make([]*node, len(conf.RemoteAddress))
	for i, ip := range conf.RemoteAddress {
		n, nerr := newNode(ip, conf.RemoteHttpPort[i], conf.RemoteTcpPort[i], conf.SecurityTLSEnable, i)
		if nerr != nil {
			return nil, nerr
		}
		nodes[i] = n
	}

	return &Http{
		httpSrv: &http.Server{
			Handler:     mux,
			ReadTimeout: conf.HTTPRequestTimeout,
		},
		httpCli:   newClient(conf),
		redisCli:  redisCli,
		quitMain:  quitMain,
		requestID: 1,
		nodes:     nodes,
		send:      send,
		recv:      recv,
		stopped:   make(chan struct{}),
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
	h.wg.Add(2)
	go h.readTcpLoop()
	go h.switchSignalLoop()
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
		// load server certificate and key
		serverCert, err := tls.LoadX509KeyPair(h.conf.SecurityTLSCertPath, h.conf.SecurityTLSPrivPath)
		if err != nil {
			h.log.Errorf("failed to load server certificate and key: %v", err)
			return err
		}
		// load CA certificate
		caCert, err := os.ReadFile(h.conf.SecurityTLSCAPath)
		if err != nil {
			h.log.Errorf("failed to read CA certificate: %v", err)
			return err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		// create a TLS config
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{serverCert},
			ClientCAs:    caCertPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
		}
		// start http listener with secure connection and http/2
		if listener, lerr = tls.Listen("tcp", fmt.Sprintf(":%d", h.conf.HTTPPort), tlsConfig); lerr != nil {
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
		h.log.Infof("starting http/1.1 service at port %v ... , secure connection is disable.", h.conf.HTTPPort)
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
			h.log.Info("quit readTcpLoop")
			return
		case data := <-h.recv:
			// receive data from tcp layer
			h.sendHttpRequest(data)
		}
	}
}

// 定时去给对面主从proxy发送http请求，如果所有http地址都失败多次达到了阀值，则触发切换本地pier
func (h *Http) switchSignalLoop() {
	defer h.wg.Done()

	dur := h.conf.HTTPRetryDuration // 整个goroutine多久循环一次
	if dur == 0 {
		dur = 10 * time.Second
	}
	requestTimeout := h.conf.HTTPCancelTimeout // 单个http请求的超时时间
	if requestTimeout == 0 {
		requestTimeout = 5 * time.Second
	}
	allBadCountLimit := h.conf.HTTPAllFailedLimit // 所有http地址都失败的最大次数
	if allBadCountLimit == 0 {
		allBadCountLimit = 30
	}
	allBadCount := 0 // 统计所有http地址都失败的次数

	timer := time.NewTimer(dur)
	defer timer.Stop()

	for {
		select {
		case <-h.stopped:
			h.log.Info("quit switchSignalLoop")
			return
		case <-timer.C:
			badCount := 0
			for _, n := range h.nodes {
				url := n.httpUrl
				data := newJsonData(1, httpReconnectMsg, httpReconnectMsg, []byte("signal"))
				body, err := json.Marshal(data)
				if err != nil {
					h.log.Error(err)
					badCount++
					continue
				}
				req, nerr := newHttpRequest(http.MethodPost, url, body)
				ctx, cancel := context.WithCancel(req.Context())
				req.WithContext(ctx)
				if nerr != nil {
					h.log.Error(nerr)
					badCount++
					cancel()
					continue
				}
				h.log.Debug("send keepalive signal to " + url)
				requestTimer := time.NewTimer(requestTimeout)
				quitGoroutine := make(chan struct{})
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					select {
					case <-requestTimer.C:
						cancel()
					case <-quitGoroutine:
					}
				}()
				response, err := h.httpCli.Do(req)
				close(quitGoroutine)
				if err != nil {
					h.log.Debug(err)
					badCount++
					h.log.Infof("keepalive error: %s, failedCount = %v", err.Error(), badCount)
					continue
				}
				if response == nil {
					badCount++
					h.log.Errorf("response is nil, failedCount = %v", badCount)
					continue
				}
				if response.StatusCode == http.StatusOK {
					h.log.Debug("node " + url + " keepalive signal Success!")
				} else {
					badCount++
					h.log.Errorf("response status code is %v, failedCount = %v", response.StatusCode, badCount)
				}

				response.Body.Close()
				requestTimer.Stop()
				cancel()

				wg.Wait()
			}
			if badCount == len(h.nodes) {
				allBadCount++
			} else {
				allBadCount = 0
			}

			if allBadCount >= allBadCountLimit {
				// 不需要重置timer
				h.log.Info("all nodes failed, network may be broken, swap pier now")
				h.quitMain.ReleaseMain()
			} else {
				timer.Reset(dur)
			}
		}
	}
}
