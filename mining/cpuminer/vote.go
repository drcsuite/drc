package cpuminer

import (
	"github.com/drcsuite/drc/btcec"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/peer"
	"github.com/drcsuite/drc/wire"
	"math/big"
)

// The number of leading reference blocks required to evaluate a scale
// 求scale值需要的前置参考块的数量
const BlockCount = 10

// 理想发块节点数
// Ideal number of block nodes
const IdealBlockNum = 50

// 理想投票节点数
// Ideal number of voting nodes
const IdealVoteNum = 300

// 区块签名的票池
// Block signature of the ticket pool
var TicketPool = make(map[chainhash.Hash][]SignAndKey)

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

// 新块验证投票
// New block validation vote
func (m *CPUMiner) BlockVote(p peer.Peer, msg *wire.MsgBlock) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()

	// 获取本节点公私钥
	// Gets the node's public key and private key
	privateKey := m.privKey
	publicKey := privateKey.PubKey()

	// 验证区块
	// Verify the block
	if CheckBlock(*msg) {
		pubKey, err := chainhash.NewHash33(publicKey.SerializeCompressed())
		if err != nil {
			log.Errorf("Format conversion error: %s", err)
		}
		// 散列两次区块内容
		// Hash the contents of the block twice
		headerHash := msg.Header.BlockHash()

		// 之前未签名过本区块，分析处理
		// Had not signed this block before, analyse processing
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
				TicketPool[headerHash] = append(TicketPool[headerHash], signAndKey)

				// 传播签名和区块
				// Propagate signatures and blocks
				msgSign := &wire.MsgSign{
					BlockHeaderHash: headerHash,
					Signature:       *sign,
					PublicKey:       *pubKey,
				}
				p.QueueMessage(msgSign, nil)
				p.QueueMessage(msg, nil)

				// weight不符合情况，传播区块
				// weight don't conform to the situation, spread the block
			} else {
				p.QueueMessage(msg, nil)
			}
		}
	}
}

// 收集签名投票
// Collect signatures and vote
func (m *CPUMiner) CollectVotes(msg *wire.MsgSign) bool {
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

			// 维护本地票池
			// Maintain local ticket pool
			signAndKey := SignAndKey{
				Signature: msg.Signature,
				PublicKey: msg.PublicKey,
			}
			TicketPool[msg.BlockHeaderHash] = append(TicketPool[msg.BlockHeaderHash], signAndKey)

			return true
		}
	}
	return false
}

// 避免重复投票或多次记录投票记录，如果没有重复，返回true
// Avoid duplicate voting or multiple records voting records, if no repeat, return true
func preventRepeatSign(blockHeaderHash chainhash.Hash, publicKey chainhash.Hash33) bool {

	// 查看签名池里是否有自己的签名,有的话返回true
	// Checks the signature pool to see if it has its own signature and returns true
	for key, value := range TicketPool {
		if key == blockHeaderHash {
			// 遍历收到的区块所有的签名公钥
			// Iterates through all the signed public keys of the received block
			for _, signAndKey := range value {
				if publicKey == signAndKey.PublicKey {
					return false
				}
			}
		}
	}
	return true
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

// 从一段时间收到的所有区块中选择最适合的区块进行签名
func ChooseBlock(p peer.Peer, msg *wire.MsgBlock) {

	// 首先查看weight的大小

	//weight := chainhash.DoubleHashB(msg.Header.Signature.CloneBytes())
	//bigWeight := new(big.Int).SetBytes(weight)

	// 查看有没有积累了票数很多的区块

	// 验证是否为恶意块

}
