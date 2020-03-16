###### 1.defer函数

延迟（defer）语句：可以在函数中添加多个defer语句。当函数执行到最后时，这些defer语句会按照逆序执行，最后该函数返回。特别是当你在进行一些打开资源的操作时，遇到错误需要提前返回，在返回前你需要关闭相应的资源，不然很容易造成资源泄露等问题。

例：

```go
func ReadWrite() bool {
    file.Open("file")
// 做一些工作
    if failureX {
        file.Close()
        return false
    }

    if failureY {
        file.Close()
        return false
    }

    file.Close()
    return true
}
```

defer版本：

```go
func ReadWrite() bool {
    file.Open("file")
    defer file.Close()
    if failureX {
        return false
    }
    if failureY {
        return false
    }
    return true
}
```

###### 2.goalng内置new函数

这是一个用来分配内存的内置函数，它的第一个参数是一个类型，不是一个值，它的返回值是一个指向新分配的 t 类型的零值的指针。

```go
func new(t Type) *Type 
```

和struct{}初始化的方式相比，struct{}初始化返回的是一个struct类型的值，而不是指针。

例：

```go
package main

import "fmt"

type Student struct{
	id int
	name string
}

func main(){
	var s_1 *Student = new(Student)
	s_1.id = 100
	s_1.name = "cat"
	var s_2 Student = Student{id:1,name:"tom"}
	fmt.Println(s_1,s_2)
}
```

输出：

```go
&{100 cat} {1 tom}
```



###### 3.initGenesis方法

对应着cmd指令：geth --datadir  /data  genesis.json init

对于initCommand的描述：

```go
	initCommand = cli.Command{
		Action:    utils.MigrateFlags(initGenesis),
		Name:      "init",
		Usage:     "Bootstrap and initialize a new genesis block",
		ArgsUsage: "<genesisPath>",
		Flags: []cli.Flag{
			utils.DataDirFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The init command initializes a new genesis block and definition for the network.
This is a destructive action and changes the network in which you will be
participating.

It expects the genesis file as argument.`
        
```

调用的函数：initGenesis

解析了一个flag的值：DataDirFlag

指令：init

flag之外的参数说明：genesis路径

代码：

```go
// initGenesis will initialise the given JSON format genesis file and writes it as
// the zero'd block (i.e. genesis) or will fail hard if it can't succeed.
func initGenesis(ctx *cli.Context) error {
	// Make sure we have a valid genesis JSON
	genesisPath := ctx.Args().First()
	if len(genesisPath) == 0 {
		utils.Fatalf("Must supply path to genesis JSON file")
	}
	file, err := os.Open(genesisPath)
	if err != nil {
		utils.Fatalf("Failed to read genesis file: %v", err)
	}
	defer file.Close()

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		utils.Fatalf("invalid genesis file: %v", err)
	}
	// Open an initialise both full and light databases
	stack := makeFullNode(ctx)
	defer stack.Close()

	for _, name := range []string{"chaindata", "lightchaindata"} {
		chaindb, err := stack.OpenDatabase(name, 0, 0, "")
		if err != nil {
			utils.Fatalf("Failed to open database: %v", err)
		}
		_, hash, err := core.SetupGenesisBlock(chaindb, genesis)
		if err != nil {
			utils.Fatalf("Failed to write genesis block: %v", err)
		}
		chaindb.Close()
		log.Info("Successfully wrote genesis state", "database", name, "hash", hash)
	}
	return nil
}
```

**首先读取指令**

```go
genesisPath := ctx.Args().First()
```

它指明了genesisPath变量是cmd指令的flag之外的第一个变量：

context.go文件

```go
type Args []string

// Args returns the command line arguments associated with the context.
func (c *Context) Args() Args {
	args := Args(c.flagSet.Args())
	return args
}

// Args returns the non-flag arguments.
func (f *FlagSet) Args() []string { return f.args }

func (a Args) First() string {
	return a.Get(0)
}

// Get returns the nth argument, or else a blank string
func (a Args) Get(n int) string {
	if len(a) > n {
		return a[n]
	}
	return ""
}
```

flag.go文件

```go

