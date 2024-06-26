package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/hashicorp/go-plugin"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/meshplus/bitxhub-core/agency"
	appchainmgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/meshplus/bitxhub-kit/storage"
	"github.com/meshplus/bitxhub-model/pb"
	rpcx "github.com/meshplus/go-bitxhub-client"
	_ "github.com/meshplus/pier/imports"
	"github.com/meshplus/pier/internal/adapt/appchain_adapter"
	"github.com/meshplus/pier/internal/adapt/bxh_adapter"
	"github.com/meshplus/pier/internal/adapt/direct_adapter"
	"github.com/meshplus/pier/internal/adapt/union_adapter"
	"github.com/meshplus/pier/internal/exchanger"
	"github.com/meshplus/pier/internal/loggers"
	"github.com/meshplus/pier/internal/peermgr"
	"github.com/meshplus/pier/internal/proxy"
	"github.com/meshplus/pier/internal/repo"
	"github.com/meshplus/pier/internal/txcrypto"
	"github.com/meshplus/pier/pkg/plugins"
	"github.com/meshplus/pier/pkg/rediscli"
	"github.com/meshplus/pier/pkg/redisha"
	"github.com/meshplus/pier/pkg/redisha/signal"
	"github.com/meshplus/pier/pkg/single"
	_ "github.com/meshplus/pier/pkg/single"
	"github.com/sirupsen/logrus"
	"github.com/wonderivan/logger"
	"path/filepath"
	"strconv"
	"strings"
)

const DEFAULT_UNION_PIER_ID = "default_union_pier_id"

// Pier represents the necessary data for starting the pier app
type Pier struct {
	privateKey  crypto.PrivateKey
	plugin      plugins.Client
	grpcPlugin  *plugin.Client
	pierHA      agency.PierHA
	storage     storage.Storage
	exchanger   exchanger.IExchanger
	ctx         context.Context
	cancel      context.CancelFunc
	appchain    *appchainmgr.Appchain
	serviceMeta map[string]*pb.Interchain
	config      *repo.Config
	logger      logrus.FieldLogger

	redisCli rediscli.Wrapper
	quitMain signal.QuitMainSignal
	proxy    proxy.Proxy

	// 负责接收Start过程中的error，外面会尝试做stop，但stop可以做防重入
	errCh chan error
}

// Start starts three main components of pier app
func (pier *Pier) Start() error {
	if err := pier.pierHA.Start(); err != nil {
		return fmt.Errorf("pier ha start fail")
	}
	go pier.startPierHA()
	return nil
}

