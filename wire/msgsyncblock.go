// Copyright (c) 2013-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wire

import (
	"io"
)

// TypeParameter为1，增量同步块
// TypeParameter为2，软状态块
type MsgSyncBlock struct {
	TypeParameter int8
	MsgCandidate  MsgCandidate
}

func (msg *MsgSyncBlock) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readElement(r, &msg.TypeParameter)
	if err != nil {
		return err
	}

	err = msg.MsgCandidate.BtcDecode(r, pver, enc)
	if err != nil {
		return err
	}

	return nil
}

func (msg *MsgSyncBlock) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := writeElement(w, &msg.TypeParameter)
	if err != nil {
		return err
	}

	err = msg.MsgCandidate.BtcEncode(w, pver, enc)
	if err != nil {
		return err
	}

	return nil
}

func (msg *MsgSyncBlock) Command() string {
	return CmdSyncBlock
}

func (msg *MsgSyncBlock) MaxPayloadLength(pver uint32) uint32 {

	return MaxBlockPayload
}

func NewMsgSyncBlock(TypeParameter int8, msgCandidate MsgCandidate) *MsgSyncBlock {
	return &MsgSyncBlock{
		TypeParameter: TypeParameter,
		MsgCandidate:  msgCandidate,
	}
}
