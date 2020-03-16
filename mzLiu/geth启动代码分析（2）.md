##### 记录一下geth app 运行的过程

###### 1.geth的启动函数

启动一个节点，阻塞运行。

```go
func geth(ctx *cli.Context) error {
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}
	prepare(ctx)
	node := makeFullNode(ctx)
	defer node.Close()
	startNode(ctx, node)
	node.Wait()
	return nil
}
```

**关于cli.Context**

上下文是在CLI应用程序中传递给每个处理程序操作的类型。上下文可用于检索上下文特定的参数和已分析的命令行选项。

```go
type Context struct {
	App           *App
	Command       Command
	shellComplete bool
	flagSet       *flag.FlagSet
	setFlags      map[string]bool
	parentContext *Context
}
```

它包含指向cli启动app的指针，command指令，app是否运行完，flag集合等

**Errorf的解释**：****

error类型本身就是一个预定义好的接口，里面定义了一个method

```go
type error interface {
    Error() string
}
```

而fmt.Errorf是将错误信息以string字符串的方式打出，而且不会退出进程，像Fatalf，panic等会直接退出当前进程。

```go
// Errorf formats according to a format specifier and returns the string
// as a value that satisfies error.
func Errorf(format string, a ...interface{}) error {
	return errors.New(Sprintf(format, a...))
}
// New returns an error that formats as the given text.
func New(text string) error {
	return &errorString{text}
}
// errorString is a trivial implementation of error.
type errorString struct {
	s string
}
```

###### 2..prepare

定义如下：

```go

// prepare manipulates memory cache allowance and setups metric system.
// This function should be called before launching devp2p stack.
func prepare(ctx *cli.Context) {
	// If we're a full node on mainnet without --cache specified, bump default cache allowance
	if ctx.GlobalString(utils.SyncModeFlag.Name) != "light" && !ctx.GlobalIsSet(utils.CacheFlag.Name) && !ctx.GlobalIsSet(utils.NetworkIdFlag.Name) {
		// Make sure we're not on any supported preconfigured testnet either
		if !ctx.GlobalIsSet(utils.TestnetFlag.Name) && !ctx.GlobalIsSet(utils.RinkebyFlag.Name) && !ctx.GlobalIsSet(utils.GoerliFlag.Name) && !ctx.GlobalIsSet(utils.DeveloperFlag.Name) {
			// Nope, we're really on mainnet. Bump that cache up!
			log.Info("Bumping default cache on mainnet", "provided", ctx.GlobalInt(utils.CacheFlag.Name), "updated", 4096)
			ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(4096))
		}
	}
	// If we're running a light client on any network, drop the cache to some meaningfully low amount
	if ctx.GlobalString(utils.SyncModeFlag.Name) == "light" && !ctx.GlobalIsSet(utils.CacheFlag.Name) {
		log.Info("Dropping default light client cache", "provided", ctx.GlobalInt(utils.CacheFlag.Name), "updated", 128)
		ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(128))
	}
	// Cap the cache allowance and tune the garbage collector
	var mem gosigar.Mem
	// Workaround until OpenBSD support lands into gosigar
	// Check https://github.com/elastic/gosigar#supported-platforms
	if runtime.GOOS != "openbsd" {
		if err := mem.Get(); err == nil {
			allowance := int(mem.Total / 1024 / 1024 / 3)
			if cache := ctx.GlobalInt(utils.CacheFlag.Name); cache > allowance {
				log.Warn("Sanitizing cache to Go's GC limits", "provided", cache, "updated", allowance)
				ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(allowance))
			}
		}
	}
	// Ensure Go's GC ignores the database cache for trigger percentage
	cache := ctx.GlobalInt(utils.CacheFlag.Name)
	gogc := math.Max(20, math.Min(100, 100/(float64(cache)/1024)))

	log.Debug("Sanitizing Go's GC trigger", "percent", int(gogc))
	godebug.SetGCPercent(int(gogc))

	// Start metrics export if enabled
	utils.SetupMetrics(ctx)

	// Start system runtime metrics collection
	go metrics.CollectProcessMetrics(3 * time.Second)
}

```

这个prepare函数不是很会，以后有机会再解析

###### 3.makeFullNode

