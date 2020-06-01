# 一、Blockchain的数据结构

```go
type BlockChain struct {
	chainConfig *params.ChainConfig // 初始化配置
	cacheConfig *CacheConfig        // 缓存配置

	db     ethdb.Database // Low level persistent database to store final content in
	triegc *prque.Prque   // Priority queue mapping block numbers to tries to gc
	gcproc time.Duration  // Accumulates canonical block processing for trie dumping

	hc            *HeaderChain // 区块头组成的链
	rmLogsFeed    event.Feed
	chainFeed     event.Feed
	chainSideFeed event.Feed
	chainHeadFeed event.Feed
	logsFeed      event.Feed
	blockProcFeed event.Feed
	scope         event.SubscriptionScope
	genesisBlock  *types.Block // 创世区块

	chainmu sync.RWMutex // 区块链插入锁

	currentBlock     atomic.Value // 主链的头区块
	currentFastBlock atomic.Value // 快速同步模式下链的头区块，这种情况下可能比主链长

	stateCache    state.Database // State database to reuse between imports (contains state cache)
	bodyCache     *lru.Cache     // Cache for the most recent block bodies
	bodyRLPCache  *lru.Cache     // Cache for the most recent block bodies in RLP encoded format
	receiptsCache *lru.Cache     // Cache for the most recent receipts per block
	blockCache    *lru.Cache     // Cache for the most recent entire blocks
	txLookupCache *lru.Cache     // Cache for the most recent transaction lookup data.
	futureBlocks  *lru.Cache     // future blocks are blocks added for later processing

	quit    chan struct{} // blockchain quit channel
	running int32         // running must be called atomically
	// procInterrupt must be atomically called
	procInterrupt int32          // interrupt signaler for block processing
	wg            sync.WaitGroup // chain processing wait group for shutting down

	engine     consensus.Engine // 用来验证区块的接口
	validator  Validator  // 验证数据有效性的接口
	prefetcher Prefetcher // 块状态预取接口
	processor  Processor  // 块交易处理接口
	vmConfig   vm.Config  // 虚拟机的配置

	badBlocks       *lru.Cache                     // 错误区块的缓存.
	shouldPreserve  func(*types.Block) bool        // 用于确定是否应保留给定块的函数。
	terminateInsert func(common.Hash, uint64) bool // Testing hook used to terminate ancient receipt chain insertion.
}
```

1、BlockChain 表示了一个规范的链, 这个链通过一个包含了创世区块的数据库指定. BlockChain管理了链的插入,还原,重建等操作.

2、插入一个区块需要通过一系列指定的规则指定的两阶段的验证器. 

3、使用Processor来对区块的交易进行处理. 状态的验证是第二阶段的验证器. 错误将导致插入终止.

4、GetBlock可能返回任意不在当前规范区块链中的区块,GetBlockByNumber 总是返回当前规范区块链中的区块.



**关键元素**

1）**db**：连接到底层数据储存，包括两部分 KeyvalueStore和AncientStore
2）**hc**：headerchain区块头链，可以用于快速延长链，验证通过后再下载blockchain，或者可以与blockchain进行相互验证；
3）**genesisBlock**：创始区块；
4）**currentBlock**：当前区块，blockchain中并不是储存链所有的block，而是通过currentBlock向前回溯直到genesisBlock，这样就构成了区块链。
5）**bodyCache、bodyRLPCache、receiptsCache、blockCache、futureBlocks**：区块链中的缓存结构，用于加快区块链的读取和构建；
6）**engine**：是consensus模块中的接口，用来验证block的接口；

7)  **prefetcher**：块预取接口，目的是在主块处理器开始执行之前从磁盘预取可能有用的状态数据。

8)  **processor**：执行区块链交易的接口，收到一个新的区块时，要对区块中的所有交易执行一遍，一方面是验证，一方面是更新世界状态；
9）**validator**：验证数据有效性的接口
10）**futureBlocks**：收到的区块时间大于当前头区块时间15s而小于30s的区块，可作为当前节点待处理的区块。

# 二、**blockchain.go中的部分函数和方法** 

## 1、NewBlockChain

使用数据库里面的可用信息构造了一个初始化好的区块链. 同时初始化了以太坊默认的 验证器和处理器，预取器等。

