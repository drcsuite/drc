package cpuminer

import (
	"github.com/drcsuite/drc/btcec"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/wire"
	"math/big"
	"time"
)

const (
	// The number of leading reference blocks required to evaluate a scale
	// 求scale值需要的前置参考块的数量
	BlockCount = 10

	// 理想发块节点数
	// Ideal number of block nodes
	IdealBlockNum = 50

	// 理想投票节点数
	// Ideal number of voting nodes
	IdealVoteNum = 300

	// 成为优势区块所需的票数差
	// The number of votes needed to become the dominant block
	AdvantageVoteNum = 100

	// 发块时间间隔
	// Block time interval
	BlockTimeInterval = 10 * time.Second
)

// 区块签名的票池
// Block signature of the ticket pool
var ticketPool = make(map[chainhash.Hash][]SignAndKey)

// 前一区块的签名票池
// The signature ticket pool for the previous block
var prevTicketPool = make(map[chainhash.Hash][]SignAndKey)

// 具有投票权的节点对区块的签名值和验证时用的公钥
// The signed value of the block by the voting node and the public key used for verification
type SignAndKey struct {
	Signature chainhash.Hash64

	PublicKey chainhash.Hash33
}

// 估算全网节点总数
// Estimate the total number of nodes in the whole network
func EstimateScale(prevVoteNums []uint16, prevScales []uint16) (scale uint16) {

	meanScale := mean(prevScales)
	meanVoteNum := mean(prevVoteNums)

	if meanScale == 0 {
		meanScale = 1
	}
	if meanVoteNum == 0 {
		meanVoteNum = 1
	}
	//上一个区块的Scale小于等于300，说明全网节点总数很少，之前收到多少投票就可估算为当前的节点总数。
	//The Scale of the last block is less than or equal to 300,
	// indicating that the total number of nodes in the whole network is very small,
	// and the number of votes received before can be estimated as the current total number of nodes.
	if meanScale <= IdealVoteNum {

		scale = meanVoteNum

		// 上一个区块的Scale大于300，说明全网节点总数大于300，需计算使符合投票的节点数更接近300.
		// The Scale of the last block is greater than 300,
		// indicating that the total number of nodes in the whole network is greater than 300.
		// The number of nodes that meet the voting needs to be calculated to be closer to 300.
	} else {

		scale = uint16(uint32(meanScale) * uint32(meanVoteNum) / IdealVoteNum)

	}
	return
}

// 求[]uint16类型切片的平均值
// Find the average value of []uint16 type slice
func mean(values []uint16) (meanValue uint16) {
	var totalValue uint16 = 0
	for _, value := range values {
		totalValue = totalValue + value
	}
	meanValue = totalValue / uint16(len(values))
	return
}

// 计算投票的∏值
// To calculate the ∏ value of a vote
func VoteVerge(scale uint16) *big.Int {

	// bigint格式的全网节点数
	// Number of nodes in the whole network
	bigScale := big.NewInt(int64(scale))
	// bigint格式的理想投票节点数
	// Ideal number of voting nodes
	bigIdealVoteNum := big.NewInt(IdealVoteNum)
	// bigint格式的2的256次方的值
	// The value of the power
	max256, _ := new(big.Int).SetString("10000000000000000000000000000000000000000000000000000000000000000", 16)

	// 投票∏值计算
	// The vote ∏ value calculation
	max256.Mul(max256, bigIdealVoteNum)
	max256.Quo(max256, bigScale)

	return max256
}

// 计算发块的∏值
// Calculates the ∏ value of the block
func BlockVerge(scale uint16) *big.Int {

	bigIdealBlockNum := big.NewInt(IdealBlockNum)
	bigIdealVoteNum := big.NewInt(IdealVoteNum)
	verge := VoteVerge(scale)

	// 发块∏值计算
	// A block ∏ value calculation
	verge.Mul(verge, bigIdealBlockNum)
	verge.Quo(verge, bigIdealVoteNum)
	return verge
}

// 新块验证投票，需广播块返回true
// New block validation vote
func (m *CPUMiner) BlockVote(msg *wire.MsgBlock) bool {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	// 获取本节点公私钥
	// Gets the node's public key and private key
	privateKey := m.privKey
	publicKey := privateKey.PubKey()
	// 散列两次区块内容
	// Hash the contents of the block twice
	headerHash := msg.Header.BlockHash()

	// 验证区块
	// Verify the block
	if CheckBlock(*msg) {

		// 判断该块票数是否符合被投票的资格
		// Determine whether the block is eligible to be voted on
		if isAdvantage(headerHash) {

			pubKey, err := chainhash.NewHash33(publicKey.SerializeCompressed())
			if err != nil {
				log.Errorf("Format conversion error: %s", err)
			}

			// 判断本节点在之前是否为该块投过票
			// Determines whether this node has voted for this block before
			if preventRepeatSign(headerHash, *pubKey) {

				// 用自己的私钥签名区块
				// Sign the block with your own private key
				headerSign, err := privateKey.Sign(headerHash.CloneBytes())
				if err != nil {
					log.Errorf("Signature error: %s", err)
				}
				// 计算本节点的投票weight
				//The voting weight of this node is calculated
				weight := chainhash.DoubleHashB(headerSign.Serialize())
				bigWeight := new(big.Int).SetBytes(weight)

				voteVerge := VoteVerge(msg.Header.Scale)

				// weight值小于voteVerge，有投票权，进行投票签名
				// Weight is less than the voteVerge, has the right to vote, does the voting signature
				if bigWeight.Cmp(voteVerge) <= 0 {

					sign, err := chainhash.NewHash64(headerSign.Serialize())
					if err != nil {
						log.Errorf("Format conversion error: %s", err)
					}

					// 维护本地票池
					// Maintain local ticket pool
					signAndKey := SignAndKey{
						Signature: *sign,
						PublicKey: *pubKey,
					}
					ticketPool[headerHash] = append(ticketPool[headerHash], signAndKey)

					// 传播签名
					// Propagate signatures
					msgSign := &wire.MsgSign{
						BlockHeaderHash: headerHash,
						Signature:       *sign,
						PublicKey:       *pubKey,
					}
					m.cfg.SendSign(msgSign)
					return true

					// weight不符合情况，传播区块,返回true
					// weight don't conform to the situation, spread the block
				} else {
					return true
				}
			}
		}
	}

	return false
}

