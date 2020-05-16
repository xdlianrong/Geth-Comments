

## HD分层钱包（BIP32）

基于BIP32标准

HD钱包是从一个根种子创建的，它可以是128-bit,256-bit或者512-bit的随机数。通常情况下，由前面章节所述，这个种子通过助记码生成。

HD钱包里的每个私钥都是从根种子生成决定的，因此在任何兼容的钱包里可以从这个根种子重新创建整个HD钱包。这样备份，保存，导出导入一个包含成千上万个私钥的HD钱包都很容易，只要简单地转移一下用来生成根种子的助记码。

分层确定性钱包(BIP-32)和路径(BIP-43/44)

大部分HD钱包遵循BIP-32标准，这实际上已经成为了确定性密钥生成的行业标准。详细的规格说明请参照：

https://github.com/bitcoin/bips/blob/master/bip-0032.mediawiki



## BIP32标准

### 默认钱包布局

HDW被组织为多个“帐户”。帐户已编号，默认帐户（“”）的编号为0。客户不需要支持多个帐户-如果不是，则仅使用默认帐户。

每个帐户由两个密钥对链组成：一个内部和一个外部。外部钥匙串用于生成新的公共地址，而内部钥匙串用于所有其他操作（更改地址，生成地址等），不需要进行任何通信。不支持单独钥匙串的客户应使用外部钥匙串进行所有操作。

- m / i H / 0 / k对应于从主m派生的HDW的帐号i的外部链的第k个密钥对。
- m / i H / 1 / k对应于从母版m导出的HDW帐号i的内部链的第k个密钥对。



扩展公钥和私钥按照BIP-32的定义，用来生成子密钥的父密钥被叫做扩展密钥。如果是私钥，那就是扩展私钥，用前缀xprv标识，扩展公钥用前缀xpub标识。



HD钱包一个非常有用的特点就是在没有私钥的情况下可以用父公钥导出子公钥。这给我们得到子公钥的两个途径：通过子私钥或者直接从父公钥。



### HD钱包密钥标识（路径）

HD钱包里的密钥约定用“路径”标识，树结构里的每一层用“/”分开(参见HD wallet path examples)。主私钥导出的私钥用“m.”开头，主公钥导出的公钥用“M.”开头。因此，主私钥的第一个子私钥是m/0。第一个子公钥是M/0。子私钥的第二个孙私钥是m/0/1，等等。

一个密钥的“祖先”可以从右到左读出，直到读到主密钥。比如，标识符m/x/y/z表示这个密钥是m/x/y的第z+1个子密钥，而m/x/y是密钥m/x的第y+1个子密钥，m/x,是m的第x+1个子密钥。

HD钱包路径示例

| HD 路径     | 密钥描述                                                     |
| ----------- | ------------------------------------------------------------ |
| m/0         | 从主私钥(m) 推导出的第一代的第一个子私钥                     |
| m/0/0       | 从第一代子密钥(m/0)推导出的第二代第一个孙私钥                |
| m/0'/0      | 从第一代增强密钥 (m/0')推导出的第二代第一个孙密钥            |
| m/1/0       | 从第一代的第二个子密钥推导出的第二代第一个孙密钥             |
| M/23/17/0/0 | 从第一代第24个子公钥的第18个孙公钥的第一个曾孙公钥推导出的第一个玄孙公钥 |

增强子密钥推导

从一个xpub推导一组公钥的功能非常有用，但是也有潜在风险。Xpub并没有访问子私钥的权限。但是，xpub包含了链码，如果一个子私钥已知或者说泄露，两者一起就可以推导出其它所有的子私钥。泄露一个子私钥，加上父链码，就泄露了所有的子私钥。更糟糕的是，子私钥加上父链码可以用作推算父私钥。

为了防范这种风险，HD钱包采用了另一种叫做“增强推导”的推导函数，它隔绝了父公钥和子链码的关系。增强推导函数使用父私钥推导子链码，而不是用父公钥。这样通过不影响父私钥和同级子私钥的链码，在父/子结构中间加了一道“防火墙”。

