// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"

	"github.com/drcsuite/drc/database"
	"github.com/drcsuite/drc/drcutil"
)

// 可能会接受一个区块进入区块链,如果接受，返回是否在主链上。它执行取决于它在区块链中的位置的几个验证检查
// flag也被传递给checkBlockContext和connectBestChain
// 这个函数必须在持有链状态锁的情况下调用(用于写操作)。
// maybeAcceptBlock potentially accepts a block into the block chain and, if
// accepted, returns whether or not it is on the main chain.  It performs
// several validation checks which depend on its position within the block chain
// before adding it.  The block is expected to have already gone through
// ProcessBlock before calling this function with it.
//
// 标志也被传递给checkBlockContext和connectBestChain。有关标志如何修改其行为，请参阅它们的文档。
// The flags are also passed to checkBlockContext and connectBestChain.  See
// their documentation for how the flags modify their behavior.
//
// 此函数必须在持有链状态锁的情况下调用(用于写操作)。
// This function MUST be called with the chain state lock held (for writes).
func (b *BlockChain) maybeAcceptBlock(block *drcutil.Block, flags BehaviorFlags) (bool, error) {
	// The height of this block is one more than the referenced previous
	// block.
	//这个块的高度比前面引用的块高1。
	prevHash := &block.MsgBlock().Header.PrevBlock
	prevNode := b.index.LookupNode(prevHash)
	if prevNode == nil {
		str := fmt.Sprintf("previous block %s is unknown", prevHash)
		return false, ruleError(ErrPreviousBlockUnknown, str)
	} else if b.index.NodeStatus(prevNode).KnownInvalid() {
		str := fmt.Sprintf("previous block %s is known to be invalid", prevHash)
		return false, ruleError(ErrInvalidAncestorBlock, str)
	}

	blockHeight := prevNode.height + 1
	block.SetHeight(blockHeight)

	// 块必须通过所有依赖于块在块链中的位置的验证规则。
	// The block must pass all of the validation rules which depend on the
	// position of the block within the block chain.
	err := b.checkBlockContext(block, prevNode, flags)
	if err != nil {
		return false, err
	}

	//如果还没有块，就将它插入数据库。
	// 尽管该块最终可能无法连接，但它已经通过了所有的工作证明和有效性测试，这意味着攻击者用一堆无法连接的块填充磁盘的代价将非常高昂。
	// 这是必要的，因为它允许块下载从昂贵得多的连接逻辑解耦。
	// 它还具有其他一些很好的特性，比如使从不成为主链的一部分的块或连接失败的块可用作进一步的分析。
	// Insert the block into the database if it's not already there.  Even
	// though it is possible the block will ultimately fail to connect, it
	// has already passed all proof-of-work and validity tests which means
	// it would be prohibitively expensive for an attacker to fill up the
	// disk with a bunch of blocks that fail to connect.  This is necessary
	// since it allows block download to be decoupled from the much more
	// expensive connection logic.  It also has some other nice properties
	// such as making blocks that never become part of the main chain or
	// blocks that fail to connect available for further analysis.
	err = b.db.Update(func(dbTx database.Tx) error {
		return dbStoreBlock(dbTx, block)
	})
	if err != nil {
		return false, err
	}

	// 为块创建一个新的块节点，并将其添加到节点索引中。即使这个块最终连接到主链，它也从侧链开始。
	// Create a new block node for the block and add it to the node index. Even
	// if the block ultimately gets connected to the main chain, it starts out
	// on a side chain.

	blockHeader := &block.MsgBlock().Header
	newNode := newBlockNode(blockHeader, prevNode, block.Votes)
	newNode.status = statusDataStored

	b.index.AddNode(newNode)
	err = b.index.flushToDB()
	if err != nil {
		return false, err
	}
	log.Info("开始写入区块链")
	// 将通过的模块连接到链条上，同时要根据链条的正确选择，并提供最多的工作证明。这还处理事务脚本的验证。
	// Connect the passed block to the chain while respecting proper chain
	// selection according to the chain with the most proof of work.  This
	// also handles validation of the transaction scripts.
	isMainChain, err := b.connectBestChain(newNode, block, flags)
	log.Info("区块上链成功")
	log.Info("高度: ", block.Height())
	log.Info("version: ", block.MsgBlock().Header.Version)
	log.Info("scale: ", block.MsgBlock().Header.Scale)
	log.Info("timestamp: ", block.MsgBlock().Header.Timestamp)
	log.Info("blockhash: ", block.MsgBlock().BlockHash())

	if err != nil {
		return false, err
	}

	//通知调用者新块已被接受到块链中。调用方通常希望通过将库存转发给其他对等方来做出反应。
	// Notify the caller that the new block was accepted into the block
	// chain.  The caller would typically want to react by relaying the
	// inventory to other peers.
	//b.chainLock.Unlock()
	//b.sendNotification(NTBlockAccepted, block)
	//b.chainLock.Lock()

	return isMainChain, nil
}
