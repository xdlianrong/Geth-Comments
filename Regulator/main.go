package main

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"regulator/regdb"
	"regulator/utils"
)

const (
	clientIdentifier = "regulator" // Client identifier to advertise over the network
	clientVersion    = "1.0.0"
	clientUsage      = "Regulatory side server for ethereumZKP"
)

var (
	app       = cli.NewApp()
	baseFlags = []cli.Flag{
		utils.DatabaseFlag,
		utils.DataportFlag,
		utils.ListenPortFlag,
		utils.DbPasswdPortFlag,
	}
	regDb *redis.Client
)

func init() {
	app.Action = regulator
	app.Name = clientIdentifier
	app.Version = clientVersion
	app.Usage = clientUsage
	app.Commands = []cli.Command{regdb.InitCommand}
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

func prepare(ctx *cli.Context) error {
	//连接并初始化Redis, 接收数据库端口，密码，数据库号三个参数
	regDb = regdb.ConnectToDB(ctx.String("dataport"), ctx.String("passwd"), ctx.Int("database"))
	if !regdb.Exists(regDb, "chainConfig") {
		utils.Fatalf("Failed to start server,please initialise first")
	}
	//TODO:检查是否有公私钥匙：无则报错退出程序
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
	e.POST("/register", register)
	e.POST("/verify", verify)
	//TODO:写一个get路由regkey，接收参数chainid，如果此chainid正确就返回监管者公钥PublicKey
	//TODO:写一个post路由decrypto，接收参数监管者私钥X和需要解密的数据，如果X正确就返回解密后的明文数据
	// Start server
	e.Logger.Fatal(e.Start(":" + port))
}

func register(c echo.Context) error {
	u := new(regdb.Identity)
	if err := c.Bind(u); err != nil {
		return err
	}
	//fmt.Println(u.Hashky)
	hash := utils.Hash(u.Hashky)
	if err := regdb.Set(regDb, hash, u); err != nil {
		utils.Fatalf("Failed to set : %v", err)
		return c.String(http.StatusOK, "Fail!")
	}
	return c.String(http.StatusOK, "Successful!")
	//return c.JSON(http.StatusCreated, u)
}

func verify(c echo.Context) error {
	//publicKey := c.FormValue("publicKey")
	u := new(regdb.Identity)
	if err := c.Bind(u); err != nil {
		return err
	}
	if !regdb.Exists(regDb, utils.Hash(u.Hashky)) {
		return c.String(http.StatusOK, "False")
	}
	return c.String(http.StatusOK, "True")
}
