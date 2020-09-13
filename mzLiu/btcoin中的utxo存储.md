##### package blockchain

Package blockchain implements bitcoin block handling and chain selection rules.

The bitcoin block handling and chain selection rules are an integral, and quite likely the most important, part of bitcoin. Unfortunately, at the time of this writing, these rules are also largely undocumented and had to be ascertained from the bitcoind source code. At its core, bitcoin is a distributed consensus of which blocks are valid and which ones will comprise the main block chain (public ledger) that ultimately determines accepted transactions, so it is extremely important that fully validating nodes agree on all rules.

##### files

utxoset的操作中涉及到chainio.go upgrade.go validate.go

##### const

```go
latestUtxoSetBucketVersion = 2
```

最新的UtxoSetBuket版本

##### type

```go
utxoSetVersionKeyName = []byte("utxosetversion")
utxoSetBucketName = []byte("utxosetv2")
```

leveldb中存储utxoVersion的Key的名字

leveldb中存储utxoBuket的Key的名字

##### func dbFetchUtxoEntryByHash

```go
func dbFetchUtxoEntryByHash(dbTx database.Tx, hash *chainhash.Hash) (*UtxoEntry, error) 
```

用给定的hash取得对应的utxo，使用光标实现（为实现最效率），若没有则返回null

##### func dbFetchUtxoEntry

```go
func dbFetchUtxoEntry(dbTx database.Tx, outpoint wire.OutPoint) (*UtxoEntry, error)
```

用数据库中已经存在的交易从utxo中取出交易的输出

##### func dbPutUtxoView

```go
func dbPutUtxoView(dbTx database.Tx, view *UtxoViewpoint) error 
	for outpoint, entry := range view.entries {
		...
        // No need to update the database if the entry was not modified.
    }
        // Remove the utxo entry if it is spent.
		...
        // Serialize and store the utxo entry.
...
        // NOTE: The key is intentionally not recycled here since the
		// database interface contract prohibits modifications.  It will
		// be garbage collected normally when the database is done with
		// it.
```

dbPutUtxoView使用现有的数据库的交易，根据提供的utxo视图内容和状态，更新数据库中的utxo集。特别是，只有标记为修改的条目才会写入数据库。

这个地方比较重要，基本上就是根据tx把把utxo集一顿修改，

```go
func (b *BlockChain) createChainState() error
```

建立创世区块，初始化各种数据库Bucket，在这个过程中调用了dbPutVersion(dbTx,utxoSetVersionKeyName,latestUtxoSetBucketVersion)函数和CreateBucket(utxoSetBucketName)存储utxo集的元数据和utxo集数据库Bucket

##### func upgradeUtxoSetToV2

```go
func upgradeUtxoSetToV2(db database.DB, interrupt <-chan struct{}) error
```

批量将utxo集条目从版本1迁移到2。

##### func (b *BlockChain) maybeUpgradeDbBuckets

```
func (b *BlockChain) maybeUpgradeDbBuckets(interrupt <-chan struct{}) error
```

检查此包里buckets的database是否为最新，并升级到最新

##### func (b *BlockChain) checkConnectBlock

```
func (b *BlockChain) checkConnectBlock(node *blockNode, block *btcutil.Block, view *UtxoViewpoint, stxos *[]SpentTxOut) error
```