func NewPier(repoRoot string, config *repo.Config) (*Pier, error) {
	var (
		ex          exchanger.IExchanger
		pierHA      agency.PierHA
		redisCli    rediscli.Wrapper      = &rediscli.MockWrapperImpl{}
		quitMain    signal.QuitMainSignal = &signal.MockQuitMainSignal{}
		peerManager peermgr.PeerManager
		errch       chan error = make(chan error)
	)

	logger := loggers.Logger(loggers.App)
	privateKey, err := repo.LoadPrivateKey(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("repo load key: %w", err)
	}

	nodePrivKey, err := repo.LoadNodePrivateKey(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("repo load node key: %w", err)
	}

	switch config.Mode.Type {
	case repo.DirectMode:
		var cryptor txcrypto.Cryptor
		// in ha=redis mode, peerManager should be re-new at each time when startHA called
		if config.HA.Mode != "redis" {
			peerManager, err = peermgr.New(config, nodePrivKey, privateKey, 1, errch, loggers.Logger(loggers.PeerMgr))
			if err != nil {
				return nil, fmt.Errorf("peerMgr create: %w", err)
			}
			// todo: DirectCryptor cannot be used by direct+redis mode
			cryptor, err = txcrypto.NewDirectCryptor(peerManager, privateKey, loggers.Logger(loggers.Cryptor))
			if err != nil {
				return nil, fmt.Errorf("crypto create: %w", err)
			}
		}
		appchainAdapter, err := appchain_adapter.NewAppchainAdapter(repo.DirectMode, config, loggers.Logger(loggers.Appchain), cryptor)
		if err != nil {
			return nil, fmt.Errorf("new appchain adapter: %w", err)
		}

		directAdapter, err := direct_adapter.New(peerManager, appchainAdapter, loggers.Logger(loggers.Direct), config.Mode.Direct.RemoteAppchainID)
		if err != nil {
			return nil, fmt.Errorf("new direct adapter: %w", err)
		}

		ex, err = exchanger.New(repo.DirectMode, config.Appchain.ID, "", errch,
			exchanger.WithSrcAdapt(appchainAdapter),
			exchanger.WithDestAdapt(directAdapter),
			exchanger.WithLogger(loggers.Logger(loggers.Exchanger)))
		if err != nil {
			return nil, fmt.Errorf("exchanger create: %w", err)
		}

		switch config.HA.Mode {
		case "single":
			pierHA = single.New(nil, config.Appchain.ID)
		case "redis":
			redisHA := redisha.New(config.Redis, config.Appchain.ID, errch)
			redisCli = redisHA.GetRedisCli()
			pierHA = redisHA
			quitMain = redisHA
		default:
			return nil, fmt.Errorf("unsupported ha mode %s, should be `single` or `redis`", config.HA.Mode)
		}
		loggers.Logger(loggers.Direct).Infof("create direct pier instance finished")
	case repo.RelayMode:
		client, err := newBitXHubClient(logger, privateKey, config)
		if err != nil {
			return nil, fmt.Errorf("create bitxhub client: %w", err)
		}

		cryptor, err := txcrypto.NewRelayCryptor(client, privateKey, loggers.Logger(loggers.Cryptor))
		appchainAdapter, err := appchain_adapter.NewAppchainAdapter(repo.RelayMode, config, loggers.Logger(loggers.Appchain), cryptor)
		if err != nil {
			return nil, fmt.Errorf("new appchain adapter: %w", err)
		}

		bxhAdapter, err := bxh_adapter.New(repo.RelayMode, appchainAdapter.ID(), client, loggers.Logger(loggers.Syncer), config.TSS)
		if err != nil {
			return nil, fmt.Errorf("new bxh adapter: %w", err)
		}
		pierHAConstructor, err := agency.GetPierHAConstructor(config.HA.Mode)
		if err != nil {
			return nil, fmt.Errorf("pier ha constructor not found")
		}

		pierHA = pierHAConstructor(client, config.Appchain.ID)

		if config.Mode.Relay.EnableOffChainTransmission {
			offChainTransmissionConstructor, err := agency.GetOffchainTransmissionConstructor("offChain_transmission")
			if err != nil {
				return nil, fmt.Errorf("offchain transmission constructor not found")
			}
			peerManager, err = peermgr.New(config, nodePrivKey, privateKey, 1, errch, loggers.Logger(loggers.PeerMgr))
			if err != nil {
				return nil, fmt.Errorf("peerMgr create: %w", err)
			}
			offChainTransmissionMgr := offChainTransmissionConstructor(appchainAdapter.ID(), peerManager, appchainAdapter.(*appchain_adapter.AppchainAdapter).GetPluginClient())
			if err := offChainTransmissionMgr.Start(); err != nil {
				return nil, fmt.Errorf("start offchain transmission: %w", err)
			}
		}

		ex, err = exchanger.New(repo.RelayMode, config.Appchain.ID, config.Mode.Relay.BitXHubID, errch,
			exchanger.WithSrcAdapt(appchainAdapter),
			exchanger.WithDestAdapt(bxhAdapter),
			exchanger.WithLogger(loggers.Logger(loggers.Exchanger)))
		if err != nil {
			return nil, fmt.Errorf("exchanger create: %w", err)
		}
	case repo.UnionMode:
		client, err := newBitXHubClient(logger, privateKey, config)
		if err != nil {
			return nil, fmt.Errorf("create bitxhub client: %w", err)
		}

		bxhAdapter, err := bxh_adapter.New(repo.UnionMode, DEFAULT_UNION_PIER_ID, client, loggers.Logger(loggers.Syncer), config.TSS)
		if err != nil {
			return nil, fmt.Errorf("new bitxhub adapter: %w", err)
		}

		peerManager, err := peermgr.New(config, nodePrivKey, privateKey, config.Mode.Union.Providers, errch, loggers.Logger(loggers.PeerMgr))
		if err != nil {
			return nil, fmt.Errorf("peerMgr create: %w", err)
		}

		unionAdapt, err := union_adapter.New(peerManager, bxhAdapter, loggers.Logger(loggers.Union))
		if err != nil {
			return nil, fmt.Errorf("new union adapter: %w", err)
		}

		ex, err = exchanger.New(repo.UnionMode, "", bxhAdapter.ID(), errch,
			exchanger.WithSrcAdapt(bxhAdapter),
			exchanger.WithDestAdapt(unionAdapt),
			exchanger.WithLogger(loggers.Logger(loggers.Exchanger)))
		if err != nil {
			return nil, fmt.Errorf("exchanger create: %w", err)
		}

		pierHA = single.New(nil, DEFAULT_UNION_PIER_ID)
	default:
		return nil, fmt.Errorf("unsupported mode")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Pier{
		privateKey: privateKey,
		exchanger:  ex,
		pierHA:     pierHA,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		config:     config,
		redisCli:   redisCli,
		quitMain:   quitMain,
		errCh:      errch,
	}, nil
}

func (pier *Pier) getSelfP2PPort(nodePrivKey crypto.PrivateKey) (int, error) {
	ecdsaPrivKey, ok := nodePrivKey.(*ecdsa.PrivateKey)
	if !ok {
		return 0, fmt.Errorf("convert to libp2p private key: not ecdsa private key")
	}

	libp2pPrivKey, _, err := crypto2.ECDSAKeyPairFromKey(ecdsaPrivKey.K)
	if err != nil {
		return 0, err
	}

	networkConfig, err := repo.LoadNetworkConfig(pier.config.RepoRoot, libp2pPrivKey)
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

func (pier *Pier) startPierHA() {
	logger.Info("pier HA manager start")

	nodePrivKey, _ := repo.LoadNodePrivateKey(pier.config.RepoRoot)
	selfP2pPort, err := pier.getSelfP2PPort(nodePrivKey)
	if err != nil {
		pier.logger.Errorf("failed to get local tcp port, err: %s", err.Error())
		panic("failed to get local tcp port")
	}

	status := false
	for {
		select {
		case isMain := <-pier.pierHA.IsMain():
			logger.Info("receive from isMain: %v, status: %v", isMain, status)
			if isMain {
				if status {
					continue
				}
				//for serviceID, meta := range pier.serviceMeta {
				//	pier.logger.WithFields(logrus.Fields{
				//		"id":                        serviceID,
				//		"interchain_counter":        meta.InterchainCounter,
				//		"receipt_counter":           meta.ReceiptCounter,
				//		"source_interchain_counter": meta.SourceInterchainCounter,
				//		"source_receipt_counter":    meta.SourceReceiptCounter,
				//	}).Infof("Pier information of service %s", serviceID)
				//}

				if pier.config.Mode.Type == repo.DirectMode && pier.config.Proxy.Enable {
					// initialize proxy component
					px, nerr := proxy.NewProxy(filepath.Join(pier.config.RepoRoot, repo.ProxyConfigName), pier.redisCli, pier.quitMain, selfP2pPort, loggers.Logger(loggers.Proxy))
					if nerr != nil {
						pier.logger.Errorf("failed to init proxy, err: %s", nerr.Error())
						panic("failed to init proxy")
					}
					pier.proxy = px
					// start up proxy component
					if serr := pier.proxy.Start(); serr != nil {
						pier.logger.Errorf("start up proxy error: %s", serr.Error())
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
				if pier.config.Mode.Type == repo.DirectMode && pier.config.HA.Mode == "redis" {
					peerManager, err := peermgr.New(pier.config, nodePrivKey, pier.privateKey, 1, pier.errCh, loggers.Logger(loggers.PeerMgr))
					if err != nil {
						pier.logger.Errorf("renew create peerManager error: %s", err.Error())
						panic("create peerManager error")
					}
					pier.exchanger.RenewPeerManager(peerManager)
				}
				if err := pier.exchanger.Start(); err != nil {
					pier.logger.Errorf("exchanger start: %s", err.Error())
					// todo: how to handle error?
					return
				}
				status = true
			} else {
				if !status {
					continue
				}
				if err := pier.StopExchanger(); err != nil {
					pier.logger.Errorf("pier stop exchanger: %w", err)
					return
				}
				if pier.proxy != nil {
					if err := pier.proxy.Stop(); err != nil {
						pier.logger.Errorf("stop proxy error: %v", err)
					}
				}
				// gw：原来的逻辑不行，是因为同一侧的两个pier，可能都能从isMain这个channel收到true（这件事是redis抢锁控制的），
				// 但是如果对端的pier没有启动的话，那么这边的pier连stop的机会都没有；
				// 现在的逻辑是：pier的Start函数go起来调用，无论执行到哪里，如果出现了Stop，那么Stop函数负责打断Start函数的goroutine，
				// 并且确保由本函数启动的goroutine全部都停下来了；这样Start就不会一直阻塞了，至少有被Stop打断的机会；
				// 最后，status的赋值和修改应该在本函数内同步完成；
				status = false
			}
		case <-pier.ctx.Done():
			pier.logger.Infof("receiving done signal, exit pier HA...")
			return
		}
	}
}

// Stop stops three main components of pier app
func (pier *Pier) Stop() error {
	pier.cancel()

	if err := pier.exchanger.Stop(); err != nil {
		return fmt.Errorf("exchanger stop: %w", err)
	}

	// todo: proxy stop

	if err := pier.pierHA.Stop(); err != nil {
		return fmt.Errorf("pierHA stop: %w", err)
	}
	return nil
}

func (pier *Pier) StopExchanger() error {
	if err := pier.exchanger.Stop(); err != nil {
		return fmt.Errorf("exchanger stop: %w", err)
	}
	return nil
}

// Type gets the application blockchain type the pier is related to
func (pier *Pier) Type() string {
	if pier.config.Mode.Type != repo.UnionMode {
		return pier.plugin.Type()
	}
	return repo.UnionMode
}

func filterServiceMeta(serviceInterchain map[string]*pb.Interchain, bxhID, appchainID string, serviceIDs []string) map[string]*pb.Interchain {
	result := make(map[string]*pb.Interchain)

	for _, id := range serviceIDs {
		fullServiceID := fmt.Sprintf("%s:%s:%s", bxhID, appchainID, id)
		val, ok := serviceInterchain[fullServiceID]
		if !ok {
			val = &pb.Interchain{
				ID:                      fullServiceID,
				InterchainCounter:       make(map[string]uint64),
				ReceiptCounter:          make(map[string]uint64),
				SourceInterchainCounter: make(map[string]uint64),
				SourceReceiptCounter:    make(map[string]uint64),
			}
		}
		result[fullServiceID] = val
	}

	return result
}

func newBitXHubClient(logger logrus.FieldLogger, privateKey crypto.PrivateKey, config *repo.Config) (rpcx.Client, error) {
	opts := []rpcx.Option{
		rpcx.WithLogger(logger),
		rpcx.WithPrivateKey(privateKey),
	}
	addrs := make([]string, 0)
	if strings.EqualFold(repo.RelayMode, config.Mode.Type) {
		addrs = config.Mode.Relay.Addrs
	} else if strings.EqualFold(repo.UnionMode, config.Mode.Type) {
		addrs = config.Mode.Union.Addrs
	}
	nodesInfo := make([]*rpcx.NodeInfo, 0, len(addrs))
	for index, addr := range addrs {
		nodeInfo := &rpcx.NodeInfo{Addr: addr}
		if config.Security.EnableTLS {
			nodeInfo.CertPath = filepath.Join(config.RepoRoot, config.Security.Tlsca)
			nodeInfo.EnableTLS = config.Security.EnableTLS
			nodeInfo.CommonName = config.Security.CommonName
			nodeInfo.AccessCert = filepath.Join(config.RepoRoot, config.Security.AccessCert[index])
			nodeInfo.AccessKey = filepath.Join(config.RepoRoot, config.Security.AccessKey)
		}
		nodesInfo = append(nodesInfo, nodeInfo)
	}
	opts = append(opts, rpcx.WithNodesInfo(nodesInfo...), rpcx.WithTimeoutLimit(config.Mode.Relay.TimeoutLimit))
	return rpcx.New(opts...)
}