简单的说，如果你用xpub的方便性推导一组公钥，同时不想泄露链码，就应该用增强推导，而不是用常规的推导。最正确的做法是，从主密钥推导第一级子密钥时，永远使用增强推导，避免泄露主密钥。



## hd.go

hd.go该代码没有运用到其它代码的任何一个包，仅仅用到了go语言中一些基础的包，该文件比较独立。

```go
import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strings"
)
```

```go
var DefaultRootDerivationPath = DerivationPath{0x80000000 + 44, 0x80000000 + 60, 0x80000000 + 0, 0}
//DefaultRootDerivationPath是自定义派生端点的根路径附加。第一个帐户位于m / 44'/ 60'/ 0'/ 0，第二个帐户位于在m / 44'/ 60'/ 0'/ 1等处。
var DefaultBaseDerivationPath = DerivationPath{0x80000000 + 44, 0x80000000 + 60, 0x80000000 + 0, 0, 0}
// DefaultBaseDerivationPath是自定义派生终结点的基本路径递增。第一个帐户位于m / 44'/ 60'/ 0'/ 0/0，第二个帐户位于在m / 44'/ 60'/ 0'/ 0/1等处
var LegacyLedgerBaseDerivationPath = DerivationPath{0x80000000 + 44, 0x80000000 + 60, 0x80000000 + 0, 0}
// LegacyLedgerBaseDerivationPath是用于自定义派生的旧版基本路径端点增加。 因此，第一个帐户将为m / 44'/ 60'/ 0'/ 0，第二个账户以m / 44'/ 60'/ 0'/ 1等表示，
```

后面的参数可参见BIP32标准：

（2013年4月16日）添加了i≥0x80000000的私有派生（降低了父私有密钥泄漏的风险），（2014年1月15日）将索引≥0x80000000的密钥重命名为强化密钥。

```go
func ParseDerivationPath(path string) (DerivationPath, error) {
	var result DerivationPath

	// Handle absolute or relative paths
    //处理绝对或相对的路径
	components := strings.Split(path, "/")
	switch {
	case len(components) == 0:
		return nil, errors.New("empty derivation path")

	case strings.TrimSpace(components[0]) == "":
		return nil, errors.New("ambiguous path: use 'm/' prefix for absolute paths, or no leading '/' for relative ones")

	case strings.TrimSpace(components[0]) == "m":
		components = components[1:]

	default:
		result = append(result, DefaultRootDerivationPath...)
	}
	// All remaining components are relative, append one by one
	if len(components) == 0 {
		return nil, errors.New("empty derivation path") // Empty relative paths
	}
	for _, component := range components {
		// Ignore any user added whitespace
		component = strings.TrimSpace(component)
		var value uint32

		// Handle hardened paths
		if strings.HasSuffix(component, "'") {
			value = 0x80000000
			component = strings.TrimSpace(strings.TrimSuffix(component, "'"))
		}
		// Handle the non hardened component
		bigval, ok := new(big.Int).SetString(component, 0)
		if !ok {
			return nil, fmt.Errorf("invalid component: %s", component)
		}
		max := math.MaxUint32 - value
		if bigval.Sign() < 0 || bigval.Cmp(big.NewInt(int64(max))) > 0 {
			if value == 0 {
				return nil, fmt.Errorf("component %v out of allowed range [0, %d]", bigval, max)
			}
			return nil, fmt.Errorf("component %v out of allowed hardened range [0, %d]", bigval, max)
		}
		value += uint32(bigval.Uint64())

		// Append and repeat
		result = append(result, value)
	}
	return result, nil
}
```

该函数用于对派生路径的解析，先判断是否路径是否为空，为绝对路径，摸棱两可的路径。若为绝对路径，则添加到result中。接下来对加强路径和非加强路径进行处理，处理完后添加到result中。

```go
func (path DerivationPath) String() string {
	result := "m"
	for _, component := range path {
		var hardened bool
		if component >= 0x80000000 {
			component -= 0x80000000
			hardened = true
		}
		result = fmt.Sprintf("%s/%d", result, component)
		if hardened {
			result += "'"
		}
	}
	return result
}
```

