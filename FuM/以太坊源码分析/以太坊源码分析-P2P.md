# 以太坊源码分析-P2P

[TOC]

## Node

```go
// Node is a container on which services can be registered.
type Node struct {
	eventmux *event.TypeMux // Event multiplexer used between the services of a stack
	config   *Config
	accman   *accounts.Manager

	ephemeralKeystore string            // if non-empty, the key directory that will be removed by Stop
	instanceDirLock   fileutil.Releaser // prevents concurrent use of instance directory

	serverConfig p2p.Config
	server       *p2p.Server // Currently running P2P networking layer

	serviceFuncs []ServiceConstructor     // Service constructors (in dependency order)
	services     map[reflect.Type]Service // Currently running services

	rpcAPIs       []rpc.API   // List of APIs currently provided by the node
	inprocHandler *rpc.Server // In-process RPC request handler to process the API requests

	ipcEndpoint string       // IPC endpoint to listen at (empty = IPC disabled)
	ipcListener net.Listener // IPC RPC listener socket to serve API requests
	ipcHandler  *rpc.Server  // IPC RPC request handler to process the API requests

	httpEndpoint  string       // HTTP endpoint (interface + port) to listen at (empty = HTTP disabled)
	httpWhitelist []string     // HTTP RPC modules to allow through this endpoint
	httpListener  net.Listener // HTTP RPC listener socket to server API requests
	httpHandler   *rpc.Server  // HTTP RPC request handler to process the API requests

	wsEndpoint string       // Websocket endpoint (interface + port) to listen at (empty = websocket disabled)
	wsListener net.Listener // Websocket RPC listener socket to server API requests
	wsHandler  *rpc.Server  // Websocket RPC request handler to process the API requests

	stop chan struct{} // Channel to wait for termination notifications
	lock sync.RWMutex

	log log.Logger
}
```

node在go-ethereum中代表了一个节点。 可能是全节点，可能是轻量级节点。 node可以理解为一个进程，以太坊由运行在世界各地的很多中类型的node组成。一个典型的node就是一个p2p的节点。 运行了p2p网络协议，同时根据节点类型不同，运行了不同的业务层协议(以区别网络层协议。 参考p2p peer中的Protocol接口)。

## 流程图

![](./images/P2P启动流程图.png)



## 启动

以太坊客户端启动时，会根据命令行参数(有无均可)初始化一个全节点`node`，然后调用`startNode(ctx, node)`启动此节点，P2P服务的启动过程也在此开始。

### 1&2.utils.StartNode(stack)&stack.Start()

1和2都是直接调用下层函数，没有发现额外关于P2P的处理事件。



### 3.node.Start()

```go
// Start create a live P2P node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()//锁定读写
	defer n.lock.Unlock()//方法退出是解锁读写锁定

	// Short circuit if the node's already running
	if n.server != nil {
		return ErrNodeRunning
	}
	if err := n.openDataDir(); err != nil {
		return err
	}

	// Initialize the p2p server. This creates the node key and
	// discovery databases.
	n.serverConfig = n.config.P2P
	n.serverConfig.PrivateKey = n.config.NodeKey()//每一个节点都会有一个NodeKey，此文件为{$DataDir}/geth/nodekey，没有的话会自动创建
	n.serverConfig.Name = n.config.NodeName()
	n.serverConfig.Logger = n.log
	if n.serverConfig.StaticNodes == nil {
		n.serverConfig.StaticNodes = n.config.StaticNodes()//静态节点文件，此文件为{$DataDir}/static-nodes.json
	}
	if n.serverConfig.TrustedNodes == nil {
		n.serverConfig.TrustedNodes = n.config.TrustedNodes()//信任节点文件，此文件为{$DataDir}/trusted-nodes.json
	}
	if n.serverConfig.NodeDatabase == "" {
		n.serverConfig.NodeDatabase = n.config.NodeDB()//节点数据库文件目录，此目录为{$DataDir}/geth/nodes
	}
	//创建p2p服务器
	running := &p2p.Server{Config: n.serverConfig}
	n.log.Info("Starting peer-to-peer node", "instance", n.serverConfig.Name)

	// Otherwise copy and specialize the P2P configuration
	services := make(map[reflect.Type]Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular service
		ctx := &ServiceContext{
			Config:         *n.config,
			services:       make(map[reflect.Type]Service),
			EventMux:       n.eventmux,
			AccountManager: n.accman,
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.services[kind] = s
		}
		// Construct and save the service
		// 创建所有注册的服务
		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(service)
		if _, exists := services[kind]; exists {
			return &DuplicateServiceError{Kind: kind}
		}
		services[kind] = service
	}
	// Gather the protocols and start the freshly assembled P2P server
	// 收集所有的p2p的protocols并插入p2p.Rrotocols
	for _, service := range services {
		running.Protocols = append(running.Protocols, service.Protocols()...)
	}
	//启动p2p服务器
	if err := running.Start(); err != nil {
		return convertFileLockError(err)
	}
	// Start each of the services
	var started []reflect.Type
	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		// 启动每一个服务
		if err := service.Start(running); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}
			running.Stop()

			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}
	// Lastly start the configured RPC interfaces
	// 启动RPC服务
	if err := n.startRPC(services); err != nil {
		for _, service := range services {
			service.Stop()
		}
		running.Stop()
		return err
	}
	// Finish initializing the startup
	n.services = services
	n.server = running
	n.stop = make(chan struct{})
	return nil
}
```

