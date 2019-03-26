// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package netsync

import (
	"container/list"
	"github.com/drcsuite/drc/mining/cpuminer"

	"sync"
	"sync/atomic"
	"time"

	"github.com/drcsuite/drc/blockchain"
	"github.com/drcsuite/drc/chaincfg"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/database"
	"github.com/drcsuite/drc/drcutil"
	"github.com/drcsuite/drc/mempool"
	peerpkg "github.com/drcsuite/drc/peer"
	"github.com/drcsuite/drc/wire"
	//"github.com/drcsuite/drc/mining/cpuminer"
)

const (
	// 是在请求更多块之前，头优先模式的请求队列中应该包含的最小块数。
	// minInFlightBlocks is the minimum number of blocks that should be
	// in the request queue for headers-first mode before requesting
	// more.
	minInFlightBlocks = 10

	// maxRejectedTxns是要存储在内存中的最大被拒绝事务散列数。
	// maxRejectedTxns is the maximum number of rejected transactions
	// hashes to store in memory.
	maxRejectedTxns = 1000

	// maxRequestedBlocks是要存储在内存中的最大请求块散列数。
	// maxRequestedBlocks is the maximum number of requested block
	// hashes to store in memory.
	maxRequestedBlocks = wire.MaxInvPerMsg

	// maxRequestedTxns是要存储在内存中的最大请求事务散列数。
	// maxRequestedTxns is the maximum number of requested transactions
	// hashes to store in memory.
	maxRequestedTxns = wire.MaxInvPerMsg
)

// zeroHash是零值散列(全部为零)。它被定义为一种便利。
// zeroHash is the zero value hash (all zeros).  It is defined as a convenience.
var zeroHash chainhash.Hash

// newPeerMsg表示一个新连接到块处理程序的对等点。
// newPeerMsg signifies a newly connected peer to the block handler.
type newPeerMsg struct {
	peer *peerpkg.Peer
}

// blockMsg将比特币块消息及其来自的对等方打包在一起，因此块处理程序可以访问该信息。
// blockMsg packages a bitcoin block message and the peer it came from together
// so the block handler has access to that information.
type blockMsg struct {
	block *drcutil.Block
	peer  *peerpkg.Peer
	reply chan struct{}
}

type candidateMsg struct {
	block *drcutil.Block
	peer  *peerpkg.Peer
	reply chan struct{}
}

// invMsg将比特币inv消息及其来自的对等方打包在一起，以便块处理程序能够访问该信息。
// invMsg packages a bitcoin inv message and the peer it came from together
// so the block handler has access to that information.
type invMsg struct {
	inv  *wire.MsgInv
	peer *peerpkg.Peer
}

// headersMsg将比特币头部消息及其来自的对等方打包在一起，因此块处理程序可以访问该信息。
// headersMsg packages a bitcoin headers message and the peer it came from
// together so the block handler has access to that information.
type headersMsg struct {
	headers *wire.MsgHeaders
	peer    *peerpkg.Peer
}

// donePeerMsg表示块处理程序的一个新断开连接的对等点。
// donePeerMsg signifies a newly disconnected peer to the block handler.
type donePeerMsg struct {
	peer *peerpkg.Peer
}

// txMsg将比特币tx消息及其来自的对等方打包在一起，因此块处理程序可以访问该信息。
// txMsg packages a bitcoin tx message and the peer it came from together
// so the block handler has access to that information.
type txMsg struct {
	tx    *drcutil.Tx
	peer  *peerpkg.Peer
	reply chan struct{}
}

// getSyncPeerMsg是一种通过消息通道发送的消息类型，用于检索当前同步对等点。
// getSyncPeerMsg is a message type to be sent across the message channel for
// retrieving the current sync peer.
type getSyncPeerMsg struct {
	reply chan int32
}

// processBlockResponse是发送到processBlockMsg的应答通道的响应。
// processBlockResponse is a response sent to the reply channel of a
// processBlockMsg.
type processBlockResponse struct {
	isOrphan bool
	err      error
}
type sendBlockResponse struct {
	isOrphan bool
	err      error
}

type sendSignResponse struct {
	isOrphan bool
	err      error
}

// processBlockMsg是一种通过消息通道发送的消息类型，用于处理请求的块。
// 注意，这个调用不同于上面的块msg，因为块msg是为来自对等点并具有额外处理的块而设计的，而这个消息实际上只是在内部块链实例上调用ProcessBlock的一种并发安全方法。
// processBlockMsg is a message type to be sent across the message channel
// for requested a block is processed.  Note this call differs from blockMsg
// above in that blockMsg is intended for blocks that came from peers and have
// extra handling whereas this message essentially is just a concurrent safe
// way to call ProcessBlock on the internal block chain instance.
type processBlockMsg struct {
	block *drcutil.Block
	flags blockchain.BehaviorFlags
	reply chan processBlockResponse
}
type sendBlockMsg struct {
	block *drcutil.Block
	//flags blockchain.BehaviorFlags
	reply chan sendBlockResponse
}

type sendSignMsg struct {
	msgSign *wire.MsgSign
	reply   chan sendSignResponse
}

// isCurrentMsg是一种通过消息通道发送的消息类型，用于请求sync manager是否认为它与当前连接的对等点同步。
// isCurrentMsg is a message type to be sent across the message channel for
// requesting whether or not the sync manager believes it is synced with the
// currently connected peers.
type isCurrentMsg struct {
	reply chan bool
}

// pauseMsg是通过消息通道发送的消息类型，用于暂停同步管理器。这有效地为调用者提供了通过管理器的独占访问，直到在unpause通道上执行接收为止。
// pauseMsg is a message type to be sent across the message channel for
// pausing the sync manager.  This effectively provides the caller with
// exclusive access over the manager until a receive is performed on the
// unpause channel.
type pauseMsg struct {
	unpause <-chan struct{}
}

// headerNode用作检查点之间链接在一起的报头列表中的节点。
// headerNode is used as a node in a list of headers that are linked together
// between checkpoints.
type headerNode struct {
	height int32
	hash   *chainhash.Hash
}

// peerSyncState存储同步管理器跟踪的关于对等点的其他信息。
// peerSyncState stores additional information that the SyncManager tracks
// about a peer.
type peerSyncState struct {
	syncCandidate   bool
	requestQueue    []*wire.InvVect
	requestedTxns   map[chainhash.Hash]struct{}
	requestedBlocks map[chainhash.Hash]struct{}
}

// SyncManager用于与对等方通信块相关的消息。通过在goroutine中执行Start()启动SyncManager。启动后，它选择要同步的对等点，并启动初始块下载。一旦链同步，SyncManager将处理传入的块和头通知，并将新块的通知转发给对等方。
// SyncManager is used to communicate block related messages with peers. The
// SyncManager is started as by executing Start() in a goroutine. Once started,
// it selects peers to sync from and starts the initial block download. Once the
// chain is in sync, the SyncManager handles incoming block and header
// notifications and relays announcements of new blocks to peers.
type SyncManager struct {
	peerNotifier   PeerNotifier
	started        int32
	shutdown       int32
	chain          *blockchain.BlockChain
	txMemPool      *mempool.TxPool
	chainParams    *chaincfg.Params
	progressLogger *blockProgressLogger
	msgChan        chan interface{}
	wg             sync.WaitGroup
	quit           chan struct{}

	// These fields should only be accessed from the blockHandler thread
	rejectedTxns    map[chainhash.Hash]struct{}
	requestedTxns   map[chainhash.Hash]struct{}
	requestedBlocks map[chainhash.Hash]struct{}
	syncPeer        *peerpkg.Peer
	peerStates      map[*peerpkg.Peer]*peerSyncState

	// The following fields are used for headers-first mode.
	headersFirstMode bool
	headerList       *list.List
	startHeader      *list.Element
	//nextCheckpoint   *chaincfg.Checkpoint

	// An optional fee estimator.
	feeEstimator *mempool.FeeEstimator
}

