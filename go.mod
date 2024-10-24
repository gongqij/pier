module github.com/meshplus/pier

go 1.13

require (
	github.com/Rican7/retry v0.1.0
	github.com/btcsuite/btcd v0.24.2-beta.-rc1
	github.com/cavaliercoder/grab v2.0.0+incompatible
	github.com/cbergoon/merkletree v0.2.0
	github.com/fatih/color v1.9.0
	github.com/fsnotify/fsnotify v1.4.9
	github.com/garyburd/redigo v1.6.4 // indirect
	github.com/gobuffalo/packd v1.0.1
	github.com/gobuffalo/packr/v2 v2.8.3
	github.com/golang/mock v1.6.0
	github.com/google/btree v1.0.0
	github.com/google/uuid v1.3.0
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd
	github.com/hashicorp/go-plugin v1.3.0
	github.com/ipfs/go-cid v0.0.7
	github.com/json-iterator/go v1.1.11
	github.com/libp2p/go-buffer-pool v0.1.0 // indirect
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
	github.com/tidwall/gjson v1.9.3
	github.com/tjfoc/gmsm v1.4.1
	github.com/urfave/cli v1.22.1
	github.com/whyrusleeping/tar-utils v0.0.0-20201201191210-20a61371de5b // indirect
	github.com/wonderivan/logger v1.0.0
	go.uber.org/atomic v1.7.0
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/net v0.28.0
	google.golang.org/grpc v1.38.0
	gopkg.in/bsm/ratelimit.v1 v1.0.0-20170922094635-f56db5e73a5e // indirect
	gopkg.in/redis.v4 v4.2.4
)

replace (
	github.com/btcsuite/btcd => ./third/btcd
	github.com/hyperledger/fabric => github.com/hyperledger/fabric v2.0.1+incompatible
	github.com/libp2p/go-libp2p => ./third/go-libp2p
	github.com/libp2p/go-libp2p-core => github.com/libp2p/go-libp2p-core v0.5.6
	github.com/meshplus/go-lightp2p => ./third/lightp2p
	github.com/pkg/sftp => github.com/pkg/sftp v1.13.1
)
