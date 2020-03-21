```java
@author FuMing
@data 2020.03.12
```

# go-ethereum启动

[TOC]



### 前置准备



当``go-ethereum``客户端安装好之后，在命令行运行geth就相当于运行了代码目录中的``/go-ethereum/cmd/geth/main.go``

先阅读[Go 包的初始化及init()函数](./引用/Go 包的初始化及init()函数)作为前置条件。

根据Go初始化方式，``/go-ethereum/cmd/geth/main.go``文件初始化会先初始化导入的包，再对包块中声明的变量进行计算和分配初始值，最后执行``init``函数。

### 变量初始化

先看``/go-ethereum/cmd/geth/main.go``文件中变量声明中的句子：

```go
// /go-ethereum/cmd/geth/main.go
app = utils.NewApp(gitCommit, gitDate, "the go-ethereum command line interface")
```

其中``NewApp()``方法在``/go-ethereum/cmd/utils/flags.go``中定义，

```go
// /go-ethereum/cmd/utils/flags.go
// NewApp creates an app with sane defaults.
func NewApp(gitCommit, gitDate, usage string) *cli.App {
	app := cli.NewApp()// 现在app中参数为默认值
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	app.Email = ""
	app.Version = params.VersionWithCommit(gitCommit, gitDate)
	app.Usage = usage
	return app
}
```

由

```go
app := cli.NewApp()// 现在app中参数为默认值
```