一个node节点可以拥有不同的功能，这些功能不是New的时候就已经存在了，而是需要去注册服务。所以上述方法的主要流程如下：

+ 初始化P2P节点中的节点私钥，已存静态节点，受信任节点和数据库
+ 创建P2P服务器
+ 创建P2P节点所注册的所有服务
+ 启动P2P服务器`图中4`
+ 启动P2P节点所注册的所有服务
+ 启动RPC服务

### 4.p2p.server.start()

```go
// Start starts running the server.
// Servers can not be re-used after stopping.
func (srv *Server) Start() (err error) {
	srv.lock.Lock()//锁定读写
	defer srv.lock.Unlock()//方法退出是解锁读写锁定
	if srv.running {
		return errors.New("server already running")
	}
	srv.running = true
	srv.log = srv.Config.Logger
	if srv.log == nil {
		srv.log = log.Root()
	}
	if srv.clock == nil {
		srv.clock = mclock.System{}
	}
	if srv.NoDial && srv.ListenAddr == "" {
		srv.log.Warn("P2P server will be useless, neither dialing nor listening")
	}

	// static fields
	if srv.PrivateKey == nil {
		return errors.New("Server.PrivateKey must be set to a non-nil key")
	}
	if srv.newTransport == nil {
		srv.newTransport = newRLPX//RLPX通信的实现
	}
	if srv.listenFunc == nil {
		srv.listenFunc = net.Listen//监听
	}
	srv.quit = make(chan struct{})//退出通道
	srv.delpeer = make(chan peerDrop)//删除节点通道
	srv.checkpointPostHandshake = make(chan *conn)//连接已通过加密握手，因此可以知道远程身份（但尚未验证）。
	srv.checkpointAddPeer = make(chan *conn)//连接已通过协议握手。 它的功能已知，并且远程身份已验证。
	srv.addtrusted = make(chan *enode.Node)//添加信任节点
	srv.removetrusted = make(chan *enode.Node)//移除信任节点
	srv.peerOp = make(chan peerOpFunc)
	srv.peerOpDone = make(chan struct{})
	//初始化本地节点
	if err := srv.setupLocalNode(); err != nil {
		return err
	}
	//设置tcp监听服务
	if srv.ListenAddr != "" {
		if err := srv.setupListening(); err != nil {
			return err
		}
	}
	//设置udp监听服务
	if err := srv.setupDiscovery(); err != nil {
		return err
	}
	srv.setupDialScheduler()//设置对外广播消息服务

	srv.loopWG.Add(1)
	go srv.run()
	return nil
}
```

此函数启动了P2P服务器，主要做了以下工作：

+ 使用读写锁保证此服务器只启动一次
+ 初始化日志和时钟对象，检查节点私钥，初始化RLPX通信的实现和监听实例对象
+ 创建一系列通道作为事件接收器，包括
  + 本地服务停止
  + 删除远程节点
  + 发现已通过加密握手的远程连接
  + 发现已通过协议握手的远程连接
  + 新增信任节点
  + 移除信任节点
  + peerOp和peerOpDone目前还不知道用途
+ `srv.setupLocalNode()`初始化本地节点
+ `srv.setupListening()`设置tcp监听服务
+ `srv.setupDiscovery()`设置ump监听服务
+ `srv.setupDialScheduler()`设置对外广播消息服务
+ `go srv.run()`启动协程，监听上面所提到的事件

下面主要说一下`srv.setupLocalNode()`、`srv.setupListening()`、`srv.setupDiscovery()`、`srv.setupDialScheduler()`和`go srv.run()`