这是一个接口，转换为一个二进制的派生路径。判断是否是一个增强路径，如果为了一个增强路径，将在result上加一个‘。

```go
func (path DerivationPath) MarshalJSON() ([]byte, error) {
	return json.Marshal(path.String())
}

// UnmarshalJSON a json-serialized string back into a derivation path
func (path *DerivationPath) UnmarshalJSON(b []byte) error {
	var dp string
	var err error
	if err = json.Unmarshal(b, &dp); err != nil {
		return err
	}
	*path, err = ParseDerivationPath(dp)
	return err
}
```

 MarshalJSON：将数据编码成json字符串

UnmarshalJSON：将json字符串解码到相应的数据结构

这是json工具包中的两个方法

总的来说，该代码文件是对派生路径的一些处理。关于分层性钱包，在account.go里面有关于它们的两个接口，具体解释可参见之前写的文件。



## Block、StateDB与Trie的关系

每个Block包含若干交易，每个交易都包含账户From和To(除部署合约)，全部的账户凑在一起组成了StateDB，每个块的StateDB都用一颗Trie树来组织账户信息。

![13637985-eed86ddc97cbdcec](C:\Users\43798\Desktop\13637985-eed86ddc97cbdcec.webp)

保存账户信息的 StateDB 通常会存储在磁盘上，通过 Block.StateRoot 来进行加载，StateRoot 是树根，也是 leveldb 中的一个 key, 这个根只对应当前块的交易相关的账户信息，value 是这棵树的全部叶子节点，加载的时候会用叶子节点来构建下图中的树型结构

![](C:\Users\43798\Desktop\13637985-1b76fa49e36de010.webp)

智能合约的地址也被当作账户管理，当 Account 为一个智能合约时，那么这个 stateObject 也会包含一颗树，用来保存智能合约的最新状态信息，这些信息是每次执行 evm 中 SSTORE 这个指令时的输入信息，key 是合约的变量名，value 是最新值，这棵树的加载过程与上图中的过程完全一致

![13637985-2f7a01891cbe6287](C:\Users\43798\Desktop\13637985-2f7a01891cbe6287.webp)

### StateDB的关键方法：

```go
func (self *StateDB) AddBalance(addr common.Address, amount *big.Int) 
func (s *StateDB) Commit(deleteEmptyObjects bool) (root common.Hash, err error) 
func (s *StateDB) Finalise(deleteEmptyObjects bool) 
```

StateDB的操作流程：

当执行 `AddBalance` 方法会设置 `addr` 的 `stateObject.Account.Balance += amount` ，这时 `addr` 对应的 `stateObject` 会被放在 `StateDB.stateObjects` 中缓存起来，
 当执行 `Commit` 方法时，会将 `StateDB.stateObjects` 中的数据构建成一颗默克尔树存放在 `StateDB.trie` 中，修改数据时会产生 `journal` 日志，为了方便出错时的回滚，
 当执行 `Finalise` 方法时会删除 `journal` 刷新 `stateObject` 的 `trie.root` (如果 stateObject 存在 trie)

执行到此时树的结构已经确定，最终的树根也已经确定，但是在以太坊中数据此时还没有进入数据库
 为了提高效率StateDB对数据做了缓存处理，大部分时间都是放在内存中的，StateDB.db 是 state.Database 接口的实现，定义如下：

```go
// Database wraps access to tries and contract code.
type Database interface {
    // blockchain 的 树，root == block.stateRoot
    OpenTrie(root common.Hash) (Trie, error)
    // account 的 树，addrHash == account.address
    OpenStorageTrie(addrHash, root common.Hash) (Trie, error)
    ......
    // TrieDB retrieves the low level trie database used for data storage.
    TrieDB() *trie.Database
}
```

需要使用 TrieDB 这个方法返回的 trie.Database 对象来最终完成 trei 写入 leveldb 的操作，这个方法在 WriteBlockWithState 中会被有条件调用。



### statedb.go

