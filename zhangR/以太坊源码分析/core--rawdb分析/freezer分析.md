# freezer分析

freezer也是持久化的存储。目前来说，我个人的理解是，freezer是把一些旧区块的数据，和最近新建的区块分开来放置。可以观察一下我们搭建私链的文件目录 geth/chaindb 的内容。

当我们第一次初始化私链时，也就是执行 **init genesis.json**后，chaindb下是这样的：

![1586069311092](images\1586069311092.png)

而当我们挖完矿，区块高度变高，然后关闭geth后，chaindb下是这样的：

![1586069482980](images\1586069482980.png)

可以看到的是，除了本身产生了很多新的ldb文件外，还产生了一个ancient文件夹，里面是这样的：

![1586069623288](images\1586069623288.png)

这里包含的信息就是旧块的信息了，主要是**hash header body receipt diffs**五个部分和FLOCK锁。

**PS**：这里的文件后缀是代码生成的，由**r/c(raw和compressed)**和**idx/dat(index和data)**组成。

先来看几个数据结构

```go
// freezer.go 68行
type freezer struct {
	frozen uint64 // Number of blocks already frozen
	tables       map[string]*freezerTable // Data tables for storing everything
	instanceLock fileutil.Releaser        // File-system lock to prevent double opens
}

// freezer_Table.go 78行
type freezerTable struct {
	// 表中存储的项目数（包括从尾部删除的项目）
	items uint64 
    // 如果为true，则禁用快速压缩。
	noCompression bool   
	maxFileSize   uint32 // Max file size for data-files
	name          string
	path          string
	// 表数据头的文件描述符
	head   *os.File            // File descriptor for the data head of the table
	files  map[uint32]*os.File // open files
    // 当前活动头文件的编号
	headId uint32              // number of the currently active head file
    // 最早的文件号
	tailId uint32              // number of the earliest file
    // indexEntry文件的文件描述符
	index  *os.File            // File descriptor for the indexEntry file of the table

	// In the case that old items are deleted (from the tail), we use itemOffset
	// to count how many historic items have gone missing.
	itemOffset uint32 // Offset (number of discarded items)丢弃件数

	headBytes  uint32        // Number of bytes written to the head file
	readMeter  metrics.Meter // Meter for measuring the effective amount of data read
	writeMeter metrics.Meter // Meter for measuring the effective amount of data written
	sizeGauge  metrics.Gauge // Gauge for tracking the combined size of all freezer tables

	logger log.Logger   // Logger with database path and table name ambedded
	lock   sync.RWMutex // Mutex protecting the data file descriptors
}

// schema.go 93行
// freezerNoSnappy配置是否对旧表禁用压缩，哈希和难度不能压缩。
var freezerNoSnappy = map[string]bool{   
    freezerHeaderTable:     false,   
    freezerHashTable:       true,   
    freezerBodiesTable:     false,   
    freezerReceiptTable:    false,   
    freezerDifficultyTable: true,
}
```

## freezer.go

newFreezer方法通过给定目录将table文件读入内存中，构成freezer对象。

```go
func newFreezer(datadir string, namespace string) (*freezer, error) {
	// Create the initial freezer object
	// metrics启用统计信息
	var (
		readMeter  = metrics.NewRegisteredMeter(namespace+"ancient/read", nil)
		writeMeter = metrics.NewRegisteredMeter(namespace+"ancient/write", nil)
		sizeGauge  = metrics.NewRegisteredGauge(namespace+"ancient/size", nil)
	)
	// Ensure the datadir is not a symbolic link if it exists.
	// 确保目录不是软链接
	if info, err := os.Lstat(datadir); !os.IsNotExist(err) {
		if info.Mode()&os.ModeSymlink != 0 {
			log.Warn("Symbolic link ancient database is not supported", "path", datadir)
			return nil, errSymlinkDatadir
		}
	}
	// Leveldb uses LOCK as the filelock filename. To prevent the
	// name collision, we use FLOCK as the lock name.
	// 避免名字冲突，采用FLOCK作为文件锁名
	lock, _, err := fileutil.Flock(filepath.Join(datadir, "FLOCK"))
	if err != nil {
		return nil, err
	}
	// Open all the supported data tables
	freezer := &freezer{
		tables:       make(map[string]*freezerTable),
		instanceLock: lock,
	}
	// freezerNoSnappy包含 hash header body receipt difficulty五部分
	for name, disableSnappy := range freezerNoSnappy {
        // newTable方法在freezer_table.go文件中。
		table, err := newTable(datadir, name, readMeter, writeMeter, sizeGauge, disableSnappy)
		if err != nil {
			for _, table := range freezer.tables {
				table.Close()
			}
			lock.Release()
			return nil, err
		}
		freezer.tables[name] = table
	}
    // repair对读入的table进行检查和修复，
    // 确保不会存在比如headers里面有5个块的信息，而bodies里面有6个这种情况。
	if err := freezer.repair(); err != nil {
		for _, table := range freezer.tables {
			table.Close()
		}
		lock.Release()
		return nil, err
	}
	log.Info("Opened ancient database", "database", datadir)
	return freezer, nil
}
```

然后是对freezer这个对象的一些方法，目前来看的结果是这些方法内部再调用freezer_table的方法，而freezer_table里面的方法又是调用的table的方法，再往前则是回到了ethdb里面的方法。

```go
Close()
// 判断number号区块的kind是否存在
HasAncient(kind string, number uint64)
// 返回number号区块的kind信息
Ancient(kind string, number uint64)
// 返回此时ancient里面所有items的长度
Ancients()
// 返回某个信息的长度
AncientSize(kind string)
// 往freezer对象的tables中添加信息（此处要求5种信息同时添加，一旦一个出错直接返回err）
AppendAncient(number uint64, hash, header, body, receipts, td []byte)
// 根据一个确定长度截断每个信息
TruncateAncients(items uint64)
// sync将所有数据表刷新到磁盘。
Sync()
// 将一个db对象冷藏化
freeze(db ethdb.KeyValueStore)
// 修复tables
repair()
```

