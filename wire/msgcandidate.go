package wire

import (
	"fmt"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"io"
)

// MsgBlock implements the Message interface and represents a bitcoin
// block message.  It is used to deliver block and transaction information in
// response to a getdata message (MsgGetData) for a given block hash.
type MsgCandidate struct {
	Header       BlockHeader
	Transactions []*MsgTx
	Sigwit       *MsgSigwit
}

func (msg *MsgCandidate) AddTransaction(tx *MsgTx) error {
	msg.Transactions = append(msg.Transactions, tx)
	return nil

}

func (msg *MsgCandidate) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	err := readBlockHeader(r, pver, &msg.Header)
	if err != nil {
		return err
	}

	txCount, err := ReadVarInt(r, pver)
	if err != nil {
		return err
	}

	// Prevent more transactions than could possibly fit into a block.
	// It would be possible to cause memory exhaustion and panics without
	// a sane upper bound on this count.
	if txCount > maxTxPerBlock {
		str := fmt.Sprintf("too many transactions to fit into a block "+
			"[count %d, max %d]", txCount, maxTxPerBlock)
		return messageError("MsgBlock.BtcDecode", str)
	}

	msg.Transactions = make([]*MsgTx, 0, txCount)
	for i := uint64(0); i < txCount; i++ {
		tx := MsgTx{}
		err := tx.BtcDecode(r, pver, enc)
		if err != nil {
			return err
		}
		msg.Transactions = append(msg.Transactions, &tx)
	}

	err = msg.Sigwit.BtcDecode(r, pver, enc)
	if err != nil {
		return err
	}

	return nil
}

func (msg *MsgCandidate) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	err := writeBlockHeader(w, pver, &msg.Header)
	if err != nil {
		return err
	}

	err = WriteVarInt(w, pver, uint64(len(msg.Transactions)))
	if err != nil {
		return err
	}

	for _, tx := range msg.Transactions {
		err = tx.BtcEncode(w, pver, enc)
		if err != nil {
			return err
		}
	}

	err = msg.Sigwit.BtcEncode(w, pver, enc)
	if err != nil {
		return err
	}

	return nil
}

func (msg *MsgCandidate) Command() string {
	return CmdCandidate
}

func (msg *MsgCandidate) MaxPayloadLength(pver uint32) uint32 {
	// Block header at 80 bytes + transaction count + max transactions
	// which can vary up to the MaxBlockPayload (including the block header
	// and transaction count).
	return MaxBlockPayload
}

func (msg *MsgCandidate) SerializeSizeStripped() int {
	// Block header bytes + Serialized varint size for the number of
	// transactions.
	n := blockHeaderLen + VarIntSerializeSize(uint64(len(msg.Transactions)))

	for _, tx := range msg.Transactions {
		n += tx.SerializeSizeStripped()
	}

	return n
}

type MsgSigwit struct {
	Height int32
	Votes  []*MsgVote
}

func (msg *MsgSigwit) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return readElements(r, msg.Height)
}

func (msg *MsgSigwit) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return writeElements(w, &msg.Height)
}

type MsgVote struct {
	Sign   *chainhash.Hash64
	PubKey *chainhash.Hash33
}

func (msg *MsgVote) BtcDecode(r io.Reader, pver uint32, enc MessageEncoding) error {
	return readElements(r, &msg.Sign, &msg.PubKey)
}

func (msg *MsgVote) BtcEncode(w io.Writer, pver uint32, enc MessageEncoding) error {
	return writeElements(w, &msg.Sign, &msg.PubKey)
}