// A FlagSet represents a set of defined flags. The zero value of a FlagSet
// has no name and has ContinueOnError error handling.
type FlagSet struct {
	// Usage is the function called when an error occurs while parsing flags.
	// The field is a function (not a method) that may be changed to point to
	// a custom error handler. What happens after Usage is called depends
	// on the ErrorHandling setting; for the command line, this defaults
	// to ExitOnError, which exits the program after calling Usage.
	Usage func()

	name          string
	parsed        bool
	actual        map[string]*Flag
	formal        map[string]*Flag
	args          []string // arguments after flags
	errorHandling ErrorHandling
	output        io.Writer // nil means stderr; use out() accessor
}
```

这里可能有点绕，我说下自己的解释说明：

首先ctx.Args函数从FlagSet结构体中获取args，再把它转换为本包中的Args类型，但其实都是字符串切片。再通过First函数，First函数再调用get函数获得命令行的第一个参数。

**第二步打开文件**

```go
file, err := os.Open(genesisPath)
defer file.Close()
```

以只读的方式打开文件，返回指向文件的指针file

```go
func Open(name string) (*File, error) {
	return OpenFile(name, O_RDONLY, 0)
}
```

**第三步创建genesis对象**

```go
genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		utils.Fatalf("invalid genesis file: %v", err)
	}
```

core/genesis.go中有定义Genesis结构体：

```go
type Genesis struct {
	Config     *params.ChainConfig `json:"config"`
	Nonce      uint64              `json:"nonce"`
	Timestamp  uint64              `json:"timestamp"`
	ExtraData  []byte              `json:"extraData"`
	GasLimit   uint64              `json:"gasLimit"   gencodec:"required"`
	Difficulty *big.Int            `json:"difficulty" gencodec:"required"`
	Mixhash    common.Hash         `json:"mixHash"`
	Coinbase   common.Address      `json:"coinbase"`
	Alloc      GenesisAlloc        `json:"alloc"      gencodec:"required"`

	// These fields are used for consensus tests. Please don't use them
	// in actual genesis blocks.
	Number     uint64      `json:"number"`
	GasUsed    uint64      `json:"gasUsed"`
	ParentHash common.Hash `json:"parentHash"`
}
```

之后检查genesis的文件格式是否正确：

```go
// A Decoder reads and decodes JSON values from an input stream.
type Decoder struct {
	r       io.Reader
	buf     []byte
	d       decodeState
	scanp   int   // start of unread data in buf
	scanned int64 // amount of data already scanned
	scan    scanner
	err     error

	tokenState int
	tokenStack []int
}

//NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// Decode reads the next JSON-encoded value from its
// input and stores it in the value pointed to by v.
func (dec *Decoder) Decode(v interface{}) 
```

关于如何检查文件的先不深究，具体源码在上面，是go的源码包中的流处理的函数

**第四步创建节点**

```go
stack := makeFullNode(ctx)
defer stack.Close()
```

具体节点如何配置以后详细看

**第五步创建leveldb数据库**

chaincmd.go:

```go
	for _, name := range []string{"chaindata", "lightchaindata"} {
		chaindb, err := stack.OpenDatabase(name, 0, 0, "")
		if err != nil {
			utils.Fatalf("Failed to open database: %v", err)
		}
		_, hash, err := core.SetupGenesisBlock(chaindb, genesis)
		if err != nil {
			utils.Fatalf("Failed to write genesis block: %v", err)
		}
		chaindb.Close()
		log.Info("Successfully wrote genesis state", "database", name, "hash", hash)
	}

```

node.go：似乎还可以不指定目录名称创建一个临时数据库，暂不深究

```go
// OpenDatabase opens an existing database with the given name (or creates one if no
// previous can be found) from within the node's instance directory. If the node is
// ephemeral, a memory database is returned.
func (n *Node) OpenDatabase(name string, cache, handles int, namespace string) (ethdb.Database, error) {
	if n.config.DataDir == "" {
		return rawdb.NewMemoryDatabase(), nil
	}
	return rawdb.NewLevelDBDatabase(n.config.ResolvePath(name), cache, handles, namespace)
}
```

database.go:和leveldb的api进行交互，创建数据库

```go
// NewLevelDBDatabase creates a persistent key-value database without a freezer
// moving immutable chain segments into cold storage.
func NewLevelDBDatabase(file string, cache int, handles int, namespace string) (ethdb.Database, error) {
	db, err := leveldb.New(file, cache, handles, namespace)
	if err != nil {
		return nil, err
	}
	return NewDatabase(db), nil
}