// resetHeaderState将headers-first模式状态设置为适合从新对等点同步的值。
// resetHeaderState sets the headers-first mode state to values appropriate for
// syncing from a new peer.
func (sm *SyncManager) resetHeaderState(newestHash *chainhash.Hash, newestHeight int32) {
	sm.headersFirstMode = false
	sm.headerList.Init()
	sm.startHeader = nil

	// When there is a next checkpoint, add an entry for the latest known
	// block into the header pool.  This allows the next downloaded header
	// to prove it links to the chain properly.
	//if sm.nextCheckpoint != nil {
	//	node := headerNode{height: newestHeight, hash: newestHash}
	//	sm.headerList.PushBack(&node)
	//}
}

// findNextHeaderCheckpoint在传递的高度之后返回下一个检查点。当没有空值时，它返回nil，要么是因为高度已经比最终检查点晚，要么是因为禁用检查点等其他原因。
// findNextHeaderCheckpoint returns the next checkpoint after the passed height.
// It returns nil when there is not one either because the height is already
// later than the final checkpoint or some other reason such as disabled
// checkpoints.
//func (sm *SyncManager) findNextHeaderCheckpoint(height int32) *chaincfg.Checkpoint {
//	checkpoints := sm.chain.Checkpoints()
//	if len(checkpoints) == 0 {
//		return nil
//	}
//
//	// There is no next checkpoint if the height is already after the final
//	// checkpoint.
//	finalCheckpoint := &checkpoints[len(checkpoints)-1]
//	if height >= finalCheckpoint.Height {
//		return nil
//	}
//
//	// Find the next checkpoint.
//	nextCheckpoint := finalCheckpoint
//	for i := len(checkpoints) - 2; i >= 0; i-- {
//		if height >= checkpoints[i].Height {
//			break
//		}
//		nextCheckpoint = &checkpoints[i]
//	}
//	return nextCheckpoint
//}

// startSync将从可用的候选对等点中选择最好的对等点来下载/同步区块链。当同步已经在运行时，它只是返回。它还检查候选人中是否有任何不再是候选人的，并根据需要删除他们。
// startSync will choose the best peer among the available candidate peers to
// download/sync the blockchain from.  When syncing is already running, it
// simply returns.  It also examines the candidates for any which are no longer
// candidates and removes them as needed.
func (sm *SyncManager) startSync() {
	// Return now if we're already syncing.
	if sm.syncPeer != nil {
		return
	}

	// 一旦segwit软分叉包被激活，我们只想从启用了witness的节点进行同步，以确保我们完全验证所有区块链数据。
	// Once the segwit soft-fork package has activated, we only
	// want to sync from peers which are witness enabled to ensure
	// that we fully validate all blockchain data.
	segwitActive, err := sm.chain.IsDeploymentActive(chaincfg.DeploymentSegwit)
	if err != nil {
		log.Errorf("Unable to query for segwit soft-fork state: %v", err)
		return
	}

	best := sm.chain.BestSnapshot()
	var bestPeer *peerpkg.Peer
	for peer, state := range sm.peerStates {
		if !state.syncCandidate {
			continue
		}

		if segwitActive && !peer.IsWitnessEnabled() {
			log.Debugf("peer %v not witness enabled, skipping", peer)
			continue
		}

		// Remove sync candidate peers that are no longer candidates due
		// to passing their latest known block.  NOTE: The < is
		// intentional as opposed to <=.  While technically the peer
		// doesn't have a later block when it's equal, it will likely
		// have one soon so it is a reasonable choice.  It also allows
		// the case where both are at 0 such as during regression test.
		if peer.LastBlock() < best.Height {
			state.syncCandidate = false
			continue
		}

		// TODO(davec): Use a better algorithm to choose the best peer.
		// For now, just pick the first available candidate.
		bestPeer = peer
	}

	// Start syncing from the best peer if one was selected.
	if bestPeer != nil {
		// Clear the requestedBlocks if the sync peer changes, otherwise
		// we may ignore blocks we need that the last sync peer failed
		// to send.
		sm.requestedBlocks = make(map[chainhash.Hash]struct{})

		//locator, err := sm.chain.LatestBlockLocator()
		if err != nil {
			log.Errorf("Failed to get block locator for the "+
				"latest block: %v", err)
			return
		}

		log.Infof("Syncing to block height %d from peer %v",
			bestPeer.LastBlock(), bestPeer.Addr())

		// When the current height is less than a known checkpoint we
		// can use block headers to learn about which blocks comprise
		// the chain up to the checkpoint and perform less validation
		// for them.  This is possible since each header contains the
		// hash of the previous header and a merkle root.  Therefore if
		// we validate all of the received headers link together
		// properly and the checkpoint hashes match, we can be sure the
		// hashes for the blocks in between are accurate.  Further, once
		// the full blocks are downloaded, the merkle root is computed
		// and compared against the value in the header which proves the
		// full block hasn't been tampered with.
		//
		// Once we have passed the final checkpoint, or checkpoints are
		// disabled, use standard inv messages learn about the blocks
		// and fully validate them.  Finally, regression test mode does
		// not support the headers-first approach so do normal block
		// downloads when in regression test mode.
		//if sm.nextCheckpoint != nil &&
		//	best.Height < sm.nextCheckpoint.Height &&
		//	sm.chainParams != &chaincfg.RegressionNetParams {
		//
		//	bestPeer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
		//	sm.headersFirstMode = true
		//	log.Infof("Downloading headers for blocks %d to "+
		//		"%d from peer %s", best.Height+1,
		//		sm.nextCheckpoint.Height, bestPeer.Addr())
		//} else {
		//	bestPeer.PushGetBlocksMsg(locator, &zeroHash)
		//}
		sm.syncPeer = bestPeer
	} else {
		log.Warnf("No sync peer candidates available")
	}
}

// isSyncCandidate 返回该对等点是否是考虑同步的候选对象。
// isSyncCandidate returns whether or not the peer is a candidate to consider
// syncing from.
func (sm *SyncManager) isSyncCandidate(peer *peerpkg.Peer) bool {
	// Typically a peer is not a candidate for sync if it's not a full node,
	// however regression test is special in that the regression tool is
	// not a full node and still needs to be considered a sync candidate.
	//if sm.chainParams == &chaincfg.RegressionNetParams {
	//	// The peer is not a candidate if it's not coming from localhost
	//	// or the hostname can't be determined for some reason.
	//	host, _, err := net.SplitHostPort(peer.Addr())
	//	if err != nil {
	//		return false
	//	}
	//
	//	if host != "127.0.0.1" && host != "localhost" {
	//		return false
	//	}
	//} else {
	// The peer is not a candidate for sync if it's not a full
	// node. Additionally, if the segwit soft-fork package has
	// activated, then the peer must also be upgraded.
	segwitActive, err := sm.chain.IsDeploymentActive(chaincfg.DeploymentSegwit)
	if err != nil {
		log.Errorf("Unable to query for segwit "+
			"soft-fork state: %v", err)
	}
	nodeServices := peer.Services()
	if nodeServices&wire.SFNodeNetwork != wire.SFNodeNetwork ||
		(segwitActive && !peer.IsWitnessEnabled()) {
		return false
	}
	//}

	// Candidate if all checks passed.
	return true
}

// handleNewPeerMsg与新同行的交易表明，它们可能被视为同步同行(它们已经成功谈判)。
// 如果需要，它还会开始同步。它是从syncHandler goroutine调用的。
// handleNewPeerMsg deals with new peers that have signalled they may
// be considered as a sync peer (they have already successfully negotiated).  It
// also starts syncing if needed.  It is invoked from the syncHandler goroutine.
func (sm *SyncManager) handleNewPeerMsg(peer *peerpkg.Peer) {
	// Ignore if in the process of shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	log.Infof("New valid peer %s (%s)", peer, peer.UserAgent())

	// Initialize the peer state
	isSyncCandidate := sm.isSyncCandidate(peer)
	sm.peerStates[peer] = &peerSyncState{
		syncCandidate:   isSyncCandidate,
		requestedTxns:   make(map[chainhash.Hash]struct{}),
		requestedBlocks: make(map[chainhash.Hash]struct{}),
	}

	// Start syncing by choosing the best candidate if needed.
	if isSyncCandidate && sm.syncPeer == nil {
		sm.startSync()
	}
}

