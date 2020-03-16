###### 1.内置函数append

append主要用于给某个切片追加元素，如果切片长度cap足够，则直接追加，长度变长,如果空间不足，则扩展切片的长度，再加入元素

例：

```go
func main() {
    s1 := []int{}
    fmt.Printf("len = %d, cap = %d\n", len(s1), cap(s1))
    fmt.Println("s1 = ", s1)
 
    //在原切片的末尾添加元素
    s1 = append(s1, 1)
    s1 = append(s1, 2)
    s1 = append(s1, 3)
    fmt.Printf("len = %d, cap = %d\n", len(s1), cap(s1))
    fmt.Println("s1 = ", s1)
 
    s2 := []int{1, 2, 3}
    fmt.Println("s2 = ", s2)
    s2 = append(s2, 5)
    s2 = append(s2, 5)
    s2 = append(s2, 5)
    fmt.Println("s2 = ", s2)
}
```

输出：

```
len = 0, cap = 0
s1 =  []
len = 3, cap = 4
s1 =  [1 2 3]
 
s2 =  [1 2 3]
s2 =  [1 2 3 5 5 5]
```

如果第二个参数是另一个切片，则将它里面的所有元素拷贝追加到第一个切片后边。要注意的是，这种用法函数的参数只能接收两个slice，并且末尾要加三个点

```go
slice := append([]int{1,2,3},[]int{4,5,6}...)
fmt.Println(slice) //[1 2 3 4 5 6]
```

还有种特殊用法，将字符串当作[]byte类型作为第二个参数传入

```go
bytes := append([]byte("hello"),"world"...)
```

·append函数返回值必须有变量接收

###### 2.urfave/cli：golang命令行构建工具

根据源码大概讲一下：

```go
var(
// The app that holds all commands and flags.
	app = utils.NewApp(gitCommit, gitDate, "the go-ethereum command line interface")
	// flags that configure the node
	nodeFlags = []cli.Flag{
        。。。
    }
    rpcFlags = []cli.Flag{
        ...
    }
    whisperFlags = []cli.Flag{
        ...
    }
    metricsFlags = []cli.Flag{
        ...
    }
)

func NewApp(gitCommit, gitDate, usage string) *cli.App {
	app := cli.NewApp()
	app.Name = filepath.Base(os.Args[0])
	app.Author = ""
	app.Email = ""
	app.Version = params.VersionWithCommit(gitCommit, gitDate)
	app.Usage = usage
	return app
}

func init() {
	// Initialize the CLI app and start Geth
	app.Action = geth
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2020 The go-ethereum Authors"
	app.Commands = []cli.Command{
        ...
    }
    	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, consoleFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, whisperFlags...)
	app.Flags = append(app.Flags, metricsFlags...)

	app.Before = func(ctx *cli.Context) error {
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}
}
```

在该例中，首先通过new实例化一个App对象，源码如下：

```go
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

其中结构体没有被定义的地方很多都有默认的初始化参数，说明一下：

filepath.Base : 获取path中最后一个分隔符之后的部分(不包含分隔符)

Action是一个空接口，要定义的是app执行的方法

compileTime返回一个标准包的time.Time类型，想了解可参考[以下博客](https://yezzi.xyz/2019/10/08/go%E7%9A%84time%E5%8C%85%E4%BD%BF%E7%94%A8/)

其中比较重要的是Action，因为命令最后会在这里开始执行，其实已经定义好了，就是geth函数，之后再看geth函数，先写一下其他的构成。

Flags：

flag是构建启动参数，以太的Flag都写在了flags.go文件里。

cli包中flag的定义如下：

```go
type Flag interface {
	fmt.Stringer
	// Apply Flag settings to the given flag set
	Apply(*flag.FlagSet)
	GetName() string
}
```

Flag是一个接口，可以自己定义，但至少包含一个Stringer接口，和Apply，GetName两个方法

Stringer接口定义如下：

```go
// Stringer is implemented by any value that has a String method,
// which defines the ``native'' format for that value.
// The String method is used to print values passed as an operand
// to any format that accepts a string or to an unformatted printer
// such as Print.
type Stringer interface {
	String() string
}
```

至少要有返回string的String（）方法，就可以成为Stringer接口。

以太坊中既有用到cli包内置的flag类型，如：StringFlag, BoolFlag, Float64Flag等，也有自定义的Flag，放在customflags.go中，如DirectoryFlag等。

其中cli中类似StringFlag的定义如下

```go
// StringFlag is a flag with type string
type StringFlag struct {
	Name        string
	Usage       string
	EnvVar      string
	FilePath    string
	Required    bool
	Hidden      bool
	TakesFile   bool
	Value       string
	Destination *string
}
```

如其说明一样，就是个带有string值的flag.

自定义的flag的值一般都有特殊含义和用处，DirectoryFlag定义如下

```go
type DirectoryString string

