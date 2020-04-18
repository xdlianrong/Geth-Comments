# database分析

这个文件中主要是数据库的初始化方法。需要注意的是freezerdb这个结构，把一些已经确定好的旧区块的数据，和最近新建的区块分开来放置。ethdb.KeyValueStore是从leveldb中实例化的数据库，ethdb.AncientStore则是冷藏室中的实例化出来的数据库。

```go
type freezerdb struct {
	ethdb.KeyValueStore
	ethdb.AncientStore
}
```

```go
// NewDatabase在给定的键值数据存储之上创建一个高级数据库，而无需使用freezer
func NewDatabase(db ethdb.KeyValueStore) ethdb.Database {
	return &nofreezedb{
		KeyValueStore: db,
	}
}

// 带有freezer，在给定的键值数据存储之上创建一个高级数据库，包含一个正常的键值数据库和一个freezerdb
func NewDatabaseWithFreezer(db ethdb.KeyValueStore, freezer string, namespace string) (ethdb.Database, error) {
    frdb, err := newFreezer(freezer, namespace)
    
    // 中间是对leveldb与冷藏库的处理。
    ......
    // 启动一个线程按一定的频率来冷藏区块
    go frdb.freeze(db)
	return &freezerdb{
		KeyValueStore: db,
		AncientStore:  frdb,
	}, nil
}

// 创建一个内存数据库
func NewMemoryDatabase() ethdb.Database {
	return NewDatabase(memorydb.New())
}

// 创建一个确定大小的内存数据库
func NewMemoryDatabaseWithCap(size int) ethdb.Database {
	return NewDatabase(memorydb.NewWithCap(size))
}

// 实例化一个leveldb数据库，这个方法在通过创世块创建私链的时候会使用到
func NewLevelDBDatabase(file string, cache int, handles int, namespace string) (ethdb.Database, error) {
    db, err := leveldb.New(file, cache, handles, namespace)
	if err != nil {
		return nil, err
	}
	return NewDatabase(db), nil
}


// 在上面的NewLevelDBDatabase的基础上添加了一个freezerdb
func NewLevelDBDatabaseWithFreezer(file string, cache int, handles int, freezer string, namespace string) (ethdb.Database, error) {
    kvdb, err := leveldb.New(file, cache, handles, namespace)
	if err != nil {
		return nil, err
	}
	frdb, err := NewDatabaseWithFreezer(kvdb, freezer, namespace)
	if err != nil {
		kvdb.Close()
		return nil, err
	}
	return frdb, nil
}

// InspectDatabase遍历整个数据库并检查所有不同类别数据的大小。
func InspectDatabase(db ethdb.Database) error {
}
```