// handleDonePeerMsg与那些表示已经完成交易的同行进行交易。
// 它删除了对等点作为同步的候选对象，如果它是当前同步对等点，则尝试选择要同步的新最佳对等点。
// 它是从syncHandler goroutine调用的。
// handleDonePeerMsg deals with peers that have signalled they are done.  It
// removes the peer as a candidate for syncing and in the case where it was
// the current sync peer, attempts to select a new best peer to sync from.  It
// is invoked from the syncHandler goroutine.
func (sm *SyncManager) handleDonePeerMsg(peer *peerpkg.Peer) {
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received done peer message for unknown peer %s", peer)
		return
	}

	// Remove the peer from the list of candidate peers.
	delete(sm.peerStates, peer)

	log.Infof("Lost peer %s", peer)

	// Remove requested transactions from the global map so that they will
	// be fetched from elsewhere next time we get an inv.
	for txHash := range state.requestedTxns {
		delete(sm.requestedTxns, txHash)
	}

	// Remove requested blocks from the global map so that they will be
	// fetched from elsewhere next time we get an inv.
	// TODO: we could possibly here check which peers have these blocks
	// and request them now to speed things up a little.
	for blockHash := range state.requestedBlocks {
		delete(sm.requestedBlocks, blockHash)
	}

	// Attempt to find a new peer to sync from if the quitting peer is the
	// sync peer.  Also, reset the headers-first state if in headers-first
	// mode so
	if sm.syncPeer == peer {
		sm.syncPeer = nil
		if sm.headersFirstMode {
			best := sm.chain.BestSnapshot()
			sm.resetHeaderState(&best.Hash, best.Height)
		}
		sm.startSync()
	}
}

// handleTxMsg处理来自所有对等点的事务消息。
// handleTxMsg handles transaction messages from all peers.
func (sm *SyncManager) handleTxMsg(tmsg *txMsg) {
	peer := tmsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received tx message from unknown peer %s", peer)
		return
	}

	// NOTE:  BitcoinJ, and possibly other wallets, don't follow the spec of
	// sending an inventory message and allowing the remote peer to decide
	// whether or not they want to request the transaction via a getdata
	// message.  Unfortunately, the reference implementation permits
	// unrequested data, so it has allowed wallets that don't follow the
	// spec to proliferate.  While this is not ideal, there is no check here
	// to disconnect peers for sending unsolicited transactions to provide
	// interoperability.
	txHash := tmsg.tx.Hash()

	// Ignore transactions that we have already rejected.  Do not
	// send a reject message here because if the transaction was already
	// rejected, the transaction was unsolicited.
	if _, exists = sm.rejectedTxns[*txHash]; exists {
		log.Debugf("Ignoring unsolicited previously rejected "+
			"transaction %v from %s", txHash, peer)
		return
	}

	// Process the transaction to include validation, insertion in the
	// memory pool, orphan handling, etc.
	acceptedTxs, err := sm.txMemPool.ProcessTransaction(tmsg.tx,
		true, true, mempool.Tag(peer.ID()))

	// Remove transaction from request maps. Either the mempool/chain
	// already knows about it and as such we shouldn't have any more
	// instances of trying to fetch it, or we failed to insert and thus
	// we'll retry next time we get an inv.
	delete(state.requestedTxns, *txHash)
	delete(sm.requestedTxns, *txHash)

	if err != nil {

		// 在处理新块之前，不要再次请求此交易。
		// Do not request this transaction again until a new block
		// has been processed.
		sm.rejectedTxns[*txHash] = struct{}{}
		sm.limitMap(sm.rejectedTxns, maxRejectedTxns)

		// When the error is a rule error, it means the transaction was
		// simply rejected as opposed to something actually going wrong,
		// so log it as such.  Otherwise, something really did go wrong,
		// so log it as an actual error.
		if _, ok := err.(mempool.RuleError); ok {
			log.Debugf("Rejected transaction %v from %s: %v",
				txHash, peer, err)
		} else {
			log.Errorf("Failed to process transaction %v: %v",
				txHash, err)
		}

		// Convert the error into an appropriate reject message and
		// send it.
		code, reason := mempool.ErrToRejectErr(err)
		peer.PushRejectMsg(wire.CmdTx, code, reason, txHash, false)
		return
	}

	sm.peerNotifier.AnnounceNewTransactions(acceptedTxs)
}

// 如果我们相信我们与同行同步了，那么当前的返回值为true;如果我们仍然有块要检查，那么返回值为false
// current returns true if we believe we are synced with our peers, false if we
// still have blocks to check
func (sm *SyncManager) current() bool {
	if !sm.chain.IsCurrent() {
		return false
	}

	// if blockChain thinks we are current and we have no syncPeer it
	// is probably right.
	if sm.syncPeer == nil {
		return true
	}

	// No matter what chain thinks, if we are below the block we are syncing
	// to we are not current.
	if sm.chain.BestSnapshot().Height < sm.syncPeer.LastBlock() {
		return false
	}
	return true
}

