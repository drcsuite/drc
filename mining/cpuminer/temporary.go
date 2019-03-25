package cpuminer

import (
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/wire"
	"time"
)

// 临时接口，最后删除

// 块池
var blockPool = make(map[chainhash.Hash]wire.BlockHeader)

// 获取块池
func GetBlockPool() map[chainhash.Hash]wire.BlockHeader {

	return blockPool
}

// 获取创世时间
func GetCreationTime() time.Time {

	return time.Time{}

}

// 获取最新块的高度
func GetBlockHeight() time.Duration {

	return 0

}

// 上链
func WrittenChain() {

}

// 写入可能区块缓存
func MayBlock(wire.BlockHeader) {

}
