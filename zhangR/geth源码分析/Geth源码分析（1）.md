# Geth源码分析（1）

## 初始化全局变量app

 geth程序的入口函数在cmd/geth/main.go 里面，包括main，以及初始化等。函数开头初始化程序的全局变量app。 

![app](D:\Destop\geth\app.png)

 cmd/utils/flags.go  111行
 ![NewApp](D:\Destop\geth\NewApp.png)

主要是新建了一个cli.NewApp结构，代码在gopkg.in/urfave/cli.v1/app.go 里面，管理程序启动，App包括几个基本的接口： Command()， Run()， Setup() 。

## 初始化子命令，程序入口函数设置

 `app.Action = geth `
 默认的操作，就是启动一个节点 ，如果有其他命令行参数，会调用到下面的Commands 里面去。

![1584207070078](D:\Destop\geth\commands.png)

[详细命令用法和参数详解](https://learnblockchain.cn/2017/11/29/geth_cmd_options/)

## main入口

```go
func main() {   
	if err := app.Run(os.Args); err != nil {
    	fmt.Fprintln(os.Stderr, err)
		os.Exit(1)   
	}
}
```

直接调用app.Run。
如果是geth命令行启动，不带子命令，那么直接调用app.Action = geth（）函数；
如果带有子命令比如build/bin/geth console 启动，那么会调用对应命令的Command.Action。

consoleCommand位于cmd/geth/consolecmd中

```go
    consoleCommand = cli.Command{      
    Action:   utils.MigrateFlags(localConsole),     
    Name:     "console",      
    Usage:    "Start an interactive JavaScript environment", 
    Flags:    append(append(append(nodeFlags, rpcFlags...), consoleFlags...), whisperFlags...),
    Category: "CONSOLE COMMANDS",      
    Description: `
    The Geth console is an interactive shell for the JavaScript runtime environmentwhich exposes a node admin interface as well as the Ðapp JavaScript API.See https://github.com/ethereum/go-ethereum/wiki/JavaScript-Console.`,   	}
```

其中Action中的localConsole函数如下：

```go
	func localConsole(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	prepare(ctx)
	node := makeFullNode(ctx)
	startNode(ctx, node)
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
	console.Interactive()

	return nil
}
```

这个函数是控制台的主要函数，主要功能有四个

* 首先创建一个节点、同时启动该节点

	主要是节点创建函数makeFullNode 和启动函数 startNode。

```go

func makeFullNode(ctx *cli.Context) *node.Node 
func startNode(ctx *cli.Context, stack *node.Node)

//此处的cli.Context位于gopkg.in/urfave/cli.v1/context.go中
type Context struct {
	App           *App
	Command       Command
	shellComplete bool
	flagSet       *flag.FlagSet
	setFlags      map[string]bool
	parentContext *Context
}
```

* 创建一个console的实例

	主要是监听控制台输入的命令。

* 显示Welcome信息

```go
	func (c *Console) Welcome() {
		// Print some generic Geth metadata
		fmt.Fprintf(c.printer, "Welcome to the Geth JavaScript console!\n\n")
		c.jsre.Run(`
			console.log("instance: " + web3.version.node);
			console.log("coinbase: " + eth.coinbase);
			console.log("at block: " + eth.blockNumber + " (" + new Date(1000 * eth.getBlock(eth.blockNumber).timestamp) + ")");
			console.log(" datadir: " + admin.datadir);
		`)
		// List all the supported modules for the user to call
		if apis, err := c.client.SupportedModules(); err == nil {
			modules := make([]string, 0, len(apis))
			for api, version := range apis {
				modules = append(modules, fmt.Sprintf("%s:%s", api, version))
			}
			sort.Strings(modules)
			fmt.Fprintln(c.printer, " modules:", strings.Join(modules, " "))
		}
		fmt.Fprintln(c.printer)
	}
	
```

	首先打印Welcome to the Geth JavaScript console!信息，接着使用上面创建console执行四个控制台输出用来写版本、区块等信息。

* 创建一个无限循环用于在控制台交互
