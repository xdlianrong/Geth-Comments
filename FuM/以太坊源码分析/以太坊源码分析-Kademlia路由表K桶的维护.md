# 以太坊源码分析-Kademlia路由表K桶的维护

[TOC]

Kademlia协议中的路由表由`p2p/discover/table.go`实现。路由表就被称为table。

## table的结构和字段

```go
const (
	alpha           = 3  // Kademlia concurrency factor
	bucketSize      = 16 // Kademlia bucket size
	maxReplacements = 10 // Size of per-bucket replacement list

	// We keep buckets for the upper 1/15 of distances because
	// it's very unlikely we'll ever encounter a node that's closer.
	hashBits          = len(common.Hash{}) * 8 // 32Bytes * 8 = 256bits
	nBuckets          = hashBits / 15       // Number of buckets
	bucketMinDistance = hashBits - nBuckets // Log distance of closest bucket

	// IP address limits.
	bucketIPLimit, bucketSubnet = 2, 24 // at most 2 addresses from the same /24
	tableIPLimit, tableSubnet   = 10, 24

	refreshInterval    = 30 * time.Minute
	revalidateInterval = 10 * time.Second
	copyNodesInterval  = 30 * time.Second
	seedMinTableTime   = 5 * time.Minute
	seedCount          = 30
	seedMaxAge         = 5 * 24 * time.Hour
)

// Table is the 'node table', a Kademlia-like index of neighbor nodes. The table keeps
// itself up-to-date by verifying the liveness of neighbors and requesting their node
// records when announcements of a new record version are received.
type Table struct {
	mutex   sync.Mutex        // protects buckets, bucket content, nursery, rand
	buckets [nBuckets]*bucket // index of known nodes by distance
	nursery []*node           // bootstrap nodes
	rand    *mrand.Rand       // source of randomness, periodically reseeded
	ips     netutil.DistinctNetSet

	log        log.Logger
	db         *enode.DB // database of known nodes
	net        transport
	refreshReq chan chan struct{}
	initDone   chan struct{}
	closeReq   chan struct{}
	closed     chan struct{}

	nodeAddedHook func(*node) // for testing
}
```

静态变量含义主要如下：

 + `alpha = 3`，Kademlia 协议所声明的，在节点路由时，并发寻找的节点数。
 + `bucketSize = 16`，Kademlia 协议所声明的，一个桶的所能存储的可用节点最大容量。
 + `maxReplacements = 10`，一个桶所能存储的替换节点的最大容量。
 + `hashBits = len(common.Hash{}) * 8`，一个哈希的比特数，`len(common.Hash{})`是hash的字节数(此项目中为32)，`hashBits`为256。
 + `nBuckets = hashBits / 15`，桶的数量，=17。
 + `bucketMinDistance = hashBits - nBuckets`=239，最近桶的对数距离，目前还不知用处。
 + `bucketIPLimit, bucketSubnet = 2, 24 `，和IP相关，不清楚其意义。
 + `tableIPLimit, tableSubnet   = 10, 24`，和IP相关，不清楚其意义。
 + `refreshInterval = 30 * time.Minute`，桶的刷新间隔，30分钟。
 + `revalidateInterval = 10 * time.Second`，重新验证时间间隔，10秒。
 + `copyNodesInterval = 30 * time.Second`，拷贝节点间隔，30秒。
 + `seedMinTableTime = 5 * time.Minute`，和种子随机相关，不清楚其意义。
 + `seedCount = 30`，和种子随机相关，不清楚其意义。
 + `seedMaxAge = 5 * 24 * time.Hour`，和种子随机相关，不清楚其意义。

`Table`结构中的变量分析如下：

