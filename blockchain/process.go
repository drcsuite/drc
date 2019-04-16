// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/database"
	"github.com/drcsuite/drc/drcutil"
	"github.com/drcsuite/drc/vote"
	"github.com/drcsuite/drc/wire"
	"math/big"
)

// behavior flags是一个位掩码，它定义了在执行链处理和一致规则检查时对正常行为的调整。
// BehaviorFlags is a bitmask defining tweaks to the normal behavior when
// performing chain processing and consensus rules checks.
type BehaviorFlags uint32

const (
	// BFFastAdd may be set to indicate that several checks can be avoided
	// for the block since it is already known to fit into the chain due to
	// already proving it correct links into the chain up to a known
	// checkpoint.  This is primarily used for headers-first mode.
	BFFastAdd BehaviorFlags = 1 << iota

	// BFNoPoWCheck may be set to indicate the proof of work check which
	// ensures a block hashes to a value less than the required target will
	// not be performed.
	BFNoPoWCheck

	// BFNone是一个方便的值，专门指出没有标志。
	// BFNone is a convenience value to specifically indicate no flags.
	BFNone BehaviorFlags = 0
)

var (
	// 前一轮块池
	PrevCandidatePool map[chainhash.Hash]*wire.MsgCandidate
	// 当前轮块池
	CurrentCandidatePool map[chainhash.Hash]*wire.MsgCandidate

	// 当前轮指向池
	CurrentPointPool map[chainhash.Hash][]*wire.MsgCandidate
)

// 获取当前轮块池，多数指向的前一轮块的Hash
func GetBestPointBlockH() chainhash.Hash {
	var bestHash chainhash.Hash
	best := 0
	for k, v := range CurrentPointPool {
		l := len(v)
		if l > best {
			best = l
			bestHash = k
		}
	}
	return bestHash
}

// block exists确定具有给定散列的块是否存在于主链或任何侧链中。
// blockExists determines whether a block with the given hash exists either in
// the main chain or any side chains.
//
// This function is safe for concurrent access.
func (b *BlockChain) blockExists(hash *chainhash.Hash) (bool, error) {
	// Check block index first (could be main chain or side chain blocks).
	if b.index.HaveBlock(hash) {
		return true, nil
	}

	// Check in the database.
	var exists bool
	err := b.db.View(func(dbTx database.Tx) error {
		var err error
		exists, err = dbTx.HasBlock(hash)
		if err != nil || !exists {
			return err
		}

		// Ignore side chain blocks in the database.  This is necessary
		// because there is not currently any record of the associated
		// block index data such as its block height, so it's not yet
		// possible to efficiently load the block and do anything useful
		// with it.
		//
		// Ultimately the entire block index should be serialized
		// instead of only the current main chain so it can be consulted
		// directly.
		_, err = dbFetchHeightByHash(dbTx, hash)
		if isNotInMainChainErr(err) {
			exists = false
			return nil
		}
		return err
	})
	return exists, err
}

// 前一轮块池是否存在该hash
func (b *BlockChain) PrevCandidateExists(hash *chainhash.Hash) bool {
	// Check in the database.
	exist := PrevCandidatePool[*hash]
	if exist != nil {
		return true
	}
	return false
}

// 当前轮块池是否存在该hash
func (b *BlockChain) CandidateExists(hash *chainhash.Hash) bool {
	// Check in the database.
	exist := CurrentCandidatePool[*hash]
	if exist != nil {
		return true
	}
	return false
}

// 选出块池中weight最小的块和weight
func GetMinWeightBlock() (*wire.MsgCandidate, *big.Int) {
	min, _ := new(big.Int).SetString("10000000000000000000000000000000000000000000000000000000000000000", 16)
	var minCandidate *wire.MsgCandidate
	if CurrentCandidatePool != nil {
		for _, v := range CurrentCandidatePool {
			weight := new(big.Int).SetBytes(chainhash.DoubleHashB(v.Header.Signature.CloneBytes()))
			if weight.Cmp(min) < 0 {
				min = weight
				minCandidate = v
			}
		}
	}
	return minCandidate, min
}