该代码位于state/journal.go

```go
type journal struct {
   entries []journalEntry         // Current changes tracked by the journal
   dirties map[common.Address]int // Dirty accounts and the number of changes
}
```

```go
type revision struct {
	id           int
	journalIndex int
}
```

日志存的是账户变动的反向操作。对以太坊实现快照功能以及回滚世界状态非常有用。validRevisions是一个revision的切片，后者存的是日志（journal）的索引。通过revision的journalIndex可以索引到日志切片的位置，回滚到某个revision就只要去查找当前journals切片中大于revision.journalIndex的那些日志，并执行，即可回滚到当前revision指定的世界状态。

```go
type StateDB struct {
	db   Database
	trie Trie

	// 这个map相当于是stateDB的缓存，存放活动的账户，如果有账户在这里找不到，则通过trie从数据库中找到目标账户，然后存到这个map里
	stateObjects        map[common.Address]*stateObject
	stateObjectsPending map[common.Address]struct{} // State objects finalized but not yet written to the trie
	stateObjectsDirty   map[common.Address]struct{} // State objects modified in the current execution

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// The refund counter, also used by state transitioning.
	refund uint64

	thash, bhash common.Hash
	txIndex      int
	logs         map[common.Hash][]*types.Log
	logSize      uint

	preimages map[common.Hash][]byte

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int

	// Measurements gathered during execution for debugging purposes
	AccountReads   time.Duration
	AccountHashes  time.Duration
	AccountUpdates time.Duration
	AccountCommits time.Duration
	StorageReads   time.Duration
	StorageHashes  time.Duration
	StorageUpdates time.Duration
	StorageCommits time.Duration
}
```

stateObjects 是一个 map，用来缓存所有从数据库（也就是 trie 字段）中读取出来的账户信息，无论这些信息是否被修改过都会缓存在这里。

stateObjectsDirty 很显然是用来记录哪些账户信息被修改过了。需要注意的是，这个字段并不时刻与 stateObjects 对应，并且也不会在账户信息被修改时立即修改这个字段。

journal 字段记录了 StateDB 对象的所有操作，以便用来进行回滚操作。需要注意的是，当 stateObjects 中的所有信息被写入到 `trie` 字段代表的 trie 树中后，`journal` 字段会被清空，无法再进行回滚了。



根据给定的树根创建一个世界状态

```go
func New(root common.Hash, db Database) (*StateDB, error) {
	tr, err := db.OpenTrie(root)
	if err != nil {
		return nil, err
	}
	return &StateDB{
		db:                db,
		trie:              tr,
		stateObjects:      make(map[common.Address]*stateObject),
		stateObjectsDirty: make(map[common.Address]struct{}),
		logs:              make(map[common.Hash][]*types.Log),
		preimages:         make(map[common.Hash][]byte),
		journal:           newJournal(),
	}, nil
}

```

调用：

1、BlockChain插入区块链时进行状态验证：BlockChain.inertChain —> state.New
2、BlockChain初始化时验证状态是否可读：BlockChain.loadLastState —> state.New



该函数步骤：1.实例化一个stateObject对象；2.在日志中添加创建新账户事件;将新账号添加到缓存中。

```go
func (self *StateDB) createObject(addr common.Address) (newobj, prev *stateObject) {
        // 先看下这个地址之前的账户状态，如果有就不用新建账户
	prev = self.getStateObject(addr)
	newobj = newObject(self, addr, Account{})
 
        // 设置nonce的初始值
	newobj.setNonce(0) 
 
	if prev == nil {
		self.journal.append(createObjectChange{account: &addr})
	} else {
		self.journal.append(resetObjectChange{prev: prev})
	}
        // 将新账户添加到缓存
	self.setStateObject(newobj)
	return newobj, prev
}
 
func (self *StateDB) setStateObject(object *stateObject) {
	self.stateObjects[object.Address()] = object
}
```

反向操作：

当prev不存在时，我们向journal中添加了createObjectChange{account： &addr}，这个对象实现的方法有：

