// Copyright (c) 2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package indexers

import (
	"errors"

	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/gcs"
	"github.com/btcsuite/btcutil/gcs/builder"
	"github.com/drcsuite/drc/blockchain"
	"github.com/drcsuite/drc/chaincfg"
	"github.com/drcsuite/drc/chaincfg/chainhash"
	"github.com/drcsuite/drc/database"
	"github.com/drcsuite/drc/wire"
)

const (
	// cfIndexName is the human-readable name for the index.
	cfIndexName = "committed filter index"
)

// Committed filters come in one flavor currently: basic. They are generated
// and dropped in pairs, and both are indexed by a block's hash.  Besides
// holding different content, they also live in different buckets.
var (
	// cfIndexParentBucketKey is the name of the parent bucket used to
	// house the index. The rest of the buckets live below this bucket.
	cfIndexParentBucketKey = []byte("cfindexparentbucket")

	// cfIndexKeys is an array of db bucket names used to house indexes of
	// block hashes to cfilters.
	cfIndexKeys = [][]byte{
		[]byte("cf0byhashidx"),
	}

	// cfHeaderKeys is an array of db bucket names used to house indexes of
	// block hashes to cf headers.
	cfHeaderKeys = [][]byte{
		[]byte("cf0headerbyhashidx"),
	}

	// cfHashKeys is an array of db bucket names used to house indexes of
	// block hashes to cf hashes.
	cfHashKeys = [][]byte{
		[]byte("cf0hashbyhashidx"),
	}

	maxFilterType = uint8(len(cfHeaderKeys) - 1)

	// zeroHash is the chainhash.Hash value of all zero bytes, defined here
	// for convenience.
	zeroHash chainhash.Hash
)

// dbFetchFilterIdxEntry从筛选器索引数据库检索一个数据blob。条目的缺失不被认为是错误。
// dbFetchFilterIdxEntry retrieves a data blob from the filter index database.
// An entry's absence is not considered an error.
func dbFetchFilterIdxEntry(dbTx database.Tx, key []byte, h *chainhash.Hash) ([]byte, error) {
	idx := dbTx.Metadata().Bucket(cfIndexParentBucketKey).Bucket(key)
	return idx.Get(h[:]), nil
}

// dbStoreFilterIdxEntry在筛选器索引数据库中存储一个数据blob。
// dbStoreFilterIdxEntry stores a data blob in the filter index database.
func dbStoreFilterIdxEntry(dbTx database.Tx, key []byte, h *chainhash.Hash, f []byte) error {
	idx := dbTx.Metadata().Bucket(cfIndexParentBucketKey).Bucket(key)
	return idx.Put(h[:], f)
}

// dbDeleteFilterIdxEntry从筛选器索引数据库中删除一个数据blob。
// dbDeleteFilterIdxEntry deletes a data blob from the filter index database.
func dbDeleteFilterIdxEntry(dbTx database.Tx, key []byte, h *chainhash.Hash) error {
	idx := dbTx.Metadata().Bucket(cfIndexParentBucketKey).Bucket(key)
	return idx.Delete(h[:])
}

// CfIndex implements a committed filter (cf) by hash index.
type CfIndex struct {
	db          database.DB
	chainParams *chaincfg.Params
}

// Ensure the CfIndex type implements the Indexer interface.
var _ Indexer = (*CfIndex)(nil)

// Ensure the CfIndex type implements the NeedsInputser interface.
var _ NeedsInputser = (*CfIndex)(nil)

// needsinput表示索引需要引用的输入，以便正确地创建索引。
// NeedsInputs signals that the index requires the referenced inputs in order
// to properly create the index.
//
// This implements the NeedsInputser interface.
func (idx *CfIndex) NeedsInputs() bool {
	return true
}

// 初始化基于散列的cf索引。这是Indexer接口的一部分。
// Init initializes the hash-based cf index. This is part of the Indexer
// interface.
func (idx *CfIndex) Init() error {
	return nil // Nothing to do.
}

//键以字节片的形式返回用于索引的数据库键。
// Key returns the database key to use for the index as a byte slice. This is
// part of the Indexer interface.
func (idx *CfIndex) Key() []byte {
	return cfIndexParentBucketKey
}

// Name返回人类可读的索引名。
// Name returns the human-readable name of the index. This is part of the
// Indexer interface.
func (idx *CfIndex) Name() string {
	return cfIndexName
}