//确定是否存在依赖于传递的块散列的孤子(如果为真，则不再是孤子)，并可能接受它们。
// processOrphans determines if there are any orphans which depend on the passed
// block hash (they are no longer orphans if true) and potentially accepts them.
// It repeats the process for the newly accepted blocks (to detect further
// orphans which may no longer be orphans) until there are no more.
//
// The flags do not modify the behavior of this function directly, however they
// are needed to pass along to maybeAcceptBlock.
//
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) processOrphans(hash *chainhash.Hash, flags BehaviorFlags) error {
	// Start with processing at least the passed hash.  Leave a little room
	// for additional orphan blocks that need to be processed without
	// needing to grow the array in the common case.
	processHashes := make([]*chainhash.Hash, 0, 10)
	processHashes = append(processHashes, hash)
	for len(processHashes) > 0 {
		// Pop the first hash to process from the slice.
		processHash := processHashes[0]
		processHashes[0] = nil // Prevent GC leak.
		processHashes = processHashes[1:]

		// 看看我们刚刚接收的街区里所有的孤块。
		// 这通常只会是一个，但是如果同时挖掘和广播多个块，则可能是多个块。
		// 工作证明最多的人最终会胜出。
		// 这里有意在一个范围上使用循环索引，因为范围不会在每次迭代时重新评估片，也不会调整修改片的索引。
		// Look up all orphans that are parented by the block we just
		// accepted.  This will typically only be one, but it could
		// be multiple if multiple blocks are mined and broadcast
		// around the same time.  The one with the most proof of work
		// will eventually win out.  An indexing for loop is
		// intentionally used over a range here as range does not
		// reevaluate the slice on each iteration nor does it adjust the
		// index for the modified slice.
		for i := 0; i < len(b.prevOrphans[*processHash]); i++ {
			orphan := b.prevOrphans[*processHash][i]
			if orphan == nil {
				log.Warnf("Found a nil entry at index %d in the "+
					"orphan dependency list for block %v", i,
					processHash)
				continue
			}

			// Remove the orphan from the orphan pool.
			orphanHash := orphan.block.Hash()
			b.removeOrphanBlock(orphan)
			i--

			// Potentially accept the block into the block chain.
			_, err := b.maybeAcceptBlock(orphan.block, flags)
			if err != nil {
				return err
			}

			// Add this block to the list of blocks to process so
			// any orphan blocks that depend on this block are
			// handled too.
			processHashes = append(processHashes, orphanHash)
		}
	}
	return nil
}