// 收集签名投票，传播投票
// Collect signatures and vote, spread the vote
func (m *CPUMiner) CollectVotes(msg *wire.MsgSign, headerBlock wire.BlockHeader) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	// 获取本节点公钥
	// Gets the node's public key
	publicKey := m.privKey.PubKey()

	// 查看是否是自己的签名投票
	// Check to see if it's your signature vote
	pubKey, err := chainhash.NewHash33(publicKey.SerializeCompressed())
	if err != nil {
		log.Errorf("Format conversion error: %s", err)
	}
	if preventRepeatSign(msg.BlockHeaderHash, *pubKey) {

		// 验证签名
		// Verify the signature
		signature, err := btcec.ParseSignature(msg.Signature.CloneBytes(), btcec.S256())
		if err != nil {
			log.Errorf("Parse error: %s", err)
		}
		pubKey, err := btcec.ParsePubKey(msg.PublicKey.CloneBytes(), btcec.S256())
		if err != nil {
			log.Errorf("Parse error: %s", err)
		}
		hash := msg.BlockHeaderHash.CloneBytes()
		if signature.Verify(hash, pubKey) {

			// 查看weight是否符合
			sign := msg.Signature.CloneBytes()
			weight := chainhash.DoubleHashB(sign)
			bigWeight := new(big.Int).SetBytes(weight)
			voteVerge := VoteVerge(headerBlock.Scale)
			// weight值小于voteVerge，此节点有投票权
			// Weight is less than the voteVerge, this node has the right to vote
			if bigWeight.Cmp(voteVerge) <= 0 {

				// 维护本地票池
				// Maintain local ticket pool
				signAndKey := SignAndKey{
					Signature: msg.Signature,
					PublicKey: msg.PublicKey,
				}
				ticketPool[msg.BlockHeaderHash] = append(ticketPool[msg.BlockHeaderHash], signAndKey)

				// 符合传播条件，传播签名
				// If propagation conditions are met, the signature is propagated
				m.cfg.SendSign(msg)
			}
		}
	}
}

// 避免重复投票或多次记录投票记录，如果没有重复，返回true
// Avoid duplicate voting or multiple records voting records, if no repeat, return true
func preventRepeatSign(blockHeaderHash chainhash.Hash, publicKey chainhash.Hash33) bool {

	// 查看签名池里是否有自己的签名,有的话返回true
	// Checks the signature pool to see if it has its own signature and returns true
	for headerHash, signAndKeys := range ticketPool {
		if headerHash == blockHeaderHash {
			// 遍历收到的区块所有的签名公钥
			// Iterates through all the signed public keys of the received block
			for _, signAndKey := range signAndKeys {
				if publicKey == signAndKey.PublicKey {
					return false
				}
			}
		}
	}
	return true
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

	for headerHash, signAndKeys := range ticketPool {
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

	signAndKeys := ticketPool[hash]

	return uint16(len(signAndKeys))
}

// 检查区块
// Check the block
func CheckBlock(msg wire.MsgBlock) bool {

	// 验证区块头的签名
	// Verify the signature of the block header
	signature, err := btcec.ParseSignature(msg.Header.Signature.CloneBytes(), btcec.S256())
	if err != nil {
		return false
	}
	pubKey, err := btcec.ParsePubKey(msg.Header.PublicKey.CloneBytes(), btcec.S256())
	if err != nil {
		return false
	}
	hash := msg.Header.PrevBlock.CloneBytes()
	if signature.Verify(hash, pubKey) {
		return true
	}
	return false
}

// 投票时间到，选出获胜区块上链，处理票池
// When it's time to vote, select the winner on the blockchain and process the pool of votes
func VoteProcess() {
	blockHeaderHash, _ := GetMaxVotes()
	blockPool := GetBlockPool()
	block := blockPool[blockHeaderHash]
	// 写入可能区块
	MayBlock(block)

	// 把本轮收到最多的上轮可能区块，写入区块链中
	WrittenChain()

	// 本轮投票结束，当前票池变成上一轮票池
	prevTicketPool = ticketPool
	// 清空当前票池票池
	ticketPool = make(map[chainhash.Hash][]SignAndKey)
}

// 获取当前票池
// Gets the current ticket pool
func GetTicketPool() map[chainhash.Hash][]SignAndKey {

	return ticketPool
}

// 获取之前的票池
// Gets the previous ticket pool
func GetPrevTicketPool() map[chainhash.Hash][]SignAndKey {

	return prevTicketPool
}