// NewDatabase creates a high level database on top of a given key-value data
// store without a freezer moving immutable chain segments into cold storage.
func NewDatabase(db ethdb.KeyValueStore) ethdb.Database {
	return &nofreezedb{
		KeyValueStore: db,
	}
}
```

之后就是分别创建lightchaindata和chaindata，并取哈希，存入日志中。

###### 4.SetupGenesisBlock的方法

这个比较有意思了，看代码：

```go

// SetupGenesisBlock writes or updates the genesis block in db.
// The block that will be used is:
//
//                          genesis == nil       genesis != nil
//                       +------------------------------------------
//     db has no genesis |  main-net default  |  genesis
//     db has genesis    |  from DB           |  genesis (if compatible)
//
// The stored chain configuration will be updated if it is compatible (i.e. does not
// specify a fork block below the local head block). In case of a conflict, the
// error is a *params.ConfigCompatError and the new, unwritten config is returned.
//
// The returned chain configuration is never nil.
func SetupGenesisBlock(db ethdb.Database, genesis *Genesis) (*params.ChainConfig, common.Hash, error) {
	return SetupGenesisBlockWithOverride(db, genesis, nil, nil)
}

func SetupGenesisBlockWithOverride(db ethdb.Database, genesis *Genesis, overrideIstanbul, overrideMuirGlacier *big.Int) (*params.ChainConfig, common.Hash, error) {
	if genesis != nil && genesis.Config == nil {
		return params.AllEthashProtocolChanges, common.Hash{}, errGenesisNoConfig
	}
	// Just commit the new block if there is no stored genesis block.
	stored := rawdb.ReadCanonicalHash(db, 0)
	if (stored == common.Hash{}) {
		if genesis == nil {
			log.Info("Writing default main-net genesis block")
			genesis = DefaultGenesisBlock()
		} else {
			log.Info("Writing custom genesis block")
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, common.Hash{}, err
		}
		return genesis.Config, block.Hash(), nil
	}

	// We have the genesis block in database(perhaps in ancient database)
	// but the corresponding state is missing.
	header := rawdb.ReadHeader(db, stored, 0)
	if _, err := state.New(header.Root, state.NewDatabaseWithCache(db, 0)); err != nil {
		if genesis == nil {
			genesis = DefaultGenesisBlock()
		}
		// Ensure the stored genesis matches with the given one.
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
		block, err := genesis.Commit(db)
		if err != nil {
			return genesis.Config, hash, err
		}
		return genesis.Config, block.Hash(), nil
	}

	// Check whether the genesis block is already written.
	if genesis != nil {
		hash := genesis.ToBlock(nil).Hash()
		if hash != stored {
			return genesis.Config, hash, &GenesisMismatchError{stored, hash}
		}
	}

	// Get the existing chain configuration.
	newcfg := genesis.configOrDefault(stored)
	if overrideIstanbul != nil {
		newcfg.IstanbulBlock = overrideIstanbul
	}
	if overrideMuirGlacier != nil {
		newcfg.MuirGlacierBlock = overrideMuirGlacier
	}
	if err := newcfg.CheckConfigForkOrder(); err != nil {
		return newcfg, common.Hash{}, err
	}
	storedcfg := rawdb.ReadChainConfig(db, stored)
	if storedcfg == nil {
		log.Warn("Found genesis block without chain config")
		rawdb.WriteChainConfig(db, stored, newcfg)
		return newcfg, stored, nil
	}
	// Special case: don't change the existing config of a non-mainnet chain if no new
	// config is supplied. These chains would get AllProtocolChanges (and a compat error)
	// if we just continued here.
	if genesis == nil && stored != params.MainnetGenesisHash {
		return storedcfg, stored, nil
	}

	// Check config compatibility and write the config. Compatibility errors
	// are returned to the caller unless we're already at block zero.
	height := rawdb.ReadHeaderNumber(db, rawdb.ReadHeadHeaderHash(db))
	if height == nil {
		return newcfg, stored, fmt.Errorf("missing block number for head header hash")
	}
	compatErr := storedcfg.CheckCompatible(newcfg, *height)
	if compatErr != nil && *height != 0 && compatErr.RewindTo != 0 {
		return newcfg, stored, compatErr
	}
	rawdb.WriteChainConfig(db, stored, newcfg)
	return newcfg, stored, nil
}
```

