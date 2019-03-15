package votingsys

const IdealVoteNum = 300

// 估算全网节点总数
func EstimateScale(prevVoteNum uint16, prevScale uint16) (scale uint16) {

	//上一个区块的Scale小于等于300，说明全网节点总数很少，之前收到多少投票就可估算为当前的节点总数。
	if prevScale <= IdealVoteNum {

		scale = prevVoteNum

		// 上一个区块的Scale大于300，说明全网节点总数大于300，需计算使符合投票的节点数更接近300.
	} else {

		scale = uint16(int32(prevScale) * int32(prevVoteNum) / IdealVoteNum)

	}
	return
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
//func blockSignature(blockHash []byte, key *btcec.PrivateKey) *btcec.Signature {
//
//	signature, err := key.Sign(blockHash)
//	if err != nil {
//		return nil
//	}
//	return signature
//}

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
