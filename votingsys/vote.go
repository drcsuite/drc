package votingsys

import (
	"github.com/drcsuite/drc/btcec"
)

// 获取全网节点总数，通过blockHeader获取
func computeNodes(vb *VoteBlock) uint32 {

	return vb.Header.Nonce
}

// 新块验证投票

//func BlockVote(p *peer.Peer, msg *VoteBlock,pri *btcec.PrivateKey,pub *btcec.PublicKey) {
//
//
//	if checkBlock(msg,pub) {
//
// 计算本节点的weight，确认是否有投票资格
//		nodeNumber := computeNodes(msg)
//
//var weight uint32
//
//		if weight > 200/nodeNumber*uint32(math.Pow(2, 256)) {
//
//
//			sign := blockSignature(msg.Transactions, pri)
//			blockFooter := &BlockFooter{Sign: sign, PubKey: pub}
//			msg.Footer = append(msg.Footer, blockFooter)
//
//			p.QueueMessage(msg, nil)
//
//		} else {
//			p.QueueMessage(msg, nil)
//		}
//	}
//}

// 有投票权的节点签名区块
func blockSignature(blockHash []byte, key *btcec.PrivateKey) *btcec.Signature {

	signature, err := key.Sign(blockHash)
	if err != nil {
		return nil
	}
	return signature
}

// 检查收到的区块是否是之前接收过的
//func checkBlock(msg *VoteBlock,pub *btcec.PublicKey) bool  {
//
//	// 查看是否有自己的签名
//	for _,pubKey := range msg.Footer {
//
//		if pubKey.PubKey == pub {
//			return false
//		}
//
//	}
//	return true
//}