+ `mutex   sync.Mutex`，互斥锁，保护路由表不被并发读写等。
+ `buckets [nBuckets]*bucket`，桶的列表，长度为`nBuckets`=17。
+ `nursery []*node`，信任的种子节点，一个节点启动的时候首先最多能够连接35个种子节点，其中5个是由以太坊官方提供的，另外30个是从数据库里取的。
+ `rand    *mrand.Rand`，随机量的来源
+ `ips     netutil.DistinctNetSet`，和IP相关，不清楚其意义。
+ `log        log.Logger`，日志器。
+ `db         *enode.DB`，已知节点的数据库，每次寻找节点是耗时费力的，将已知节点持久化，下次客户端启动时可以减小资源消耗。
+ `net        transport`，UDP相关。
+ 下面四个chan是事件通道：
  + `refreshReq chan chan struct{}`
  + `initDone   chan struct{}`
  + `closeReq   chan struct{}`
  + `closed     chan struct{}`
+ `nodeAddedHook func(*node)`测试用。

## bucket的结构和字段

```go
// bucket contains nodes, ordered by their last activity. the entry
// that was most recently active is the first element in entries.
type bucket struct {
   entries      []*node // live entries, sorted by time of last contact
   replacements []*node // recently seen nodes to be used if revalidation fails
   ips          netutil.DistinctNetSet
}
```

一个桶包含两个列表：

+ `entries      []*node`：存储已经发现的节点信息，以上次连接的时间先后排序。
+ `replacements []*node`：当entries是满的时候，新找到的节点不是直接抛弃，而是放到replacement列表。
+ `ips          netutil.DistinctNetSet`：和IP相关，不清楚其意义。

## Table的初始化

```go
func newTable(t transport, db *enode.DB, bootnodes []*enode.Node, log log.Logger) (*Table, error) {
	tab := &Table{
		net:        t,
		db:         db,
		refreshReq: make(chan chan struct{}),
		initDone:   make(chan struct{}),
		closeReq:   make(chan struct{}),
		closed:     make(chan struct{}),
		rand:       mrand.New(mrand.NewSource(0)),
		ips:        netutil.DistinctNetSet{Subnet: tableSubnet, Limit: tableIPLimit},
		log:        log,
	}
	if err := tab.setFallbackNodes(bootnodes); err != nil {
		return nil, err
	}
	for i := range tab.buckets {
		tab.buckets[i] = &bucket{
			ips: netutil.DistinctNetSet{Subnet: bucketSubnet, Limit: bucketIPLimit},
		}
	}
	tab.seedRand()
	tab.loadSeedNodes()

	return tab, nil
}
```

路由表初始化函数，主要做了以下工作：

+ 初始化Table类
+ 加载种子节点
+ 初始化K桶
+ 从table.buckets中随机取30个节点加载种子节点到相应的bucket

## tab.loadSeedNodes()

```go
func (tab *Table) loadSeedNodes() {
	seeds := wrapNodes(tab.db.QuerySeeds(seedCount, seedMaxAge))
	seeds = append(seeds, tab.nursery...)
	for i := range seeds {
		seed := seeds[i]
		age := log.Lazy{Fn: func() interface{} { return time.Since(tab.db.LastPongReceived(seed.ID(), seed.IP())) }}
		tab.log.Trace("Found seed node in database", "id", seed.ID(), "addr", seed.addr(), "age", age)
		tab.addSeenNode(seed)
	}
}
```

加载种子节点，主要做了如下工作：

+ 从数据库里随机选取30个节点（seedCount）
+ 使用table.addSeenNode()方法将每个节点加载到相应的bucket中。

## tab.addSeenNode()

