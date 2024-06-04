package repo

import (
	"fmt"
	"github.com/gobuffalo/packr/v2/file/resolver/encoding/hex"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"testing"

	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym/ecdsa"
	"github.com/stretchr/testify/require"
)

func Test_loadNetworkConfig(t *testing.T) {
	// TODO
	path := "./testdata"
	nodePriivKey, err := LoadNodePrivateKey(path)
	require.Nil(t, err)
	libp2pPrivKey, err := convertToLibp2pPrivKey(nodePriivKey)
	require.Nil(t, err)
	networkConfig, err := LoadNetworkConfig(path, libp2pPrivKey)
	require.Nil(t, err)
	require.Equal(t, 4, len(networkConfig.Piers))

}

func convertToLibp2pPrivKey(privateKey crypto.PrivateKey) (crypto2.PrivKey, error) {
	ecdsaPrivKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("convert to libp2p private key: not ecdsa private key")
	}

	libp2pPrivKey, _, err := crypto2.ECDSAKeyPairFromKey(ecdsaPrivKey.K)
	if err != nil {
		return nil, err
	}

	return libp2pPrivKey, nil
}

func TestMultiHosts(t *testing.T) {
	repoRoot := "../../mockpier/main/config11"
	var (
		priv        crypto.PrivateKey
		priv2       crypto2.PrivKey
		err         error
		networkConf *NetworkConfig
		piers       map[string]*peer.AddrInfo
	)
	priv, err = LoadNodePrivateKey(repoRoot)
	assert.Nil(t, err)
	priv2, err = convertToLibp2pPrivKey(priv)
	assert.Nil(t, err)
	networkConf, err = LoadNetworkConfig(repoRoot, priv2)
	assert.Nil(t, err)
	piers, err = GetNetworkPeers(networkConf)
	assert.Nil(t, err)
	for k, v := range piers {
		var output string
		for _, addr := range v.Addrs {
			output += fmt.Sprintf("{bytes: %s, string: %s} ", hex.EncodeToString(addr.Bytes()), addr.String())
		}
		t.Logf("pid: %s, piers: %s", k, output)
	}
}
