package mockpier

import (
	"fmt"
	"github.com/meshplus/bitxhub-core/agency"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/log"
	"github.com/meshplus/bitxhub-model/pb"
	network "github.com/meshplus/go-lightp2p"
	"github.com/meshplus/pier/internal/loggers"
	"github.com/meshplus/pier/internal/peermgr"
	"github.com/meshplus/pier/internal/repo"
	"github.com/meshplus/pier/pkg/redisha"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"time"
)

type MockPier struct {
	nodePriv crypto.PrivateKey
	pm       peermgr.PeerManager
	pierHA   agency.PierHA
	log      logrus.FieldLogger
	config   *repo.Config
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
		log.WithMaxAge(24*time.Hour),
		log.WithRotationTime(24*time.Hour),
	)
	if err != nil {
		return nil, fmt.Errorf("log initialize: %w", err)
	}
	loggers.InitializeLogger(config)

	mockPier := &MockPier{
		nodePriv: nodePrivKey,
		pierHA:   redisha.New(config.Redis, config.Appchain.ID),
		log:      loggers.Logger(loggers.App),
		config:   config,
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

func (mp *MockPier) startPierHA() {
	mp.log.Info("pier HA manager start")
	status := false
	//firstBeMain := true
	for {
		select {
		case isMain := <-mp.pierHA.IsMain():
			if isMain {
				if status {
					continue
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
					peerManager, err := peermgr.New(mp.config, mp.nodePriv, nil, 1, loggers.Logger(loggers.PeerMgr))
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
				//firstBeMain = false
			} else {
				if !status {
					continue
				}
				if mp.pm != nil {
					err := mp.pm.Stop()
					if err != nil {
						mp.log.Warn("stop peerManager error: %s", err.Error())
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
