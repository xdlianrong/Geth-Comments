# accessors源码分析

accessors_chain，accessors_indexes，accessors_metadata三个文件都是一些区块链与数据库之间交互的方法，侧重点不一样。accessors_chain：区块头，区块体，难度，收据等；accessors_indexes：交易和bloom；accessors_metadata：版本号，config，批处理原像等等。

首先看一些数据结构，位于core/types/block.go当中

```go
type Header struct {
	ParentHash  common.Hash    `json:"parentHash"       gencodec:"required"`//指向父区块的指针
	UncleHash   common.Hash    `json:"sha3Uncles"       gencodec:"required"`//block中叔块数组的RLP哈希值
	Coinbase    common.Address `json:"miner"            gencodec:"required"`//挖出该区块的人的地址
	Root        common.Hash    `json:"stateRoot"        gencodec:"required"`//StateDB中的stat trie的根节点的RLP哈希值
	TxHash      common.Hash    `json:"transactionsRoot" gencodec:"required"`//tx trie的根节点的哈希值
	ReceiptHash common.Hash    `json:"receiptsRoot"     gencodec:"required"`//receipt trie的根节点的哈希值
	Bloom       Bloom          `json:"logsBloom"        gencodec:"required"`//布隆过滤器，用来判断Log对象是否存在
	Difficulty  *big.Int       `json:"difficulty"       gencodec:"required"`//难度系数
	Number      *big.Int       `json:"number"           gencodec:"required"`//区块序号
	GasLimit    uint64         `json:"gasLimit"         gencodec:"required"`//区块内所有Gas消耗的理论上限
	GasUsed     uint64         `json:"gasUsed"          gencodec:"required"`//区块内消耗的总Gas
	Time        uint64         `json:"timestamp"        gencodec:"required"`//区块应该被创建的时间
	Extra       []byte         `json:"extraData"        gencodec:"required"`
	MixDigest   common.Hash    `json:"mixHash"`
	Nonce       BlockNonce     `json:"nonce"` 
}

type Body struct {
	Transactions []*Transaction
	Uncles       []*Header
}

type Block struct {
	header       *Header
	uncles       []*Header
	transactions Transactions

	// caches
	hash atomic.Value
	size atomic.Value

	// Td is used by package core to store the total difficulty
	// of the chain up to and including the block.
	td *big.Int

	// These fields are used by package eth to track
	// inter-peer block relay.
	ReceivedAt   time.Time
	ReceivedFrom interface{}
}
```

以区块头和区块体的写入为例：

**根据区块头hash将区块高度存储，再根据区块高度和区块头hash将区块头RLP编码后存储**

```go
// WriteHeaderNumber stores the hash->number mapping.
func WriteHeaderNumber(db ethdb.KeyValueWriter, hash common.Hash, number uint64) {
	key := headerNumberKey(hash)//  key="H"+hash
	enc := encodeBlockNumber(number)// 将number转化为大端
	if err := db.Put(key, enc); err != nil {
		log.Crit("Failed to store hash to number mapping", "err", err)
	}
}

// WriteHeader stores a block header into the database and also stores the hash-
// to-number mapping.
func WriteHeader(db ethdb.KeyValueWriter, header *types.Header) {
	var (
		hash   = header.Hash()
		number = header.Number.Uint64()
	)
	// Write the hash -> number mapping
	WriteHeaderNumber(db, hash, number)

	// Write the encoded header
	data, err := rlp.EncodeToBytes(header)// 对区块头进行RLP编码
	if err != nil {
		log.Crit("Failed to RLP encode header", "err", err)
	}
	key := headerKey(number, hash)// key="h"+number+hash
	if err := db.Put(key, data); err != nil {
		log.Crit("Failed to store header", "err", err)
	}
}
```

 **将区块体的RLP编码后存储** 

```go
// WriteBody stores a block body into the database.
func WriteBody(db ethdb.KeyValueWriter, hash common.Hash, number uint64, body *types.Body) {
	data, err := rlp.EncodeToBytes(body)// 对区块体进行RLP编码
	if err != nil {
		log.Crit("Failed to RLP encode body", "err", err)
	}
	WriteBodyRLP(db, hash, number, data)// 存储区块体
}

// WriteBodyRLP stores an RLP encoded block body into the database.
func WriteBodyRLP(db ethdb.KeyValueWriter, hash common.Hash, number uint64, rlp rlp.RawValue) {
	if err := db.Put(blockBodyKey(number, hash), rlp); err != nil {
		log.Crit("Failed to store block body", "err", err)
	}// key="b"+number+hash
}
```

**存储区块是分开存储 区块体和区块头**

```go
// WriteBlock serializes a block into the database, header and body separately.
func WriteBlock(db ethdb.KeyValueWriter, block *types.Block) {
	WriteBody(db, block.Hash(), block.NumberU64(), block.Body())
	WriteHeader(db, block.Header())
}
```