```go
func NewBlockChain(db ethdb.Database, cacheConfig *CacheConfig, chainConfig *params.ChainConfig, engine consensus.Engine, vmConfig vm.Config, shouldPreserve func(block *types.Block) bool) (*BlockChain, error) {
	if cacheConfig == nil {
		cacheConfig = &CacheConfig{
			TrieCleanLimit: 256,
			TrieDirtyLimit: 256,
			TrieTimeLimit:  5 * time.Minute,
		}
	}
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	receiptsCache, _ := lru.New(receiptsCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	txLookupCache, _ := lru.New(txLookupCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)
	badBlocks, _ := lru.New(badBlockLimit)

	bc := &BlockChain{
		chainConfig:    chainConfig,
		cacheConfig:    cacheConfig,
		db:             db,
		triegc:         prque.New(nil),
		stateCache:     state.NewDatabaseWithCache(db, cacheConfig.TrieCleanLimit),
		quit:           make(chan struct{}),
		shouldPreserve: shouldPreserve,
		bodyCache:      bodyCache,
		bodyRLPCache:   bodyRLPCache,
		receiptsCache:  receiptsCache,
		blockCache:     blockCache,
		txLookupCache:  txLookupCache,
		futureBlocks:   futureBlocks,
		engine:         engine,
		vmConfig:       vmConfig,
		badBlocks:      badBlocks,
	}
	bc.validator = NewBlockValidator(chainConfig, bc, engine)
	bc.prefetcher = newStatePrefetcher(chainConfig, bc, engine)
	bc.processor = NewStateProcessor(chainConfig, bc, engine)

	var err error
	bc.hc, err = NewHeaderChain(db, chainConfig, engine, bc.getProcInterrupt)
	if err != nil {
		return nil, err
	}
	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}

	var nilBlock *types.Block
	bc.currentBlock.Store(nilBlock)
	bc.currentFastBlock.Store(nilBlock)

	// Initialize the chain with ancient data if it isn't empty.
	if bc.empty() {
		rawdb.InitDatabaseFromFreezer(bc.db)
	}
	// 加载数据库里面的最新的我们知道的区块链状态.
	if err := bc.loadLastState(); err != nil {
		return nil, err
	}
	// 节点要做的第一件事是重构头块(ethash缓存或clique投票快照)的验证数据。
	bc.engine.VerifyHeader(bc, bc.CurrentHeader(), true)

	if frozen, err := bc.db.Ancients(); err == nil && frozen > 0 {
		var (
			needRewind bool
			low        uint64
		)
		// 如果头块比ancient chain还要低，截断ancient store
		fullBlock := bc.CurrentBlock()
		if fullBlock != nil && fullBlock != bc.genesisBlock && fullBlock.NumberU64() < frozen-1 {
			needRewind = true
			low = fullBlock.NumberU64()
		}
		// 在快速同步中，可能会发生ancient data被写入到ancient store，但是LastFastBlock没有被更新的情况，截断额外的数据。
		fastBlock := bc.CurrentFastBlock()
		if fastBlock != nil && fastBlock.NumberU64() < frozen-1 {
			needRewind = true
			if fastBlock.NumberU64() < low || low == 0 {
				low = fastBlock.NumberU64()
			}
		}
		if needRewind {
			var hashes []common.Hash
			previous := bc.CurrentHeader().Number.Uint64()
			for i := low + 1; i <= bc.CurrentHeader().Number.Uint64(); i++ {
				hashes = append(hashes, rawdb.ReadCanonicalHash(bc.db, i))
			}
			bc.Rollback(hashes)
			log.Warn("Truncate ancient chain", "from", previous, "to", low)
		}
	}
	// 检查块哈希的当前状态，并确保链中没有任何bad block
    // BadHashes是一些手工配置的区块hash值, 用来硬分叉使用的.
	for hash := range BadHashes {
		if header := bc.GetHeaderByHash(hash); header != nil {
			// 获取规范的区块链上面同样高度的区块头,如果这个区块头确实是在我们的规范的区块链上的话,我们需要回滚到这个区块头的高度 - 1
			headerByNumber := bc.GetHeaderByNumber(header.Number.Uint64())
			// make sure the headerByNumber (if present) is in our current canonical chain
			if headerByNumber != nil && headerByNumber.Hash() == header.Hash() {
				log.Error("Found bad hash, rewinding chain", "number", header.Number, "hash", header.ParentHash)
                // SetHead 回滚到该区块头高度-1的位置
				bc.SetHead(header.Number.Uint64() - 1)
				log.Error("Chain rewind was successful, resuming normal operation")
			}
		}
	}
	// 开启处理未来区块的go线程
	go bc.update()
	return bc, nil
}

```
  检查本地区块链上是否有bad block，如果有调用bc.SetHead回到硬分叉之前的区块头 

```go
bc.GetHeaderByHash(hash)
—> bc.hc.GetHeaderByHash(hash)
—> hc.GetBlockNumber(hash)  // 通过hash来找到这个区块的number，即用‘H’+hash为key在数据库中查找
—> hc.GetHeader(hash, *number)  // 通过hash+number来找到header
—> hc.headerCache.Get(hash)  // 先从缓存里找，找不到再去数据库找
—> rawdb.ReadHeader(hc.chainDb, hash, number)  // 在数据库中，通过'h'+num+hash为key来找到header的RLP编码值

bc.GetHeaderByNumber(number)
—> hc.GetHeaderByNumber(number)
—> raw.ReadCanonicalHash(hc.chainDb, number) 
// 在规范链上以‘h’+num+‘n’为key查找区块的hash，
// 如果找到了，说明区块链上确实有该badblock
// 如果找不到，则说明该bad block只存在数据库，没有上规范链
—> hc.GetHeader(hash,number) // 如果规范链上有这个badblock，则返回该block header
```

## 2、Rollback

