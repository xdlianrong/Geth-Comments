```go
@author FuMing
@data 2020.03.10
```

# Go 命令行cli库的使用

[TOC]



## 类库包下载

```
go get github.com/urfave/cli
```

## 使用举例

### 基本使用

一个cli应用程序可以只需要main()中的一行代码。

```go
//test.go
package main

import (
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
)
func main() {
  err := cli.NewApp().Run(os.Args)
  if err != nil {
    log.Fatal(err)
  }
}
```

运行结果

```tex
fuming@fumingdeMacBook-Pro test % go run test.go              
NAME:
   test - A new cli application

USAGE:
   test [global options] command [command options] [arguments...]

VERSION:
   0.0.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

此app运行后会显示帮助文本，但不是很有用，接下来我们写入一个action来执行它。

```go
//test.go
package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
)
func main() {
	app := cli.NewApp()
	app.Name = "boom"
	app.Usage = "make an explosive entrance"
	app.Action = func(c *cli.Context) error {
		fmt.Println("boom! I say!")
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
//执行结果
//fuming@fumingdeMacBook-Pro test % go run test.go
//boom! I say!
```

我们也可以从终端获取参数

```go
//test.go
package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
)
func main() {
	app := cli.NewApp()

	app.Action = func(c *cli.Context) error {
		log.Printf("argu:%v", c.Args())
		for i := 0; i < len(c.Args()); i++ {
			log.Printf("argu%d:%v", i,c.Args().Get(i))
		}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
//执行结果
//fuming@fumingdeMacBook-Pro test % go run test.go argu01 argu02 argu03
//2020/03/12 23:13:46 argu:[argu01 argu02 argu03]
//2020/03/12 23:13:46 argu0:argu01
//2020/03/12 23:13:46 argu1:argu02
//2020/03/12 23:13:46 argu2:argu03
```

### Flag

```go
//test.go
package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
)
func main() {
	app := cli.NewApp()

	app.Flags = []cli.Flag {
		cli.StringFlag{
			Name: "lang",
			Value: "english",
			Usage: "language for the greeting",
		},
	}

	app.Action = func(c *cli.Context) error {
		name := "Nefertiti"	//声明变量并定义默认值
		if c.NArg() > 0 {
			name = c.Args().Get(0)
		}
		if c.String("lang") == "spanish" {
			fmt.Println("Hola", name)
		} else {
			fmt.Println("Hello", name)
		}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
//运行结果
//fuming@fumingdeMacBook-Pro test % go run test.go -lang spanish programer
//Hola programer
//fuming@fumingdeMacBook-Pro test % go run test.go -lang english programer
//Hello programer
```

也可以将参数与变量绑定

```go
//test.go
package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
)
func main() {
	var language string

	app := cli.NewApp()

	app.Flags = []cli.Flag {
		cli.StringFlag{
			Name: "lang",
			Value: "english",
			Usage: "language for the greeting",
			Destination: &language,
		},
	}

	app.Action = func(c *cli.Context) error {
		name := "Nefertiti"	//声明变量并定义默认值
		if c.NArg() > 0 {
			name = c.Args().Get(0)
		}
		if language == "spanish" {
			fmt.Println("Hola", name)
		} else {
			fmt.Println("Hello", name)
		}
		return nil
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
//运行结果
//fuming@fumingdeMacBook-Pro test % go run test.go -lang spanish programer
//Hola programer
//fuming@fumingdeMacBook-Pro test % go run test.go -lang english programer
//Hello programer
```

### Subcommands

Subcommands允许我们为命令定义子命令，与Linux终端命令相似。

```go
//test.go
package main

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"log"
	"os"
)
func main() {
	app := cli.NewApp()

	app.Commands = []cli.Command{
		{
			Name:    "add",
			Aliases: []string{"a"},
			Usage:   "add a task to the list",
			Action:  func(c *cli.Context) error {
				fmt.Println("added task: ", c.Args().First())
				return nil
			},
		},
		{
			Name:    "complete",
			Aliases: []string{"c"},
			Usage:   "complete a task on the list",
			Action:  func(c *cli.Context) error {
				fmt.Println("completed task: ", c.Args().First())
				return nil
			},
		},
		{
			Name:        "template",
			Aliases:     []string{"t"},
			Usage:       "options for task templates",
			Subcommands: []cli.Command{
				{
					Name:  "add",
					Usage: "add a new template",
					Action: func(c *cli.Context) error {
						fmt.Println("new task template: ", c.Args().First())
						return nil
					},
				},
				{
					Name:  "remove",
					Usage: "remove an existing template",
					Action: func(c *cli.Context) error {
						fmt.Println("removed task template: ", c.Args().First())
						return nil
					},
				},
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
//运行结果
//fuming@fumingdeMacBook-Pro test % go run test.go add t
//added task:  t
//fuming@fumingdeMacBook-Pro test % go run test.go complete a
//completed task:  a
//fuming@fumingdeMacBook-Pro test % go run test.go template add a
//new task template:  a
//fuming@fumingdeMacBook-Pro test % go run test.go template remove a
//removed task template:  a
```

## 参考文献

+ [Github cli v1 手册](https://github.com/urfave/cli/blob/master/docs/v1/manual.md)

+ [cli文档](http://godoc.org/github.com/urfave/cli)