```go
func (ch createObjectChange) revert(s *StateDB) {
	delete(s.stateObjects, *ch.account)
	delete(s.stateObjectsDirty, *ch.account)
}
```

revert()方法传入StateDB，然后从stateObjects和stateObjectsDirty中把刚才createObject生成的stateObject给删除掉。

resetObjectChange{prev:prev}，这是当我们创建指定地址的账户发现该账户已经存在时在日志中添加的反向操作，目的是将StateObjcet设置成之前的对象。上面是将一个已经存在的stateObject替换成了新的stateObject，尽管地址一样，但其他的属性可能都变了。所以反向操作就是直接替换回来。

```go
func (ch resetObjectChange) revert(s *StateDB) {
	s.setStateObject(ch.prev)
}
```



```go
type journal struct {
	entries []journalEntry         // Current changes tracked by the journal
	dirties map[common.Address]int // 记录了变动的账户及账户变动的次数
}
 
func (self *StateDB) Snapshot() int {
        // 首先获取快照id，从0开始计数
	id := self.nextRevisionId
	self.nextRevisionId++
        // 然后将快照保存，即reversion{id, journal.length}
	self.validRevisions = append(self.validRevisions, revision{id, self.journal.length()})
	return id
}
 
// 恢复快照
// 1、检查快照编号是否有效
// 2、通过快照编号获取日志长度 
// 3、调用日志中的revert函数进行恢复
// 4、移除恢复点后面的快照
func (self *StateDB) RevertToSnapshot(revid int) {
	// 找出validReversion[0，n]中最小的下标偏移i，能够满足第二个函数f(i) == true
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex
 
	// Replay the journal to undo changes and remove invalidated snapshots
	self.journal.revert(self, snapshot)
	self.validRevisions = self.validRevisions[:idx]
}
 
func (j *journal) revert(statedb *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// 从后往前逐个执行revert函数
		j.entries[i].revert(statedb)
 
		// 将所有账户变动的标记删掉
		if addr := j.entries[i].dirtied(); addr != nil {
			if j.dirties[*addr]--; j.dirties[*addr] == 0 {
				delete(j.dirties, *addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}
```

调用：

1、EVM调用Call、CallCode、DelegateCall、StaticCall、Create的过程中：EVM.Create —> StateDB.Snapshot

2、应用交易的过程：Work.commitTransaction —> StateDB.Snapshot

这些函数都是在执行交易的时候进行快照的，如果交易执行失败，需要按照快照进行回滚。每次账户的变动都会在dirties中记录，回滚后要重新恢复原来的dirties标记。



销毁账户

1.获取账户的stateObject对象 2.添加删除日志 3.标记账户自杀，余额清零

```go
func (self *StateDB) Suicide(addr common.Address) bool {
        // 获取账户的stateObject对象
	stateObject := self.getStateObject(addr)
	if stateObject == nil {
		return false
	}
        // 添加日志
	self.journal.append(suicideChange{
		account:     &addr,
		prev:        stateObject.suicided,
		prevbalance: new(big.Int).Set(stateObject.Balance()),
	})
        // 标记自杀，余额清零
	stateObject.markSuicided()
	stateObject.data.Balance = new(big.Int)
 
	return true
}
 
func (ch suicideChange) revert(s *StateDB) {
	obj := s.getStateObject(*ch.account)
	if obj != nil {
		obj.suicided = ch.prev
		obj.setBalance(ch.prevbalance)
	}
}
```

反向操作获取该stateObject，恢复自杀标记，然后余额返回账户



获取stateObject,返回余额。无日志，因为没用更改账户。

```go
func (self *StateDB) GetBalance(addr common.Address) *big.Int {
	stateObject := self.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Balance()
	}
	return common.Big0
}
```

GetNonce、GetCode、GetCodeSize，GerCodeHash等函数操作与之类似。



生成快照

调用关系：

1、EVM调用Call、CallCode、DelegateCall、StaticCall、Create的过程中：EVM.Create —> StateDB.Snapshot

2、应用交易的过程：Work.commitTransaction —> StateDB.Snapshot

