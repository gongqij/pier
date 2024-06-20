package proxy

import (
	"errors"
	"fmt"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/meshplus/pier/internal/proxy/config"
	"github.com/meshplus/pier/internal/proxy/http"
	"github.com/meshplus/pier/internal/proxy/tcp"
	"github.com/meshplus/pier/pkg/rediscli"
	"github.com/meshplus/pier/pkg/redisha/signal"
	"github.com/sirupsen/logrus"
	"sync/atomic"
)

type Proxy interface {
	Start() error
	Stop() error
}

type ProxyImpl struct {
	config *config.ProxyConfig
	log    logrus.FieldLogger

	myTCP  *tcp.Tcp
	myHTTP *http.Http

	sendCh chan *common.Data
	recvCh chan *common.Data

	// 1 means the proxy has stopped or not started
	stopped uint64
}

func NewProxy(cfgFilePath string, redisCli rediscli.Wrapper, quitMain signal.QuitMainSignal, log logrus.FieldLogger) (Proxy, error) {
	conf, lerr := config.LoadProxyConfig(cfgFilePath)
	if lerr != nil {
		return nil, fmt.Errorf("failed to load config %s, err: %s", cfgFilePath, lerr.Error())
	}

	sendCh := make(chan *common.Data, 1)
	recvCh := make(chan *common.Data, 1)

	mtcp, nerr := tcp.New(recvCh, sendCh, conf, log)
	if nerr != nil {
		return nil, fmt.Errorf("failed to init tcp component, err: %s", nerr.Error())
	}
	mhttp, err := http.New(redisCli, quitMain, sendCh, recvCh, conf, log)
	if err != nil {
		return nil, err
	}

	return &ProxyImpl{
		config: conf,
		log:    log,
		myTCP:  mtcp,
		myHTTP: mhttp,

		sendCh:  sendCh,
		recvCh:  recvCh,
		stopped: 1,
	}, nil
}

func (p *ProxyImpl) Start() error {
	p.log.Infof("proxy start")

	if !atomic.CompareAndSwapUint64(&p.stopped, 1, 0) {
		p.log.Errorf("proxy.stopped is not 1, start failed")
		return errors.New("proxy.stopped is not 1, start failed")
	}
	p.log.Infof("proxy actually start")

	err := p.myHTTP.Start()
	if err != nil {
		return err
	}

	err = p.myTCP.Start()
	if err != nil {
		return err
	}

	return nil
}

func (p *ProxyImpl) Stop() error {
	// 防重入
	if !atomic.CompareAndSwapUint64(&p.stopped, 0, 1) {
		p.log.Warningf("cannot call stop when stopped == 1")
		return nil
	}
	p.log.Info("start to stop proxy......")

	err := p.myHTTP.Stop()
	if err != nil {
		return err
	}

	p.log.Infof("proxy.http stopped successfully")

	err = p.myTCP.Stop()
	if err != nil {
		return err
	}

	p.log.Infof("proxy.tcp stopped successfully")

	return nil
}