### 5.srv.setupLocalNode()

```go
func (srv *Server) setupLocalNode() error {
	// Create the devp2p handshake.
	pubkey := crypto.FromECDSAPub(&srv.PrivateKey.PublicKey)
	srv.ourHandshake = &protoHandshake{Version: baseProtocolVersion, Name: srv.Name, ID: pubkey[1:]}
	for _, p := range srv.Protocols {
		srv.ourHandshake.Caps = append(srv.ourHandshake.Caps, p.cap())
	}
	sort.Sort(capsByNameAndVersion(srv.ourHandshake.Caps))

	// Create the local node.
	// 打开一个节点数据库，用于存储和检索有关网络中已知对等方的信息。
	db, err := enode.OpenDB(srv.Config.NodeDatabase)//Config.NodeDatabase为节点数据库文件目录，此目录为{$DataDir}/geth/nodes
	if err != nil {
		return err
	}
	srv.nodedb = db
	srv.localnode = enode.NewLocalNode(db, srv.PrivateKey)
	srv.localnode.SetFallbackIP(net.IP{127, 0, 0, 1})
	// TODO: check conflicts
	for _, p := range srv.Protocols {
		for _, e := range p.Attributes {
			srv.localnode.Set(e)
		}
	}
	switch srv.NAT.(type) {
	case nil:
		// No NAT interface, do nothing.
	case nat.ExtIP:
		// ExtIP doesn't block, set the IP right away.
		ip, _ := srv.NAT.ExternalIP()
		srv.localnode.SetStaticIP(ip)
	default:
		// Ask the router about the IP. This takes a while and blocks startup,
		// do it in the background.
		srv.loopWG.Add(1)
		go func() {
			defer srv.loopWG.Done()
			if ip, err := srv.NAT.ExternalIP(); err == nil {
				srv.localnode.SetStaticIP(ip)
			}
		}()
	}
	return nil
}
```

此函数创建本地的node，主要做了以下工作：

+ 配置节点protoHandshake结构（version, name, ID(即公钥pubkey)，caps（即支持的上层协议及版本））
+ 打开一个db，读取并存储网络中已知节点的信息。
+ 对这个节点设置IP地址。

### 6.srv.setupListening()

```go
func (srv *Server) setupListening() error {
	// Launch the listener.
	// 启动tcp监听
	listener, err := srv.listenFunc("tcp", srv.ListenAddr)
	if err != nil {
		return err
	}
	srv.listener = listener
	srv.ListenAddr = listener.Addr().String()

	// Update the local node record and map the TCP listening port if NAT is configured.
	if tcp, ok := listener.Addr().(*net.TCPAddr); ok {
		srv.localnode.Set(enr.TCP(tcp.Port))
		if !tcp.IP.IsLoopback() && srv.NAT != nil {
			srv.loopWG.Add(1)
			go func() {
				nat.Map(srv.NAT, srv.quit, "tcp", tcp.Port, tcp.Port, "ethereum p2p")
				srv.loopWG.Done()
			}()
		}
	}

	srv.loopWG.Add(1)
	// 使用协程持续监听消息
	go srv.listenLoop()
	return nil
}
```

此函数启动节点的tcp监听，主要做了以下工作：

+ 启动tcp监听
+ 使用协程持续监听消息

### 7.srv.setupDiscovery()

