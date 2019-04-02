package cpuminer

import (
	"fmt"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/vote"
	"github.com/drcsuite/drc/wire"
	"math/big"
)

const (
	// The number of leading reference blocks required to evaluate a scale
	// 求scale值需要的前置参考块的数量
	BlockCount = 10

	// 成为优势区块所需的票数差
	// The number of votes needed to become the dominant block
	AdvantageVoteNum = 100
)

// 区块验证投票
// Block validation vote
func (m *CPUMiner) BlockVote(msg *wire.MsgCandidate) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	// 获取本节点公私钥
	// Gets the node's public key and private key
	privateKey := m.privKey
	publicKey := privateKey.PubKey()
	// 散列两次区块内容
	// Hash the contents of the block twice
	headerHash := msg.Header.BlockHash()

	// 判断该块票数是否符合被投票的资格
	// Determine whether the block is eligible to be voted on
	//if isAdvantage(headerHash) {
	if true {

		pubKey, err := chainhash.NewHash33(publicKey.SerializeCompressed())
		if err != nil {
			log.Errorf("Format conversion error: %s", err)
		}

		// 用自己的私钥签名区块
		// Sign the block with your own private key
		headerSign, err := privateKey.Sign64(headerHash.CloneBytes())
		fmt.Printf("headersign： %x\n", headerSign)
		if err != nil {
			log.Errorf("Signature error: %s", err)
		}
		// 计算本节点的投票weight
		//The voting weight of this node is calculated
		weight := chainhash.DoubleHashB(headerSign)
		bigWeight := new(big.Int).SetBytes(weight)

		voteVerge := vote.VotesVerge(msg.Header.Scale)
		// weight值小于voteVerge，有投票权，进行投票签名
		// Weight is less than the voteVerge, has the right to vote, does the voting signature
		if bigWeight.Cmp(voteVerge) <= 0 {

			sign, err := chainhash.NewHash64(headerSign)
			if err != nil {
				log.Errorf("Format conversion error: %s", err)
			}

			// 维护本地票池
			// Maintain local ticket pool
			signAndKey := vote.SignAndKey{
				Signature: *sign,
				PublicKey: *pubKey,
			}
			vote.UpdateTicketPool(headerHash, signAndKey)

			// 传播签名
			// Propagate signatures
			msgSign := &wire.MsgSign{
				BlockHeaderHash: headerHash,
				Signature:       *sign,
				PublicKey:       *pubKey,
			}

			m.cfg.SendSign(msgSign)
		}
	}
}

// 如果当前块的票数比别的块差太多，放弃投票转发当前块
// If the current block is too many votes short of the other blocks, the current block is not forwarded
func isAdvantage(headerHash chainhash.Hash) bool {
	_, max := GetMaxVotes()
	// 当前块的票数
	// The number of votes in the current block
	count := GetVotes(headerHash)

	// 当前块与最多票数的块票数差值为100票，不需要为其投票
	// The difference between the current block and the block with the most votes is 100, and no vote is required
	if max-count >= AdvantageVoteNum {
		return false
	}

	return true

}

// 取得当前票池中获得最多投票数的区块和票数值
// Gets the block with the most votes in the current pool and the number of votes
func GetMaxVotes() (chainhash.Hash, uint16) {

	var maxVotes = 0
	var maxBlockHash chainhash.Hash
	fmt.Println("票池大小： ", len(vote.GetTicketPool()))

	for headerHash, signAndKeys := range vote.GetTicketPool() {
		if count := len(signAndKeys); count > maxVotes {
			maxVotes = count
			maxBlockHash = headerHash
		}
	}
	return maxBlockHash, uint16(maxVotes)
}

// 获取区块的当前票数
// Gets the current number of votes for the block
func GetVotes(hash chainhash.Hash) uint16 {
	ticketPool := vote.GetTicketPool()

	signAndKeys := ticketPool[hash]

	return uint16(len(signAndKeys))
}