```go
// Rollback 旨在从数据库中删除不确定有效的链片段
func (bc *BlockChain) Rollback(chain []common.Hash) {
	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	batch := bc.db.NewBatch()
	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]
		// 设置当前区块头header
		currentHeader := bc.hc.CurrentHeader()
		if currentHeader.Hash() == hash {
			newHeadHeader := bc.GetHeader(currentHeader.ParentHash, currentHeader.Number.Uint64()-1)
			// headHeaderKey = []byte("LastHeader")作为KEY
             rawdb.WriteHeadHeaderHash(batch, currentHeader.ParentHash)
			bc.hc.SetCurrentHeader(newHeadHeader)
		}
		if currentFastBlock := bc.CurrentFastBlock(); currentFastBlock.Hash() == hash {
			newFastBlock := bc.GetBlock(currentFastBlock.ParentHash(), currentFastBlock.NumberU64()-1)
			rawdb.WriteHeadFastBlockHash(batch, currentFastBlock.ParentHash())
			bc.currentFastBlock.Store(newFastBlock)
			headFastBlockGauge.Update(int64(newFastBlock.NumberU64()))
		}
		if currentBlock := bc.CurrentBlock(); currentBlock.Hash() == hash {
			newBlock := bc.GetBlock(currentBlock.ParentHash(), currentBlock.NumberU64()-1)
			rawdb.WriteHeadBlockHash(batch, currentBlock.ParentHash())
			bc.currentBlock.Store(newBlock)
			headBlockGauge.Update(int64(newBlock.NumberU64()))
		}
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to rollback chain markers", "err", err)
	}
	//截断超过current header的旧数据。值得注意的是，系统崩溃时不会截断旧数据，但在活动存储中已更新了head indicator。对于这个问题，系统将通过在安装阶段截断额外的数据进行自我恢复。
	if err := bc.truncateAncient(bc.hc.CurrentHeader().Number.Uint64()); err != nil {
		log.Crit("Truncate ancient store failed", "err", err)
	}
}
```

从高到低回滚，不断的设置当前内存中的标记，然后截断freezer中超出当前区块header的数据。

## 3、SetHead

```go
func (bc *BlockChain) SetHead(head uint64) error {
	log.Warn("Rewinding blockchain", "target", head)

	bc.chainmu.Lock()
	defer bc.chainmu.Unlock()

	updateFn := func(db ethdb.KeyValueWriter, header *types.Header) {
		// 回退区块链，确保我们不会以无状态的头区块结束
		if currentBlock := bc.CurrentBlock(); currentBlock != nil && header.Number.Uint64() < currentBlock.NumberU64() {
			// 获取我们要回退到的区块
			newHeadBlock := bc.GetBlock(header.Hash(), header.Number.Uint64())
			if newHeadBlock == nil {
				newHeadBlock = bc.genesisBlock
			} else {
				if _, err := state.New(newHeadBlock.Root(), bc.stateCache); err != nil {
					newHeadBlock = bc.genesisBlock
				}
			}
			// headBlockKey = []byte("LastBlock")
			rawdb.WriteHeadBlockHash(db, newHeadBlock.Hash())

			bc.currentBlock.Store(newHeadBlock)
			headBlockGauge.Update(int64(newHeadBlock.NumberU64()))
		}

		// Rewind the fast block in a simpleton way to the target head
		if currentFastBlock := bc.CurrentFastBlock(); currentFastBlock != nil && header.Number.Uint64() < currentFastBlock.NumberU64() {
			newHeadFastBlock := bc.GetBlock(header.Hash(), header.Number.Uint64())
			// If either blocks reached nil, reset to the genesis state
			if newHeadFastBlock == nil {
				newHeadFastBlock = bc.genesisBlock
			}
			rawdb.WriteHeadFastBlockHash(db, newHeadFastBlock.Hash())

			bc.currentFastBlock.Store(newHeadFastBlock)
			headFastBlockGauge.Update(int64(newHeadFastBlock.NumberU64()))
		}
	}

	// Rewind the header chain, deleting all block bodies until then
	// 回滚header链，在此之前删除所有块体
	delFn := func(db ethdb.KeyValueWriter, hash common.Hash, num uint64) {
		// Ignore the error here since light client won't hit this path
		frozen, _ := bc.db.Ancients()
		if num+1 <= frozen {
			// 截断所有相关数据(header、总难度、body、receipt和规范hash)。
			if err := bc.db.TruncateAncients(num + 1); err != nil {
				log.Crit("Failed to truncate ancient data", "number", num, "err", err)
			}

			// Remove the hash <-> number mapping from the active store.
			rawdb.DeleteHeaderNumber(db, hash)
		} else {
			// 从active store移除相关body和receipts。header、总难度和规范hash将在hc.SetHead()中删除
			rawdb.DeleteBody(db, hash, num)
			rawdb.DeleteReceipts(db, hash, num)
		}
		// Todo(rjl493456442) txlookup, bloombits, etc
	}
	bc.hc.SetHead(head, updateFn, delFn)

	//清除缓存中的内容
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.receiptsCache.Purge()
	bc.blockCache.Purge()
	bc.txLookupCache.Purge()
	bc.futureBlocks.Purge()

	return bc.loadLastState()
}
```

1) 首先调用bc.hc.SetHead(head, updateFn, delFn)，回滚head对应的区块头。

2) updateFn:  重新设置bc.currentBlock，bc.currentFastBlock 

3) delFn:  清除中间区块头所有的数据和缓存 

4) 调用bc.loadLastState()，重新加载本地的最新状态 

## 4、loadLastState

加载数据库里面的最新的我们知道的区块链状态. 就是要找到最新的区块头，然后设置currentBlock、currentHeader和currentFastBlock 

1) 	获取到最新区块以及它的hash

> 		// 跟踪最新的已知完整区块的哈希。
> headBlockKey = []byte("LastBlock")

2)	从stateDb中打开最新区块的状态trie，如果打开失败调用bc.repair(&currentBlock)方法进行修复。修复方法就是从当前区块一个个的往前面找，直到找到好的区块，然后赋值给currentBlock。

