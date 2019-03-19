package cpuminer

import (
	"github.com/drcsuite/drc/btcec"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/peer"
	"github.com/drcsuite/drc/wire"
	"math/big"
)

// 求scale值需要的前置参考块的数量
const BlockCount = 10

// 理想发块节点数
const IdealBlockNum = 50

// 理想投票节点数
const IdealVoteNum = 300

// 区块签名的票池
var TicketPool = make(map[chainhash.Hash][]SignAndKey)

// 具有投票权的节点对区块的签名值和验证时用的公钥
type SignAndKey struct {
	Signature chainhash.Hash64

	PublicKey chainhash.Hash33
}

// 估算全网节点总数
func EstimateScale(prevVoteNums []uint16, prevScales []uint16) (scale uint16) {

	meanScale := mean(prevScales)
	meanVoteNum := mean(prevVoteNums)

	//上一个区块的Scale小于等于300，说明全网节点总数很少，之前收到多少投票就可估算为当前的节点总数。
	if meanScale <= IdealVoteNum {

		scale = meanVoteNum

		// 上一个区块的Scale大于300，说明全网节点总数大于300，需计算使符合投票的节点数更接近300.
	} else {

		scale = uint16(uint32(meanScale) * uint32(meanVoteNum) / IdealVoteNum)

	}
	return
}

// 求[]uint16类型切片的平均值
func mean(values []uint16) (meanValue uint16) {
	var totalValue uint16 = 0
	for _, value := range values {
		totalValue = totalValue + value
	}
	meanValue = totalValue / uint16(len(values))
	return
}

// 计算投票的π值
func VoteVerge(scale uint16) *big.Int {

	// bigint格式的全网节点数
	bigScale := big.NewInt(int64(scale))
	// bigint格式的理想投票节点数
	bigIdealVoteNum := big.NewInt(IdealVoteNum)
	// bigint格式的2的256次方的值
	max256, _ := new(big.Int).SetString("10000000000000000000000000000000000000000000000000000000000000000", 16)

	// 投票π值计算
	max256.Mul(max256, bigIdealVoteNum)
	max256.Quo(max256, bigScale)

	return max256
}

// 计算发块的π值
func BlockVerge(scale uint16) *big.Int {

	bigIdealBlockNum := big.NewInt(IdealBlockNum)
	bigIdealVoteNum := big.NewInt(IdealVoteNum)
	verge := VoteVerge(scale)

	// 发块π值计算
	verge.Mul(verge, bigIdealBlockNum)
	verge.Quo(verge, bigIdealVoteNum)
	return verge
}

// 新块验证投票
func BlockVote(p peer.Peer, msg *wire.MsgBlock, publicKey *btcec.PublicKey, privateKey *btcec.PrivateKey) {

	if CheckBlock(*msg) {
		pubKey, err := chainhash.NewHash33(publicKey.SerializeCompressed())
		CheckAndPanic(err)
		// 散列两次区块内容
		headerHash := msg.Header.BlockHash()

		// 之前未签名过本区块，分析处理
		if preventRepeatSign(headerHash, *pubKey) {

			// 用自己的私钥签名区块
			headerSign, err := privateKey.Sign(headerHash.CloneBytes())
			CheckAndPanic(err)
			// 计算本节点的投票weight
			weight := chainhash.DoubleHashB(headerSign.Serialize())
			bigWeight := new(big.Int).SetBytes(weight)

			voteVerge := VoteVerge(msg.Header.Scale)

			// weight值小于voteVerge，有投票权，进行投票签名
			if bigWeight.Cmp(voteVerge) <= 0 {

				sign, err := chainhash.NewHash64(headerSign.Serialize())
				CheckAndPanic(err)

				// 扩散签名
				msgSign := &wire.MsgSign{
					BlockHeaderHash: headerHash,
					Signature:       *sign,
					PublicKey:       *pubKey,
				}
				p.QueueMessage(msgSign, nil)
			}
			// 之前对本区块签过名，直接转发出去
		} else {
			p.QueueMessage(msg, nil)
		}
	}
}

// 防止重复签名区块
func preventRepeatSign(blockHeaderHash chainhash.Hash, publicKey chainhash.Hash33) bool {

	// 查看签名池里是否有自己的签名,有的话返回true
	for key, value := range TicketPool {
		if key == blockHeaderHash {
			// 遍历收到的区块所有的签名公钥
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
func CheckBlock(msg wire.MsgBlock) bool {

	//验证区块头的签名
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
	} else {
		return false
	}
	return false
}

// Check And Panic
func CheckAndPanic(err error) {
	if err != nil {
		panic(err)
	}
}