// handleBlockMsg处理来自所有对等点的块消息。
// handleBlockMsg handles block messages from all peers.
func (sm *SyncManager) handleBlockMsg(bmsg *blockMsg) {
	peer := bmsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received block message from unknown peer %s", peer)
		return
	}

	// If we didn't ask for this block then the peer is misbehaving.
	blockHash := bmsg.block.Hash()
	//if _, exists = state.requestedBlocks[*blockHash]; !exists {
	// The regression test intentionally sends some blocks twice
	// to test duplicate block insertion fails.  Don't disconnect
	// the peer or ignore the block when we're in regression test
	// mode in this case so the chain code is actually fed the
	// duplicate blocks.
	//if sm.chainParams != &chaincfg.RegressionNetParams {
	//	log.Warnf("Got unrequested block %v from %s -- "+
	//		"disconnecting", blockHash, peer.Addr())
	//	peer.Disconnect()
	//	return
	//}
	//}

	//在heades -first模式下，如果块匹配正在获取的头列表中第一个头的哈希值，则可以减少验证，
	// 因为头已经被验证为链接在一起，并且直到下一个检查点为止都是有效的。
	// 此外，删除除检查点之外的所有块的列表条目，因为它需要正确地验证下一轮头文件链接。
	// When in headers-first mode, if the block matches the hash of the
	// first header in the list of headers that are being fetched, it's
	// eligible for less validation since the headers have already been
	// verified to link together and are valid up to the next checkpoint.
	// Also, remove the list entry for all blocks except the checkpoint
	// since it is needed to verify the next round of headers links
	// properly.
	//isCheckpointBlock := false
	behaviorFlags := blockchain.BFNone
	//if sm.headersFirstMode {
	//	firstNodeEl := sm.headerList.Front()
	//	if firstNodeEl != nil {
	//		firstNode := firstNodeEl.Value.(*headerNode)
	//		if blockHash.IsEqual(firstNode.hash) {
	//			behaviorFlags |= blockchain.BFFastAdd
	//			wire.ChangeCode()
	//			//if firstNode.hash.IsEqual(sm.nextCheckpoint.Hash) {
	//			//	isCheckpointBlock = true
	//			//} else {
	//			//	sm.headerList.Remove(firstNodeEl)
	//			//}
	//		}
	//	}
	//}

	// Remove block from request maps. Either chain will know about it and
	// so we shouldn't have any more instances of trying to fetch it, or we
	// will fail the insert and thus we'll retry next time we get an inv.
	delete(state.requestedBlocks, *blockHash)
	delete(sm.requestedBlocks, *blockHash)

	// 处理该块以包括验证、最佳链选择、孤儿处理等。
	// Process the block to include validation, best chain selection, orphan
	// handling, etc.

	// 处理发块环节收到的块，
	// 包括：验证签名，验证交易，验证weight，验证coinbase，
	// 通过验证将块放入块池
	_, isOrphan, err := sm.chain.ProcessBlock(bmsg.block, behaviorFlags)
	if err != nil {
		// When the error is a rule error, it means the block was simply
		// rejected as opposed to something actually going wrong, so log
		// it as such.  Otherwise, something really did go wrong, so log
		// it as an actual error.
		if _, ok := err.(blockchain.RuleError); ok {
			log.Infof("Rejected block %v from %s: %v", blockHash,
				peer, err)
		} else {
			log.Errorf("Failed to process block %v: %v",
				blockHash, err)
		}
		if dbErr, ok := err.(database.Error); ok && dbErr.ErrorCode ==
			database.ErrCorruption {
			panic(dbErr)
		}

		//将错误转换为适当的拒绝消息并发送。
		// Convert the error into an appropriate reject message and
		// send it.
		code, reason := mempool.ErrToRejectErr(err)
		peer.PushRejectMsg(wire.CmdBlock, code, reason, blockHash, false)
		return
	}

	// Meta-data about the new block this peer is reporting. We use this
	// below to update this peer's latest block height and the heights of
	// other peers based on their last announced block hash. This allows us
	// to dynamically update the block heights of peers, avoiding stale
	// heights when looking for a new sync peer. Upon acceptance of a block
	// or recognition of an orphan, we also use this information to update
	// the block heights over other peers who's invs may have been ignored
	// if we are actively syncing while the chain is not yet current or
	// who may have lost the lock announcement race.
	var heightUpdate int32
	var blkHashUpdate *chainhash.Hash

	// Request the parents for the orphan block from the peer that sent it.
	if isOrphan {
		// We've just received an orphan block from a peer. In order
		// to update the height of the peer, we try to extract the
		// block height from the scriptSig of the coinbase transaction.
		// Extraction is only attempted if the block's version is
		// high enough (ver 2+).
		header := &bmsg.block.MsgBlock().Header
		if blockchain.ShouldHaveSerializedBlockHeight(header) {
			coinbaseTx := bmsg.block.Transactions()[0]
			cbHeight, err := blockchain.ExtractCoinbaseHeight(coinbaseTx)
			if err != nil {
				log.Warnf("Unable to extract height from "+
					"coinbase tx: %v", err)
			} else {
				log.Debugf("Extracted height of %v from "+
					"orphan block", cbHeight)
				heightUpdate = cbHeight
				blkHashUpdate = blockHash
			}
		}

		orphanRoot := sm.chain.GetOrphanRoot(blockHash)
		locator, err := sm.chain.LatestBlockLocator()
		if err != nil {
			log.Warnf("Failed to get block locator for the "+
				"latest block: %v", err)
		} else {
			peer.PushGetBlocksMsg(locator, orphanRoot)
		}
	} else {
		// When the block is not an orphan, log information about it and
		// update the chain state.
		sm.progressLogger.LogBlockHeight(bmsg.block)

		// Update this peer's latest block height, for future
		// potential sync node candidacy.
		best := sm.chain.BestSnapshot()
		heightUpdate = best.Height
		blkHashUpdate = &best.Hash

		// Clear the rejected transactions.
		sm.rejectedTxns = make(map[chainhash.Hash]struct{})
	}

	// 更新此对等点的块高度。但是，只有当这是孤立的或我们的链是“当前的”时，才向服务器发送消息更新对等高度。如果我们从头开始同步链，这将避免发送垃圾数量的消息。
	// Update the block height for this peer. But only send a message to
	// the server for updating peer heights if this is an orphan or our
	// chain is "current". This avoids sending a spammy amount of messages
	// if we're syncing the chain from scratch.
	if blkHashUpdate != nil && heightUpdate != 0 {
		peer.UpdateLastBlockHeight(heightUpdate)
		if isOrphan || sm.current() {
			go sm.peerNotifier.UpdatePeerHeights(blkHashUpdate, heightUpdate,
				peer)
		}
	}

	//如果我们不采取“头先上”的模式，就没什么可做的了。
	// Nothing more to do if we aren't in headers-first mode.
	if !sm.headersFirstMode {
		return
	}

	// This is headers-first mode, so if the block is not a checkpoint
	// request more blocks using the header list when the request queue is
	// getting short.
	//if !isCheckpointBlock {
	//	if sm.startHeader != nil &&
	//		len(state.requestedBlocks) < minInFlightBlocks {
	//		sm.fetchHeaderBlocks()
	//	}
	//	return
	//}

	// This is headers-first mode and the block is a checkpoint.  When
	// there is a next checkpoint, get the next round of headers by asking
	// for headers starting from the block after this one up to the next
	// checkpoint.
	//prevHeight := sm.nextCheckpoint.Height
	//prevHash := sm.nextCheckpoint.Hash
	//sm.nextCheckpoint = sm.findNextHeaderCheckpoint(prevHeight)
	//if sm.nextCheckpoint != nil {
	//	locator := blockchain.BlockLocator([]*chainhash.Hash{prevHash})
	//	err := peer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
	//	if err != nil {
	//		log.Warnf("Failed to send getheaders message to "+
	//			"peer %s: %v", peer.Addr(), err)
	//		return
	//	}
	//	log.Infof("Downloading headers for blocks %d to %d from "+
	//		"peer %s", prevHeight+1, sm.nextCheckpoint.Height,
	//		sm.syncPeer.Addr())
	//	return
	//}

	//这是head -first模式，这个块是一个检查点，并且没有更多的检查点，所以切换到正常模式，从这个块之后的块请求块，直到链的末尾(零散列)。
	// This is headers-first mode, the block is a checkpoint, and there are
	// no more checkpoints, so switch to normal mode by requesting blocks
	// from the block after this one up to the end of the chain (zero hash).
	sm.headersFirstMode = false
	sm.headerList.Init()
	log.Infof("Reached the final checkpoint -- switching to normal mode")
	locator := blockchain.BlockLocator([]*chainhash.Hash{blockHash})
	err = peer.PushGetBlocksMsg(locator, &zeroHash)
	if err != nil {
		log.Warnf("Failed to send getblocks message to peer %s: %v",
			peer.Addr(), err)
		return
	}
}

// 处理发块阶段收到的块
func (sm *SyncManager) handleCadidateMsg(bmsg *candidateMsg) {
	peer := bmsg.peer
	_, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received block message from unknown peer %s", peer)
		return
	}

	// If we didn't ask for this block then the peer is misbehaving.
	blockHash := bmsg.block.CandidateHash()

	//在heades -first模式下，如果块匹配正在获取的头列表中第一个头的哈希值，则可以减少验证，
	// 因为头已经被验证为链接在一起，并且直到下一个检查点为止都是有效的。
	// 此外，删除除检查点之外的所有块的列表条目，因为它需要正确地验证下一轮头文件链接。
	// When in headers-first mode, if the block matches the hash of the
	// first header in the list of headers that are being fetched, it's
	// eligible for less validation since the headers have already been
	// verified to link together and are valid up to the next checkpoint.
	// Also, remove the list entry for all blocks except the checkpoint
	// since it is needed to verify the next round of headers links
	// properly.
	behaviorFlags := blockchain.BFNone

	// 处理该块
	// 包括：验证签名，验证交易，验证weight，验证coinbase，
	// 通过验证将该块放入块池和指向池
	// Process the block to include validation, best chain selection, orphan
	// handling, etc.
	b, err := sm.chain.ProcessCandidate(bmsg.block, behaviorFlags)
	if err != nil || !b {
		// When the error is a rule error, it means the block was simply
		// rejected as opposed to something actually going wrong, so log
		// it as such.  Otherwise, something really did go wrong, so log
		// it as an actual error.
		if _, ok := err.(blockchain.RuleError); ok {
			log.Infof("Rejected block %v from %s: %v", blockHash,
				peer, err)
		} else {
			log.Errorf("Failed to process block %v: %v",
				blockHash, err)
		}
		if dbErr, ok := err.(database.Error); ok && dbErr.ErrorCode ==
			database.ErrCorruption {
			panic(dbErr)
		}

		return
	}

	//向发送孤儿块的对等方请求父方。当块不是孤立块时，记录有关它的信息并更新链状态。
	// Request the parents for the orphan block from the peer that sent it.
	// When the block is not an orphan, log information about it and
	// update the chain state.
	//sm.progressLogger.LogBlockHeight(bmsg.block)

	// 清除被拒绝的事务。
	// Clear the rejected transactions.
	sm.rejectedTxns = make(map[chainhash.Hash]struct{})

	//如果我们不采取“头先上”的模式，就没什么可做的了。
	// Nothing more to do if we aren't in headers-first mode.
	if !sm.headersFirstMode {
		return
	}

	// 转发块信息
	bo, err := sm.SendBlock(bmsg.block)
	if err != nil || !bo {
		log.Errorf("Failed to send block message: %v", err)
	}
}