3)	获取到最新的区块头

4)	找到最新的fast模式下的block，并设置bc.currentFastBlock

```go
func (bc *BlockChain) loadLastState() error {
	// 还原最后一个已知的头块
	head := rawdb.ReadHeadBlockHash(bc.db)
	if head == (common.Hash{}) {
		// 损坏或空的数据库，初始化从零开始
		log.Warn("Empty database, resetting chain")
		return bc.Reset()
	}
	// 确保整个头块可用
	currentBlock := bc.GetBlockByHash(head)
	if currentBlock == nil {
		log.Warn("Head block missing, resetting chain", "hash", head)
		return bc.Reset()
	}
	// 确保与块关联的状态可用
	if _, err := state.New(currentBlock.Root(), bc.stateCache); err != nil {
		// Dangling block without a state associated, init from scratch
		log.Warn("Head state missing, repairing chain", "number", currentBlock.Number(), "hash", currentBlock.Hash())
		if err := bc.repair(&currentBlock); err != nil {
			return err
		}
		rawdb.WriteHeadBlockHash(bc.db, currentBlock.Hash())
	}
	// Everything seems to be fine, set as the head block
	bc.currentBlock.Store(currentBlock)
	headBlockGauge.Update(int64(currentBlock.NumberU64()))

	// 还原最后一个已知的head header
	currentHeader := currentBlock.Header()
	if head := rawdb.ReadHeadHeaderHash(bc.db); head != (common.Hash{}) {
		if header := bc.GetHeaderByHash(head); header != nil {
			currentHeader = header
		}
	}
	bc.hc.SetCurrentHeader(currentHeader)

	// 恢复最后一个已知的head fast block
	bc.currentFastBlock.Store(currentBlock)
	headFastBlockGauge.Update(int64(currentBlock.NumberU64()))

	if head := rawdb.ReadHeadFastBlockHash(bc.db); head != (common.Hash{}) {
		if block := bc.GetBlockByHash(head); block != nil {
			bc.currentFastBlock.Store(block)
			headFastBlockGauge.Update(int64(block.NumberU64()))
		}
	}
	
	currentFastBlock := bc.CurrentFastBlock()

	headerTd := bc.GetTd(currentHeader.Hash(), currentHeader.Number.Uint64())
	blockTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
	fastTd := bc.GetTd(currentFastBlock.Hash(), currentFastBlock.NumberU64())
	// 为用户发出状态日志
	log.Info("Loaded most recent local header", "number", currentHeader.Number, "hash", currentHeader.Hash(), "td", headerTd, "age", common.PrettyAge(time.Unix(int64(currentHeader.Time), 0)))
	log.Info("Loaded most recent local full block", "number", currentBlock.Number(), "hash", currentBlock.Hash(), "td", blockTd, "age", common.PrettyAge(time.Unix(int64(currentBlock.Time()), 0)))
	log.Info("Loaded most recent local fast block", "number", currentFastBlock.Number(), "hash", currentFastBlock.Hash(), "td", fastTd, "age", common.PrettyAge(time.Unix(int64(currentFastBlock.Time()), 0)))

	return nil
}
```

## 5、reorgs

该方法是在新的链的总难度大于本地链的总难度的情况下，需要用新的区块链来替换本地的区块链为规范链。

 前提条件：newBlock的总难度大于oldBlock，且newBlock的父区块不是oldBlock。