// ProcessBlock是处理将新块插入到块链中的主要工作部件。
// 它包括拒绝重复块、确保块遵循所有规则、孤立处理和插入到块链以及最佳链选择和重组等功能。
// ProcessBlock is the main workhorse for handling insertion of new blocks into
// the block chain.  It includes functionality such as rejecting duplicate
// blocks, ensuring blocks follow all rules, orphan handling, and insertion into
// the block chain along with best chain selection and reorganization.
//
// 当处理过程中没有发生错误时，第一个返回值指示该块是否在主链上，第二个返回值指示该块是否是孤儿。
// When no errors occurred during processing, the first return value indicates
// whether or not the block is on the main chain and the second indicates
// whether or not the block is an orphan.
//
// This function is safe for concurrent access.
func (b *BlockChain) ProcessBlock(block *drcutil.Block, flags BehaviorFlags) (bool, bool, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	blockHash := block.Hash()
	log.Tracef("Processing block %v", blockHash)

	// 该块不能已经存在于主链或侧链中。
	// The block must not already exist in the main chain or side chains.
	exists, err := b.blockExists(blockHash)
	if err != nil {
		return false, false, err
	}
	if exists {
		str := fmt.Sprintf("already have block %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	//该块不能作为孤儿存在。
	// The block must not already exist as an orphan.
	if _, exists := b.orphans[*blockHash]; exists {
		str := fmt.Sprintf("already have block (orphan) %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	blockHeader := &block.MsgBlock().Header
	// Handle orphan blocks.
	prevHash := &blockHeader.PrevBlock
	prevHashExists, err := b.blockExists(prevHash)
	if err != nil {
		return false, false, err
	}
	if !prevHashExists {
		log.Infof("Adding orphan block %v with parent %v", blockHash, prevHash)
		b.addOrphanBlock(block)

		return false, true, nil
	}

	//对块及其事务执行初步的完整性检查。
	// Perform preliminary sanity checks on the block and its transactions.
	node := b.index.LookupNode(prevHash)
	seed := chainhash.DoubleHashH(node.signature.CloneBytes())
	Pi := vote.BlockVerge(blockHeader.Scale)
	err = checkBlockSanity(block, &seed, Pi, b.timeSource) // seed pi
	if err != nil {
		return false, false, err
	}

	// The block has passed all context independent checks and appears sane
	// enough to potentially accept it into the block chain.
	isMainChain, err := b.maybeAcceptBlock(block, flags)
	if err != nil {
		return false, false, err
	}

	// 接受任何依赖于此块的孤立块(它们确实如此)
	// 不再是孤儿)，并重复这些接受街区，直到
	// 没有了。
	// Accept any orphan blocks that depend on this block (they are
	// no longer orphans) and repeat for those accepted blocks until
	// there are no more.
	err = b.processOrphans(blockHash, flags)
	if err != nil {
		return false, false, err
	}

	log.Debugf("Accepted block %v", blockHash)

	return isMainChain, false, nil
}

// 处理同步增量块
func (b *BlockChain) ProcessSyncBlock(block *drcutil.Block) (bool, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	blockHash := block.Hash()
	log.Tracef("Processing block %v", blockHash)

	// 该块不能已经存在于主链或侧链中。
	// The block must not already exist in the main chain or side chains.
	exists, err := b.blockExists(blockHash)
	if err != nil {
		return false, err
	}
	if exists {
		str := fmt.Sprintf("already have block %v", blockHash)
		return false, ruleError(ErrDuplicateBlock, str)
	}

	//该块不能作为孤儿存在。
	// The block must not already exist as an orphan.
	if _, exists := b.orphans[*blockHash]; exists {
		str := fmt.Sprintf("already have block (orphan) %v", blockHash)
		return false, ruleError(ErrDuplicateBlock, str)
	}

	prevHash := &block.MsgBlock().Header.PrevBlock
	prevHashExists, err := b.blockExists(prevHash)
	if err != nil {
		return false, nil
	}
	if !prevHashExists {
		log.Infof("Adding orphan block %v with parent %v", blockHash, prevHash)
		// 添加入孤块池
		b.addOrphanBlock(block)

		return true, nil
	}

	// The block has passed all context independent checks and appears sane
	// enough to potentially accept it into the block chain.
	_, err = b.maybeAcceptBlock(block, BFNone)
	if err != nil {
		return false, err
	}

	// 接受任何依赖于此块的孤立块(它们确实如此)
	// 不再是孤儿)，并重复这些接受街区，直到
	// 没有了。
	// Accept any orphan blocks that depend on this block (they are
	// no longer orphans) and repeat for those accepted blocks until
	// there are no more.
	err = b.processOrphans(blockHash, BFNone)
	if err != nil {
		return false, err
	}

	log.Debugf("Accepted sync block %v", blockHash)

	return true, nil
}

// 处理发块阶段收到的块，并将该块放入块池和指向池
// 																			是否投票，块是否符合规则，错误信息
func (b *BlockChain) ProcessCandidate(block *drcutil.Block, flags BehaviorFlags) (bool, bool, error) {
	b.chainLock.Lock()
	defer b.chainLock.Unlock()

	//fastAdd := flags&BFFastAdd == BFFastAdd
	blockHash := block.CandidateHash()
	log.Tracef("Processing block %v", blockHash)

	// 该块不能已经存在于块池中
	// The block must not already exist in the main chain or side chains.
	exists := b.CandidateExists(blockHash)
	if exists {
		str := fmt.Sprintf("already have block %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	//该块不能作为孤儿存在。
	// The block must not already exist as an orphan.
	if _, exists := b.orphans[*blockHash]; exists {
		str := fmt.Sprintf("already have block (orphan) %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	// 发块阶段收到的块BestLastCandidate不能为空
	best := b.BestLastCandidate()

	//对块及其事务执行初步的完整性检查。
	// Perform preliminary sanity checks on the block and its transactions.
	// 计算Pi,阈值规模

	seed := chainhash.DoubleHashH(best.Header.Signature.CloneBytes())
	votes, scales := make([]uint16, 0), make([]uint16, 0)
	scales = append(scales, best.Header.Scale)
	votes = append(votes, best.Votes)
	node := b.GetBlockIndex().LookupNode(&best.Header.PrevBlock)
	if node != nil {
		for i := 0; i < vote.PrevScaleNum; i++ {
			// 添加每个节点实际收到的票数和当时估算值
			if node == nil {
				break
			}
			scales = append(scales, node.Header().Scale)
			votes = append(votes, node.Votes)
			hashes := node.Header().PrevBlock
			node = b.GetBlockIndex().LookupNode(&hashes)
		}
	}
	scale := vote.EstimateScale(votes, scales)
	Pi := vote.BlockVerge(scale)

	// 完整性检查
	vb, err := checkCandidateSanity(block, &seed, Pi, b.timeSource) // seed pi
	if err != nil {
		return false, false, err
	}

	// Find the previous checkpoint and perform some additional checks based
	// on the checkpoint.  This provides a few nice properties such as
	// preventing old side chain blocks before the last checkpoint,
	// rejecting easy to mine, but otherwise bogus, blocks that could be
	// used to eat memory, and ensuring expected (versus claimed) proof of
	// work requirements since the previous checkpoint are met.
	blockHeader := &block.MsgCandidate().Header

	// 处理该块前项块
	prevHash := &blockHeader.PrevBlock
	prevHashExists := b.PrevCandidateExists(prevHash)
	if !prevHashExists {
		str := fmt.Sprintf("The preceding block of this block does not exist: %v", blockHash)
		return false, false, ruleError(ErrDuplicateBlock, str)
	}

	// 加入指向池 和 当前块池
	CurrentCandidatePool[*blockHash] = block.MsgCandidate()
	points := CurrentPointPool[best.Hash]
	points = append(points, block.MsgCandidate())
	CurrentPointPool[best.Hash] = points
	log.Debugf("Accepted block %v", blockHash)

	return vb, true, nil
}