```go
func (srv *Server) setupDiscovery() error {
	srv.discmix = enode.NewFairMix(discmixTimeout)

	// Add protocol-specific discovery sources.
	added := make(map[string]bool)
	for _, proto := range srv.Protocols {
		if proto.DialCandidates != nil && !added[proto.Name] {
			srv.discmix.AddSource(proto.DialCandidates)
			added[proto.Name] = true
		}
	}

	// Don't listen on UDP endpoint if DHT is disabled.
	if srv.NoDiscovery && !srv.DiscoveryV5 {
		return nil
	}
	//产生udp地址
	addr, err := net.ResolveUDPAddr("udp", srv.ListenAddr)
	if err != nil {
		return err
	}
	//获取监听udp端口的实例对象
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	realaddr := conn.LocalAddr().(*net.UDPAddr)
	//启动协程监听udp
	srv.log.Debug("UDP listener up", "addr", realaddr)
	if srv.NAT != nil {
		if !realaddr.IP.IsLoopback() {
			go nat.Map(srv.NAT, srv.quit, "udp", realaddr.Port, realaddr.Port, "ethereum discovery")
		}
	}
	srv.localnode.SetFallbackUDP(realaddr.Port)
	//启动节点发现功能，版本为4
	// Discovery V4
	var unhandled chan discover.ReadPacket
	var sconn *sharedUDPConn
	if !srv.NoDiscovery {
		if srv.DiscoveryV5 {
			unhandled = make(chan discover.ReadPacket, 100)
			sconn = &sharedUDPConn{conn, unhandled}
		}
		cfg := discover.Config{
			PrivateKey:  srv.PrivateKey,
			NetRestrict: srv.NetRestrict,
			Bootnodes:   srv.BootstrapNodes,
			Unhandled:   unhandled,
			Log:         srv.log,
		}
		ntab, err := discover.ListenUDP(conn, srv.localnode, cfg)
		if err != nil {
			return err
		}
		srv.ntab = ntab
		srv.discmix.AddSource(ntab.RandomNodes())
	}
	//启动节点发现功能，版本为5
	// Discovery V5
	if srv.DiscoveryV5 {
		var ntab *discv5.Network
		var err error
		if sconn != nil {
			ntab, err = discv5.ListenUDP(srv.PrivateKey, sconn, "", srv.NetRestrict)
		} else {
			ntab, err = discv5.ListenUDP(srv.PrivateKey, conn, "", srv.NetRestrict)
		}
		if err != nil {
			return err
		}
		if err := ntab.SetFallbackNodes(srv.BootstrapNodesV5); err != nil {
			return err
		}
		srv.DiscV5 = ntab
	}
	return nil
}
```

此函数主要是启动了udp服务，供节点发现等节点通信，主要做了以下工作：

+ 监听本地udp端口
+ 启动节点发现服务，目前服务有两个版本（V4和V5），应该是正在进行过渡

### 8.srv.setupDialScheduler()

```go
func (srv *Server) setupDialScheduler() {
	config := dialConfig{
		self:           srv.localnode.ID(),
		maxDialPeers:   srv.maxDialedConns(),
		maxActiveDials: srv.MaxPendingPeers,
		log:            srv.Logger,
		netRestrict:    srv.NetRestrict,
		dialer:         srv.Dialer,
		clock:          srv.clock,
	}
	if srv.ntab != nil {
		config.resolver = srv.ntab
	}
	if config.dialer == nil {
		config.dialer = tcpDialer{&net.Dialer{Timeout: defaultDialTimeout}}
	}
	srv.dialsched = newDialScheduler(config, srv.discmix, srv.SetupConn)
	for _, n := range srv.StaticNodes {
		srv.dialsched.addStatic(n)
	}
}
```

此函数初始化了一个拨号客户端，主动向node拨号，主要做了如下工作：

+ 生成dialConfig客户端config，声明了以下值
  + 我方节点ID
  + 同时拨号的最大数量
  + 最大已连接节点数
  + 日志器
  + 白名单`netRestrict`
  + 拨号器
  + 时钟
  + 静态节点列表
+ 静态节点是预设节点，存储在文件中，即使断开连接，本地节点也会一直尝试重新连接，而不会放弃该节点

### 9.srv.run()

`srv.run()`在`node.Start()`中以协程形式运行

> ```go
> srv.loopWG.Add(1)
> go srv.run()
> ```

先说一下`srv.loopWG.Add(1)`的作用。这个句子通常出现在在协程运行前。`loopWG`属于`WaitGroup`类

`WaitGroup`等待`goroutine`的集合完成。 主线程调用Add来设置自己所启动并等待的goroutine的数量。然后，每个协程运行并在完成后调用`Done`。同时，等待可以用来阻塞，直到所有goroutine完成。

下面是`srv.run()`的实现代码，可以看到在最开始就声明了`defer srv.loopWG.Done()`，也就是说在方法退出时会调用`Done`。

