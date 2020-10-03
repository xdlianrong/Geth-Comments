[TOC]

### 概述

对承诺数据库的初始化及打开，使用。对外暂提供对承诺的写，读，改。

### 在rawdb包中的database中添加CM字段

CM 承诺结构，包含承诺字段以及判断该承诺是否被使用过的spent字段，true表示已使用

```go
//cmdb 承诺数据库，可读可写
type cmdb struct {
	ethdb.KeyValueStore
}

// CM 承诺结构，包含承诺字段以及判断该承诺是否被使用过的spent字段，true表示已使用
type CM struct {
	Cm    common.Hash
	Spent bool
}
```

### CM数据库的创建及打开

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

3、从打包好的区块中取出所有交易，将交易中的承诺进行存储（v1.0 暂仅测试CMV）

```go
func WriteAllCM(db ethdb.KeyValueWriter, block *types.Block) {}
```

输入1：数据库db，区块

输出：无

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