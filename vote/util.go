package vote

import (
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"math"
	"math/big"
	"sync"
	"time"
)

const (
	// 理想发块节点数
	// Ideal number of block nodes
	IdealBlockNum = 50

	// 理想投票节点数
	// Ideal number of voting nodes
	IdealVoteNum = 300

	// 发块时间间隔
	// Block time interval
	BlockTimeInterval = 10 * time.Second

	// 为同步到最新的发块时间所需的时间间隔
	// the time interval required to synchronize to the latest block time
	SyncTimeInterval = 20 * time.Second

	// The number of items in the preceding paragraph that scale looks for
	// 计算scale时寻找的前项数量=1+PrevScaleNum
	PrevScaleNum = 9
)

var (
	Work bool
	// 读写锁
	// Read-write lock
	RWSyncMutex = new(sync.RWMutex)
	// 区块签名的票池
	// Block signature of the ticket pool
	ticketPool = make(map[chainhash.Hash][]SignAndKey)

	// 前一区块的签名票池
	// The signature ticket pool for the previous block
	prevTicketPool = make(map[chainhash.Hash][]SignAndKey)

	CurrentHeight int32 // 当前轮高度

	StartHeight int32 // 项目启动时高度

	// 第一个块生成时间
	// （当前时间-第一个块时间）/10 +1 = 当前轮高度
	FirstBLockTime = time.Unix(time.Now().Unix(), 0)
)

// 具有投票权的节点对区块的签名值和验证时用的公钥
// The signed value of the block by the voting node and the public key used for verification
type SignAndKey struct {
	Signature chainhash.Hash64

	PublicKey chainhash.Hash33
}

// 计算投票的∏值
// To calculate the ∏ value of a vote
func VotesVerge(scale uint16) *big.Int {

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
	verge := VotesVerge(scale)

	// 发块∏值计算
	// A block ∏ value calculation
	verge.Mul(verge, bigIdealBlockNum)
	verge.Quo(verge, bigIdealVoteNum)
	return verge
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
	meanValue = uint16(math.Ceil(float64(totalValue) / float64(len(values))))
	return
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

// 写入当前票池
// write to current ticket pool
func SetTicketPool(pool map[chainhash.Hash][]SignAndKey) {
	ticketPool = pool
}

// 写入之前的票池
// write to previous ticket pool
func SetPrevTicketPool(pool map[chainhash.Hash][]SignAndKey) {
	ticketPool = pool
}

// 更新当前票池，加写锁，防止同时修改票池出现错误
// Update the current ticket pool and add write lock to prevent errors in modifying the ticket pool at the same time
func UpdateTicketPool(hash chainhash.Hash, signAndKey SignAndKey) {

	RWSyncMutex.Lock()
	ticketPool := GetTicketPool()
	ticketPool[hash] = append(ticketPool[hash], signAndKey)
	SetTicketPool(ticketPool)
	RWSyncMutex.Unlock()

}
