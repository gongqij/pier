package mockpier

import (
	"fmt"
	crypto2 "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/pier/internal/repo"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenNodePriv(t *testing.T) {
	err := repo.generateNodePrivateKey("", crypto.ECDSA_P256)
	assert.Nil(t, err)
}

func TestGetNetworkID(t *testing.T) {
	privKey, _ := repo.LoadNodePrivateKey("")

	stdKey, _ := asym.PrivKeyToStdKey(privKey)

	_, pk, _ := crypto2.KeyPairFromStdKey(&stdKey)

	id, _ := peer.IDFromPublicKey(pk)
	fmt.Println(id)
	// QmP9zubwiXAmiKwP5mC7oSmHFzp42JuzjAGNwhRLzpy1sB
}