这里需要单独说一下 freeze 方法，这个方法在节点启动后会新起一个线程来监听链的变化进展，当达到阙值时则将开始将区块冷藏化。

```go
// freezerRecheckInterval是检查链进展情况的频率
freezerRecheckInterval = time.Minute
// freezerBatchLimit是在写入磁盘并将其从键值存储中删除之前，能在一次批量操作中冻结的最大块数。
freezerBatchLimit = 30000
// ImmutabilityThreshold是将链段视为不可变的块数（即软性终结）。
// 在冷冻机中作为截止阈值
ImmutabilityThreshold = 90000
// 此时冷藏库中已冷藏的区块数，从90000往后计算
f.frozen
```

```go
func (f *freezer) freeze(db ethdb.KeyValueStore) {
	nfdb := &nofreezedb{KeyValueStore: db}

	for {
		// 检索冻结阈值。
         // hash不得为空
		hash := ReadHeadBlockHash(nfdb)
		if hash == (common.Hash{}) {
			log.Debug("Current full block hash unavailable") // new chain, empty database
			time.Sleep(freezerRecheckInterval)
			continue
		}
        // number要大于 90000+f.frozen
		number := ReadHeaderNumber(nfdb, hash)
		switch {
		case number == nil:
			log.Error("Current full block number unavailable", "hash", hash)
			time.Sleep(freezerRecheckInterval)
			continue

		case *number < params.ImmutabilityThreshold:
			log.Debug("Current full block not old enough", "number", *number, "hash", hash, "delay", params.ImmutabilityThreshold)
			time.Sleep(freezerRecheckInterval)
			continue

		case *number-params.ImmutabilityThreshold <= f.frozen:
			log.Debug("Ancient blocks frozen already", "number", *number, "hash", hash, "frozen", f.frozen)
			time.Sleep(freezerRecheckInterval)
			continue
		}
        // head
		head := ReadHeader(nfdb, hash, *number)
		if head == nil {
			log.Error("Current full block unavailable", "number", *number, "hash", hash)
			time.Sleep(freezerRecheckInterval)
			continue
		}
		// 已经准备好冻结数据，可以分批处理
         // 一次最多冷藏30000个块。 
		limit := *number - params.ImmutabilityThreshold
		if limit-f.frozen > freezerBatchLimit {
			limit = f.frozen + freezerBatchLimit
		}
		var (
			start    = time.Now()
			first    = f.frozen
			ancients = make([]common.Hash, 0, limit)
		)
		for f.frozen < limit {
			// 检索规范块的所有组件
			hash := ReadCanonicalHash(nfdb, f.frozen)
			if hash == (common.Hash{}) {
				log.Error("Canonical hash missing, can't freeze", "number", f.frozen)
				break
			}
			header := ReadHeaderRLP(nfdb, hash, f.frozen)
			if len(header) == 0 {
				log.Error("Block header missing, can't freeze", "number", f.frozen, "hash", hash)
				break
			}
			body := ReadBodyRLP(nfdb, hash, f.frozen)
			if len(body) == 0 {
				log.Error("Block body missing, can't freeze", "number", f.frozen, "hash", hash)
				break
			}
			receipts := ReadReceiptsRLP(nfdb, hash, f.frozen)
			if len(receipts) == 0 {
				log.Error("Block receipts missing, can't freeze", "number", f.frozen, "hash", hash)
				break
			}
			td := ReadTdRLP(nfdb, hash, f.frozen)
			if len(td) == 0 {
				log.Error("Total difficulty missing, can't freeze", "number", f.frozen, "hash", hash)
				break
			}
			log.Trace("Deep froze ancient block", "number", f.frozen, "hash", hash)
			// 将所有组件注入相关数据表
			if err := f.AppendAncient(f.frozen, hash[:], header, body, receipts, td); err != nil {
				break
			}
			ancients = append(ancients, hash)
		}
		// Batch of blocks have been frozen, flush them before wiping from leveldb
		if err := f.Sync(); err != nil {
			log.Crit("Failed to flush frozen tables", "err", err)
		}
		// 清除活动数据库中的所有数据
		batch := db.NewBatch()
		for i := 0; i < len(ancients); i++ {
			// Always keep the genesis block in active database
			if first+uint64(i) != 0 {
				DeleteBlockWithoutNumber(batch, ancients[i], first+uint64(i))
				DeleteCanonicalHash(batch, first+uint64(i))
			}
		}
		if err := batch.Write(); err != nil {
			log.Crit("Failed to delete frozen canonical blocks", "err", err)
		}
		batch.Reset()
		// 擦除侧链。
		for number := first; number < f.frozen; number++ {
			// Always keep the genesis block in active database
			if number != 0 {
				for _, hash := range ReadAllHashes(db, number) {
					DeleteBlock(batch, hash, number)
				}
			}
		}
		if err := batch.Write(); err != nil {
			log.Crit("Failed to delete frozen side blocks", "err", err)
		}
		// Log something friendly for the user
		context := []interface{}{
			"blocks", f.frozen - first, "elapsed", common.PrettyDuration(time.Since(start)), "number", f.frozen - 1,
		}
		if n := len(ancients); n > 0 {
			context = append(context, []interface{}{"hash", ancients[n-1]}...)
		}
		log.Info("Deep froze chain segment", context...)

		// 避免因微小的写操作而导致数据库崩溃
		if f.frozen-first < freezerBatchLimit {
			time.Sleep(freezerRecheckInterval)
		}
	}
}
```



