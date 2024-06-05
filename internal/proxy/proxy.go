package proxy

import (
	"fmt"
	"github.com/meshplus/pier/internal/proxy/common"
	"github.com/meshplus/pier/internal/proxy/config"
	"github.com/meshplus/pier/internal/proxy/http"
	"github.com/meshplus/pier/internal/proxy/tcp"
	"github.com/meshplus/pier/pkg/rediscli"
	"github.com/sirupsen/logrus"
	"sync"
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
}

func NewProxy(cfgFilePath string, redisCli rediscli.Wrapper, log logrus.FieldLogger) (Proxy, error) {
	conf, lerr := config.LoadProxyConfig(cfgFilePath)
	if lerr != nil {
		return nil, fmt.Errorf("failed to load config %s, err: %s", cfgFilePath, lerr.Error())
	}

	sendCh := make(chan *common.Data)
	recvCh := make(chan *common.Data)

	mtcp, nerr := tcp.New(recvCh, sendCh, conf, log)
	if nerr != nil {
		return nil, fmt.Errorf("failed to init tcp component, err: %s", nerr.Error())
	}
	mhttp, err := http.New(redisCli, sendCh, recvCh, conf, log)
	if err != nil {
		return nil, err
	}

	return &ProxyImpl{
		config: conf,
		log:    log,
		myTCP:  mtcp,
		myHTTP: mhttp,

		sendCh: sendCh,
		recvCh: recvCh,
	}, nil
}

func (p ProxyImpl) Start() error {
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

func (p ProxyImpl) Stop() error {

	drainWG := &sync.WaitGroup{}

	err := p.myHTTP.Stop()
	if err != nil {
		return err
	}

	p.log.Infof("proxy.http stopped, start a drain goroutine for http.recv channel")
	drainWG.Add(1)
	go func() {
		defer drainWG.Done()
		for range p.recvCh {
		}
	}()

	err = p.myTCP.Stop()
	if err != nil {
		return err
	}

	p.log.Infof("proxy.tcp stopped, start to close sendCh and recvCh")
	close(p.sendCh)
	close(p.recvCh)
	drainWG.Wait()
	p.log.Infof("proxy stop finish!")

	return nil
}
