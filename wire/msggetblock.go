// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"io"
)

// TypeParameter为1，请求增量同步块
// TypeParameter为2，请求软状态块
type MsgGetBlock struct {
	TypeParameter int8
	Height        int32
	BlockHash     chainhash.Hash
}

const MaxMsgGetBlock = 5 + chainhash.HashSize

func (msg *MsgGetBlock) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {

	return readElements(r, &msg.TypeParameter, &msg.Height, &msg.BlockHash)
}

func (msg *MsgGetBlock) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {

	return writeElements(w, &msg.TypeParameter, &msg.Height, &msg.BlockHash)
}

func (msg *MsgGetBlock) Command() string {
	return CmdGetBlock
}

func (msg *MsgGetBlock) MaxPayloadLength(pver uint32) uint32 {

	return MaxMsgGetBlock
}

func NewMsgGetBlock(height int32, typePara int8, blockHash chainhash.Hash) *MsgGetBlock {
	return &MsgGetBlock{
		TypeParameter: typePara,
		Height:        height,
		BlockHash:     blockHash,
	}
}
