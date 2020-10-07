package main

import (
	"echo/utils"
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

var (
	app       = cli.NewApp()
	baseFlags = []cli.Flag{
		utils.DatabaseFlag,
		utils.DataportFlag,
		utils.ListenPortFlag,
	}
)

func init() {
	app.Action = regulator
	app.Name = clientIdentifier
	app.Version = clientVersion
	app.Flags = append(app.Flags, baseFlags...)
}
func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func regulator(ctx *cli.Context) error {
	if args := ctx.Args(); len(args) > 0 {
		return fmt.Errorf("invalid command: %q", args[0])
	}
	prepare(ctx)
	return nil
}

// Handler
func hello(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}
func prepare(ctx *cli.Context) error {
	//TODO:连接并初始化Redis, --dataport是ctx.String("dataport"), --database是ctx.String("database")
	startNetwork(ctx.String("port"))
	return nil
}
func startNetwork(port string) {

	// Echo instance
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Routes
	e.GET("/", hello)

	// Start server
	e.Logger.Fatal(e.Start(":" + port))
}
