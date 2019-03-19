package wire

import (
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"io"
)

type MsgSign struct {
	// 区块头hash
	BlockHeaderHash chainhash.Hash
	// 投票节点签名
	Signature chainhash.Hash64
	// 投票节点公钥
	PublicKey chainhash.Hash33
}

func (*MsgSign) BtcDecode(io.Reader, uint32, MessageEncoding) error {
	panic("implement me")
}

func (*MsgSign) BtcEncode(io.Writer, uint32, MessageEncoding) error {
	panic("implement me")
}

func (*MsgSign) Command() string {
	panic("implement me")
}

func (*MsgSign) MaxPayloadLength(uint32) uint32 {
	panic("implement me")
}