```go
// run is the main loop of the server.
func (srv *Server) run() {
	srv.log.Info("Started P2P networking", "self", srv.localnode.Node().URLv4())
	defer srv.loopWG.Done()
	defer srv.nodedb.Close()
	defer srv.discmix.Close()
	defer srv.dialsched.stop()

	var (
		peers        = make(map[enode.ID]*Peer)
		inboundCount = 0
		trusted      = make(map[enode.ID]bool, len(srv.TrustedNodes))
	)
	// Put trusted nodes into a map to speed up checks.
	// Trusted peers are loaded on startup or added via AddTrustedPeer RPC.
	for _, n := range srv.TrustedNodes {
		trusted[n.ID()] = true
	}

running:
	for {
		select {
		case <-srv.quit:
			// The server was stopped. Run the cleanup logic.
			break running

		case n := <-srv.addtrusted:
			// This channel is used by AddTrustedPeer to add a node
			// to the trusted node set.
			srv.log.Trace("Adding trusted node", "node", n)
			trusted[n.ID()] = true
			if p, ok := peers[n.ID()]; ok {
				p.rw.set(trustedConn, true)
			}

		case n := <-srv.removetrusted:
			// This channel is used by RemoveTrustedPeer to remove a node
			// from the trusted node set.
			srv.log.Trace("Removing trusted node", "node", n)
			delete(trusted, n.ID())
			if p, ok := peers[n.ID()]; ok {
				p.rw.set(trustedConn, false)
			}

		case op := <-srv.peerOp:
			// This channel is used by Peers and PeerCount.
			op(peers)
			srv.peerOpDone <- struct{}{}

		case c := <-srv.checkpointPostHandshake:
			// A connection has passed the encryption handshake so
			// the remote identity is known (but hasn't been verified yet).
			if trusted[c.node.ID()] {
				// Ensure that the trusted flag is set before checking against MaxPeers.
				c.flags |= trustedConn
			}
			// TODO: track in-progress inbound node IDs (pre-Peer) to avoid dialing them.
			c.cont <- srv.postHandshakeChecks(peers, inboundCount, c)

		case c := <-srv.checkpointAddPeer:
			// At this point the connection is past the protocol handshake.
			// Its capabilities are known and the remote identity is verified.
			err := srv.addPeerChecks(peers, inboundCount, c)
			if err == nil {
				// The handshakes are done and it passed all checks.
				p := srv.launchPeer(c)
				peers[c.node.ID()] = p
				srv.log.Debug("Adding p2p peer", "peercount", len(peers), "id", p.ID(), "conn", c.flags, "addr", p.RemoteAddr(), "name", truncateName(c.name))
				srv.dialsched.peerAdded(c)
				if p.Inbound() {
					inboundCount++
				}
			}
			c.cont <- err

		case pd := <-srv.delpeer:
			// A peer disconnected.
			d := common.PrettyDuration(mclock.Now() - pd.created)
			delete(peers, pd.ID())
			srv.log.Debug("Removing p2p peer", "peercount", len(peers), "id", pd.ID(), "duration", d, "req", pd.requested, "err", pd.err)
			srv.dialsched.peerRemoved(pd.rw)
			if pd.Inbound() {
				inboundCount--
			}
		}
	}

	srv.log.Trace("P2P networking is spinning down")

	// Terminate discovery. If there is a running lookup it will terminate soon.
	if srv.ntab != nil {
		srv.ntab.Close()
	}
	if srv.DiscV5 != nil {
		srv.DiscV5.Close()
	}
	// Disconnect all peers.
	for _, p := range peers {
		p.Disconnect(DiscQuitting)
	}
	// Wait for peers to shut down. Pending connections and tasks are
	// not handled here and will terminate soon-ish because srv.quit
	// is closed.
	for len(peers) > 0 {
		p := <-srv.delpeer
		p.log.Trace("<-delpeer (spindown)")
		delete(peers, p.ID())
	}
}
```

此函数是P2P服务节点发现的主要服务，主要做了以下工作：

+ 声明了3个变量
  + `peers`:以节点ID-节点的K-V方式存储节点信息，包括信任节点
  + `inboundCount` :已连接节点数量
  + `trusted`:信任节点的信息，以节点ID-节点的K-V方式存储
+ 开启主要循环，主要接收并处理一下几个事件
  + 接收到退出指令，结束循环，执行循环后面的清除代码
  + 有节点需要加入到信任节点列表中
  + 从信任节点列表中移除某节点
  + 节点握手过程中的一个环节：已通过加密握手，远程身份是已知的，但尚未经过验证
  + 节点握手过程中的一个环节：已通过加密握手，已知其功能并验证了远程身份。
  + 节点失联，需要删除节点
+ 服务关闭，主要做了以下几个清除工作：
  + 关闭udp4端口
  + 关闭节点路由表
  + 与所有已知节点断开连接

节点发现服务启动之后，便开始全面启动p2p通信

### 10.service.Start(running)

// Start实现node.Service，启动以太坊协议实现所需的所有内部goroutine。

