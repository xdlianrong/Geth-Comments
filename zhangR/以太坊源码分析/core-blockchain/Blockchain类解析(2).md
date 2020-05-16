```go
// addFutureBlock检查该块是否在允许接受的最大处理窗口内，如果该块超前太远而没有被添加，则返回一个错误。
func (bc *BlockChain) addFutureBlock(block *types.Block) error {}

// BadBlocks 处理客户端从网络上获取的最近的bad block列表
func (bc *BlockChain) BadBlocks() []*types.Block {}
 
// addBadBlock 把bad block放入缓存
func (bc *BlockChain) addBadBlock(block *types.Block) {}
 
// CurrentBlock取回主链的当前头区块，这个区块是从blockchian的内部缓存中取得
func (bc *BlockChain) CurrentBlock() *types.Block {}
 
// CurrentHeader检索规范链的当前头区块header。从HeaderChain的内部缓存中检索标头。
func (bc *BlockChain) CurrentHeader() *types.Header{}
 
// CurrentFastBlock取回主链的当前fast-sync头区块，这个区块是从blockchian的内部缓存中取得
func (bc *BlockChain) CurrentFastBlock() *types.Block {}
 
// 将活动链或其子集写入给定的编写器.
func (bc *BlockChain) Export(w io.Writer) error {}
func (bc *BlockChain) ExportN(w io.Writer, first uint64, last uint64) error {}
 
// FastSyncCommitHead快速同步，将当前头块设置为特定hash的区块。
func (bc *BlockChain) FastSyncCommitHead(hash common.Hash) error {}
 
// GasLimit返回当前头区块的gas limit
func (bc *BlockChain) GasLimit() uint64 {}
 
// Genesis 取回genesis区块
func (bc *BlockChain) Genesis() *types.Block {}
 
// 获取给定块的第n个祖先。它假设给定的块或它的近祖先是标准的。
// maxNonCanonical指向一个向下的计数器，它限制在到达标准链之前要单独检查的块的数量。
// 注意:ancestor == 0返回相同的块，1返回其父块，依此类推。
func (bc *BlockChain) GetAncestor(hash common.Hash, number, ancestor uint64, maxNonCanonical *uint64) (common.Hash, uint64) {}


// 通过hash从数据库或缓存中取到一个区块体(transactions and uncles)或RLP数据
func (bc *BlockChain) GetBody(hash common.Hash) *types.Body {}
func (bc *BlockChain) GetBodyRLP(hash common.Hash) rlp.RawValue {}
 
// GetBlock 通过hash和number取到区块
func (bc *BlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {}
// GetBlockByHash 通过hash取到区块
func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {}
// GetBlockByNumber 通过number取到区块
func (bc *BlockChain) GetBlockByNumber(number uint64) *types.Block {}

// GetBlockHashesFromHash检索从给定哈希开始的许多块哈希，并获取到genesis块。
func (bc *BlockChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {}
 
// GetCanonicalHash返回给定块号的规范hash
func (bc *BlockChain) GetCanonicalHash(number uint64) common.Hash {}

// 获取给定hash的区块的总难度
func (bc *BlockChain) GetTd(hash common.Hash, number uint64) *big.Int{}
 
// GetTdByHash通过哈希从数据库中检索规范链中的块的总难度，如果找到则缓存它。
func (bc *BlockChain) GetTdByHash(hash common.Hash) *big.Int {}

// 获取给定hash和number区块的header
func (bc *BlockChain) GetHeader(hash common.Hash, number uint64) *types.Header{}
 
// 获取给定hash的区块header
func (bc *BlockChain) GetHeaderByHash(hash common.Hash) *types.Header{}
 
// 获取给定number的区块header
func (bc *BlockChain) GetHeaderByNumber(number uint64) *types.Header{}
 
// 获取从给定hash的区块到genesis区块的所有hash
func (bc *BlockChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash{}
 
// GetReceiptsByHash 在特定的区块中取到所有交易的收据
func (bc *BlockChain) GetReceiptsByHash(hash common.Hash) types.Receipts {}
 
// GetBlocksFromHash 取到特定hash的区块及其n-1个父区块
func (bc *BlockChain) GetBlocksFromHash(hash common.Hash, n int) (blocks []*types.Block) {}

// GetTransactionLookup从缓存或数据库中检索与给定交易hash相关联的查找。
func (bc *BlockChain) GetTransactionLookup(hash common.Hash) *rawdb.LegacyTxLookupEntry {}

// GetUnclesInChain 取回从给定区块到向前回溯特定距离到区块上的所有叔区块
func (bc *BlockChain) GetUnclesInChain(block *types.Block, length int) []*types.Header {}
 
// HasBlock检验hash对应的区块是否完全存在数据库中
func (bc *BlockChain) HasBlock(hash common.Hash, number uint64) bool {}
 
// 检查给定hash和number的区块的区块头是否存在数据库
func (bc *BlockChain) HasHeader(hash common.Hash, number uint64) bool{}
 
// HasState检验state trie是否完全存在数据库中
func (bc *BlockChain) HasState(hash common.Hash) bool {}
 
// HasBlockAndState检验hash对应的block和state trie是否完全存在数据库中
func (bc *BlockChain) HasBlockAndState(hash common.Hash, number uint64) bool {}
 
// insert 将新的头块注入当前块链。 该方法假设该块确实是真正的头。
// 如果它们较旧或者它们位于不同的侧链上，它还会将头部标题和头部快速同步块重置为同一个块。
func (bc *BlockChain) insert(block *types.Block) {}
 
// InsertChain尝试将给定批量的block插入到规范链中，否则，创建一个分叉。 如果返回错误，它将返回失败块的索引号以及描述错误的错误。
//插入完成后，将触发所有累积的事件。
func (bc *BlockChain) InsertChain(chain types.Blocks) (int, error){}
 
// insertChain将执行实际的链插入和事件聚合。 
// 此方法作为单独方法存在的唯一原因是使用延迟语句使锁定更清晰。
func (bc *BlockChain) insertChain(chain types.Blocks) (int, []interface{}, []*types.Log, error){}
 
// InsertHeaderChain尝试将给定的头链插入到本地链中，可能会创建reorg。
// 如果返回一个错误，它将返回失败消息头的索引号以及描述出错原因的错误。
// verify参数可用于微调是否应该进行nonce验证。
// 可选检查背后的原因是，一些标头检索机制已经需要验证nonces，以及可以稀疏地验证nonces，而不需要逐个检查。
func (bc *BlockChain) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error){}
 
// InsertReceiptChain 使用交易和收据数据来完成已经存在的headerchain
func (bc *BlockChain) InsertReceiptChain(blockChain types.Blocks, receiptChain []types.Receipts) (int, error) {}

// 当导入批处理遇到修剪后的祖先错误时调用insertSideChain，当找到具有足够旧的fork块的sidechain时发生这种错误。
// 该方法将所有(header-and-body-valid)块写到磁盘，然后如果TD超过当前链，则尝试切换到新链。
func (bc *BlockChain) insertSideChain(block *types.Block, it *insertIterator) (int, error) {}
 
//loadLastState从数据库加载最后一个已知的链状态。
func (bc *BlockChain) loadLastState() error {}
 
// Processor 返回当前current processor.
func (bc *BlockChain) Processor() Processor {}
 
// Reset重置清除整个区块链，将其恢复到genesis state.
func (bc *BlockChain) Reset() error {}
 
// ResetWithGenesisBlock 清除整个区块链, 用特定的genesis state重塑，被Reset所引用
func (bc *BlockChain) ResetWithGenesisBlock(genesis *types.Block) error {}
 
// repair尝试通过回滚当前块来修复当前的区块链，直到找到具有关联状态的块。
// 用于修复由崩溃/断电或简单的非提交尝试导致的不完整的数据库写入。
// 此方法仅回滚当前块。 当前标头和当前快速块保持不变。
func (bc *BlockChain) repair(head **types.Block) error {}

// reportBlock记录一个严重的块错误。
func (bc *BlockChain) reportBlock(block *types.Block, receipts types.Receipts, err error) {}

// reorgs需要两个块、一个旧链以及一个新链，并将重新构建块然后将它们插入到新的规范链中，并累积潜在的缺失交易并发布有关它们的事件
func (bc *BlockChain) reorg(oldBlock, newBlock *types.Block) error{}
 
// Rollback 旨在从数据库中删除不确定有效的链片段
func (bc *BlockChain) Rollback(chain []common.Hash) {}
 
// SetReceiptsData 计算收据的所有非共识字段
func SetReceiptsData(config *params.ChainConfig, block *types.Block, receipts types.Receipts) error {}
 
// SetHead将本地链回滚到指定的头部。
// 通常可用于处理分叉时重选主链。对于Header，新Header上方的所有内容都将被删除，新的头部将被设置。
// 但如果块体丢失，则会进一步回退（快速同步后的非归档节点）。
func (bc *BlockChain) SetHead(head uint64) error {}
 
// SetProcessor设置状态修改所需要的processor
func (bc *BlockChain) SetProcessor(processor Processor) {}
 
// SetValidator 设置用于验证未来区块的validator
func (bc *BlockChain) SetValidator(validator Validator) {}
 
// State 根据当前头区块返回一个可修改的状态
func (bc *BlockChain) State() (*state.StateDB, error) {}
 
// StateAt 根据特定时间点返回新的可变状态
func (bc *BlockChain) StateAt(root common.Hash) (*state.StateDB, error) {}

// StateCache返回支持区块链实例的缓存数据库。
func (bc *BlockChain) StateCache() state.Database {}
 
// Stop 停止区块链服务，如果有正在import的进程，它会使用procInterrupt来取消。
// it will abort them using the procInterrupt.
func (bc *BlockChain) Stop() {}
 
// TrieNode从memory缓存或storage中检索与trie节点hash相关联的数据。
func (bc *BlockChain) TrieNode(hash common.Hash) ([]byte, error) {}
 
// truncateAncient将区块链回滚到指定的标头，并删除ancient store中超过指定标头的所有数据。
// Purge 用于完全清除缓存
func (bc *BlockChain) truncateAncient(head uint64) error {}

// Validator返回当前validator.
func (bc *BlockChain) Validator() Validator {}
 
// WriteBlockWithoutState仅将块及其元数据写入数据库，但不写入任何状态。 这用于构建竞争方叉，直到超过规范总难度。
func (bc *BlockChain) WriteBlockWithoutState(block *types.Block, td *big.Int) (err error){}
 
// WriteBlockWithState将块和所有关联状态写入数据库。
func (bc *BlockChain) WriteBlockWithState(block *types.Block, receipts []*types.Receipt, state *state.StateDB) {}
 
// writeHeader将标头写入本地链，因为它的父节点已知。 如果新插入的报头的总难度变得大于当前已知的TD，则重新路由规范链
func (bc *BlockChain) writeHeader(header *types.Header) error{}

// writeHeadBlock向当前块链注入一个新的头块。
// 此方法假设该块确实是一个真头。如果它们是旧的，或者它们在不同的侧链上，
// 它还会将head header和head fast sync块重置为相同的块。
// 注意，这个函数假设持有' mu '互斥锁!
func (bc *BlockChain) writeHeadBlock(block *types.Block) {}
 
// writeKnownBlock用一个已知的块更新头块标志，并在必要时引入chain reorg。
func (bc *BlockChain) writeKnownBlock(block *types.Block) error {}

// 处理未来区块链
func (bc *BlockChain) update() {}
```

