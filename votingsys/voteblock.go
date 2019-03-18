package votingsys

import (
	"github.com/drcsuite/drc/btcec"
)

// 具有投票权的节点对区块的签名值和验证时用的公钥
type SignAndKey struct {
	Sign   *btcec.Signature
	PubKey *btcec.PublicKey
}