// fetchHeaderBlocks创建一个请求，并向syncPeer发送一个请求，根据当前头列表下载下一个块列表。
// fetchHeaderBlocks creates and sends a request to the syncPeer for the next
// list of blocks to be downloaded based on the current list of headers.
func (sm *SyncManager) fetchHeaderBlocks() {
	// Nothing to do if there is no start header.
	if sm.startHeader == nil {
		log.Warnf("fetchHeaderBlocks called with no start header")
		return
	}

	// Build up a getdata request for the list of blocks the headers
	// describe.  The size hint will be limited to wire.MaxInvPerMsg by
	// the function, so no need to double check it here.
	gdmsg := wire.NewMsgGetDataSizeHint(uint(sm.headerList.Len()))
	numRequested := 0
	for e := sm.startHeader; e != nil; e = e.Next() {
		node, ok := e.Value.(*headerNode)
		if !ok {
			log.Warn("Header list node type is not a headerNode")
			continue
		}

		iv := wire.NewInvVect(wire.InvTypeBlock, node.hash)
		haveInv, err := sm.haveInventory(iv)
		if err != nil {
			log.Warnf("Unexpected failure when checking for "+
				"existing inventory during header block "+
				"fetch: %v", err)
		}
		if !haveInv {
			syncPeerState := sm.peerStates[sm.syncPeer]

			sm.requestedBlocks[*node.hash] = struct{}{}
			syncPeerState.requestedBlocks[*node.hash] = struct{}{}

			// If we're fetching from a witness enabled peer
			// post-fork, then ensure that we receive all the
			// witness data in the blocks.
			if sm.syncPeer.IsWitnessEnabled() {
				iv.Type = wire.InvTypeWitnessBlock
			}

			gdmsg.AddInvVect(iv)
			numRequested++
		}
		sm.startHeader = e.Next()
		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}
	if len(gdmsg.InvList) > 0 {
		sm.syncPeer.QueueMessage(gdmsg, nil)
	}
}

// handleHeadersMsg处理来自所有对等点的块头消息。头文件在执行头文件优先同步时被请求。
// handleHeadersMsg handles block header messages from all peers.  Headers are
// requested when performing a headers-first sync.
func (sm *SyncManager) handleHeadersMsg(hmsg *headersMsg) {
	peer := hmsg.peer
	_, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received headers message from unknown peer %s", peer)
		return
	}

	// The remote peer is misbehaving if we didn't request headers.
	msg := hmsg.headers
	numHeaders := len(msg.Headers)
	if !sm.headersFirstMode {
		log.Warnf("Got %d unrequested headers from %s -- "+
			"disconnecting", numHeaders, peer.Addr())
		peer.Disconnect()
		return
	}

	// Nothing to do for an empty headers message.
	if numHeaders == 0 {
		return
	}

	// Process all of the received headers ensuring each one connects to the
	// previous and that checkpoints match.
	receivedCheckpoint := false
	//var finalHash *chainhash.Hash
	for _, blockHeader := range msg.Headers {
		blockHash := blockHeader.BlockHash()
		//finalHash = &blockHash

		// Ensure there is a previous header to compare against.
		prevNodeEl := sm.headerList.Back()
		if prevNodeEl == nil {
			log.Warnf("Header list does not contain a previous" +
				"element as expected -- disconnecting peer")
			peer.Disconnect()
			return
		}

		// Ensure the header properly connects to the previous one and
		// add it to the list of headers.
		node := headerNode{hash: &blockHash}
		prevNode := prevNodeEl.Value.(*headerNode)
		if prevNode.hash.IsEqual(&blockHeader.PrevBlock) {
			node.height = prevNode.height + 1
			e := sm.headerList.PushBack(&node)
			if sm.startHeader == nil {
				sm.startHeader = e
			}
		} else {
			log.Warnf("Received block header that does not "+
				"properly connect to the chain from peer %s "+
				"-- disconnecting", peer.Addr())
			peer.Disconnect()
			return
		}

		// Verify the header at the next checkpoint height matches.
		//if node.height == sm.nextCheckpoint.Height {
		//	if node.hash.IsEqual(sm.nextCheckpoint.Hash) {
		//		receivedCheckpoint = true
		//		log.Infof("Verified downloaded block "+
		//			"header against checkpoint at height "+
		//			"%d/hash %s", node.height, node.hash)
		//	} else {
		//		log.Warnf("Block header at height %d/hash "+
		//			"%s from peer %s does NOT match "+
		//			"expected checkpoint hash of %s -- "+
		//			"disconnecting", node.height,
		//			node.hash, peer.Addr(),
		//			sm.nextCheckpoint.Hash)
		//		peer.Disconnect()
		//		return
		//	}
		//	break
		//}
	}

	// When this header is a checkpoint, switch to fetching the blocks for
	// all of the headers since the last checkpoint.
	if receivedCheckpoint {
		// Since the first entry of the list is always the final block
		// that is already in the database and is only used to ensure
		// the next header links properly, it must be removed before
		// fetching the blocks.
		sm.headerList.Remove(sm.headerList.Front())
		log.Infof("Received %v block headers: Fetching blocks",
			sm.headerList.Len())
		sm.progressLogger.SetLastLogTime(time.Now())
		sm.fetchHeaderBlocks()
		return
	}

	// This header is not a checkpoint, so request the next batch of
	// headers starting from the latest known header and ending with the
	// next checkpoint.
	//locator := blockchain.BlockLocator([]*chainhash.Hash{finalHash})
	//err := peer.PushGetHeadersMsg(locator, sm.nextCheckpoint.Hash)
	//if err != nil {
	//	log.Warnf("Failed to send getheaders message to "+
	//		"peer %s: %v", peer.Addr(), err)
	//	return
	//}
}

// haveInventory返回所传递的库存向量表示的库存是否已知。
// haveInventory returns whether or not the inventory represented by the passed
// inventory vector is known.  This includes checking all of the various places
// inventory can be when it is in different states such as blocks that are part
// of the main chain, on a side chain, in the orphan pool, and transactions that
// are in the memory pool (either the main pool or orphan pool).
func (sm *SyncManager) haveInventory(invVect *wire.InvVect) (bool, error) {
	switch invVect.Type {
	case wire.InvTypeWitnessBlock:
		fallthrough
	case wire.InvTypeBlock:
		// Ask chain if the block is known to it in any form (main
		// chain, side chain, or orphan).
		return sm.chain.HaveBlock(&invVect.Hash)

	case wire.InvTypeWitnessTx:
		fallthrough
	case wire.InvTypeTx:
		// Ask the transaction memory pool if the transaction is known
		// to it in any form (main pool or orphan).
		if sm.txMemPool.HaveTransaction(&invVect.Hash) {
			return true, nil
		}

		// Check if the transaction exists from the point of view of the
		// end of the main chain.  Note that this is only a best effort
		// since it is expensive to check existence of every output and
		// the only purpose of this check is to avoid downloading
		// already known transactions.  Only the first two outputs are
		// checked because the vast majority of transactions consist of
		// two outputs where one is some form of "pay-to-somebody-else"
		// and the other is a change output.
		prevOut := wire.OutPoint{Hash: invVect.Hash}
		for i := uint32(0); i < 2; i++ {
			prevOut.Index = i
			entry, err := sm.chain.FetchUtxoEntry(prevOut)
			if err != nil {
				return false, err
			}
			if entry != nil && !entry.IsSpent() {
				return true, nil
			}
		}

		return false, nil
	}

	// The requested inventory is is an unsupported type, so just claim
	// it is known to avoid requesting it.
	return true, nil
}