这些函数都是在执行交易的时候进行快照的，因为如果交易执行失败，需要按照快照进行回滚。每次账户的变动都会在dirties中记录，所以回滚后要重新恢复原来的dirties标记。

```go
type journal struct {
	entries []journalEntry         // Current changes tracked by the journal
	dirties map[common.Address]int // 记录了变动的账户及账户变动的次数
}
 
func (self *StateDB) Snapshot() int {
        // 首先获取快照id，从0开始计数
	id := self.nextRevisionId
	self.nextRevisionId++
        // 然后将快照保存，即reversion{id, journal.length}
	self.validRevisions = append(self.validRevisions, revision{id, self.journal.length()})
	return id
}
 
// 恢复快照
// 1、检查快照编号是否有效
// 2、通过快照编号获取日志长度
// 3、调用日志中的revert函数进行恢复
// 4、移除恢复点后面的快照
func (self *StateDB) RevertToSnapshot(revid int) {
	// 找出validReversion[0，n]中最小的下标偏移i，能够满足第二个函数f(i) == true
	idx := sort.Search(len(self.validRevisions), func(i int) bool {
		return self.validRevisions[i].id >= revid
	})
	if idx == len(self.validRevisions) || self.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := self.validRevisions[idx].journalIndex
 
	// Replay the journal to undo changes and remove invalidated snapshots
	self.journal.revert(self, snapshot)
	self.validRevisions = self.validRevisions[:idx]
}
 
func (j *journal) revert(statedb *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// 从后往前逐个执行revert函数
		j.entries[i].revert(statedb)
 
		// 将所有账户变动的标记删掉
		if addr := j.entries[i].dirtied(); addr != nil {
			if j.dirties[*addr]--; j.dirties[*addr] == 0 {
				delete(j.dirties, *addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}
```





更新状态树，计算状态树根

调用关系:

1、BlockChain验证一个区块的状态树根是否正确：BlockChain.insertChain —>BlockValidator.ValidateState —>stateDB.intermediateRoot。也就是区块插入规范链时，执行完交易后要验证此时本地的状态树树根与发来的区块头中的是否一致

2、Worker递交工作的过程中执行全部交易后，需要得到状态树根来填充区块头的root：worker.CommitNewWork —> Ethash.Finalize —> stateDB.IntermeidateRoot。因为挖矿之前要先执行交易，还要结算挖矿奖励，然后生成最新的状态。这时候就需要获得状态树树根，放在区块头中，一起打包用于挖矿。

函数步骤：1.遍历更新的账户，将被更新的账户写入状态树，清除变更日志、快照、返利  2.计算状态树根

所有日志及回滚都是在StateObjects这个map缓存中进行的，一旦这些状态被写进状态树，日志就没用了，不能再回滚了，所以将日志、快照、返利都清除。

```go
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) common.Hash {
	s.Finalise(deleteEmptyObjects)
 
        // Trie的hash折叠函数
	return s.trie.Hash()
}
 
// 遍历更新的账户，将被更新的账户写入状态树，清除变更日志、快照、返利
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
        // dirties中记录的是变更过的账户
	for addr := range s.journal.dirties {
                // 验证这个账户再stateObjects列表中也存在，如果不存在就跳过这个账户
		stateObject, exist := s.stateObjects[addr]
		if !exist {
			continue
		}
                // 如果账户已经销毁，从状态树中删除账户；如果账户为空，且deleteEmptyObjects标志为true，则删除商户
		if stateObject.suicided || (deleteEmptyObjects && stateObject.empty()) {
			s.deleteStateObject(stateObject)
		} else {
                        // 否则将账户的storage变更写入storage树，更新storage树根
			stateObject.updateRoot(s.db)
                        // 将当前账户写入状态树
			s.updateStateObject(stateObject)
		} 
                // 当账户被更新到状态树后，将改动的账户标记为脏账户
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournalAndRefund()
}
```



将状态树写入数据库

调用关系：BlockChain调用WriteBlockWithState写区块链的过程中：BlockChain.WriteBlockWithState —>StateDB.Commit

