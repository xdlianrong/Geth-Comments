#### 1.goja

goja 是一个 Go 实现的 ECMAScript 5.1(+)。

它不是 V8 或 SpiderMonkey 或任何其他通用 JavaScript 引擎的替代品，因为它更慢。它可以作为一种嵌入式脚本语言使用，或者可以作为避免非 Go 相关性的一种方式。

灵感来源于 [otto](https://github.com/robertkrimen/otto) 。完全支持 ECMAScript 5.1，通过几乎所有用 es5id 标记的 tc39 测试，平均比 otto 快6-7倍，同时使用相当少的内存。

简单的理解是，在go里面写javascript，这样就可以以脚本的方式实现控制台，并和geth实时交互。

#### 2.RPC

默认socket通信。本地机器的RPC框架反序列化出执行结果，函数return这个结果。

 ![img](https://upload-images.jianshu.io/upload_images/729358-839e7a32b598953e.png?imageMogr2/auto-orient/strip|imageView2/2/w/500/format/webp) 

**Call ID映射。**我们怎么告诉远程机器我们要调用Multiply，而不是Add或者FooBar呢？在本地调用中，函数体是直接通过函数指针来指定的，我们调用Multiply，编译器就自动帮我们调用它相应的函数指针。但是在远程调用中，函数指针是不行的，因为两个进程的地址空间是完全不一样的。所以，在RPC中，所有的函数都必须有自己的一个ID。这个ID在所有进程中都是唯一确定的。客户端在做远程过程调用时，必须附上这个ID。然后我们还需要在客户端和服务端分别维护一个 {函数 <--> Call ID} 的对应表。两者的表不一定需要完全相同，但相同的函数对应的Call ID必须相同。当客户端需要进行远程调用时，它就查一下这个表，找出相应的Call ID，然后把它传给服务端，服务端也通过查表，来确定客户端需要调用的函数，然后执行相应函数的代码。

**序列化和反序列化。**客户端怎么把参数值传给远程的函数呢？在本地调用中，我们只需要把参数压到栈里，然后让函数自己去栈里读就行。但是在远程过程调用时，客户端跟服务端是不同的进程，不能通过内存来传递参数。甚至有时候客户端和服务端使用的都不是同一种语言（比如服务端用C++，客户端用Java或者Python）。这时候就需要客户端把参数先转成一个字节流，传给服务端后，再把字节流转成自己能读取的格式。这个过程叫序列化和反序列化。同理，从服务端返回的值也需要序列化反序列化的过程。

**网络传输。**远程调用往往用在网络上，客户端和服务端是通过网络连接的。所有的数据都需要通过网络传输，因此就需要有一个网络传输层。网络传输层需要把Call ID和序列化后的参数字节流传给服务端，然后再把序列化后的调用结果传回客户端。只要能完成这两者的，都可以作为传输层使用。因此，它所使用的协议其实是不限的，能完成传输就行。尽管大部分RPC框架都使用TCP协议，但其实UDP也可以，而gRPC干脆就用了HTTP2。Java的Netty也属于这层的东西。

 ![img](https://upload-images.jianshu.io/upload_images/729358-de38e5423ffbc3e7.png?imageMogr2/auto-orient/strip|imageView2/2/w/589/format/webp) 

**3.控制台原理**

geth/main.go

```go
func init() {
	// Initialize the CLI app and start Geth
	app.Action = geth
	app.HideVersion = true // we have a command to print the version
	app.Copyright = "Copyright 2013-2020 The go-ethereum Authors"
	app.Commands = []cli.Command{
		// See chaincmd.go:
		...
		consoleCommand,
		...
	}
```

geth/consolecmd.go

```go
consoleCommand = cli.Command{
		Action:   utils.MigrateFlags(localConsole),
		Name:     "console",
		Usage:    "Start an interactive JavaScript environment",
		Flags:    append(append(append(nodeFlags, rpcFlags...), consoleFlags...), whisperFlags...),
		Category: "CONSOLE COMMANDS",
		Description: `
The Geth console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://github.com/ethereum/go-ethereum/wiki/JavaScript-Console.`,
	}



// localConsole starts a new geth node, attaching a JavaScript console to it at the
// same time.
func localConsole(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
    // 和geth命令有些相似，只不过没有node.wait（），就是启动一个节点
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
	//新建控制台
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
    // 说欢迎，开启交互模式
	console.Welcome()
	console.Interactive()

	return nil
}


```

console/bridge.go

```go
// bridge is a collection of JavaScript utility methods to bride the .js runtime
// environment and the Go RPC connection backing the remote method calls.
type bridge struct {
	client   *rpc.Client  // RPC client to execute Ethereum requests through
	prompter UserPrompter // Input prompter to allow interactive user feedback输入提示，允许交互式用户反馈
	printer  io.Writer    // Output writer to serialize any display strings序列化 
}

// newBridge creates a new JavaScript wrapper around an RPC client.
func newBridge(client *rpc.Client, prompter UserPrompter, printer io.Writer) *bridge {
	return &bridge{
		client:   client,
		prompter: prompter,
		printer:  printer,
	}
}
```

console/console.go

```go
// init retrieves the available APIs from the remote RPC provider and initializes
// the console's JavaScript namespaces based on the exposed modules.
func (c *Console) init(preload []string) error {
	c.initConsoleObject()

	// Initialize the JavaScript <-> Go RPC bridge.
	bridge := newBridge(c.client, c.prompter, c.printer)
	if err := c.initWeb3(bridge); err != nil {
		return err
	}
	if err := c.initExtensions(); err != nil {
		return err
	}

	// Add bridge overrides for web3.js functionality.
	c.jsre.Do(func(vm *goja.Runtime) {
		c.initAdmin(vm, bridge)
		c.initPersonal(vm, bridge)
	})

	// Preload JavaScript files.
	for _, path := range preload {
		if err := c.jsre.Exec(path); err != nil {
			failure := err.Error()
			if gojaErr, ok := err.(*goja.Exception); ok {
				failure = gojaErr.String()
			}
			return fmt.Errorf("%s: %v", path, failure)
		}
	}

	// Configure the input prompter for history and tab completion.
    // 为历史记录和制表符完成配置输入提示。

	if c.prompter != nil {
		if content, err := ioutil.ReadFile(c.histPath); err != nil {
			c.prompter.SetHistory(nil)
		} else {
			c.history = strings.Split(string(content), "\n")
			c.prompter.SetHistory(c.history)
		}
		c.prompter.SetWordCompleter(c.AutoCompleteInput)
	}
	return nil
}

func New(config Config) (*Console, error) {
	// Handle unset config values gracefully
	if config.Prompter == nil {
		config.Prompter = Stdin
	}
	if config.Prompt == "" {
		config.Prompt = DefaultPrompt
	}
	if config.Printer == nil {
		config.Printer = colorable.NewColorableStdout()
	}

	// Initialize the console and return
	console := &Console{
		client:   config.Client,
		jsre:     jsre.New(config.DocRoot, config.Printer),
		prompt:   config.Prompt,
		prompter: config.Prompter,
		printer:  config.Printer,
		histPath: filepath.Join(config.DataDir, HistoryFile),
	}
	if err := os.MkdirAll(config.DataDir, 0700); err != nil {
		return nil, err
	}
	if err := console.init(config.Preload); err != nil {
		return nil, err
	}
	return console, nil
}

// Welcome show summary of current Geth instance and some metadata about the
// console's available modules.
func (c *Console) Welcome() {
	message := "Welcome to the Geth JavaScript console!\n\n"

	// Print some generic Geth metadata
	if res, err := c.jsre.Run(`
		var message = "instance: " + web3.version.node + "\n";
		try {
			message += "coinbase: " + eth.coinbase + "\n";
		} catch (err) {}
		message += "at block: " + eth.blockNumber + " (" + new Date(1000 * eth.getBlock(eth.blockNumber).timestamp) + ")\n";
		try {
			message += " datadir: " + admin.datadir + "\n";
		} catch (err) {}
		message
	`); 
    ...
	fmt.Fprintln(c.printer, message)
}

// Interactive starts an interactive user session, where input is propted from
// the configured user prompter.
func (c *Console) Interactive() {
    ...
}
```

#### 4.一个交互举例子

console/console.go

```go
func (c *Console) initWeb3(bridge *bridge) error {
	bnJS := string(deps.MustAsset("bignumber.js"))
	web3JS := string(deps.MustAsset("web3.js"))
	if err := c.jsre.Compile("bignumber.js", bnJS); err != nil {
		return fmt.Errorf("bignumber.js: %v", err)
	}
	if err := c.jsre.Compile("web3.js", web3JS); err != nil {
		return fmt.Errorf("web3.js: %v", err)
	}
	if _, err := c.jsre.Run("var Web3 = require('web3');"); err != nil {
		return fmt.Errorf("web3 require: %v", err)
	}
	var err error
	c.jsre.Do(func(vm *goja.Runtime) {
		transport := vm.NewObject()
		transport.Set("send", jsre.MakeCallback(vm, bridge.Send))
		transport.Set("sendAsync", jsre.MakeCallback(vm, bridge.Send))
		vm.Set("_consoleWeb3Transport", transport)
		_, err = vm.RunString("var web3 = new Web3(_consoleWeb3Transport)")
	})
	return err
}
```

拿其中的send方法举例

console/bridge.go

```go
// Send implements the web3 provider "send" method.
func (b *bridge) Send(call jsre.Call) (goja.Value, error) {
	// Remarshal the request into a Go value.
	reqVal, err := call.Argument(0).ToObject(call.VM).MarshalJSON()
	var (
		rawReq = string(reqVal)
		dec    = json.NewDecoder(strings.NewReader(rawReq))
		reqs   []jsonrpcCall
		batch  bool
	)
	dec.UseNumber() // avoid float64s
	if rawReq[0] == '[' {
		batch = true
		dec.Decode(&reqs)
	} else {
		batch = false
		reqs = make([]jsonrpcCall, 1)
		dec.Decode(&reqs[0])
	}

	// Execute the requests.
	var resps []*goja.Object
	for _, req := range reqs {
		resp := call.VM.NewObject()
		resp.Set("jsonrpc", "2.0")
		resp.Set("id", req.ID)

		var result json.RawMessage
        //比较重要的一步，rpc调用
		err = b.client.Call(&result, req.Method, req.Params...)
		switch err := err.(type) {
		case nil:
			if result == nil {
				// Special case null because it is decoded as an empty
				// raw message for some reason.
				resp.Set("result", goja.Null())
			} else {
				JSON := call.VM.Get("JSON").ToObject(call.VM)
				parse, callable := goja.AssertFunction(JSON.Get("parse"))
				if !callable {
					return nil, fmt.Errorf("JSON.parse is not a function")
				}
				resultVal, err := parse(goja.Null(), call.VM.ToValue(string(result)))
				if err != nil {
					setError(resp, -32603, err.Error())
				} else {
					resp.Set("result", resultVal)
				}
			}
		case rpc.Error:
			setError(resp, err.ErrorCode(), err.Error())
		default:
			setError(resp, -32603, err.Error())
		}
		resps = append(resps, resp)
	}

	// Return the responses either to the callback (if supplied)
	// or directly as the return value.
    
    //将响应返回给回调（如果提供）
    //或直接作为返回值。
	var result goja.Value
	if batch {
		result = call.VM.ToValue(resps)
	} else {
		result = resps[0]
	}
	if fn, isFunc := goja.AssertFunction(call.Argument(1)); isFunc {
		fn(goja.Null(), goja.Null(), result)
		return goja.Undefined(), nil
	}
	return result, nil
}

```