// handleinmsg处理来自所有对等点的inv消息。我们检查远程对等方所宣传的库存，并采取相应的行动。
// handleInvMsg handles inv messages from all peers.
// We examine the inventory advertised by the remote peer and act accordingly.
func (sm *SyncManager) handleInvMsg(imsg *invMsg) {
	peer := imsg.peer
	state, exists := sm.peerStates[peer]
	if !exists {
		log.Warnf("Received inv message from unknown peer %s", peer)
		return
	}

	//尝试在清单中找到最后一个块。也许没有。
	// Attempt to find the final block in the inventory list.  There may
	// not be one.
	lastBlock := -1
	invVects := imsg.inv.InvList
	for i := len(invVects) - 1; i >= 0; i-- {
		if invVects[i].Type == wire.InvTypeBlock {
			lastBlock = i
			break
		}
	}

	//如果这个inv包含一个块声明，而这个声明不是来自我们当前的同步对等点，
	// 或者我们是当前的，那么为这个对等点更新最后一个声明的块。
	// 稍后，我们将使用这些信息根据我们已经接受的对等块更新对等块的高度。
	// If this inv contains a block announcement, and this isn't coming from
	// our current sync peer or we're current, then update the last
	// announced block for this peer. We'll use this information later to
	// update the heights of peers based on blocks we've accepted that they
	// previously announced.
	if lastBlock != -1 && (peer != sm.syncPeer || sm.current()) {
		peer.UpdateLastAnnouncedBlock(&invVects[lastBlock].Hash)
	}

	// Ignore invs from peers that aren't the sync if we are not current.
	// Helps prevent fetching a mass of orphans.
	if peer != sm.syncPeer && !sm.current() {
		return
	}

	//如果我们的链是当前的，并且一个对等点声明了一个我们已经知道的块，那么更新它们当前块的高度。
	// If our chain is current and a peer announces a block we already
	// know of, then update their current block height.
	if lastBlock != -1 && sm.current() {
		blkHeight, err := sm.chain.BlockHeightByHash(&invVects[lastBlock].Hash)
		if err == nil {
			peer.UpdateLastBlockHeight(blkHeight)
		}
	}

	// 如果我们还没有广告上的存货，请索取。
	// 另外，如果我们已经有了一个孤儿，请向父母索要。
	// 最后，尝试检测可能的摊位由于长侧链我们已经有和要求更多的方块来防止他们。
	// Request the advertised inventory if we don't already have it.  Also,
	// request parent blocks of orphans if we receive one we already have.
	// Finally, attempt to detect potential stalls due to long side chains
	// we already have and request more blocks to prevent them.
	for i, iv := range invVects {
		// Ignore unsupported inventory types.
		switch iv.Type {
		case wire.InvTypeBlock:
		case wire.InvTypeTx:
		case wire.InvTypeWitnessBlock:
		case wire.InvTypeWitnessTx:
		default:
			continue
		}

		//将库存添加到对等节点的已知库存缓存中。
		// Add the inventory to the cache of known inventory
		// for the peer.
		peer.AddKnownInventory(iv)

		//当我们处于“优先考虑”模式时，忽略库存。
		// Ignore inventory when we're in headers-first mode.
		if sm.headersFirstMode {
			continue
		}

		// Request the inventory if we don't already have it.
		haveInv, err := sm.haveInventory(iv)
		if err != nil {
			log.Warnf("Unexpected failure when checking for "+
				"existing inventory during inv message "+
				"processing: %v", err)
			continue
		}
		if !haveInv {
			if iv.Type == wire.InvTypeTx {
				// Skip the transaction if it has already been
				// rejected.
				if _, exists := sm.rejectedTxns[iv.Hash]; exists {
					continue
				}
			}

			// Ignore invs block invs from non-witness enabled
			// peers, as after segwit activation we only want to
			// download from peers that can provide us full witness
			// data for blocks.
			if !peer.IsWitnessEnabled() && iv.Type == wire.InvTypeBlock {
				continue
			}

			// Add it to the request queue.
			state.requestQueue = append(state.requestQueue, iv)
			continue
		}

		if iv.Type == wire.InvTypeBlock {
			//这个block是我们已经有的孤立块。当处理现有的孤儿时，它请求丢失的父块。
			// 当这种情况发生时，这意味着丢失的块比单个库存消息中允许的块要多。
			// 因此，一旦该对等点请求了最终的广告块，远程对等点就会注意到，并将孤立块重新发送为可用块，以表明有更多缺失的块需要请求。
			// The block is an orphan block that we already have.
			// When the existing orphan was processed, it requested
			// the missing parent blocks.  When this scenario
			// happens, it means there were more blocks missing
			// than are allowed into a single inventory message.  As
			// a result, once this peer requested the final
			// advertised block, the remote peer noticed and is now
			// resending the orphan block as an available block
			// to signal there are more missing blocks that need to
			// be requested.
			if sm.chain.IsKnownOrphan(&iv.Hash) {
				// Request blocks starting at the latest known
				// up to the root of the orphan that just came
				// in.
				orphanRoot := sm.chain.GetOrphanRoot(&iv.Hash)
				locator, err := sm.chain.LatestBlockLocator()
				if err != nil {
					log.Errorf("PEER: Failed to get block "+
						"locator for the latest block: "+
						"%v", err)
					continue
				}
				peer.PushGetBlocksMsg(locator, orphanRoot)
				continue
			}

			//我们已经在这条inv信息中公布了最后一个区块，因此强制要求更多。只有在侧链很长时才会发生这种情况。
			// We already have the final block advertised by this
			// inventory message, so force a request for more.  This
			// should only happen if we're on a really long side
			// chain.
			if i == lastBlock {
				//请求块，直到远程对等方知道的最后一个请求块为止(零停止散列)。
				// Request blocks after this one up to the
				// final one the remote peer knows about (zero
				// stop hash).
				locator := sm.chain.BlockLocatorFromHash(&iv.Hash)
				peer.PushGetBlocksMsg(locator, &zeroHash)
			}
		}
	}

	//马上提出尽可能多的要求。任何不适合请求的内容都将在下一个inv消息中被请求。
	// Request as much as possible at once.  Anything that won't fit into
	// the request will be requested on the next inv message.
	numRequested := 0
	gdmsg := wire.NewMsgGetData()
	requestQueue := state.requestQueue
	for len(requestQueue) != 0 {
		iv := requestQueue[0]
		requestQueue[0] = nil
		requestQueue = requestQueue[1:]

		switch iv.Type {
		case wire.InvTypeWitnessBlock:
			fallthrough
		case wire.InvTypeBlock:
			//如果还没有一个挂起的请求，请求该块。
			// Request the block if there is not already a pending
			// request.
			if _, exists := sm.requestedBlocks[iv.Hash]; !exists { // 挂起请求
				sm.requestedBlocks[iv.Hash] = struct{}{}
				sm.limitMap(sm.requestedBlocks, maxRequestedBlocks)
				state.requestedBlocks[iv.Hash] = struct{}{}

				if peer.IsWitnessEnabled() {
					iv.Type = wire.InvTypeWitnessBlock
				}

				gdmsg.AddInvVect(iv)
				numRequested++
			}

		case wire.InvTypeWitnessTx:
			fallthrough
		case wire.InvTypeTx:
			// Request the transaction if there is not already a
			// pending request.
			if _, exists := sm.requestedTxns[iv.Hash]; !exists {
				sm.requestedTxns[iv.Hash] = struct{}{}
				sm.limitMap(sm.requestedTxns, maxRequestedTxns)
				state.requestedTxns[iv.Hash] = struct{}{}

				// If the peer is capable, request the txn
				// including all witness data.
				if peer.IsWitnessEnabled() {
					iv.Type = wire.InvTypeWitnessTx
				}

				gdmsg.AddInvVect(iv)
				numRequested++
			}
		}

		if numRequested >= wire.MaxInvPerMsg {
			break
		}
	}
	state.requestQueue = requestQueue
	if len(gdmsg.InvList) > 0 {
		peer.QueueMessage(gdmsg, nil)
	}
}