// Create在索引管理器确定需要首次创建索引时调用。
// Create is invoked when the indexer manager determines the index needs to
// be created for the first time. It creates buckets for the two hash-based cf
// indexes (regular only currently).
func (idx *CfIndex) Create(dbTx database.Tx) error {
	meta := dbTx.Metadata()

	cfIndexParentBucket, err := meta.CreateBucket(cfIndexParentBucketKey)
	if err != nil {
		return err
	}

	for _, bucketName := range cfIndexKeys {
		_, err = cfIndexParentBucket.CreateBucket(bucketName)
		if err != nil {
			return err
		}
	}

	for _, bucketName := range cfHeaderKeys {
		_, err = cfIndexParentBucket.CreateBucket(bucketName)
		if err != nil {
			return err
		}
	}

	for _, bucketName := range cfHashKeys {
		_, err = cfIndexParentBucket.CreateBucket(bucketName)
		if err != nil {
			return err
		}
	}

	return nil
}

//storeFilter存储给定的过滤器，并执行生成过滤器头部所需的步骤。
// storeFilter stores a given filter, and performs the steps needed to
// generate the filter's header.
func storeFilter(dbTx database.Tx, block *btcutil.Block, f *gcs.Filter,
	filterType wire.FilterType) error {
	if uint8(filterType) > maxFilterType {
		return errors.New("unsupported filter type")
	}

	// Figure out which buckets to use.
	fkey := cfIndexKeys[filterType]
	hkey := cfHeaderKeys[filterType]
	hashkey := cfHashKeys[filterType]

	// Start by storing the filter.
	h := block.Hash()
	filterBytes, err := f.NBytes()
	if err != nil {
		return err
	}
	err = dbStoreFilterIdxEntry(dbTx, fkey, h, filterBytes)
	if err != nil {
		return err
	}

	// Next store the filter hash.
	filterHash, err := builder.GetFilterHash(f)
	if err != nil {
		return err
	}
	err = dbStoreFilterIdxEntry(dbTx, hashkey, h, filterHash[:])
	if err != nil {
		return err
	}

	// Then fetch the previous block's filter header.
	var prevHeader *chainhash.Hash
	ph := &block.MsgBlock().Header.PrevBlock
	if ph.IsEqual(&zeroHash) {
		prevHeader = &zeroHash
	} else {
		pfh, err := dbFetchFilterIdxEntry(dbTx, hkey, ph)
		if err != nil {
			return err
		}

		// Construct the new block's filter header, and store it.
		prevHeader, err = chainhash.NewHash(pfh)
		if err != nil {
			return err
		}
	}

	fh, err := builder.MakeHeaderForFilter(f, *prevHeader)
	if err != nil {
		return err
	}
	return dbStoreFilterIdxEntry(dbTx, hkey, h, fh[:])
}

// ConnectBlock由索引管理器在连接到主链的新块时调用。
// ConnectBlock is invoked by the index manager when a new block has been
// connected to the main chain. This indexer adds a hash-to-cf mapping for
// every passed block. This is part of the Indexer interface.
func (idx *CfIndex) ConnectBlock(dbTx database.Tx, block *btcutil.Block,
	stxos []blockchain.SpentTxOut) error {

	prevScripts := make([][]byte, len(stxos))
	for i, stxo := range stxos {
		prevScripts[i] = stxo.PkScript
	}

	f, err := builder.BuildBasicFilter(block.MsgBlock(), prevScripts)
	if err != nil {
		return err
	}

	return storeFilter(dbTx, block, f, wire.GCSFilterRegular)
}

//当一个块从主链断开时，索引管理器调用DisconnectBlock。
// DisconnectBlock is invoked by the index manager when a block has been
// disconnected from the main chain.  This indexer removes the hash-to-cf
// mapping for every passed block. This is part of the Indexer interface.
func (idx *CfIndex) DisconnectBlock(dbTx database.Tx, block *btcutil.Block,
	_ []blockchain.SpentTxOut) error {

	for _, key := range cfIndexKeys {
		err := dbDeleteFilterIdxEntry(dbTx, key, block.Hash())
		if err != nil {
			return err
		}
	}

	for _, key := range cfHeaderKeys {
		err := dbDeleteFilterIdxEntry(dbTx, key, block.Hash())
		if err != nil {
			return err
		}
	}

	for _, key := range cfHashKeys {
		err := dbDeleteFilterIdxEntry(dbTx, key, block.Hash())
		if err != nil {
			return err
		}
	}

	return nil
}

// entryByBlockHash获取特定类型的筛选器索引项
// entryByBlockHash fetches a filter index entry of a particular type
// (eg. filter, filter header, etc) for a filter type and block hash.
func (idx *CfIndex) entryByBlockHash(filterTypeKeys [][]byte,
	filterType wire.FilterType, h *chainhash.Hash) ([]byte, error) {

	if uint8(filterType) > maxFilterType {
		return nil, errors.New("unsupported filter type")
	}
	key := filterTypeKeys[filterType]

	var entry []byte
	err := idx.db.View(func(dbTx database.Tx) error {
		var err error
		entry, err = dbFetchFilterIdxEntry(dbTx, key, h)
		return err
	})
	return entry, err
}