func (s *DirectoryString) String() string {
	return string(*s)
}

func (s *DirectoryString) Set(value string) error {
	*s = DirectoryString(expandPath(value))
	return nil
}

// Custom cli.Flag type which expand the received string to an absolute path.
// e.g. ~/.ethereum -> /home/username/.ethereum
type DirectoryFlag struct {
	Name   string
	Value  DirectoryString
	Usage  string
	EnvVar string
}

func (f DirectoryFlag) String() string {
	return cli.FlagStringer(f)
}

// called by cli library, grabs variable from environment (if in env)
// and adds variable to flag set for parsing.
func (f DirectoryFlag) Apply(set *flag.FlagSet) {
	eachName(f.Name, func(name string) {
		set.Var(&f.Value, f.Name, f.Usage)
	})
}

func (f DirectoryFlag) GetName() string {
	return f.Name
}

func (f *DirectoryFlag) Set(value string) {
	f.Value.Set(value)
}

```

Commands:

首先Command的定义如下：

```go
type Command struct {
	// The name of the command
	Name string
	// short name of the command. Typically one character (deprecated, use `Aliases`)
	ShortName string
	// A list of aliases for the command
	Aliases []string
	// A short description of the usage of this command
	Usage string
	// Custom text to show on USAGE section of help
	UsageText string
	// A longer explanation of how the command works
	Description string
	// A short description of the arguments of this command
	ArgsUsage string
	// The category the command is part of
	Category string
	// The function to call when checking for bash command completions
	BashComplete BashCompleteFunc
	// An action to execute before any sub-subcommands are run, but after the context is ready
	// If a non-nil error is returned, no sub-subcommands are run
	Before BeforeFunc
	// An action to execute after any subcommands are run, but after the subcommand has finished
	// It is run even if Action() panics
	After AfterFunc
	// The function to call when this command is invoked
	Action interface{}
	// TODO: replace `Action: interface{}` with `Action: ActionFunc` once some kind
	// of deprecation period has passed, maybe?

	// Execute this function if a usage error occurs.
	OnUsageError OnUsageErrorFunc
	// List of child commands
	Subcommands Commands
	// List of flags to parse
	Flags []Flag
	// Treat all flags as normal arguments if true
	SkipFlagParsing bool
	// Skip argument reordering which attempts to move flags before arguments,
	// but only works if all flags appear after all arguments. This behavior was
	// removed n version 2 since it only works under specific conditions so we
	// backport here by exposing it as an option for compatibility.
	SkipArgReorder bool
	// Boolean to hide built-in help command
	HideHelp bool
	// Boolean to hide this command from help or completion
	Hidden bool
	// Boolean to enable short-option handling so user can combine several
	// single-character bool arguments into one
	// i.e. foobar -o -v -> foobar -ov
	UseShortOptionHandling bool

	// Full name of command for help, defaults to full command name, including parent commands.
	HelpName        string
	commandNamePath []string

	// CustomHelpTemplate the text template for the command help topic.
	// cli.go uses text/template to render templates. You can
	// render custom help text by setting this variable.
	CustomHelpTemplate string
}
```

其中比较重要的是：

name：启动的命令

Action：输入参数之后的运行函数

Flags：要解析的flag

现在看一下初始化区块链的命令：

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

其utils.MigrateFlags方法可以先不管，大概就是格式化一下

其解析了DataDirFlag，对应着输入的命令 geth --datadir /data genesis.json init 这行启动命令中的datadir设置。

Before和After：

源码定义如下：

```go
	Before BeforeFunc
	// An action to execute after any subcommands are run, but after the subcommand has finished
	// It is run even if Action() panics
	After AfterFunc

	// The action to execute when no subcommands are specified
	// Expects a `cli.ActionFunc` but will accept the *deprecated* signature of `func(*cli.Context) {}`
	// *Note*: support for the deprecated `Action` signature will be removed in a future version
```

就是在命令构建之前和之后跑的工具

函数定义：

```go
	app.Before = func(ctx *cli.Context) error {
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		console.Stdin.Close() // Resets terminal mode.
		return nil
	}
```

大概干了这几件事：

开始之前启动日志系统，结束之后关闭日志系统，并且重置终端。

###### 3.geth启动函数：

定义如下：

```go
// geth is the main entry point into the system if no special subcommand is ran.
// It creates a default node based on the command line arguments and runs it in
// blocking mode, waiting for it to be shut down.
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

基于命令行参数创建一个默认节点，并以阻塞模式运行它，等待它关闭。

###### 4.initGenesis方法：

完整的定义如下：

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

大概是先检查json文件格式是否存在，是否符合要求，之后构建数据库