// limitMap是一个帮助函数，用于映射，如果添加一个新值会导致它超出允许的最大值，则通过删除随机事务来要求最大限制。
// limitMap is a helper function for maps that require a maximum limit by
// evicting a random transaction if adding a new value would cause it to
// overflow the maximum allowed.
func (sm *SyncManager) limitMap(m map[chainhash.Hash]struct{}, limit int) {
	if len(m)+1 > limit {
		// Remove a random entry from the map.  For most compilers, Go's
		// range statement iterates starting at a random item although
		// that is not 100% guaranteed by the spec.  The iteration order
		// is not important here because an adversary would have to be
		// able to pull off preimage attacks on the hashing function in
		// order to target eviction of specific entries anyways.
		for txHash := range m {
			delete(m, txHash)
			return
		}
	}
}

// 块处理程序是同步管理器的主要处理程序。它必须像goroutine一样运行。
// 它在与对等处理程序分开的goroutine中处理块和inv消息，因此块(MsgBlock)消息由一个线程处理，而不需要锁定内存数据结构。
// 这一点很重要，因为同步管理器控制需要哪些块以及抓取应该如何进行。
// blockHandler is the main handler for the sync manager.  It must be run as a
// goroutine.  It processes block and inv messages in a separate goroutine
// from the peer handlers so the block (MsgBlock) messages are handled by a
// single thread without needing to lock memory data structures.  This is
// important because the sync manager controls which blocks are needed and how
// the fetching should proceed.
func (sm *SyncManager) blockHandler() {
out:
	for {
		select {
		case m := <-sm.msgChan:
			switch msg := m.(type) {
			case *newPeerMsg:
				sm.handleNewPeerMsg(msg.peer)

			case *txMsg:
				sm.handleTxMsg(msg)
				msg.reply <- struct{}{}

			case *blockMsg:
				sm.handleBlockMsg(msg)
				msg.reply <- struct{}{}

			case *candidateMsg:
				sm.handleCadidateMsg(msg)
				msg.reply <- struct{}{}

			case *invMsg:
				sm.handleInvMsg(msg)

			case *headersMsg:
				sm.handleHeadersMsg(msg)

			case *donePeerMsg:
				sm.handleDonePeerMsg(msg.peer)

			case getSyncPeerMsg:
				var peerID int32
				if sm.syncPeer != nil {
					peerID = sm.syncPeer.ID()
				}
				msg.reply <- peerID

			case processBlockMsg:
				_, isOrphan, err := sm.chain.ProcessBlock(msg.block, msg.flags)
				if err != nil {
					msg.reply <- processBlockResponse{
						isOrphan: false,
						err:      err,
					}
				}

				msg.reply <- processBlockResponse{
					isOrphan: isOrphan,
					err:      nil,
				}
			case sendBlockMsg:
				sm.peerNotifier.SendBlock(msg.block.MsgCandidate())
			case sendSignMsg:
				sm.peerNotifier.SendSign(msg.msgSign)

			case isCurrentMsg:
				msg.reply <- sm.current()

			case pauseMsg:
				// Wait until the sender unpauses the manager.
				<-msg.unpause

			default:
				log.Warnf("Invalid message type in block "+
					"handler: %T", msg)
			}

		case <-sm.quit:
			break out
		}
	}

	sm.wg.Done()
	log.Trace("Block handler done")
}

// 处理投票结果，是个独立线程
// Processing the poll result is a separate thread
func (sm *SyncManager) VoteHandle() {
	creationTime := cpuminer.GetCreationTime()
	blockHeight := cpuminer.GetBlockHeight()

	// 根据最新块，计算10秒发块定时器启动的时间
	// According to the latest block, calculate the start time of the 10-second block timer
	laterTime := creationTime.Add(blockHeight*cpuminer.BlockTimeInterval + 20*time.Second)
	nowTime := time.Now()
	t := time.NewTimer(laterTime.Sub(nowTime))
	<-t.C
	t.Stop()

	// 处理当前轮的写块和投票
	// Handles write blocks and polls for the current round
	cpuminer.VoteProcess()

	// 10秒处理一波投票结果
	// Process one wave of voting results 10 second
	handlingTime := time.NewTimer(10 * time.Second)
	for {
		select {
		case <-handlingTime.C:
			cpuminer.VoteProcess()
		}
	}
}

// handleblockchain通知处理来自区块链的通知。它做的事情包括请求孤立块父块和将接受的块转发给连接的对等点。
// handleBlockchainNotification handles notifications from blockchain.  It does
// things such as request orphan block parents and relay accepted blocks to
// connected peers.
func (sm *SyncManager) handleBlockchainNotification(notification *blockchain.Notification) {
	switch notification.Type {
	// A block has been accepted into the block chain.  Relay it to other
	// peers.
	case blockchain.NTBlockAccepted:
		// Don't relay if we are not current. Other peers that are
		// current should already know about it.
		if !sm.current() {
			return
		}

		block, ok := notification.Data.(*drcutil.Block)
		if !ok {
			log.Warnf("Chain accepted notification is not a block.")
			break
		}

		// Generate the inventory vector and relay it.
		iv := wire.NewInvVect(wire.InvTypeBlock, block.Hash())
		sm.peerNotifier.RelayInventory(iv, block.MsgBlock().Header) // blockHash和header

	// A block has been connected to the main block chain.
	case blockchain.NTBlockConnected:
		block, ok := notification.Data.(*drcutil.Block)
		if !ok {
			log.Warnf("Chain connected notification is not a block.")
			break
		}

		// Remove all of the transactions (except the coinbase) in the
		// connected block from the transaction pool.  Secondly, remove any
		// transactions which are now double spends as a result of these
		// new transactions.  Finally, remove any transaction that is
		// no longer an orphan. Transactions which depend on a confirmed
		// transaction are NOT removed recursively because they are still
		// valid.
		for _, tx := range block.Transactions()[1:] {
			sm.txMemPool.RemoveTransaction(tx, false)
			sm.txMemPool.RemoveDoubleSpends(tx)
			sm.txMemPool.RemoveOrphan(tx)
			sm.peerNotifier.TransactionConfirmed(tx)
			acceptedTxs := sm.txMemPool.ProcessOrphans(tx)
			sm.peerNotifier.AnnounceNewTransactions(acceptedTxs)
		}

		// Register block with the fee estimator, if it exists.
		if sm.feeEstimator != nil {
			err := sm.feeEstimator.RegisterBlock(block)

			// If an error is somehow generated then the fee estimator
			// has entered an invalid state. Since it doesn't know how
			// to recover, create a new one.
			if err != nil {
				sm.feeEstimator = mempool.NewFeeEstimator(
					mempool.DefaultEstimateFeeMaxRollback,
					mempool.DefaultEstimateFeeMinRegisteredBlocks)
			}
		}

	// A block has been disconnected from the main block chain.
	case blockchain.NTBlockDisconnected:
		block, ok := notification.Data.(*drcutil.Block)
		if !ok {
			log.Warnf("Chain disconnected notification is not a block.")
			break
		}

		// Reinsert all of the transactions (except the coinbase) into
		// the transaction pool.
		for _, tx := range block.Transactions()[1:] {
			_, _, err := sm.txMemPool.MaybeAcceptTransaction(tx,
				false, false)
			if err != nil {
				// Remove the transaction and all transactions
				// that depend on it if it wasn't accepted into
				// the transaction pool.
				sm.txMemPool.RemoveTransaction(tx, true)
			}
		}

		// Rollback previous block recorded by the fee estimator.
		if sm.feeEstimator != nil {
			sm.feeEstimator.Rollback(block.Hash())
		}
	}
}

