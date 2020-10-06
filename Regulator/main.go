package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/urfave/cli"
	"net/http"
	"os"
)

const (
	clientIdentifier = "regulator" // Client identifier to advertise over the network
	clientVersion    = "1.0.0"
)

func init() {
	//TODO:初始化
	app := cli.NewApp()
	app.Name = clientIdentifier
	app.Version = clientVersion
	app.Commands = []cli.Command{
		{
			// 命令的名字
			Name: "",
			// 命令的缩写，就是不输入language只输入lang也可以调用命令
			Aliases: []string{""},
			// 命令的用法注释，这里会在输入 程序名 -help的时候显示命令的使用方法
			Usage: "",
			// 命令的处理函数
			Action: func(c *cli.Context) error {
				fmt.Println(c.Args().First())
				return nil
			},
		},
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "datadir, d",
			Usage: "Data directory for the databases",
		},
		cli.IntFlag{
			Name:  "port, p",
			Usage: "Network listening port",
			Value: 1323,
		},
	}
	app.Run(os.Args)
}
func main() {
	startNetwork(os.Args)
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
func startNetwork(arguments []string) {

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)

	// Start server
	e.Logger.Fatal(e.Start(":1323"))
}