函数步骤:1.遍历日志列表，更新脏账户（因为我们只需要重写那些改动了的账户，没有改动的账户不需要处理）  2.遍历被更新的账户，将被更新的账户写入状态树   3.将状态树写入数据库

```go
func (s *StateDB) Commit(deleteEmptyObjects bool) (root common.Hash, err error) {
	defer s.clearJournalAndRefund()
        // 遍历脏账户，将被更新的账户写入状态树
	for addr := range s.journal.dirties {
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// 遍历被更新账户，将其写入状态树
	for addr, stateObject := range s.stateObjects {
		_, isDirty := s.stateObjectsDirty[addr]
		switch {
		case stateObject.suicided || (isDirty && deleteEmptyObjects && stateObject.empty()):
			// 如果账户标记为销毁，则从状态树中删除之
                        // 如果账户为空，且deleteEmptyObjects标志为true，从状态树中删除账户
			s.deleteStateObject(stateObject)
		case isDirty:
			// 如果有代码更新，则将code以codeHash为key，以code为value存入db数据库
			if stateObject.code != nil && stateObject.dirtyCode {
				s.db.TrieDB().InsertBlob(common.BytesToHash(stateObject.CodeHash()), stateObject.code)
				stateObject.dirtyCode = false
			}
			// 更新storage树
			if err := stateObject.CommitTrie(s.db); err != nil {
				return common.Hash{}, err
			}
			// 更新状态树
			s.updateStateObject(stateObject)
		}
		delete(s.stateObjectsDirty, addr)
	}
	// task3：将状态树写入数据库
	root, err = s.trie.Commit(func(leaf []byte, parent common.Hash) error {
		var account Account
		if err := rlp.DecodeBytes(leaf, &account); err != nil {
			return nil
		}
		if account.Root != emptyState {
			s.db.TrieDB().Reference(account.Root, parent)
		}
		code := common.BytesToHash(account.CodeHash)
		if code != emptyCode {
			s.db.TrieDB().Reference(code, parent)
		}
		return nil
	})
	log.Debug("Trie cache stats after commit", "misses", trie.CacheMisses(), "unloads", trie.CacheUnloads())
	return root, err
}
```



将状态写入状态树

```go
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	for addr := range s.journal.dirties {
		obj, exist := s.stateObjects[addr]
		if !exist {
			// ripeMD is 'touched' at block 1714175, in tx 0x1237f737031e40bcde4a8b7e717b2d15e3ecadfe49bb1bbc71ee9deb09c6fcf2
			// That tx goes out of gas, and although the notion of 'touched' does not exist there, the
			// touch-event will still be recorded in the journal. Since ripeMD is a special snowflake,
			// it will persist in the journal even though the journal is reverted. In this special circumstance,
			// it may exist in `s.journal.dirties` but not in `s.stateObjects`.
			// Thus, we can safely ignore it here
			continue
		}
		if obj.suicided || (deleteEmptyObjects && obj.empty()) {
			obj.deleted = true
		} else {
			obj.finalise()
		}
		s.stateObjectsPending[addr] = struct{}{}
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournalAndRefund()
}
```

小结：

stateDB对象就是对以太坊状态MPT进行管理的对象。其管理功能包括：
1、初始化：New
2、增加：StateDB.createObject
3、删除：StateDB.Suicide
4、修改：StateDB.AddBalance
5、查询：StateDB.GetBalance
6、拍摄快照：StateDB.Snopshot
7、恢复快照：StateDB.RevertToSnopshot
8、将状态写入状态树：StateDB.Finalise
9、获得树根：StateDB.IntermediateRoot
10、将状态写入数据库：StateDB.Commit

### stateDB生命周期

 stateDB是用来管理世界状态。

世界状态改变时候：

1.打包交易进行挖矿时

2.收到区块广播执行同步时

stateDB是从挖矿时从交易池取出交易并执行，然后打包等待挖矿，最后当区块挖矿成功后，将stateDB中的账户改变刷入数据库后，stateDB的使命就结束了，就可以从内存中删除了。