package mockpier

import (
	"errors"
	"fmt"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/pb"
	network "github.com/meshplus/go-lightp2p"
	"github.com/meshplus/pier/internal/loggers"
	"github.com/meshplus/pier/internal/peermgr"
	"github.com/meshplus/pier/internal/proxy"
	"github.com/meshplus/pier/internal/repo"
	"github.com/meshplus/pier/pkg/rediscli"
	"github.com/meshplus/pier/pkg/redisha"
	"github.com/meshplus/pier/pkg/redisha/signal"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type MockPier struct {
	nodePriv crypto.PrivateKey
	pm       peermgr.PeerManager
	pierHA   agency.PierHA
	redisCli rediscli.Wrapper
	log      logrus.FieldLogger
	config   *repo.Config
	repoRoot string
	proxy    proxy.Proxy
	quitMain signal.QuitMainSignal

	// 负责接收Start过程中的error，外面会尝试做stop，但stop可以做防重入
	errCh chan error
}

func NewMockPier(repoRoot string) (*MockPier, error) {
	config, err := repo.UnmarshalConfig(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("init config error: %s", err)
	}
	nodePrivKey, err := repo.LoadNodePrivateKey(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("repo load node key: %w", err)
	}
	err = log.Initialize(
		log.WithReportCaller(config.Log.ReportCaller),
		log.WithPersist(true),
		log.WithFilePath(filepath.Join(repoRoot, config.Log.Dir)),
		log.WithFileName(config.Log.Filename),
		log.WithMaxSize(2*1024*1024),
		log.WithMaxAge(time.Duration(config.Log.Day)*24*time.Hour),
		log.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return nil, fmt.Errorf("log initialize: %w", err)
	}
	loggers.InitializeLogger(config)
	loggers.Logger(loggers.App).Infof("redis.send_lease_timeout: %d", config.Redis.SendLeaseTimeout)
	loggers.Logger(loggers.App).Infof("redis.proxy_enable: %v", config.Proxy.Enable)

	errCh := make(chan error)

	rpm := redisha.New(config.Redis, config.Appchain.ID, errCh)
	mockPier := &MockPier{
		nodePriv: nodePrivKey,
		pierHA:   rpm,
		redisCli: rpm.GetRedisCli(),
		log:      loggers.Logger(loggers.App),
		config:   config,
		repoRoot: repoRoot,
		quitMain: rpm,
		errCh:    errCh,
	}

	return mockPier, nil
}

func (mp *MockPier) Start() error {
	if err := mp.pierHA.Start(); err != nil {
		return fmt.Errorf("pier ha start fail")
	}
	go mp.startPierHA()
	return nil
}

func (mp *MockPier) Stop() error {
	err := mp.pm.Stop()
	if err != nil {
		mp.log.Errorf("peerManager stop error: %s", err.Error())
		return err
	}
	err = mp.pierHA.Stop()
	if err != nil {
		mp.log.Errorf("pierHA stop error: %s", err.Error())
		return err
	}
	return nil
}

func (mp *MockPier) getSelfP2PPort(nodePrivKey crypto.PrivateKey) (int, error) {
	ecdsaPrivKey, ok := nodePrivKey.(*ecdsa.PrivateKey)
	if !ok {
		return 0, fmt.Errorf("convert to libp2p private key: not ecdsa private key")
	}

	libp2pPrivKey, _, err := crypto2.ECDSAKeyPairFromKey(ecdsaPrivKey.K)
	if err != nil {
		return 0, err
	}

	networkConfig, err := repo.LoadNetworkConfig(mp.config.RepoRoot, libp2pPrivKey)
	if err != nil {
		return 0, fmt.Errorf("load network config err: %w", err)
	}
	if networkConfig == nil || networkConfig.LocalAddr == "" {
		return 0, errors.New("not found local address in network.toml")
	}

	idx := strings.LastIndex(networkConfig.LocalAddr, "/tcp/")
	if idx == -1 {
		return 0, fmt.Errorf("not found tcp protocol on address %v", networkConfig.LocalAddr)
	}

	tcpPort, aerr := strconv.Atoi(networkConfig.LocalAddr[idx+5:])
	if aerr != nil {
		return 0, fmt.Errorf("failed to get tcp port on address %v", networkConfig.LocalAddr)
	}

	return tcpPort, nil
}

func (mp *MockPier) startPierHA() {
	mp.log.Info("pier HA manager start")
	status := false
	selfP2pPort, err := mp.getSelfP2PPort(mp.nodePriv)
	if err != nil {
		mp.log.Errorf("failed to get local tcp port, err: %s", err.Error())
		panic("failed to get local tcp port")
	}
	//firstBeMain := true
	for {
		select {
		case isMain := <-mp.pierHA.IsMain():
			if isMain {
				if status {
					continue
				}

				if mp.config.Proxy.Enable {
					// initialize proxy component
					px, nerr := proxy.NewProxy(filepath.Join(mp.repoRoot, repo.ProxyConfigName), mp.redisCli, mp.quitMain, selfP2pPort, loggers.Logger(loggers.Proxy))
					if nerr != nil {
						mp.log.Errorf("failed to init proxy, err: %s", nerr.Error())
						panic("failed to init proxy")
					}
					mp.proxy = px
					// start up proxy component
					if serr := mp.proxy.Start(); serr != nil {
						mp.log.Errorf("start up proxy error: %s", serr.Error())
						panic("failed to start proxy")
					}
				}

				// special check for direct && pierHA_redis && not-first-time-in
				// single mode will never enter this logic again, so nothing to do;
				// union mode SKIP!!!!
				// only direct+redis_ha is related
				// golightp2p.Network wrap libp2p.Host, Host.New wrap host.Start(),
				// so Host does not contain Start() interface;
				// Besides, Host.Close() has a once.Do logic, so golightp2p.Network can not restart
				//if pier.config.Mode.Type == repo.DirectMode && pier.config.HA.Mode == "redis" && !firstBeMain {
				if mp.config.Mode.Type == repo.DirectMode && mp.config.HA.Mode == "redis" {
					peerManager, err := peermgr.New(mp.config, mp.nodePriv, nil, 1, mp.errCh, loggers.Logger(loggers.PeerMgr))
					if err != nil {
						mp.log.Errorf("renew create peerManager error: %s", err.Error())
						panic("create peerManager error")
					}
					mp.pm = peerManager
					handler := &additionalHandler{
						appchainID: mp.config.Appchain.ID,
						pm:         peerManager,
						log:        mp.log,
					}
					err = mp.pm.RegisterMsgHandler(pb.Message_ADDRESS_GET, handler.handleGetAddressMessage)
					if err != nil {
						panic(fmt.Sprintf("register msg handler error: %s", err.Error()))
					}
					err = mp.pm.Start()
					if err != nil {
						mp.log.Errorf("start peerManager error: %s", err.Error())
						panic("start peerManager error")
					}
				}
				status = true
			} else {
				if !status {
					continue
				}
				if mp.pm != nil {
					err := mp.pm.Stop()
					if err != nil {
						mp.log.Warnf("stop peerManager error: %s", err.Error())
					}
				}
				if mp.proxy != nil {
					if err := mp.proxy.Stop(); err != nil {
						mp.log.Errorf("stop proxy error: %v", err)
					}
				}

				status = false
			}
		}
	}
}

type additionalHandler struct {
	appchainID string
	pm         peermgr.PeerManager
	log        logrus.FieldLogger
}

func (h *additionalHandler) handleGetAddressMessage(stream network.Stream, message *pb.Message) {
	addr := h.appchainID

	retMsg := peermgr.Message(pb.Message_ACK, true, []byte(addr))

	err := h.pm.AsyncSendWithStream(stream, retMsg)
	if err != nil {
		h.log.Error(err)
		return
	}
}