```go
// addSeenNode adds a node which may or may not be live to the end of a bucket. If the
// bucket has space available, adding the node succeeds immediately. Otherwise, the node is
// added to the replacements list.
//
// The caller must not hold tab.mutex.
func (tab *Table) addSeenNode(n *node) {
	if n.ID() == tab.self().ID() {
		return
	}

	tab.mutex.Lock()
	defer tab.mutex.Unlock()
	b := tab.bucket(n.ID())
	if contains(b.entries, n.ID()) {
		// Already in bucket, don't add.
		return
	}
	if len(b.entries) >= bucketSize {
		// Bucket full, maybe add as replacement.
		tab.addReplacement(b, n)
		return
	}
	if !tab.addIP(b, n.IP()) {
		// Can't add: IP limit reached.
		return
	}
	// Add to end of bucket:
	b.entries = append(b.entries, n)
	b.replacements = deleteNode(b.replacements, n)
	n.addedAt = time.Now()
	if tab.nodeAddedHook != nil {
		tab.nodeAddedHook(n)
	}
}
```

增加已发现节点函数，主要做了以下工作：

+ 判断节点是否为本路由表的节点本身。
+ 开启互斥锁，保证不会发生并行读写。
+ 计算要加入节点与路由节点的距离，并找到相应的K桶。
+ 检查此K桶是否已经有此节点。
+ 如果K桶的entries已经满了，就把节点加入到replacement中，return。
+ 将节点IP加入到某列表。(具体还不清楚)
+ 将节点加入对应K桶的entries，并从此K桶的replacement中删除此节点。

## 维护路由表的loop事件监听

```go
// loop schedules runs of doRefresh, doRevalidate and copyLiveNodes.
func (tab *Table) loop() {
	var (
		revalidate     = time.NewTimer(tab.nextRevalidateTime())
		refresh        = time.NewTicker(refreshInterval)
		copyNodes      = time.NewTicker(copyNodesInterval)
		refreshDone    = make(chan struct{})           // where doRefresh reports completion
		revalidateDone chan struct{}                   // where doRevalidate reports completion
		waiting        = []chan struct{}{tab.initDone} // holds waiting callers while doRefresh runs
	)
	defer refresh.Stop()
	defer revalidate.Stop()
	defer copyNodes.Stop()

	// Start initial refresh.
	go tab.doRefresh(refreshDone)

loop:
	for {
		select {
		case <-refresh.C:// 定时刷新k桶事件，refreshInterval=30 min
			tab.seedRand()
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				go tab.doRefresh(refreshDone)
			}
		case req := <-tab.refreshReq:// 刷新k桶的请求事件
			waiting = append(waiting, req)
			if refreshDone == nil {
				refreshDone = make(chan struct{})
				go tab.doRefresh(refreshDone)
			}
		case <-refreshDone:// 刷新k桶的完成事件
			for _, ch := range waiting {
				close(ch)
			}
			waiting, refreshDone = nil, nil
		case <-revalidate.C:// 验证k桶节点有效性，10 second
			revalidateDone = make(chan struct{})
			go tab.doRevalidate(revalidateDone)
		case <-revalidateDone:// 验证k桶节点有效性完成事件
			revalidate.Reset(tab.nextRevalidateTime())
			revalidateDone = nil
		case <-copyNodes.C:// 定时（30秒）将节点存入数据库
			go tab.copyLiveNodes()
		case <-tab.closeReq://结束事件监听事件
			break loop
		}
	}

	if refreshDone != nil {
		<-refreshDone
	}
	for _, ch := range waiting {
		close(ch)
	}
	if revalidateDone != nil {
		<-revalidateDone
	}
	close(tab.closed)
}
```

table.go中生命了table以及相关方法，当启动table时，此loop函数也会被启动用于监听相关事件，主要有刷新K桶，验证K桶等维护K桶的操作。此loop函数首先声明了几个变量：

+ `revalidate`定时器，每10秒钟就ping路由表中的节点，验证节点是否还可用。
+ `refresh`定时器，每30分钟自动刷新k-桶，刷新k-桶可以补充或保持table是满的状态，刚初始化的table可能并不是满的，需要不断的补充和更新。
+ `copyNodes`定时器，每30秒就将k-桶中存在超过5分钟的节点存入本地数据库，视作稳定节点；

