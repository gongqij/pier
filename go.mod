module github.com/meshplus/pier

go 1.13

require (
	github.com/Rican7/retry v0.1.0
	github.com/btcsuite/btcd v0.21.0-beta
	github.com/bytecodealliance/wasmtime-go v0.37.0 // indirect
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/cbergoon/merkletree v0.2.0
	github.com/fatih/color v1.9.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-redis/redis/v8 v8.11.5
	github.com/gobuffalo/packd v1.0.1
	github.com/gobuffalo/packr/v2 v2.8.3
	github.com/golang/mock v1.6.0
	github.com/google/btree v1.0.0
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.3.0
	github.com/ipfs/go-cid v0.0.7
	github.com/json-iterator/go v1.1.11
	github.com/libp2p/go-libp2p-core v0.6.1
	github.com/meshplus/bitxhub-core v1.3.1-0.20220511024304-f7458609c30a
	github.com/meshplus/bitxhub-kit v1.2.1-0.20220412092457-5836414df781
	github.com/meshplus/bitxhub-model v1.2.1-0.20220616031805-96a66092bc97
	github.com/meshplus/go-bitxhub-client v1.4.1-0.20220412093230-11ca79f069fc
	github.com/meshplus/go-lightp2p v0.0.0-20200817105923-6b3aee40fa54
	github.com/mitchellh/go-homedir v1.1.0
	github.com/multiformats/go-multiaddr v0.3.0
	github.com/pelletier/go-toml v1.9.3
	github.com/rs/cors v1.7.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/viper v1.8.1
	github.com/stretchr/testify v1.8.3
	github.com/tidwall/gjson v1.6.8
	github.com/ultramesh/flato-msp v0.2.10
	github.com/ultramesh/flato-msp-cert v0.2.30
	github.com/urfave/cli v1.22.1
	github.com/wonderivan/logger v1.0.0
	go.uber.org/atomic v1.7.0
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d
	google.golang.org/grpc v1.38.0
)

replace github.com/libp2p/go-libp2p-core => github.com/libp2p/go-libp2p-core v0.5.6

replace github.com/hyperledger/fabric => github.com/hyperledger/fabric v2.0.1+incompatible

replace golang.org/x/net => golang.org/x/net v0.0.0-20210405180319-a5a99cb37ef4

replace github.com/binance-chain/tss-lib => github.com/dawn-to-dusk/tss-lib v1.3.3-0.20220330081758-f404e10a1268

replace google.golang.org/grpc => google.golang.org/grpc v1.33.0

replace (
	github.com/ultramesh/crypto-gm => git.hyperchain.cn/ultramesh/crypto-gm.git v0.2.18
	github.com/ultramesh/crypto-standard => git.hyperchain.cn/ultramesh/crypto-standard.git v0.2.4
	github.com/ultramesh/fancylogger => git.hyperchain.cn/ultramesh/fancylogger.git v0.1.3
	github.com/ultramesh/flato-common => git.hyperchain.cn/ultramesh/flato-common.git v0.2.46-0.20220408031420-7fe84146a5f0
	github.com/ultramesh/flato-db-interface => git.hyperchain.cn/ultramesh/flato-db-interface.git v0.1.29
	github.com/ultramesh/flato-msp => git.hyperchain.cn/ultramesh/flato-msp.git v0.2.10
	github.com/ultramesh/flato-msp-cert => git.hyperchain.cn/ultramesh/flato-msp-cert.git v0.2.31
	github.com/ultramesh/flato-statedb => git.hyperchain.cn/ultramesh/flato-statedb.git v0.2.16
)