可再向前追溯至[cli库](https://godoc.org/gopkg.in/urfave/cli.v1)，该库的基本用法可以参考[Go 命令行cli库的使用](./引用/Go 命令行cli库的使用)。

上述代码中``NewApp()``是cli库``app.go``中的函数，声明如下

```go
// /cli.v1@v1.20.0/app.go
// NewApp creates a new cli Application with some reasonable defaults for Name,
// Usage, Version and Action.
func NewApp() *App {
	return &App{
		Name:         filepath.Base(os.Args[0]),
		HelpName:     filepath.Base(os.Args[0]),
		Usage:        "A new cli application",
		UsageText:    "",
		Version:      "0.0.0",
		BashComplete: DefaultAppComplete,
		Action:       helpCommand.Action,
		Compiled:     compileTime(),
		Writer:       os.Stdout,
	}
}
```

``NewApp()``返回了App类型的指针。``App``在``app.go``中被定义，是``struct``类型数据，声明如下

```go
// App is the main structure of a cli application. It is recommended that
// an app be created with the cli.NewApp() function
type App struct {
	// The name of the program. Defaults to path.Base(os.Args[0])
	Name string
	// Full name of command for help, defaults to Name
	HelpName string
	// Description of the program.
	Usage string
	// Text to override the USAGE section of help
	UsageText string
	// Description of the program argument format.
	ArgsUsage string
	// Version of the program
	Version string
	// Description of the program
	Description string
	// List of commands to execute
	Commands []Command
	// List of flags to parse
	Flags []Flag
	// Boolean to enable bash completion commands
	EnableBashCompletion bool
	// Boolean to hide built-in help command
	HideHelp bool
	// Boolean to hide built-in version flag and the VERSION section of help
	HideVersion bool
	// Populate on app startup, only gettable through method Categories()
	categories CommandCategories
	// An action to execute when the bash-completion flag is set
	BashComplete BashCompleteFunc
	// An action to execute before any subcommands are run, but after the context is ready
	// If a non-nil error is returned, no subcommands are run
	Before BeforeFunc
	// An action to execute after any subcommands are run, but after the subcommand has finished
	// It is run even if Action() panics
	After AfterFunc

	// The action to execute when no subcommands are specified
	// Expects a `cli.ActionFunc` but will accept the *deprecated* signature of `func(*cli.Context) {}`
	// *Note*: support for the deprecated `Action` signature will be removed in a future version
	Action interface{}

	// Execute this function if the proper command cannot be found
	CommandNotFound CommandNotFoundFunc
	// Execute this function if an usage error occurs
	OnUsageError OnUsageErrorFunc
	// Compilation date
	Compiled time.Time
	// List of all authors who contributed
	Authors []Author
	// Copyright of the binary if any
	Copyright string
	// Name of Author (Note: Use App.Authors, this is deprecated)
	Author string
	// Email of Author (Note: Use App.Authors, this is deprecated)
	Email string
	// Writer writer to write output to
	Writer io.Writer
	// ErrWriter writes error output
	ErrWriter io.Writer
	// Other custom info
	Metadata map[string]interface{}
	// Carries a function which returns app specific info.
	ExtraInfo func() map[string]string
	// CustomAppHelpTemplate the text template for app help topic.
	// cli.go uses text/template to render templates. You can
	// render custom help text by setting this variable.
	CustomAppHelpTemplate string

	didSetup bool
}
```

``/cli.v1@v1.20.0/app.go:NewApp()``对其中几个量进行了初始化之后将其地址返还给``/go-ethereum/cmd/utils/flags.go:NewApp()``函数，然后``/go-ethereum/cmd/utils/flags.go:NewApp()``再对``app``进行赋值返回给``/go-ethereum/cmd/geth/main.go app``

此时``/go-ethereum/cmd/geth/main.go``中变量初始化完成，开始执行``init()``函数。

### init()函数的执行

掌握了cli库的基本用法之后，init()函数的执行也就没那么复杂了。主要是对app参数，Flags和Commands、执行前后的一些基本操作。主要功能是对命令的解析与任务的分配。

### main()函数的执行

**所有工作都从``app.Run(os.Args)``触发**

```go
// /cli.v1@v1.20.0/app.go
// Run is the entry point to the cli app. Parses the arguments slice and routes
// to the proper flag/args combination
func (a *App) Run(arguments []string) (err error) {
	a.Setup()//进行简单初始化，未进行个性化设置

	// handle the completion flag separately from the flagset since
	// completion could be attempted after a flag, but before its value was put
	// on the command line. this causes the flagset to interpret the completion
	// flag name as the value of the flag before it which is undesirable
	// note that we can only do this because the shell autocomplete function
	// always appends the completion flag at the end of the command
	shellComplete, arguments := checkShellCompleteFlag(a, arguments)

	// parse flags
	set, err := flagSet(a.Name, a.Flags)//拿到a中的names和flags存入set
	if err != nil {
		return err
	}

	set.SetOutput(ioutil.Discard)// 对set赋予io能力
	err = set.Parse(arguments[1:])//对set中的命令进行解析，解析后存入set.actual中，最后一个参数存入set.args中。第1个参数是启动命令，不被解析
	nerr := normalizeFlags(a.Flags, set)//a.Flags是所有flag 寻找set中是否使用了同一命令的两种形式，如-help和-h
	context := NewContext(a, set, nil)//不知道这句的目的
	if nerr != nil {
		fmt.Fprintln(a.Writer, nerr)
		ShowAppHelp(context)
		return nerr
	}
	context.shellComplete = shellComplete

	if checkCompletions(context) {
		return nil
	}

	if err != nil {
		if a.OnUsageError != nil {
			err := a.OnUsageError(context, err, false)
			HandleExitCoder(err)
			return err
		}
		fmt.Fprintf(a.Writer, "%s %s\n\n", "Incorrect Usage.", err.Error())
		ShowAppHelp(context)
		return err
	}

	if !a.HideHelp && checkHelp(context) {
		ShowAppHelp(context)
		return nil
	}

	if !a.HideVersion && checkVersion(context) {
		ShowVersion(context)
		return nil
	}

	if a.After != nil {
		defer func() {
			if afterErr := a.After(context); afterErr != nil {
				if err != nil {
					err = NewMultiError(err, afterErr)
				} else {
					err = afterErr
				}
			}
		}()
	}

	if a.Before != nil {
		beforeErr := a.Before(context)
		if beforeErr != nil {
			ShowAppHelp(context)
			HandleExitCoder(beforeErr)
			err = beforeErr
			return err
		}
	}

	args := context.Args()
	//如果args有值
	if args.Present() {
		name := args.First()//获取最后一个命令，
		c := a.Command(name)//从a中存储的所有command寻找名为name的command并返回a中存储的此command地址，没找到则返回nil
		if c != nil {
			return c.Run(context)//命令解析完毕，开始执行
		}
	}
	//处理没有command的情况
	if a.Action == nil {
		a.Action = helpCommand.Action
	}

	// Run default Action
	err = HandleAction(a.Action, context)

	HandleExitCoder(err)
	return err
}
```

[/go/src/flag/flag.go Parse函数解析](./引用/Go flag.go部分解析)

对命令解析完后会执行至``/go-ethereum/cmd/geth/consolecmd.go func localConsole(ctx *cli.Context) error``

```go
// /go-ethereum/cmd/geth/consolecmd.go
/ localConsole starts a new geth node, attaching a JavaScript console to it at the
// same time.
func localConsole(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	prepare(ctx)
	node := makeFullNode(ctx)/* FuM:初始化一些配置信息 */
	startNode(ctx, node)/* FuM:开启节点 */
	defer node.Close()

	// Attach to the newly started node and start the JavaScript console
	client, err := node.Attach()
	if err != nil {
		utils.Fatalf("Failed to attach to the inproc geth: %v", err)
	}
	config := console.Config{
		DataDir: utils.MakeDataDir(ctx),
		DocRoot: ctx.GlobalString(utils.JSpathFlag.Name),
		Client:  client,
		Preload: utils.MakeConsolePreloads(ctx),
	}

	console, err := console.New(config)
	if err != nil {
		utils.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	// If only a short execution was requested, evaluate and return
	if script := ctx.GlobalString(utils.ExecFlag.Name); script != "" {
		console.Evaluate(script)
		return nil
	}
	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()/* FuM:进入命令行 */

	return nil
}
```

先大致解析一下``/go-ethereum/cmd/geth/main.go prepare``函数。``prepare``函数主要是对虚拟机内存进行了配置。

```go
// 准备操作内存高速缓存配额并设置度量系统。
// 启动devp2p堆栈之前，应先调用此函数。
func prepare(ctx *cli.Context) {
	// 如果我们是主网上的完整节点，但未指定--cache，则增加默认缓存配额
	//utils.SyncModeFlag.Name其实就是一个类似静态变量的东西，ctx.GlobalString(utils.SyncModeFlag.Name)就是获取ctx中的同步模式
	if ctx.GlobalString(utils.SyncModeFlag.Name) != "light" && !ctx.GlobalIsSet(utils.CacheFlag.Name) && !ctx.GlobalIsSet(utils.NetworkIdFlag.Name) {
		// 确保我们也不在任何受支持的预配置测试网上
		if !ctx.GlobalIsSet(utils.TestnetFlag.Name) && !ctx.GlobalIsSet(utils.RinkebyFlag.Name) && !ctx.GlobalIsSet(utils.GoerliFlag.Name) && !ctx.GlobalIsSet(utils.DeveloperFlag.Name) {
			// 不，我们真的在主网上。 增加缓存!
			log.Info("Bumping default cache on mainnet", "provided", ctx.GlobalInt(utils.CacheFlag.Name), "updated", 4096)
			ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(4096))
		}
	}
	// 如果我们在任何网络上都运行轻客户端，则将缓存降低到有意义的低水平
	if ctx.GlobalString(utils.SyncModeFlag.Name) == "light" && !ctx.GlobalIsSet(utils.CacheFlag.Name) {
		log.Info("Dropping default light client cache", "provided", ctx.GlobalInt(utils.CacheFlag.Name), "updated", 128)
		ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(128))
	}
	// 限制缓存配额并调整垃圾收集器
	var mem gosigar.Mem// 用的是VMware的虚拟机
	// Workaround until OpenBSD support lands into gosigar
	// Check https://github.com/elastic/gosigar#supported-platforms
	if runtime.GOOS != "openbsd" {
		//mem.Get();拿到了缓存使用情况
		if err := mem.Get(); err == nil {
			allowance := int(mem.Total / 1024 / 1024 / 3)//已经使用的缓存总容量 不知道单位是啥，也不知道为什么/3
			if cache := ctx.GlobalInt(utils.CacheFlag.Name); cache > allowance {//cache是当前设置缺省最大允许缓存
				log.Warn("Sanitizing cache to Go's GC limits", "provided", cache, "updated", allowance)
				ctx.GlobalSet(utils.CacheFlag.Name, strconv.Itoa(allowance))//将cache更新为allowance
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

之后就是**makeFullNode**，源码如下

```go
/* FuM:该函数先构造了一个节点，然后注册一个Ethereum Service */
func makeFullNode(ctx *cli.Context) *node.Node {
	stack, cfg := makeConfigNode(ctx) /* FuM:构造了一个节点 */
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

继续往下看``makeConfigNode``函数，此函数主要是初始化了``gethConfig``变量进行了一些基本配置

```go
func makeConfigNode(ctx *cli.Context) (*node.Node, gethConfig) {
	// Load defaults.
	cfg := gethConfig{
		Eth:  eth.DefaultConfig,     /* FuM: 配置eth的一些基本信息，里面设置NetWorkId等 */
		Shh:  whisper.DefaultConfig, /* FuM: 配置了两个参数：1.MaxMessageSize 2.MinimumAcceptedPOW */
		Node: defaultNodeConfig(),   /* FuM: 初始化了节点的配置，主要有网络的一些配置，还有数据的存储路径 */
	}

	// Load config file.
	if file := ctx.GlobalString(configFileFlag.Name); file != "" {
		if err := loadConfig(file, &cfg); err != nil {
			utils.Fatalf("%v", err)
		}
	}

	// Apply flags.
	utils.SetNodeConfig(ctx, &cfg.Node) /* FuM:设置了P2P,IPC,HTTP,WS,DataDir,KeyStoreDir的值 */
	stack, err := node.New(&cfg.Node)/* FuM:构建新节点 */
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	utils.SetEthConfig(ctx, stack, &cfg.Eth) /* FuM:设置NetWorkId的值 */
	if ctx.GlobalIsSet(utils.EthStatsURLFlag.Name) {
		cfg.Ethstats.URL = ctx.GlobalString(utils.EthStatsURLFlag.Name)
	}
	utils.SetShhConfig(ctx, stack, &cfg.Shh)

	return stack, cfg
}
```

之后运行``localConsole(ctx *cli.Context)``中的``startNode(ctx, node)``开启节点，监听必要的端口。节点启动以后就在控制台输出相关信息后输出欢迎语句。``localConsole(ctx *cli.Context)``中的``console.Interactive()``处理用户于控制台之间的交互，对命令的合法性进行检测并运行命令，使用一个无限循环处理用户输入。