// NewPeer通知同步管理器一个新活动的对等点。
// NewPeer informs the sync manager of a newly active peer.
func (sm *SyncManager) NewPeer(peer *peerpkg.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}
	sm.msgChan <- &newPeerMsg{peer: peer}
}

// QueueTx将传递的事务消息和对等点添加到块处理队列中。在处理tx消息之后响应done通道参数。
// QueueTx adds the passed transaction message and peer to the block handling
// queue. Responds to the done channel argument after the tx message is
// processed.
func (sm *SyncManager) QueueTx(tx *drcutil.Tx, peer *peerpkg.Peer, done chan struct{}) {

	// 如果我们要关闭，不要接受更多的事务。
	// Don't accept more transactions if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.msgChan <- &txMsg{tx: tx, peer: peer, reply: done}
}

// QueueBlock将传递的块消息和对等点添加到块处理队列中。在块消息被处理后，响应done通道参数。
// QueueBlock adds the passed block message and peer to the block handling
// queue. Responds to the done channel argument after the block message is
// processed.
func (sm *SyncManager) QueueBlock(block *drcutil.Block, peer *peerpkg.Peer, done chan struct{}) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.msgChan <- &blockMsg{block: block, peer: peer, reply: done}
}

func (sm *SyncManager) QueueCandidate(block *drcutil.Block, peer *peerpkg.Peer, done chan struct{}) {
	// Don't accept more blocks if we're shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		done <- struct{}{}
		return
	}

	sm.msgChan <- &candidateMsg{block: block, peer: peer, reply: done}
}

// QueueInv将传递的inv消息和对等点添加到块处理队列中。
// QueueInv adds the passed inv message and peer to the block handling queue.
func (sm *SyncManager) QueueInv(inv *wire.MsgInv, peer *peerpkg.Peer) {

	// 这里没有通道处理，因为对等点不需要阻塞inv消息。
	// No channel handling here because peers do not need to block on inv
	// messages.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &invMsg{inv: inv, peer: peer}
}

// QueueHeaders将传递的headers消息和对等点添加到块处理队列中。
// QueueHeaders adds the passed headers message and peer to the block handling
// queue.
func (sm *SyncManager) QueueHeaders(headers *wire.MsgHeaders, peer *peerpkg.Peer) {
	// No channel handling here because peers do not need to block on
	// headers messages.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &headersMsg{headers: headers, peer: peer}
}

// DonePeer通知块管理器某个对等点已断开连接。
// DonePeer informs the blockmanager that a peer has disconnected.
func (sm *SyncManager) DonePeer(peer *peerpkg.Peer) {
	// Ignore if we are shutting down.
	if atomic.LoadInt32(&sm.shutdown) != 0 {
		return
	}

	sm.msgChan <- &donePeerMsg{peer: peer}
}

// Start启动处理块和inv消息的核心块处理程序。
// Start begins the core block handler which processes block and inv messages.
func (sm *SyncManager) Start() {
	// Already started?
	if atomic.AddInt32(&sm.started, 1) != 1 {
		return
	}

	log.Trace("Starting sync manager")
	sm.wg.Add(1)
	go sm.blockHandler()
	go sm.VoteHandle()
}

// Stop通过停止所有异步处理程序并等待它们完成，优雅地关闭同步管理器。
// Stop gracefully shuts down the sync manager by stopping all asynchronous
// handlers and waiting for them to finish.
func (sm *SyncManager) Stop() error {
	if atomic.AddInt32(&sm.shutdown, 1) != 1 {
		log.Warnf("Sync manager is already in the process of " +
			"shutting down")
		return nil
	}

	log.Infof("Sync manager shutting down")
	close(sm.quit)
	sm.wg.Wait()
	return nil
}

// 返回当前同步对等点的ID，如果没有，则返回0。
// SyncPeerID returns the ID of the current sync peer, or 0 if there is none.
func (sm *SyncManager) SyncPeerID() int32 {
	reply := make(chan int32)
	sm.msgChan <- getSyncPeerMsg{reply: reply}
	return <-reply
}

// ProcessBlock在块链的内部实例上使用ProcessBlock。
// ProcessBlock makes use of ProcessBlock on an internal instance of a block
// chain.
func (sm *SyncManager) ProcessBlock(block *drcutil.Block, flags blockchain.BehaviorFlags) (bool, error) {
	reply := make(chan processBlockResponse, 1)
	sm.msgChan <- processBlockMsg{block: block, flags: flags, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

func (sm *SyncManager) SendBlock(block *drcutil.Block) (bool, error) {
	reply := make(chan sendBlockResponse, 1)
	sm.msgChan <- sendBlockMsg{block: block, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

// 广播投票签名
func (sm *SyncManager) SendSign(msg *wire.MsgSign) (bool, error) {
	reply := make(chan sendSignResponse, 1)
	sm.msgChan <- sendSignMsg{msgSign: msg, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

// IsCurrent返回同步管理器是否认为它已与连接的对等点同步。
// IsCurrent returns whether or not the sync manager believes it is synced with
// the connected peers.
func (sm *SyncManager) IsCurrent() bool {
	reply := make(chan bool)
	sm.msgChan <- isCurrentMsg{reply: reply}
	return <-reply
}

// Pause暂停同步管理器，直到返回的通道关闭。
// Pause pauses the sync manager until the returned channel is closed.

// 注意，当暂停时，所有对等和块处理都将停止。消息发送者应该避免长时间地暂停同步管理器。
// Note that while paused, all peer and block processing is halted.  The
// message sender should avoid pausing the sync manager for long durations.
func (sm *SyncManager) Pause() chan<- struct{} {
	c := make(chan struct{})
	sm.msgChan <- pauseMsg{c}
	return c
}

// New构造一个新的SyncManager。使用Start开始处理异步块、tx和inv更新。
// New constructs a new SyncManager. Use Start to begin processing asynchronous
// block, tx, and inv updates.
func New(config *Config) (*SyncManager, error) {
	sm := SyncManager{
		peerNotifier:    config.PeerNotifier,
		chain:           config.Chain,
		txMemPool:       config.TxMemPool,
		chainParams:     config.ChainParams,
		rejectedTxns:    make(map[chainhash.Hash]struct{}),
		requestedTxns:   make(map[chainhash.Hash]struct{}),
		requestedBlocks: make(map[chainhash.Hash]struct{}),
		peerStates:      make(map[*peerpkg.Peer]*peerSyncState),
		progressLogger:  newBlockProgressLogger("Processed", log),
		msgChan:         make(chan interface{}, config.MaxPeers*3),
		headerList:      list.New(),
		quit:            make(chan struct{}),
		feeEstimator:    config.FeeEstimator,
	}

	//best := sm.chain.BestSnapshot()
	//if !config.DisableCheckpoints {
	//	// Initialize the next checkpoint based on the current height.
	//	sm.nextCheckpoint = sm.findNextHeaderCheckpoint(best.Height)
	//	if sm.nextCheckpoint != nil {
	//		sm.resetHeaderState(&best.Hash, best.Height)
	//	}
	//} else {
	//	log.Info("Checkpoints are disabled")
	//}

	sm.chain.Subscribe(sm.handleBlockchainNotification)

	return &sm, nil
}
