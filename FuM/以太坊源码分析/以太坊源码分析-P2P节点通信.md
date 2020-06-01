# 以太坊源码分析-P2P节点通信

在**以太坊源码分析-P2P结构**中，我们提到了有一个重要的协程srv.run()。

此协程主要处理了与新节点的初次连接操作，如下：

+ 有节点需要加入到信任节点列表中
+ 从信任节点列表中移除某节点
+ 节点握手过程中的一个环节：已通过加密握手，远程身份是已知的，但尚未经过验证
+ 节点握手过程中的一个环节：已通过加密握手，已知其功能并验证了远程身份。
+ 节点失联，需要删除节点

下面主要说一下与已通过加密握手，已知其功能并验证了远程身份的节点的通信建立：

```go
		// 节点握手过程中的一个环节：已通过加密握手，已知其功能并验证了远程身份。
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
```

+ 再对节点身份进行一次握手验证
+ `srv.launchPeer(c)`将远程节点实例化并启动
+ 更新已知节点集

下面看下节点启动过程

### srv.launchPeer(c)

```go
func (srv *Server) launchPeer(c *conn) *Peer {
	p := newPeer(srv.log, c, srv.Protocols)
	if srv.EnableMsgEvents {
		// If message events are enabled, pass the peerFeed
		// to the peer.
		p.events = &srv.peerFeed
	}
	go srv.runPeer(p)
	return p
}
```

此函数讲peer实例化之后，创建了协程：`go srv.runPeer(p)`

### srv.runPeer(p) 协程

此协程调用了`remoteRequested, err := p.run()`，进行下一步的启动。

### p.run()

此函数启动了此节点的协议处理对象，即`p.startProtocols(writeStart, writeErr)`

### p.startProtocols(writeStart, writeErr)

```go
func (p *Peer) startProtocols(writeStart <-chan struct{}, writeErr chan<- error) {
	p.wg.Add(len(p.running))
	for _, proto := range p.running {
		proto := proto
		proto.closed = p.closed
		proto.wstart = writeStart
		proto.werr = writeErr
		var rw MsgReadWriter = proto
		if p.events != nil {
			rw = newMsgEventer(rw, p.events, p.ID(), proto.Name, p.Info().Network.RemoteAddress, p.Info().Network.LocalAddress)
		}
		p.log.Trace(fmt.Sprintf("Starting protocol %s/%d", proto.Name, proto.Version))
		go func() {
			err := proto.Run(p, rw)
			if err == nil {
				p.log.Trace(fmt.Sprintf("Protocol %s/%d returned", proto.Name, proto.Version))
				err = errProtocolReturned
			} else if err != io.EOF {
				p.log.Trace(fmt.Sprintf("Protocol %s/%d failed", proto.Name, proto.Version), "err", err)
			}
			p.protoErr <- err
			p.wg.Done()
		}()
	}
}
```

主要看下面的协程匿名函数，调用了`proto.Run(p, rw)`。

### proto.Run(p, rw)

此函数在**以太坊源码分析-P2P结构**中的流程图中的`makeProtocol()`函数中体现，下面是此函数的具体实现

```go
func (pm *ProtocolManager) makeProtocol(version uint) p2p.Protocol {
	length, ok := protocolLengths[version]
	if !ok {
		panic("makeProtocol for unknown version")
	}

	return p2p.Protocol{
		Name:    protocolName,
		Version: version,
		Length:  length,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			peer := pm.newPeer(int(version), p, rw, pm.txpool.Get)
			select {
			case pm.newPeerCh <- peer:
				pm.wg.Add(1)
				defer pm.wg.Done()
				return pm.handle(peer)
			case <-pm.quitSync:
				return p2p.DiscQuitting
			}
		},
		NodeInfo: func() interface{} {
			return pm.NodeInfo()
		},
		PeerInfo: func(id enode.ID) interface{} {
			if p := pm.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
				return p.Info()
			}
			return nil
		},
	}
}
```

可以看出返回的结构体中有Run函数，此函数在这种情况下被调用。最重要的是Run函数在新节点加入时所返回的`pm.handle(peer)`

### pm.handle(peer)

```go
// handle is the callback invoked to manage the life cycle of an eth peer. When
// this function terminates, the peer is disconnected.
func (pm *ProtocolManager) handle(p *peer) error {
	// Ignore maxPeers if this is a trusted peer
	if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("Ethereum peer connected", "name", p.Name())

	// Execute the Ethereum handshake
	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
		hash    = head.Hash()
		number  = head.Number.Uint64()
		td      = pm.blockchain.GetTd(hash, number)
	)
	if err := p.Handshake(pm.networkID, td, hash, genesis.Hash(), forkid.NewID(pm.blockchain), pm.forkFilter); err != nil {
		p.Log().Debug("Ethereum handshake failed", "err", err)
		return err
	}
	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("Ethereum peer registration failed", "err", err)
		return err
	}
	defer pm.removePeer(p.id)

	// Register the peer in the downloader. If the downloader considers it banned, we disconnect
	if err := pm.downloader.RegisterPeer(p.id, p.version, p); err != nil {
		return err
	}
	// Propagate existing transactions. new transactions appearing
	// after this will be sent via broadcasts.
	pm.syncTransactions(p)

	// If we have a trusted CHT, reject all peers below that (avoid fast sync eclipse)
	if pm.checkpointHash != (common.Hash{}) {
		// Request the peer's checkpoint header for chain height/weight validation
		if err := p.RequestHeadersByNumber(pm.checkpointNumber, 1, 0, false); err != nil {
			return err
		}
		// Start a timer to disconnect if the peer doesn't reply in time
		p.syncDrop = time.AfterFunc(syncChallengeTimeout, func() {
			p.Log().Warn("Checkpoint challenge timed out, dropping", "addr", p.RemoteAddr(), "type", p.Name())
			pm.removePeer(p.id)
		})
		// Make sure it's cleaned up if the peer dies off
		defer func() {
			if p.syncDrop != nil {
				p.syncDrop.Stop()
				p.syncDrop = nil
			}
		}()
	}
	// If we have any explicit whitelist block hashes, request them
	for number := range pm.whitelist {
		if err := p.RequestHeadersByNumber(number, 1, 0, false); err != nil {
			return err
		}
	}
	// Handle incoming messages until the connection is torn down
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("Ethereum message handling failed", "err", err)
			return err
		}
	}
}
```