```go
// Start implements node.Service, starting all internal goroutines needed by the
// Ethereum protocol implementation.
func (s *Ethereum) Start(srvr *p2p.Server) error {
	s.startEthEntryUpdate(srvr.LocalNode())

	// Start the bloom bits servicing goroutines
	s.startBloomHandlers(params.BloomBitsBlocks)

	// Start the RPC service
	s.netRPCService = ethapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}
```

此函数开始启动P2P通信，主要做了以下工作：

+ 启动ENR更新程序循环（暂不清楚用途）
+ 启动布隆过滤器（暂不清楚用途）
+ 启动RPC服务
+ 根据server的设置计算还能连接的最大可连接节点数量，计算方法如下：
  + 如果已连接的节点数（如静态节点和信任节点）已经大于最大可连接节点数量，则会报错
  + 否则，最大可连接节点数量等于server的设置最大可连接节点数量减去已连接的节点数
+ 开启网络层`	s.protocolManager.Start(maxPeers)`
+ 如果指定要求开启轻节点服务，则会启动lesServer，即Light Ethereum Server

### 11.s.protocolManager.Start(maxPeers)

```go
func (pm *ProtocolManager) Start(maxPeers int) {
	pm.maxPeers = maxPeers

	// broadcast transactions
	pm.txsCh = make(chan core.NewTxsEvent, txChanSize)
	pm.txsSub = pm.txpool.SubscribeNewTxsEvent(pm.txsCh)
	go pm.txBroadcastLoop()

	// broadcast mined blocks
	pm.minedBlockSub = pm.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go pm.minedBroadcastLoop()

	// start sync handlers
	go pm.syncer()
	go pm.txsyncLoop64() // TODO(karalabe): Legacy initial tx echange, drop with eth/64.
}
```

此方法是P2P网络层的核心代码，启动了4个协程来监听网络事件。此方法主要做了以下工作：

+ 拿到最大可连接节点数，也就是上面计算出的还能连接的最大可连接节点数量
+ 建立一个广播交易的通道后从交易池订阅一个广播交易的事件，然后启动**交易广播协程**
+ 订阅一个新块产生的事件，然后启动**新区快广播协程**
+ 启动最后两个协程：
  + 同步器，定期与网络进行同步，下载哈希和区块
  + 新节点连接同步器，当新节点连接到我方节点时触发操作

下面主要分析上述四个协程的主要功能：

#### 11.1 pm.txBroadcastLoop()

```go
func (pm *ProtocolManager) txBroadcastLoop() {
	for {
		select {
		case event := <-pm.txsCh:
			// For testing purpose only, disable propagation
			if pm.broadcastTxAnnouncesOnly {
				pm.BroadcastTransactions(event.Txs, false)
				continue
			}
			pm.BroadcastTransactions(event.Txs, true)  // First propagate transactions to peers
			pm.BroadcastTransactions(event.Txs, false) // Only then announce to the rest

		// Err() channel will be closed when unsubscribing.
		case <-pm.txsSub.Err():
			return
		}
	}
}
```

此函数是交易广播的入口函数，主要做了以下工作：

+ 有一个测试模块，只会在测试时触发。

+ `pm.BroadcastTransactions(event.Txs, true)`

+ `pm.BroadcastTransactions(event.Txs, false) `

  下面分析一下`BroadcastTransactions()`方法

```go
// BroadcastTransactions will propagate a batch of transactions to all peers which are not known to
// already have the given transaction.
func (pm *ProtocolManager) BroadcastTransactions(txs types.Transactions, propagate bool) {
	var (
		txset = make(map[*peer][]common.Hash)
		annos = make(map[*peer][]common.Hash)
	)
	// Broadcast transactions to a batch of peers not knowing about it
	if propagate {
		for _, tx := range txs {
			peers := pm.peers.PeersWithoutTx(tx.Hash())

			// Send the block to a subset of our peers
			transfer := peers[:int(math.Sqrt(float64(len(peers))))]
			for _, peer := range transfer {
				txset[peer] = append(txset[peer], tx.Hash())
			}
			log.Trace("Broadcast transaction", "hash", tx.Hash(), "recipients", len(peers))
		}
		for peer, hashes := range txset {
			peer.AsyncSendTransactions(hashes)
		}
		return
	}
	// Otherwise only broadcast the announcement to peers
	for _, tx := range txs {
		peers := pm.peers.PeersWithoutTx(tx.Hash())
		for _, peer := range peers {
			annos[peer] = append(annos[peer], tx.Hash())
		}
	}
	for peer, hashes := range annos {
		if peer.version >= eth65 {
			peer.AsyncSendPooledTransactionHashes(hashes)
		} else {
			peer.AsyncSendTransactions(hashes)
		}
	}
}
```