```go
// reorgs需要两个块、一个旧链以及一个新链，并将重新构建块然后将它们插入到新的规范链中，并累积潜在的缺失交易并发布有关它们的事件
func (bc *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
	var (
		newChain    types.Blocks
		oldChain    types.Blocks
		commonBlock *types.Block

		deletedTxs types.Transactions
		addedTxs   types.Transactions

		deletedLogs [][]*types.Log
		rebirthLogs [][]*types.Log

		// collectLogs 会收集我们已经生成的日志信息
		collectLogs = func(hash common.Hash, removed bool) {
			......
		}
		// mergeLogs返回具有指定排序顺序的合并日志片。
		mergeLogs = func(logs [][]*types.Log, reverse bool) []*types.Log {
			......
		}
	)
    
	// 第一步：找到新链和老链的共同祖先
	// 将较长的链减少到与较短的链相同的数目
	if oldBlock.NumberU64() > newBlock.NumberU64() {
		// 如果老的链比新的链高。那么需要减少老的链，让它和新链一样高
        // 并且收集老链分支上的交易和日志
		for ; oldBlock != nil && oldBlock.NumberU64() != newBlock.NumberU64(); oldBlock = bc.GetBlock(oldBlock.ParentHash(), oldBlock.NumberU64()-1) {
			oldChain = append(oldChain, oldBlock)
			deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
			collectLogs(oldBlock.Hash(), true)
		}
	} else {
		// 如果新链比老链要高，那么减少新链。
		for ; newBlock != nil && newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = bc.GetBlock(newBlock.ParentHash(), newBlock.NumberU64()-1) {
			newChain = append(newChain, newBlock)
		}
	}
	if oldBlock == nil {
		return fmt.Errorf("invalid old chain")
	}
	if newBlock == nil {
		return fmt.Errorf("invalid new chain")
	}
	//等到共同高度后，去找到共同祖先（共同回退），继续收集日志和交易
	for {
		// If the common ancestor was found, bail out
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}
		// Remove an old block as well as stash away a new block
		oldChain = append(oldChain, oldBlock)
		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
		collectLogs(oldBlock.Hash(), true)

		newChain = append(newChain, newBlock)

		// Step back with both chains
		oldBlock = bc.GetBlock(oldBlock.ParentHash(), oldBlock.NumberU64()-1)
		if oldBlock == nil {
			return fmt.Errorf("invalid old chain")
		}
		newBlock = bc.GetBlock(newBlock.ParentHash(), newBlock.NumberU64()-1)
		if newBlock == nil {
			return fmt.Errorf("invalid new chain")
		}
	}
	// 打印规则
	if len(oldChain) > 0 && len(newChain) > 0 {
		logFn := log.Info
		msg := "Chain reorg detected"
		if len(oldChain) > 63 {
			msg = "Large chain reorg detected"
			logFn = log.Warn
		}
		logFn(msg, "number", commonBlock.Number(), "hash", commonBlock.Hash(),
			"drop", len(oldChain), "dropfrom", oldChain[0].Hash(), "add", len(newChain), "addfrom", newChain[0].Hash())
		blockReorgAddMeter.Mark(int64(len(newChain)))
		blockReorgDropMeter.Mark(int64(len(oldChain)))
	} else {
		log.Error("Impossible reorg, please file an issue", "oldnum", oldBlock.Number(), "oldhash", oldBlock.Hash(), "newnum", newBlock.Number(), "newhash", newBlock.Hash())
	}
	// 插入新链
	for i := len(newChain) - 1; i >= 1; i-- {
		// Insert the block in the canonical way, re-writing history
		bc.writeHeadBlock(newChain[i])

		// Collect reborn logs due to chain reorg
		collectLogs(newChain[i].Hash(), false)

		// Collect the new added transactions.
		addedTxs = append(addedTxs, newChain[i].Transactions()...)
	}
	// 立即删除无用的索引，其中包括非规范交易索引，以及head上方的规范链索引。
	indexesBatch := bc.db.NewBatch()
    // TxDifference返回一个a-b的差集合
	for _, tx := range types.TxDifference(deletedTxs, addedTxs) {
		rawdb.DeleteTxLookupEntry(indexesBatch, tx.Hash())
	}
	// Delete any canonical number assignments above the new head
	number := bc.CurrentBlock().NumberU64()
	for i := number + 1; ; i++ {
		hash := rawdb.ReadCanonicalHash(bc.db, i)
		if hash == (common.Hash{}) {
			break
		}
		rawdb.DeleteCanonicalHash(indexesBatch, i)
	}
	if err := indexesBatch.Write(); err != nil {
		log.Crit("Failed to delete useless indexes", "err", err)
	}
	// If any logs need to be fired, do it now. In theory we could avoid creating
	// this goroutine if there are no events to fire, but realistcally that only
	// ever happens if we're reorging empty blocks, which will only happen on idle
	// networks where performance is not an issue either way.
  	// 向外发送区块被reorgs的事件，以及日志删除事件
	if len(deletedLogs) > 0 {
		bc.rmLogsFeed.Send(RemovedLogsEvent{mergeLogs(deletedLogs, true)})
	}
	if len(rebirthLogs) > 0 {
		bc.logsFeed.Send(mergeLogs(rebirthLogs, false))
	}
	if len(oldChain) > 0 {
		for i := len(oldChain) - 1; i >= 0; i-- {
			bc.chainSideFeed.Send(ChainSideEvent{Block: oldChain[i]})
		}
	}
	return nil
}
```

1)  找到新链和老链的共同祖先

2)  将新链插入到规范链中，同时收集插入到规范链中的所有交易 

3)  删除无用的索引，其中包括非规范交易索引（deletedTxs - addedTxs），以及head上方的规范链索引。

4）向外发送区块被reorgs的事件，以及日志删除事件

## 6、writeBlockWithState

