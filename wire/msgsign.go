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

func (msg *MsgSign) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return readSign(r, pver, msg)
}

func readSign(r io.Reader, pver uint32, msg *MsgSign) error {
	return readElements(r, &msg.BlockHeaderHash, &msg.Signature, &msg.PublicKey)
}

func (msg *MsgSign) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return writeSign(w, pver, msg)
}

func writeSign(w io.Writer, pver uint32, msg *MsgSign) error {

	return writeElements(w, &msg.BlockHeaderHash, &msg.Signature, &msg.PublicKey)
}

func (msg *MsgSign) Command() string {
	return CmdSign
}

func (msg *MsgSign) MaxPayloadLength(pver uint32) uint32 {
	return MaxBlockPayload
}