此函数是对新节点的处理函数，做了如下工作

+ 如果节点是信任的节点（我们手动加入的），无视最大可连接节点数量的限制直接加入
+ 执行以太坊握手操作
+ **注册节点** 
+ 对downloader注册此节点
+ 同步交易
+ 启动循环持续监听来自此节点的信息

下面主要说注册节点

### Register(p *peer)

```go
// Register injects a new peer into the working set, or returns an error if the
// peer is already known. If a new peer it registered, its broadcast loop is also
// started.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p

	go p.broadcastBlocks()
	go p.broadcastTransactions()
	go p.announceTransactions()

	return nil
}
```

检查检查节点集是不是关闭，此节点是不是已经注册过，然后将此节点放入节点集，启动三个协程。

+ go p.broadcastBlocks()
+ go p.broadcastTransactions()
+ go p.announceTransactions()

下面说一下三个协程。

### p.broadcastBlocks() 协程 重要

```go
// broadcastBlocks is a write loop that multiplexes blocks and block accouncements
// to the remote peer. The goal is to have an async writer that does not lock up
// node internals and at the same time rate limits queued data.
func (p *peer) broadcastBlocks() {
	for {
		select {
		case prop := <-p.queuedBlocks:
			if err := p.SendNewBlock(prop.block, prop.td); err != nil {
				return
			}
			p.Log().Trace("Propagated block", "number", prop.block.Number(), "hash", prop.block.Hash(), "td", prop.td)

		case block := <-p.queuedBlockAnns:
			if err := p.SendNewBlockHashes([]common.Hash{block.Hash()}, []uint64{block.NumberU64()}); err != nil {
				return
			}
			p.Log().Trace("Announced block", "number", block.Number(), "hash", block.Hash())

		case <-p.term:
			return
		}
	}
}
```

有三个case，最后一个case为终止情况，只分析前两个。

1. 要同步一个完整的区块，称作同步。p.SendNewBlock(prop.block, prop.td);
2. 要同步一个区块的哈希，称作通知。p.SendNewBlockHashes([]common.Hash{block.Hash()}, []uint64{block.NumberU64()});

#### SendNewBlock(block *types.Block, td *big.Int)

```go
// SendNewBlock propagates an entire block to a remote peer.
func (p *peer) SendNewBlock(block *types.Block, td *big.Int) error {
	// Mark all the block hash as known, but ensure we don't overflow our limits
	for p.knownBlocks.Cardinality() >= maxKnownBlocks {
		p.knownBlocks.Pop()
	}
	p.knownBlocks.Add(block.Hash())
	return p2p.Send(p.rw, NewBlockMsg, []interface{}{block, td})
}
```

此函数先使本地远程节点实例已知要同步的区块，然后就开始在网络层进行发送区块。

NewBlockMsg是一个静态值0x07，是ETH协议消息的代码

#### Send(w MsgWriter, msgcode uint64, data interface{})

```go
// Send writes an RLP-encoded message with the given code.
// data should encode as an RLP list.
func Send(w MsgWriter, msgcode uint64, data interface{}) error {
   size, r, err := rlp.EncodeToReader(data)
   if err != nil {
      return err
   }
   return w.WriteMsg(Msg{Code: msgcode, Size: uint32(size), Payload: r})
}
```

先对要发送的区块和td进行rlp编码，得到其大小size和编码后的结果r。传入的w是peer的读写成员。

然后就

```go
// WriteMsg sends a message on the pipe.
// It blocks until the receiver has consumed the message payload.
func (p *MsgPipeRW) WriteMsg(msg Msg) error {
	if atomic.LoadInt32(p.closed) == 0 {
		consumed := make(chan struct{}, 1)
		msg.Payload = &eofSignal{msg.Payload, msg.Size, consumed}
		select {
		case p.w <- msg:
			if msg.Size > 0 {
				// wait for payload read or discard
				// 如果没有default，select会一直等待直到一个case执行成功。
				// 消息要么被接受（consumed），要么就没被接受，符合尽力但不保证原则
				// 如果对方离线或者是发送超时，p.closing会触发，因为以太坊服务一直在维持与远程节点的心跳（就是Kad协议的一部分）
				select {
				case <-consumed:
				case <-p.closing:
				}
			}
			return nil
		case <-p.closing:
		}
	}
	return ErrPipeClosed
}
```

我一直在找，到底哪个函数哪一行向网络层发送了数据包真正把数据发送给了对方，但是发现以太坊对这个处理很巧妙。他没有关注底层的网络层到底是怎么通信的。以太坊把远程节点进行了实例化，就好像那些远程本来就在本地一样，而两者的通信也像本地进程间通信一样拥有一个管道，两者通信只需要往这个管道发送数据就行了，对方会一直监听着这个管道。所以，追踪到此，就可以看到上述第8行把负载送进了管道，进而等待对方返回的回执（类似已读回执）。