```go
const (
	NonStatTy WriteStatus = iota
	CanonStatTy
	SideStatTy
)

func (bc *BlockChain) writeBlockWithState(block *types.Block, receipts []*types.Receipt, logs []*types.Log, state *state.StateDB, emitHeadEvent bool) (status WriteStatus, err error) {
	bc.wg.Add(1)
	defer bc.wg.Done()

	// 获取父区块总难度
	ptd := bc.GetTd(block.ParentHash(), block.NumberU64()-1)
	if ptd == nil {
		return NonStatTy, consensus.ErrUnknownAncestor
	}
	// 获取当前本地规范链头区块的总难度
	currentBlock := bc.CurrentBlock()
	localTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
    // 计算待插入区块的总难度
	externTd := new(big.Int).Add(block.Difficulty(), ptd)

	// Irrelevant of the canonical status, write the block itself to the database.
	//
	// Note all the components of block(td, hash->number map, header, body, receipts)
	// should be written atomically. BlockBatch is used for containing all components.
    // 将块写入数据库
	blockBatch := bc.db.NewBatch()
	rawdb.WriteTd(blockBatch, block.Hash(), block.NumberU64(), externTd)
	rawdb.WriteBlock(blockBatch, block)
	rawdb.WriteReceipts(blockBatch, block.Hash(), block.NumberU64(), receipts)
	rawdb.WritePreimages(blockBatch, state.Preimages())
	if err := blockBatch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
	// 将所有缓存的状态更改提交到底层内存数据库.
	root, err := state.Commit(bc.chainConfig.IsEIP158(block.Number()))
	if err != nil {
		return NonStatTy, err
	}
	triedb := bc.stateCache.TrieDB()
	// 中间的过程是将新的trie树内容写入数据库
    ......
    
	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
    // 将待插入的区块写入规范链
	reorg := externTd.Cmp(localTd) > 0
	currentBlock = bc.CurrentBlock()    
	// 如果待插入区块的总难度等于本地规范链的总难度，
    // 但是区块号小于或等于当前规范链的头区块号，均认为待插入的区块所在分叉更有效，需要处理分叉并更新规范链
	if !reorg && externTd.Cmp(localTd) == 0 {
		// Split same-difficulty blocks by number, then preferentially select
		// the block generated by the local miner as the canonical block.
		if block.NumberU64() < currentBlock.NumberU64() {
			reorg = true
		} else if block.NumberU64() == currentBlock.NumberU64() {
			var currentPreserve, blockPreserve bool
			if bc.shouldPreserve != nil {
				currentPreserve, blockPreserve = bc.shouldPreserve(currentBlock), bc.shouldPreserve(block)
			}
			reorg = !currentPreserve && (blockPreserve || mrand.Float64() < 0.5)
		}
	}
    // 如果待插入区块的总难度大于本地规范链的总难度，那Block必定要插入规范链
    // 如果待插入区块的总难度小于本地规范链的总难度，待插入区块在另一个分叉上，不用插入规范链
	if reorg {
		// Reorganise the chain if the parent is not the head block
		if block.ParentHash() != currentBlock.Hash() {
			if err := bc.reorg(currentBlock, block); err != nil {
				return NonStatTy, err
			}
		}
		status = CanonStatTy
	} else {
		status = SideStatTy
	}
	// Set new head.
	if status == CanonStatTy {
		bc.writeHeadBlock(block)
	}
    // 从futureBlock中删除刚才插入的区块
	bc.futureBlocks.Remove(block.Hash())

	if status == CanonStatTy {
		bc.chainFeed.Send(ChainEvent{Block: block, Hash: block.Hash(), Logs: logs})
		if len(logs) > 0 {
			bc.logsFeed.Send(logs)
		}
		// In theory we should fire a ChainHeadEvent when we inject
		// a canonical block, but sometimes we can insert a batch of
		// canonicial blocks. Avoid firing too much ChainHeadEvents,
		// we will fire an accumulated ChainHeadEvent and disable fire
		// event here.
		if emitHeadEvent {
			bc.chainHeadFeed.Send(ChainHeadEvent{Block: block})
		}
	} else {
		bc.chainSideFeed.Send(ChainSideEvent{Block: block})
	}
	return status, nil
}

```

## 7、InsertChain

首先逐个检查区块的区块号是否连续以及hash链是否连续

```go
// InsertChain尝试将给定批量的block插入到规范链中，否则，创建一个分叉。 如果返回错误，它将返回失败块的索引号以及描述错误的错误。
// 插入完成后，将触发所有累积的事件。
func (bc *BlockChain) InsertChain(chain types.Blocks) (int, error) {
	// Sanity check that we have something meaningful to import
	if len(chain) == 0 {
		return 0, nil
	}

	bc.blockProcFeed.Send(true)
	defer bc.blockProcFeed.Send(false)

	// Remove already known canon-blocks
	var (
		block, prev *types.Block
	)
	// 逐个检查区块的区块号是否连续以及hash链是否连续
	for i := 1; i < len(chain); i++ {
		block = chain[i]
		prev = chain[i-1]
		if block.NumberU64() != prev.NumberU64()+1 || block.ParentHash() != prev.Hash() {
			// Chain broke ancestry, log a message (programming error) and skip insertion
			log.Error("Non contiguous block insert", "number", block.Number(), "hash", block.Hash(),
				"parent", block.ParentHash(), "prevnumber", prev.Number(), "prevhash", prev.Hash())

			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, prev.NumberU64(),
				prev.Hash().Bytes()[:4], i, block.NumberU64(), block.Hash().Bytes()[:4], block.ParentHash().Bytes()[:4])
		}
	}
	// Pre-checks passed, start the full block imports
	bc.wg.Add(1)
	bc.chainmu.Lock()
	n, err := bc.insertChain(chain, true)
	bc.chainmu.Unlock()
	bc.wg.Done()

	return n, err
}
```
等所有检验通过以后，使用go语言的waitGroup.Add(1)来增加一个需要等待的goroutine，waitGroup.Done()来减1。在尚未Done()之前，具有waitGroup.wait()的函数就会停止，等待某处的waitGroup.Done()执行完才能执行。比如waitGroup.wait()在blockchain.Stop()函数里，意味着如果在插入区块的时候，突然有人执行Stop()函数，那么必须要等insertChain()执行完。

验证区块头

* headers切片
* seals切片（headers切片里的每个索引的header是否需要检查）
* abort通道和results通道（前者用来传递退出命令，后者用来传递检查结果） 

```go
// insertChain将执行实际的链插入和事件聚合。
func (bc *BlockChain) insertChain(chain types.Blocks, verifySeals bool) (int, error) {
	......
	// 准备数据
	headers := make([]*types.Header, len(chain))
	seals := make([]bool, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
        // headers切片里的每个索引的header是否需要检查
		seals[i] = verifySeals
	}
```
验证区块内容，header和body，bc.engine.VerifyHeaders和ValidateBody。
```go
	// 验证区块头，所有结果返回到results通道里
	abort, results := bc.engine.VerifyHeaders(bc, headers, seals)
	defer close(abort)
	// Peek the error for the first block to decide the directing import logic
	it := newInsertIterator(chain, results, bc.validator)
	//it.next()执行ValidateBody
	block, err := it.next()
```
处理验证过程中出现的err

