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
	processor  Processor  // 块事务处理接口
	vmConfig   vm.Config  // //虚拟机的配置

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

# 二、 **blockchain.go中的部分函数和方法** 

**1、NewBlockChain**，使用数据库里面的可用信息构造了一个初始化好的区块链. 同时初始化了以太坊默认的 验证器和处理器，预取器等。

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
	// 检查块哈希的当前状态，并确保链中没有任何坏块
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

**2、loadLastState**, 加载数据库里面的最新的我们知道的区块链状态. 就是要找到最新的区块头，然后设置currentBlock、currentHeader和currentFastBlock 

1) 	获取到最新区块以及它的hash

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
	// 为用户发出状态日志
	currentFastBlock := bc.CurrentFastBlock()

	headerTd := bc.GetTd(currentHeader.Hash(), currentHeader.Number.Uint64())
	blockTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
	fastTd := bc.GetTd(currentFastBlock.Hash(), currentFastBlock.NumberU64())

	log.Info("Loaded most recent local header", "number", currentHeader.Number, "hash", currentHeader.Hash(), "td", headerTd, "age", common.PrettyAge(time.Unix(int64(currentHeader.Time), 0)))
	log.Info("Loaded most recent local full block", "number", currentBlock.Number(), "hash", currentBlock.Hash(), "td", blockTd, "age", common.PrettyAge(time.Unix(int64(currentBlock.Time()), 0)))
	log.Info("Loaded most recent local fast block", "number", currentFastBlock.Number(), "hash", currentFastBlock.Hash(), "td", fastTd, "age", common.PrettyAge(time.Unix(int64(currentFastBlock.Time()), 0)))

	return nil
}
```

