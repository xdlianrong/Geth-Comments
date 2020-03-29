# ethdb源码分析（2）

## memorydb.go

首先来看这里的Database结构，基本上就是封装了一个内存的Map结构。然后使用了一把锁来对多线程进行资源保护。

```go
type Database struct {   
    db   map[string][]byte   
    lock sync.RWMutex
}
```

然后是new一个database出来，两种方式New和NewWithCap，后者是规定大小的创建。

```go
//New
db: make(map[string][]byte)
//NewWithCap
db: make(map[string][]byte, size)
```

接着是对于database这个对象的一些方法，与leveldb的基本一致，只不过这里是对内存中的这个存储进行修改。包括Close，Has，Get，Put，Delete，NewBatch，NewIterator，NewIteratorWithStart，NewIteratorWithPrefix，Stat，Compact（对于内存存储来说，stat和Compact显然不支持的，所以这里的代码也只是简单的返回空字符串和nil），Len。基本功能在leveldb.go中已经提到，leveldb中时直接封装的goleveldb的方法调用，这里的则是有具体的代码实现。以NewIteratorWithPrefix来举例。

```go
func (db *Database) NewIteratorWithPrefix(prefix []byte) ethdb.Iterator {
	db.lock.RLock()
	defer db.lock.RUnlock()

	var (
		pr     = string(prefix)
		keys   = make([]string, 0, len(db.db))
		values = make([][]byte, 0, len(db.db))
	)
	// Collect the keys from the memory database corresponding to the given prefix
	for key := range db.db {
		//HasPrefix检测字符串是否以指定的前缀开头。
		if strings.HasPrefix(key, pr) {
			keys = append(keys, key)
		}
	}
	// Sort the items and retrieve the associated values
	sort.Strings(keys)
	for _, key := range keys {
		values = append(values, db.db[key])
	}
	return &iterator{
		keys:   keys,
		values: values,
	}
}
```

此处先跳过中间的batch代码，先从296行开始看迭代器的代码

```go
// 迭代器可以遍历内存键值存储的（可能是部分）键空间。 在内部，它是整个迭代状态的深层副本，按键排序。
type iterator struct {
	inited bool
	keys   []string
	values [][]byte
}

// Next moves the iterator to the next key/value pair. It returns whether the
// iterator is exhausted.
func (it *iterator) Next() bool {
	// If the iterator was not yet initialized, do it now
	if !it.inited {
		it.inited = true
		return len(it.keys) > 0
	}
	// Iterator already initialize, advance it
	if len(it.keys) > 0 {
		it.keys = it.keys[1:]
		it.values = it.values[1:]
	}
	return len(it.keys) > 0
}

func (it *iterator) Error() error {
	return nil
}

func (it *iterator) Key() []byte {
	if len(it.keys) > 0 {
		return []byte(it.keys[0])
	}
	return nil
}

func (it *iterator) Value() []byte {
	if len(it.values) > 0 {
		return it.values[0]
	}
	return nil
}

func (it *iterator) Release() {
	it.keys, it.values = nil, nil
}
```

再来看batch，先规定两个结构体。

```go
type keyvalue struct {
	key    []byte
	value  []byte
	delete bool
}

type batch struct {
	db     *Database
	writes []keyvalue
	size   int
}
```

然后是相关的方法。

```go
// Put inserts the given value into the batch for later committing.
func (b *batch) Put(key, value []byte) error {
	b.writes = append(b.writes, keyvalue{common.CopyBytes(key), common.CopyBytes(value), false})
	b.size += len(value)
	return nil
}

// Delete inserts the a key removal into the batch for later committing.
func (b *batch) Delete(key []byte) error {
	b.writes = append(b.writes, keyvalue{common.CopyBytes(key), nil, true})
	b.size += 1
	return nil
}

// ValueSize retrieves the amount of data queued up for writing.
func (b *batch) ValueSize() int {
	return b.size
}

// Write flushes any accumulated data to the memory database.
// 将所有累积的数据刷新到内存数据库
func (b *batch) Write() error {
	b.db.lock.Lock()
	defer b.db.lock.Unlock()

	for _, keyvalue := range b.writes {
		if keyvalue.delete {
			delete(b.db.db, string(keyvalue.key))
			continue
		}
		b.db.db[string(keyvalue.key)] = keyvalue.value
	}
	return nil
}

// Reset resets the batch for reuse.
func (b *batch) Reset() {
	b.writes = b.writes[:0]
	b.size = 0
}

// Replay replays the batch contents.
//此处暂没看懂，后续修改
func (b *batch) Replay(w ethdb.KeyValueWriter) error {
	for _, keyvalue := range b.writes {
		if keyvalue.delete {
			if err := w.Delete(keyvalue.key); err != nil {
				return err
			}
			continue
		}
		if err := w.Put(keyvalue.key, keyvalue.value); err != nil {
			return err
		}
	}
	return nil
}
```

