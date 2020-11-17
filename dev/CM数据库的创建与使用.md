[TOC]

### 概述

对承诺数据库的初始化及打开，使用。对外暂提供对承诺的写，读，改。

### 在type包中新增commitment.go文件，包含CM字段及相关接口

CM 承诺结构，包含承诺字段以及判断该承诺是否被使用过的spent字段，true表示已使用

```go
type CM struct {
	Cm    uint64
	Spent bool
}
```

**Hash**:对CM中的Cm字段取hash

```go
func (cm *CM) Hash() common.Hash {
	return rlpHash(cm.Cm)
}
```

**New**：两种新建CM结构的函数第一种默认spent字段为`false`

```go
func NewDefaultCM(Cm uint64) *CM {}

func NewCM(Cm uint64, Spent bool) *CM {}
```

### CM数据库的创建与初始化

1、CM数据库的创建

```go
// cmd/geth/chaincmd.go 226行
	CMdb, err := stack.OpenDatabase("CMdata", 0, 0, "")
	if err != nil {
		utils.Fatalf("Failed to open database: %v", err)
	}
	CMdb.Close()
```

2、数据库的打开

```go
// eth/backend.go 141行
CMdb, err := ctx.OpenDatabase("CMdata", config.DatabaseCache, config.DatabaseHandles, "eth/db/CMdata/")
```

3、ethereum和blockchain新增字段

```go
// eth/backend.go 67行
type Ethereum struct {
	......
	// DB interfaces
	chainDb ethdb.Database // Block chain database
	CMdb ethdb.Database // CM database

	......
}
// eth/backend.go 163行
eth := &Ethereum{
		config:         config,
		chainDb:        chainDb,
		CMdb: 			CMdb,
		......
	}
// eth/backend.go 605行
s.CMdb.Close()
// core/blockchain.go  139行
type Blockchain struct {
    ......
    Cmdb	ethdb.Database
    ......
}
```

### CM数据库的调用

增加函数，对承诺的读，写，查，删操作 ，可通过`rawdb.xxxx()`进行调用。

`2020.9.18 ` 首次添加，暂未尝试调用

1、检查承诺是否存在

```go
func HasCM(db ethdb.Reader, hash common.Hash) bool {}
```
输入：数据库db，给定承诺的hash值

输出：bool值

2、向数据库中写入经过rlp编码后的承诺

```go
func WriteCM(db ethdb.KeyValueWriter, hash common.Hash, CM types.CM) {}
```
输入：数据库db，给定承诺的hash值，承诺CM

输出：无

3、从打包好的区块中取出所有交易，将交易中的承诺进行存储（v1.0 暂仅测试CMV） （已舍弃该函数）

```go
func WriteAllCM(db ethdb.KeyValueWriter, block *types.Block) {}
```

输入1：数据库db，区块

输出：无

**新增**：写入购币交易与转账交易的各个承诺

功能：默认将CmV有效，CmO无效，CmS有效，CmR有效写入数据库中，将验证过程放在交易进交易池时。

4、根据给定hash从数据库中取出承诺CM

```go
func ReadCMRLP(db ethdb.Reader, hash common.Hash) rlp.RawValue {}
func ReadCM(db ethdb.Reader,hash common.Hash) *types.CM  {}
```
输入1：数据库db，给定承诺的hash值

输出1：CM的rlp编码值

输入2：数据库db，给定承诺的hash值

输出2：承诺CM

函数2内部调用函数1

5、根据给定hash从数据库中删除承诺CM

```go
func DeleteCM(db ethdb.KeyValueWriter, hash common.Hash) {}
```

输入：数据库db，给定承诺的hash值

输出：无



`2020.9.21 ` 测试写入与读取成功

```go
//chaincmd.go 230行 写入与读取测试成功
	cm1 := types.CM{
		Cm  : 0x1,
		Spent : true,
	}
rawdb.WriteCM(CMdb,common.HexToHash("d3d6bb893a6e274cab241245d5df1274c58d664fbb1bfd6e59141c2e0bc5304a"),cm1)
x :=new(types.CM)
x=rawdb.ReadCM(CMdb,common.HexToHash("d3d6bb893a6e274cab241245d5df1274c58d664fbb1bfd6e59141c2e0bc5304a"))
log.Info("Successfully wrote x", "x", x)
```

数据库中如下
> [63d3d6bb893a6e274cab241245d5df1274c58d664fbb1bfd6e59141c2e0bc5304a]:c20101

输出如下

> INFO [09-25|23:24:46.380] Successfully wrote x                     x="&{Cm:1 Spent:true}"



### CM的验证

对即将进入交易池的交易内承诺验证

三种情况报错：

* 购币交易的购币承诺已存在于CMdb中
* 转账交易的CmO **不存在** 或 **存在但已使用**
* 交易ID既不为0也不为1,暂未知类型交易

```go
// core/tx_pool.go 613行
func (pool *TxPool) validateCM(tx *types.Transaction) error 
```

validateCM调用

```go
// core/tx_pool.go 658行
func (pool *TxPool) add(tx *types.Transaction, local bool) (replaced bool, err error)	{
	...
    // 若交易承诺检验未通过，丢弃
	if err := pool.validateCM(tx); err != nil {
		log.Trace("Discarding invalid transaction", "hash", hash, "err", err)
		invalidTxMeter.Mark(1)
		return false, err
	}
    ...
}
```

### CM的相关处理

无论是从本地节点得交易还是其他节点或区块同步来得交易，最后都是要调用tx_pool最下面得Add和Remove方法，所以在`pool.all.Add`和`pool.all.Remove`两个方法前分别添加processCM和reorgCM来进行处理

```go
// processCM 根据ID分别处理交易中的CM
func (pool *TxPool) processCM(tx *types.Transaction) {
}

// reorgCM 回滚因将交易从交易池中舍弃导致的CM状态改变
func (pool *TxPool) reorgCM(tx *types.Transaction) {
}
```

