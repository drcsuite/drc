package votingsys

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/wire"
	"io"
)

// 具有投票权的节点对区块的签名值和验证时用的公钥
type SignAndKey struct {
	Sign   *btcec.Signature
	PubKey *btcec.PublicKey
}

// 区块签名的票池
type SignaturePool struct {
	BlockHashValue []byte
	SignAndKeys    []SignAndKey
}

// 待验证区块信息
type VoteBlock struct {
	Header       wire.BlockHeader
	Transactions []byte
}

func (*VoteBlock) BtcDecode(io.Reader, uint32, wire.MessageEncoding) error {
	panic("implement me")
}

func (*VoteBlock) BtcEncode(io.Writer, uint32, wire.MessageEncoding) error {
	panic("implement me")
}

func (*VoteBlock) Command() string {
	panic("implement me")
}

func (*VoteBlock) MaxPayloadLength(uint32) uint32 {
	panic("implement me")
}