此函数会将一批交易广播给所有未拥有这些交易的节点，主要做了如下工作：

+ 生成一个交易集`txset`和通知集`annos`，均为map，K:节点 V:交易哈希
+ 若传播变量`propagate`为true
  + 获取自己已知节点中未拥有这些交易的节点peers[]
  + 向节点`peers[:int(math.Sqrt(float64(len(peers))))]`发送交易的哈希
+ 若传播变量`propagate`为false
  + 获取自己已知节点中未拥有这些交易的节点peers[]
  + 向`peers[]`中所有节点发送交易的哈希

由上层调用可以看出来，交易通过`pm.BroadcastTransactions(event.Txs, true)`和`pm.BroadcastTransactions(event.Txs, false) `被广播两遍。也就是一批交易分两次进行了广播，第一次广播给了节点的子集，剩下的节点通过第二次广播才能接收到交易。这么做的目的是减少本节点的瞬时网络负载，免得一次广播太多的`tx`造成自己的网络拥堵。

#### 11.2 pm.minedBroadcastLoop()

```go
// Mined broadcast loop
func (pm *ProtocolManager) minedBroadcastLoop() {
	// automatically stops if unsubscribe
	for obj := range pm.minedBlockSub.Chan() {
		if ev, ok := obj.Data.(core.NewMinedBlockEvent); ok {
			pm.BroadcastBlock(ev.Block, true)  // First propagate block to peers
			pm.BroadcastBlock(ev.Block, false) // Only then announce to the rest
		}
	}
}
```

此函数以协程形式执行，当接收到pm.minedBlockSub.Chan()管道的通知时，开始广播新出的区块，只有一个区块。可以从

>```go
>		pm.BroadcastBlock(ev.Block, true)  // First propagate block to peers
>		pm.BroadcastBlock(ev.Block, false) // Only then announce to the rest
>```

看出，广播区块的过程也是先向一小部分节点广播区块，再向剩下的节点广播区块。这么做的目的是减少本节点的瞬时网络负载，免得一次广播太多的block造成自己的网络拥堵。

下面看`pm.BroadcastBlock()`函数

```go
// BroadcastBlock will either propagate a block to a subset of its peers, or
// will only announce its availability (depending what's requested).
func (pm *ProtocolManager) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := pm.peers.PeersWithoutBlock(hash)

	// If propagation is requested, send to a subset of the peer
	if propagate {
		// Calculate the TD of the block (it's not imported yet, so block.Td is not valid)
		var td *big.Int
		if parent := pm.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1); parent != nil {
			td = new(big.Int).Add(block.Difficulty(), pm.blockchain.GetTd(block.ParentHash(), block.NumberU64()-1))
		} else {
			log.Error("Propagating dangling block", "number", block.Number(), "hash", hash)
			return
		}
		// Send the block to a subset of our peers
		transfer := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range transfer {
			peer.AsyncSendNewBlock(block, td)
		}
		log.Trace("Propagated block", "hash", hash, "recipients", len(transfer), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
		return
	}
	// Otherwise if the block is indeed in out own chain, announce it
	if pm.blockchain.HasBlock(hash, block.NumberU64()) {
		for _, peer := range peers {
			peer.AsyncSendNewBlockHash(block)
		}
		log.Trace("Announced block", "hash", hash, "recipients", len(peers), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
	}
}
```

此函数是区块广播的主要实现，主要做了以下工作：

+ 取哈希后获取自己已知节点中未拥有此区块的节点peers[]
+ 若传播变量`propagate`为true
  + 先判断区块的父区块是否存在，不存在要向上抛错
  + 根据本区块难度和父区块td计算本区块td（td：total difficulty，从区块链的第一个区块到此区块的难度的总和。）
  + 向节点`peers[:int(math.Sqrt(float64(len(peers))))]`发送区块
+ 若传播变量`propagate`为false
  + 向`peers[]`中所有节点发送区块

#### 11.3 pm.syncer()

