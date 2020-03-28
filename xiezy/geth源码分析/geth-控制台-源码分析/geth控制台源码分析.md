# cmd包源码分析

## 简介

cmd包实现了geth客户端主要的命令行程序，是geth项目的一个重要入口。

## 源码分析

geth程序的入口函数在cmd/geth/main.go 里面，在执行正式函数之前，函数开头初始化程序的全局变量app

```go
import cli "gopkg.in/urfave/cli.v1"
// 导入命令行控制台cli包
var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
    // 这两个参数参与了版本配置
	app = utils.NewApp(gitCommit, gitDate, "the go-ethereum command line interface")
    // 新建一个全局的app结构，用来管理程序启动，命令行配置等，下面重点分析
    
	// 节点配置
	nodeFlags = []cli.Flag{
        ...
	}
	// rpc通信协议配置
	rpcFlags = []cli.Flag{
        ...
	}
	// whisper协议配置
	whisperFlags = []cli.Flag{
        ...
	}
	// 提供磁盘计数器
	metricsFlags = []cli.Flag{
        ...
	}
)
```

cmd/utils/flags.go中定义了NewApp函数

```go
func NewApp(gitCommit, gitDate, usage string) *cli.App {
	app := cli.NewApp()
    /* 主要就是新建了一个cli.NewApp结构，代码在vendor/gopkg.in/urfave/cli.v1/app.go 里面，这个包是用来管理程序启动的，App包括 几个基本的接口： Command（）， Run（）， Setup（）， 这些接口将来会在main里面被调用到。*/
	app.Name = filepath.Base(os.Args[0])
    // os.Args[0],args的第一个片是文件路径
    // filepath.Base : 获取path中最后一个分隔符之后的部分(不包含分隔符)
    // 该指令获取了文件名
	app.Author = ""
	app.Email = ""
	app.Version = params.VersionWithCommit(gitCommit, gitDate)
	app.Usage = usage
	return app
}
```

初始化完程序的app，在main()函数之前还要再执行init()函数，geth的init做了很重要的工作：设置程序的子命令集，以及程序入口函数，调用准备函数app.Before，以及负责扫尾的app.After().
对于各项Command， app会解析参数比如如果参数有console， 那么会由app类去调度，调用consoleCommand对应的处理函数。

```go
func init() {
	app.Action = geth
	// 默认的操作，就是用默认的配置启动一个节点，如果有其他命令行参数，会调用到下面的app.Commands里面去
	app.HideVersion = true 
	app.Copyright = "Copyright 2013-2020 The go-ethereum Authors"
	app.Commands = []cli.Command{
    // 如果命令行参数里面有下面的指令，就会直接调用下面的Command.Run方法，而不调用默认的app.Action方法
        // 比如调用localConsole， initGenesis
		// See chaincmd.go:
		initCommand,
        // 初始化创世块，改变网络状态
		importCommand,
		exportCommand,
		importPreimagesCommand,
		exportPreimagesCommand,
		copydbCommand,
		removedbCommand,
		dumpCommand,
		inspectCommand,
		// See accountcmd.go:
		accountCommand,
		walletCommand,
		// See consolecmd.go:
		consoleCommand,
        // js命令行终端
		attachCommand,
		javascriptCommand,
		// See misccmd.go:
		makecacheCommand,
		makedagCommand,
		versionCommand,
		licenseCommand,
		// See config.go
		dumpConfigCommand,
		// See retesteth.go
		retestethCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))
    // 大概就是排序

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, consoleFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, whisperFlags...)
	app.Flags = append(app.Flags, metricsFlags...)
    // 通过调用append函数不断拓展切片

	app.Before = func(ctx *cli.Context) error {
		return debug.Setup(ctx, "")
	}
    //before函数在app.Run的开始会先调用，也就是gopkg.in/urfave/cli.v1/app.go Run函数的前面
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}
    //after函数在最后调用，app.Run 里面会设置defer function
}
```

main函数定义了app.run()，app.run()的功能其实在上面注释里说过：

1. 如果是geth命令行启动，不带子命令，那么直接调用app.Action = geth（）函数
2. 如果带有子命令比如build/bin/geth console 启动，那么会调用对应命令的Command.Action， 对于console来说就是调用的 localConsole()函数；

```go
func main() {
   if err := app.Run(os.Args); err != nil {
      fmt.Fprintln(os.Stderr, err)
      os.Exit(1)
   }
}
```

下面是不带任何子命令时调用的geth函数

```go
func geth(ctx *cli.Context) error {
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
        // 以这种形式
	}
	prepare(ctx)
	//准备操作内存缓存配额并设置度量系统，在启动devp2p堆栈之前，应调用此函数。
	node := makeFullNode(ctx)
	// 根据上文的参数ctx来初始化全节点
	defer node.Close()
	startNode(ctx, node)
	node.Wait()
	return nil
}
```

指针*cli.Context，用于检索上下文参数，数据结构如下

```go
type Context struct {
       App           *App
       Command       Command
       flagSet       *flag.FlagSet
       setFlags      map[string]bool
       parentContext *Context
}
```

创建全节点的函数makeFullNode如下

```go
func makeFullNode(ctx *cli.Context) *node.Node {
   stack, cfg := makeConfigNode(ctx)
   //生成node.Node一个结构，里面会有任务函数栈, 这里设置各个服务（轻节点，全节点，dashboard，shh，以及状态stats服务）到serviceFuncs 里面
   if ctx.GlobalIsSet(utils.OverrideIstanbulFlag.Name) {
      cfg.Eth.OverrideIstanbul = new(big.Int).SetUint64(ctx.GlobalUint64(utils.OverrideIstanbulFlag.Name))
   }
   if ctx.GlobalIsSet(utils.OverrideMuirGlacierFlag.Name) {
      cfg.Eth.OverrideMuirGlacier = new(big.Int).SetUint64(ctx.GlobalUint64(utils.OverrideMuirGlacierFlag.Name))
   }
   utils.RegisterEthService(stack, &cfg.Eth)
   //在stack上增加一个以太坊节点，其实就是new一个Ethereum 后加到后者的AddLesServer serviceFuncs 里面去
   //然后在stack.Run的时候会盗用这些service
    
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

下面是，在stack堆栈新增以太坊节点函数RegisterEthService，从下面代码可以看到，根据配置不同启动不同的节点，stack其实就是一个节点，以太坊的节点就是一个启动的geth程序。在stack上增加一个以太坊节点，其实就是new一个Ethereum 后加到stack中的AddLesServer serviceFuncs 里面去。

stack.Register 是关键，传入匿名函数，函数代码很简单，就是创建一个类型为轻节点LightEthereum 或者 全节点类型为Ethereum的结构。

```go
func RegisterEthService(stack *node.Node, cfg *eth.Config) {
	var err error
	if cfg.SyncMode == downloader.LightSync {
        // 判断以太坊节点是否是轻节点，如果是就创建成一个轻节点并返回
		err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
			return les.New(ctx, cfg)
		})
	} else {
		err = stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
            // 如果是以太坊全节点，在stack上通过函数Register调用增加一个serviceFuncs函数，增加以太坊                服务
			fullNode, err := eth.New(ctx, cfg)
            // 设置全节点类型为Ethereum
			if fullNode != nil && cfg.LightServ > 0 {
				ls, _ := les.NewLesServer(fullNode, cfg)
                // 新建一个LesServer轻量级节点，设置到全节点上
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