+ `refreshDone`通道，用于发送刷新K桶完成事件。
+ `revalidateDone`通道，用于发送验证K桶节点完成事件。
+ `waiting`通道，在刷新K桶时，保留等待的呼叫者（来自其他节点的ping）

然后开始进行事件监听，for循环中主要事件在注释中已经写明。下面主要分析K桶时如何刷新维护的。

## doRefresh()

```go
// doRefresh performs a lookup for a random target to keep buckets full. seed nodes are
// inserted if the table is empty (initial bootstrap or discarded faulty peers).
func (tab *Table) doRefresh(done chan struct{}) {
	defer close(done)

	// Load nodes from the database and insert
	// them. This should yield a few previously seen nodes that are
	// (hopefully) still alive.
	tab.loadSeedNodes()

	// Run self lookup to discover new neighbor nodes.
	tab.net.lookupSelf()

	// The Kademlia paper specifies that the bucket refresh should
	// perform a lookup in the least recently used bucket. We cannot
	// adhere to this because the findnode target is a 512bit value
	// (not hash-sized) and it is not easily possible to generate a
	// sha3 preimage that falls into a chosen bucket.
	// We perform a few lookups with a random target instead.
	for i := 0; i < 3; i++ {
		tab.net.lookupRandom()
	}
}
```

此函数执行了K桶的刷新操作，主要做了以下工作：

+ 将本地受信任节点加入到K桶。
+ 像K桶中已知节点查询一下自己本身的节点ID。
+ 随机生成目标节点进行三次查找。

## doRevalidate()

```go
// doRevalidate checks that the last node in a random bucket is still live and replaces or
// deletes the node if it isn't.
func (tab *Table) doRevalidate(done chan<- struct{}) {
	defer func() { done <- struct{}{} }()

	last, bi := tab.nodeToRevalidate()
	if last == nil {
		// No non-empty bucket found.
		return
	}

	// Ping the selected node and wait for a pong.
	remoteSeq, err := tab.net.ping(unwrapNode(last))

	// Also fetch record if the node replied and returned a higher sequence number.
	if last.Seq() < remoteSeq {
		n, err := tab.net.RequestENR(unwrapNode(last))
		if err != nil {
			tab.log.Debug("ENR request failed", "id", last.ID(), "addr", last.addr(), "err", err)
		} else {
			last = &node{Node: *n, addedAt: last.addedAt, livenessChecks: last.livenessChecks}
		}
	}

	tab.mutex.Lock()
	defer tab.mutex.Unlock()
	b := tab.buckets[bi]
	if err == nil {
		// The node responded, move it to the front.
		last.livenessChecks++
		tab.log.Debug("Revalidated node", "b", bi, "id", last.ID(), "checks", last.livenessChecks)
		tab.bumpInBucket(b, last)
		return
	}
	// No reply received, pick a replacement or delete the node if there aren't
	// any replacements.
	if r := tab.replace(b, last); r != nil {
		tab.log.Debug("Replaced dead node", "b", bi, "id", last.ID(), "ip", last.IP(), "checks", last.livenessChecks, "r", r.ID(), "rip", r.IP())
	} else {
		tab.log.Debug("Removed dead node", "b", bi, "id", last.ID(), "ip", last.IP(), "checks", last.livenessChecks)
	}
}
```

此函数实现了K桶节点的刷新，每一次随机选一个桶，取出桶中最后面的那个节点进行ping操作。主要流程如下：

+ 从随机桶中取出其最后一个节点并对其进行ping操作。
+ 如果节点应答，则将其移至此桶最前端。
+ 如果节点无应答，则使用replacements列表中的节点将其替换或直接从entries中删除此节点。

## 总结

Kademlia协议路由表中K桶的维护主要就是如上所述，并没有提及节点的路由和查找。因为K桶的维护主要由`table.go`执行，而节点的查找由另一个名为lookup的成员定义在`lookup.go`中。