```go
// syncer is responsible for periodically synchronising with the network, both
// downloading hashes and blocks as well as handling the announcement handler.
func (pm *ProtocolManager) syncer() {
	// Start and ensure cleanup of sync mechanisms
	pm.blockFetcher.Start()
	pm.txFetcher.Start()
	defer pm.blockFetcher.Stop()
	defer pm.txFetcher.Stop()
	defer pm.downloader.Terminate()

	// Wait for different events to fire synchronisation operations
	forceSync := time.NewTicker(forceSyncCycle)
	defer forceSync.Stop()

	for {
		select {
		case <-pm.newPeerCh:
			// Make sure we have peers to select from, then sync
			if pm.peers.Len() < minDesiredPeerCount {
				break
			}
			go pm.synchronise(pm.peers.BestPeer())

		case <-forceSync.C:
			// Force a sync even if not enough peers are present
			go pm.synchronise(pm.peers.BestPeer())

		case <-pm.noMorePeers:
			return
		}
	}
}
```

此函数负责定期与网络同步，获取交易哈希与块，主要做了以下工作：

+ 开启区块接收器
+ 开启交易接收器
+ 声明一个定时器，时间为10秒
+ 开启for循环接收事件，主要接收如下事件
  + 有新节点加入本地已知节点列表时，触发`case <-pm.newPeerCh:`。如果本节点所连接的节点数少于`minDesiredPeerCount=5`，则不进行同步。否则，执行协程将与已知td最高的节点同步。
  + 每10秒，触发`case <-forceSync.C:`,与已知td最高的节点同步一次。
  + 如果本地已知节点列表一个节点都没有，直接退出整个方法。

#### 11.4 pm.txsyncLoop64()

```go
// txsyncLoop64 takes care of the initial transaction sync for each new
// connection. When a new peer appears, we relay all currently pending
// transactions. In order to minimise egress bandwidth usage, we send
// the transactions in small packs to one peer at a time.
func (pm *ProtocolManager) txsyncLoop64() {
	var (
		pending = make(map[enode.ID]*txsync)
		sending = false               // whether a send is active
		pack    = new(txsync)         // the pack that is being sent
		done    = make(chan error, 1) // result of the send
	)
	// send starts a sending a pack of transactions from the sync.
	send := func(s *txsync) {
		if s.p.version >= eth65 {
			panic("initial transaction syncer running on eth/65+")
		}
		// Fill pack with transactions up to the target size.
		size := common.StorageSize(0)
		pack.p = s.p
		pack.txs = pack.txs[:0]
		for i := 0; i < len(s.txs) && size < txsyncPackSize; i++ {
			pack.txs = append(pack.txs, s.txs[i])
			size += s.txs[i].Size()
		}
		// Remove the transactions that will be sent.
		s.txs = s.txs[:copy(s.txs, s.txs[len(pack.txs):])]
		if len(s.txs) == 0 {
			delete(pending, s.p.ID())
		}
		// Send the pack in the background.
		s.p.Log().Trace("Sending batch of transactions", "count", len(pack.txs), "bytes", size)
		sending = true
		go func() { done <- pack.p.SendTransactions64(pack.txs) }()
	}

	// pick chooses the next pending sync.
	pick := func() *txsync {
		if len(pending) == 0 {
			return nil
		}
		n := rand.Intn(len(pending)) + 1
		for _, s := range pending {
			if n--; n == 0 {
				return s
			}
		}
		return nil
	}

	for {
		select {
		case s := <-pm.txsyncCh:
			pending[s.p.ID()] = s
			if !sending {
				send(s)
			}
		case err := <-done:
			sending = false
			// Stop tracking peers that cause send failures.
			if err != nil {
				pack.p.Log().Debug("Transaction send failed", "err", err)
				delete(pending, pack.p.ID())
			}
			// Schedule the next send.
			if s := pick(); s != nil {
				send(s)
			}
		case <-pm.quitSync:
			return
		}
	}
}
```

此方法处理新节点连接是的同步事务，此方法会转发交易池中Pending的交易，为了尽可能的降低宽带占用，这些交易使用多个packs进行包装并一次送出。此方法具体做了如下工作：

> 此方法声明了4个变量，两个匿名函数，之后便启动了for循环接收事件，我们直接分析for循环。

+ 当新节点需要同步交易时，`case s := <-pm.txsyncCh:`触发，`s`中含有新节点和需同步的交易信息。
  + 将s以节点ID(公钥)为key存入名为pending的map中
  + 如果当前没有在运行send函数，则运行send匿名函数，否则什么都不做
+ 当上述同步出错，`case err := <-done:`触发，从pending中删除相关信息，通过pick匿名函数确定下一次发送的时间。
+ 如果收到外部退出指令，`case <-pm.quitSync:`触发，直接退出此方法。