当插入的块已存在于数据库当中时，如果待插入块往上的总难度比现在规范链的要小的话，直接忽略该块；否则调用bc.writeKnownBlock(block)，内部判断待插入块的parenthash与current hash是否相同以决定是否调用reorgs。

```go
	// Left-trim all the known blocks
	if err == ErrKnownBlock {
		var (
			current  = bc.CurrentBlock()
			localTd  = bc.GetTd(current.Hash(), current.NumberU64())
			externTd = bc.GetTd(block.ParentHash(), block.NumberU64()-1) // The first block can't be nil
		)
		for block != nil && err == ErrKnownBlock {
			externTd = new(big.Int).Add(externTd, block.Difficulty())
			if localTd.Cmp(externTd) < 0 {
				break
			}
			log.Debug("Ignoring already known block", "number", block.Number(), "hash", block.Hash())
			stats.ignored++

			block, err = it.next()
		}
		for block != nil && err == ErrKnownBlock {
			log.Debug("Writing previously known block", "number", block.Number(), "hash", block.Hash())
			if err := bc.writeKnownBlock(block); err != nil {
				return it.index, err
			}
			lastCanon = block

			block, err = it.next()
		}
		// Falls through to the block import
	}

```
* 当验证一个块需要一个已知的、但状态不可用的祖先时，返回ErrPrunedAncestor错误。然后调用bc.insertSideChain(block, it)
*  当验证一个块需要一个未知的祖先时，将返回ErrUnknownAncestor错误。当待插入块时间戳大于该节点时间戳是，返回ErrFutureBlock错误。该case表明待插入区块的第一个区块的父区块可能都还没在链中，因此将区块添加进FutureBlocks中
* 当出现错误不是上述这些的话，移除区块并记录错误

```go
switch {
	// 当验证一个块需要一个已知的、但状态不可用的祖先时，返回ErrPrunedAncestor错误
	case err == consensus.ErrPrunedAncestor:
		log.Debug("Pruned ancestor, inserting as sidechain", "number", block.Number(), "hash", block.Hash())
		return bc.insertSideChain(block, it)

	// First block is future, shove it (and all children) to the future queue (unknown ancestor)
	case err == consensus.ErrFutureBlock || (err == consensus.ErrUnknownAncestor && bc.futureBlocks.Contains(it.first().ParentHash())):
		for block != nil && (it.index == 0 || err == consensus.ErrUnknownAncestor) {
			log.Debug("Future block, postponing import", "number", block.Number(), "hash", block.Hash())
			if err := bc.addFutureBlock(block); err != nil {
				return it.index, err
			}
			block, err = it.next()
		}
		stats.queued += it.processed()
		stats.ignored += it.remaining()

		// If there are any still remaining, mark as ignored
		return it.index, err

	// Some other error occurred, abort
	case err != nil:
		bc.futureBlocks.Remove(block.Hash())
		stats.ignored += len(it.chain)
		bc.reportBlock(block, nil, err)
		return it.index, err
	}
```
 对待插入区块的交易状态进行验证 

>  // 执行区块中的交易拿到receipt
> receipts, logs, usedGas, err := bc.processor.Process(block, statedb, bc.vmConfig)

> // 使用默认的验证器验证状态
> bc.validator.ValidateState(block, statedb, receipts, usedGas)

