package main

import (
	"echo-demo/utils"
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/urfave/cli"
	"net/http"
	"os"
	"strconv"
)
var (
	app = cli.NewApp()
	baseFlags = []cli.Flag{
		utils.PortFlag,
	}

)


func init() {
	app.Name = "exchange"
	app.Usage = "user exchange from there"
	app.Action = exchange
	app.Flags = append(app.Flags,baseFlags...)

}



func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func exchange(ctx *cli.Context)  {
	startNetwork(ctx)
}

func startNetwork(ctx *cli.Context) error {
	e := echo.New()
	port := ctx.String("port")

	// Root level middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/buy", buy)
	e.Logger.Fatal(e.Start(":" + port))
	return nil
}

func buy(c echo.Context) error {
	u := new(utils.Purchase)
	if err := c.Bind(u); err != nil {
		return err
	}
	publickey := c.FormValue("publickey")
	//pk,_ := strconv.Atoi(publickey)
	amount, _ := strconv.Atoi(c.FormValue("amount"))
	// TODO: 发送http请求到监管者服务器
	utils.Verify(publickey)
	utils.CreateCM_v(publickey, amount)

	return c.JSON(http.StatusCreated, publickey)
}