```go
func makeFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx)
	if ctx.GlobalIsSet(utils.OverrideIstanbulFlag.Name) {
		cfg.Eth.OverrideIstanbul = new(big.Int).SetUint64(ctx.GlobalUint64(utils.OverrideIstanbulFlag.Name))
	}
	if ctx.GlobalIsSet(utils.OverrideMuirGlacierFlag.Name) {
		cfg.Eth.OverrideMuirGlacier = new(big.Int).SetUint64(ctx.GlobalUint64(utils.OverrideMuirGlacierFlag.Name))
	}
	utils.RegisterEthService(stack, &cfg.Eth)

	// Whisper must be explicitly enabled by specifying at least 1 whisper flag or in dev mode
	shhEnabled := enableWhisper(ctx)
	shhAutoEnabled := !ctx.GlobalIsSet(utils.WhisperEnabledFlag.Name) && ctx.GlobalIsSet(utils.DeveloperFlag.Name)
	if shhEnabled || shhAutoEnabled {
		if ctx.GlobalIsSet(utils.WhisperMaxMessageSizeFlag.Name) {
			cfg.Shh.MaxMessageSize = uint32(ctx.Int(utils.WhisperMaxMessageSizeFlag.Name))
		}
		if ctx.GlobalIsSet(utils.WhisperMinPOWFlag.Name) {
			cfg.Shh.MinimumAcceptedPOW = ctx.Float64(utils.WhisperMinPOWFlag.Name)
		}
		if ctx.GlobalIsSet(utils.WhisperRestrictConnectionBetweenLightClientsFlag.Name) {
			cfg.Shh.RestrictConnectionBetweenLightClients = true
		}
		utils.RegisterShhService(stack, &cfg.Shh)
	}
	// Configure GraphQL if requested
	if ctx.GlobalIsSet(utils.GraphQLEnabledFlag.Name) {
		utils.RegisterGraphQLService(stack, cfg.Node.GraphQLEndpoint(), cfg.Node.GraphQLCors, cfg.Node.GraphQLVirtualHosts, cfg.Node.HTTPTimeouts)
	}
	// Add the Ethereum Stats daemon if requested.
	if cfg.Ethstats.URL != "" {
		utils.RegisterEthStatsService(stack, cfg.Ethstats.URL)
	}
	return stack
}
```

其中大概可以分为以下几个步骤：

```go
stack, cfg := makeConfigNode(ctx)    //载入默认配置，再载入配置文件中的配置，然后通过上下文的配置进行设置
utils.RegisterEthService(stack, &cfg.Eth)  //如果同步模式是轻量级同步模式，启动轻量级客户端，否则启动全节点
再加载一些别的配置
// whether enable whisper ...
// whether register eth stats ...
```

makeConfigNode(ctx)函数

```go
func makeConfigNode(ctx *cli.Context) (*node.Node, gethConfig) {
	// Load defaults.
	cfg := gethConfig{
		Eth:  eth.DefaultConfig,
		Shh:  whisper.DefaultConfig,
		Node: defaultNodeConfig(),
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node)
	stack, err := node.New(&cfg.Node)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	utils.SetEthConfig(ctx, stack, &cfg.Eth)
	if ctx.GlobalIsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.GlobalString(utils.EthStatsURLFlag.Name)
	}
	utils.SetShhConfig(ctx, stack, &cfg.Shh)

	return stack, cfg
}
```

先是进行了eth默认选项

```go
type gethConfig struct {
	Eth      eth.Config
	Shh      whisper.Config
	Node     node.Config
	Ethstats ethstatsConfig
}
```

之后加载文件配置

```go
func loadConfig(file string, cfg *gethConfig) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = tomlSettings.NewDecoder(bufio.NewReader(f)).Decode(cfg)
	// Add file name to errors that have a line number.
	if _, ok := err.(*toml.LineError); ok {
		err = errors.New(file + ", " + err.Error())
	}
	return err
}

```

之后加载节点配置

```go
// SetNodeConfig applies node-related command line flags to the config.
func SetNodeConfig(ctx *cli.Context, cfg *node.Config) {
	SetP2PConfig(ctx, &cfg.P2P)
	setIPC(ctx, cfg)
	setHTTP(ctx, cfg)
	setGraphQL(ctx, cfg)
	setWS(ctx, cfg)
	setNodeUserIdent(ctx, cfg)
	setDataDir(ctx, cfg)
	setSmartCard(ctx, cfg)

	if ctx.GlobalIsSet(ExternalSignerFlag.Name) {
		cfg.ExternalSigner = ctx.GlobalString(ExternalSignerFlag.Name)
	}

	if ctx.GlobalIsSet(KeyStoreDirFlag.Name) {
		cfg.KeyStoreDir = ctx.GlobalString(KeyStoreDirFlag.Name)
	}
	if ctx.GlobalIsSet(LightKDFFlag.Name) {
		cfg.UseLightweightKDF = ctx.GlobalBool(LightKDFFlag.Name)
	}
	if ctx.GlobalIsSet(NoUSBFlag.Name) {
		cfg.NoUSB = ctx.GlobalBool(NoUSBFlag.Name)
	}
	if ctx.GlobalIsSet(InsecureUnlockAllowedFlag.Name) {
		cfg.InsecureUnlockAllowed = ctx.GlobalBool(InsecureUnlockAllowedFlag.Name)
	}
}
```