// entriesByBlockHashes批处理获取特定类型的筛选器索引项
// entriesByBlockHashes batch fetches a filter index entry of a particular type
// (eg. filter, filter header, etc) for a filter type and slice of block hashes.
func (idx *CfIndex) entriesByBlockHashes(filterTypeKeys [][]byte,
	filterType wire.FilterType, blockHashes []*chainhash.Hash) ([][]byte, error) {

	if uint8(filterType) > maxFilterType {
		return nil, errors.New("unsupported filter type")
	}
	key := filterTypeKeys[filterType]

	entries := make([][]byte, 0, len(blockHashes))
	err := idx.db.View(func(dbTx database.Tx) error {
		for _, blockHash := range blockHashes {
			entry, err := dbFetchFilterIdxEntry(dbTx, key, blockHash)
			if err != nil {
				return err
			}
			entries = append(entries, entry)
		}
		return nil
	})
	return entries, err
}

// FilterByBlockHash返回块的基本过滤器或提交过滤器的序列化内容。
// FilterByBlockHash returns the serialized contents of a block's basic or
// committed filter.
func (idx *CfIndex) FilterByBlockHash(h *chainhash.Hash,
	filterType wire.FilterType) ([]byte, error) {
	return idx.entryByBlockHash(cfIndexKeys, filterType, h)
}

// FiltersByBlockHashes通过散列返回一组块的基本过滤器或提交过滤器的序列化内容。
// FiltersByBlockHashes returns the serialized contents of a block's basic or
// committed filter for a set of blocks by hash.
func (idx *CfIndex) FiltersByBlockHashes(blockHashes []*chainhash.Hash,
	filterType wire.FilterType) ([][]byte, error) {
	return idx.entriesByBlockHashes(cfIndexKeys, filterType, blockHashes)
}

// FilterHeaderByBlockHash返回一个块的基本提交过滤器头的序列化内容。
// FilterHeaderByBlockHash returns the serialized contents of a block's basic
// committed filter header.
func (idx *CfIndex) FilterHeaderByBlockHash(h *chainhash.Hash,
	filterType wire.FilterType) ([]byte, error) {
	return idx.entryByBlockHash(cfHeaderKeys, filterType, h)
}

// FilterHeadersByBlockHashes returns the serialized contents of a block's
// basic committed filter header for a set of blocks by hash.
func (idx *CfIndex) FilterHeadersByBlockHashes(blockHashes []*chainhash.Hash,
	filterType wire.FilterType) ([][]byte, error) {
	return idx.entriesByBlockHashes(cfHeaderKeys, filterType, blockHashes)
}

// FilterHashByBlockHash返回一个块的基本提交过滤器散列的序列化内容。
// FilterHashByBlockHash returns the serialized contents of a block's basic
// committed filter hash.
func (idx *CfIndex) FilterHashByBlockHash(h *chainhash.Hash,
	filterType wire.FilterType) ([]byte, error) {
	return idx.entryByBlockHash(cfHashKeys, filterType, h)
}

// filterhashesbyblockhash根据哈希值返回一组块的基本提交的筛选器哈希的序列化内容。
// FilterHashesByBlockHashes returns the serialized contents of a block's basic
// committed filter hash for a set of blocks by hash.
func (idx *CfIndex) FilterHashesByBlockHashes(blockHashes []*chainhash.Hash,
	filterType wire.FilterType) ([][]byte, error) {
	return idx.entriesByBlockHashes(cfHashKeys, filterType, blockHashes)
}

//NewCfIndex返回一个索引器的新实例，该实例用于创建一个映射，将区块链中所有块的散列映射到各自提交的过滤器。
// NewCfIndex returns a new instance of an indexer that is used to create a
// mapping of the hashes of all blocks in the blockchain to their respective
// committed filters.
//
// It implements the Indexer interface which plugs into the IndexManager that
// in turn is used by the blockchain package. This allows the index to be
// seamlessly maintained along with the chain.
func NewCfIndex(db database.DB, chainParams *chaincfg.Params) *CfIndex {
	return &CfIndex{db: db, chainParams: chainParams}
}

// DropCfIndex如果存在，则从提供的数据库中删除CF索引。
// DropCfIndex drops the CF index from the provided database if exists.
func DropCfIndex(db database.DB, interrupt <-chan struct{}) error {
	return dropIndex(db, cfIndexParentBucketKey, cfIndexName, interrupt)
}