```go
// No validation errors for the first block (or chain prefix skipped)
for ; block != nil && err == nil || err == ErrKnownBlock; block, err = it.next() {
		// 如果链终止，则停止处理块
		if atomic.LoadInt32(&bc.procInterrupt) == 1 {
			log.Debug("Premature abort during blocks processing")
			break
		}
		// 如果是badhash，直接终止
		if BadHashes[block.Hash()] {
			bc.reportBlock(block, nil, ErrBlacklistedHash)
			return it.index, ErrBlacklistedHash
		}
		// If the block is known (in the middle of the chain), it's a special case for
		// Clique blocks where they can share state among each other, so importing an
		// older block might complete the state of the subsequent one. In this case,
		// just skip the block (we already validated it once fully (and crashed), since
		// its header and body was already in the database).
		if err == ErrKnownBlock {
			logger := log.Debug
			if bc.chainConfig.Clique == nil {
				logger = log.Warn
			}
			logger("Inserted known block", "number", block.Number(), "hash", block.Hash(),
				"uncles", len(block.Uncles()), "txs", len(block.Transactions()), "gas", block.GasUsed(),
				"root", block.Root())

			if err := bc.writeKnownBlock(block); err != nil {
				return it.index, err
			}
			stats.processed++

			// We can assume that logs are empty here, since the only way for consecutive
			// Clique blocks to have the same state is if there are no transactions.
			lastCanon = block
			continue
		}
		// 检索父块及其在其上执行的状态
		start := time.Now()

		parent := it.previous()
		if parent == nil {
			parent = bc.GetHeader(block.ParentHash(), block.NumberU64()-1)
		}
    	// 将这个block的父亲的状态树从数据库中读取出来，并实例化成StateDB
		statedb, err := state.New(parent.Root, bc.stateCache)
		if err != nil {
			return it.index, err
		}
		// 如果我们有一个后续块，那么在当前状态下运行它，
    	// 以便预先缓存交易和一些account/storage trie节点。
		var followupInterrupt uint32
        //TrieCleanNoPrefetch:是否为后续块禁用启发式状态预取
		if !bc.cacheConfig.TrieCleanNoPrefetch {
            //it.peek()返回下一个块，但index不加1
			if followup, err := it.peek(); followup != nil && err == nil {
				throwaway, _ := state.New(parent.Root, bc.stateCache)
				go func(start time.Time, followup *types.Block, throwaway *state.StateDB, interrupt *uint32) {
                      // 预先缓存交易签名和状态trie节点。
					bc.prefetcher.Prefetch(followup, throwaway, bc.vmConfig, interrupt)

					blockPrefetchExecuteTimer.Update(time.Since(start))
					if atomic.LoadUint32(interrupt) == 1 {
						blockPrefetchInterruptMeter.Mark(1)
					}
				}(time.Now(), followup, throwaway, &followupInterrupt)
			}
		}
		// Process block using the parent state as reference point
		substart := time.Now()
    	// 执行区块中的交易拿到receipt
		receipts, logs, usedGas, err := bc.processor.Process(block, statedb, bc.vmConfig)
		if err != nil {
			bc.reportBlock(block, receipts, err)
			atomic.StoreUint32(&followupInterrupt, 1)
			return it.index, err
		}
		// Update the metrics touched during block processing
		accountReadTimer.Update(statedb.AccountReads)     // Account reads are complete, we can mark them
		storageReadTimer.Update(statedb.StorageReads)     // Storage reads are complete, we can mark them
		accountUpdateTimer.Update(statedb.AccountUpdates) // Account updates are complete, we can mark them
		storageUpdateTimer.Update(statedb.StorageUpdates) // Storage updates are complete, we can mark them

		triehash := statedb.AccountHashes + statedb.StorageHashes // Save to not double count in validation
		trieproc := statedb.AccountReads + statedb.AccountUpdates
		trieproc += statedb.StorageReads + statedb.StorageUpdates

		blockExecutionTimer.Update(time.Since(substart) - trieproc - triehash)

		// 使用默认的验证器验证状态
		substart = time.Now()
		if err := bc.validator.ValidateState(block, statedb, receipts, usedGas); err != nil {
			bc.reportBlock(block, receipts, err)
			atomic.StoreUint32(&followupInterrupt, 1)
			return it.index, err
		}
		proctime := time.Since(start)

		// Update the metrics touched during block validation
		accountHashTimer.Update(statedb.AccountHashes) // Account hashes are complete, we can mark them
		storageHashTimer.Update(statedb.StorageHashes) // Storage hashes are complete, we can mark them

		blockValidationTimer.Update(time.Since(substart) - (statedb.AccountHashes + statedb.StorageHashes - triehash))

		// 将块写入链并获得状态。
		substart = time.Now()
		status, err := bc.writeBlockWithState(block, receipts, logs, statedb, false)
		if err != nil {
			atomic.StoreUint32(&followupInterrupt, 1)
			return it.index, err
		}
		atomic.StoreUint32(&followupInterrupt, 1)

		// Update the metrics touched during block commit
		accountCommitTimer.Update(statedb.AccountCommits) // Account commits are complete, we can mark them
		storageCommitTimer.Update(statedb.StorageCommits) // Storage commits are complete, we can mark them

		blockWriteTimer.Update(time.Since(substart) - statedb.AccountCommits - statedb.StorageCommits)
		blockInsertTimer.UpdateSince(start)

		switch status {
		case CanonStatTy:
			log.Debug("Inserted new block", "number", block.Number(), "hash", block.Hash(),
				"uncles", len(block.Uncles()), "txs", len(block.Transactions()), "gas", block.GasUsed(),
				"elapsed", common.PrettyDuration(time.Since(start)),
				"root", block.Root())

			lastCanon = block

			// Only count canonical blocks for GC processing time
			bc.gcproc += proctime

		case SideStatTy:
			log.Debug("Inserted forked block", "number", block.Number(), "hash", block.Hash(),
				"diff", block.Difficulty(), "elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()),
				"root", block.Root())

		default:
			// This in theory is impossible, but lets be nice to our future selves and leave
			// a log, instead of trying to track down blocks imports that don't emit logs.
			log.Warn("Inserted block with unknown status", "number", block.Number(), "hash", block.Hash(),
				"diff", block.Difficulty(), "elapsed", common.PrettyDuration(time.Since(start)),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()),
				"root", block.Root())
		}
		stats.processed++
		stats.usedGas += usedGas

		dirty, _ := bc.stateCache.TrieDB().Size()
		stats.report(chain, it.index, dirty)
	}
```
最后再检查一下是否有块剩下了，只关心future block。
```go
	// Any blocks remaining here? The only ones we care about are the future ones
	if block != nil && err == consensus.ErrFutureBlock {
		if err := bc.addFutureBlock(block); err != nil {
			return it.index, err
		}
		block, err = it.next()

		for ; block != nil && err == consensus.ErrUnknownAncestor; block, err = it.next() {
			if err := bc.addFutureBlock(block); err != nil {
				return it.index, err
			}
			stats.queued++
		}
	}
	stats.ignored += it.remaining()

	return it.index, err
```