之后创建一个节点

```go
// New creates a new P2P node, ready for protocol registration.
func New(conf *Config) (*Node, error) {
	// Copy config and resolve the datadir so future changes to the current
	// working directory don't affect the node.
	confCopy := *conf
	conf = &confCopy
	if conf.DataDir != "" {
		absdatadir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		conf.DataDir = absdatadir
	}
	// Ensure that the instance name doesn't cause weird conflicts with
	// other files in the data directory.
	if strings.ContainsAny(conf.Name, `/\`) {
		return nil, errors.New(`Config.Name must not contain '/' or '\'`)
	}
	if conf.Name == datadirDefaultKeyStore {
		return nil, errors.New(`Config.Name cannot be "` + datadirDefaultKeyStore + `"`)
	}
	if strings.HasSuffix(conf.Name, ".ipc") {
		return nil, errors.New(`Config.Name cannot end in ".ipc"`)
	}
	// Ensure that the AccountManager method works before the node has started.
	// We rely on this in cmd/geth.
	am, ephemeralKeystore, err := makeAccountManager(conf)
	if err != nil {
		return nil, err
	}
	if conf.Logger == nil {
		conf.Logger = log.New()
	}
	// Note: any interaction with Config that would create/touch files
	// in the data directory or instance directory is delayed until Start.
	return &Node{
		accman:            am,
		ephemeralKeystore: ephemeralKeystore,
		config:            conf,
		serviceFuncs:      []ServiceConstructor{},
		ipcEndpoint:       conf.IPCEndpoint(),
		httpEndpoint:      conf.HTTPEndpoint(),
		wsEndpoint:        conf.WSEndpoint(),
		eventmux:          new(event.TypeMux),
		log:               conf.Logger,
	}, nil
}
```

之后加载和eth有关的command

```go
// SetEthConfig applies eth-related command line flags to the config.
func SetEthConfig(ctx *cli.Context, stack *node.Node, cfg *eth.Config) {
	// Avoid conflicting network flags
    。。。
	// TODO(fjl): force-enable this in --dev mode
	。。。
	// Override any default configs for hard coded networks.
	。。。
		cfg.Genesis = core.DeveloperGenesisBlock(uint64(ctx.GlobalInt(DeveloperPeriodFlag.Name)), developer.Address)
	}
}

```

再加载ssh有关的command

```go
// SetShhConfig applies shh-related command line flags to the config.
func SetShhConfig(ctx *cli.Context, stack *node.Node, cfg *whisper.Config) {
	if ctx.GlobalIsSet(WhisperMaxMessageSizeFlag.Name) {
		cfg.MaxMessageSize = uint32(ctx.GlobalUint(WhisperMaxMessageSizeFlag.Name))
	}
	if ctx.GlobalIsSet(WhisperMinPOWFlag.Name) {
		cfg.MinimumAcceptedPOW = ctx.GlobalFloat64(WhisperMinPOWFlag.Name)
	}
	if ctx.GlobalIsSet(WhisperRestrictConnectionBetweenLightClientsFlag.Name) {
		cfg.RestrictConnectionBetweenLightClients = true
	}
}
```

之后是调用RegisterEthService：将以太坊客户端加入堆栈中：

```go
// RegisterEthService adds an Ethereum client to the stack.
func RegisterEthService(stack *node.Node, cfg *eth.Config) {
	var err error
	if cfg.SyncMode == downloader.LightSync {
		err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, cfg)
		})
	} else {
		err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			fullNode, err := eth.New(ctx, cfg)
			if fullNode != nil && cfg.LightServ > 0 {
				ls, _ := les.NewLesServer(fullNode, cfg)
				fullNode.AddLesServer(ls)
			}
			return fullNode, err
		})
	}
	if err != nil {
		Fatalf("Failed to register the Ethereum service: %v", err)
	}
}
```

###### 4.startNode

```go
// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func startNode(ctx *cli.Context, stack *node.Node){
	。。。
	}
```

staetNode方法启动节点，会开启所有已经注册的协议，解锁请求的账户，开启 RPC/IPC 接口，并开始挖矿。 

涉及到节点时再详细看。

###### 5.node.Wait()

```go
// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
func (n *Node) Wait() {
	n.lock.RLock()
	if n.server == nil {
		n.lock.RUnlock()
		return
	}
	stop := n.stop
	n.lock.RUnlock()

	<-stop
}
```

节点运行，阻塞进程。