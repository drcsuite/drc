package wire

import "github.com/drcsuite/drc/chaincfg/chainhash"

type MsgSign struct {
	// 区块头hash
	BlockHeaderHash chainhash.Hash
	// 投票节点签名
	Signature chainhash.Hash64
	// 投票节点公钥
	PublicKey chainhash.Hash33
}
