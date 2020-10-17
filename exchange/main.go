package main

import (
	"echo-demo/crypto"
	"echo-demo/utils"
	"fmt"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/urfave/cli"
	"net/http"
	"os"
)
var (
	app = cli.NewApp()
	baseFlags = []cli.Flag{
		utils.PortFlag,
		utils.KeyFlag,
	}
	publisherpub   = crypto.PublicKey{}
	publisherpriv  = crypto.PrivateKey{}
	regulatorpub   = crypto.PublicKey{}
	cm_and_r       = crypto.Commitment{}
	elgamal        = crypto.CypherText{}
	signature      = crypto.Signature{}
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
	gk := ctx.String("generatekey")
	publisherpub, publisherpriv, _ = utils.GenerateKey(gk)
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
	amount    := c.FormValue("amount")
	//pk,_ := strconv.Atoi(publickey)
	//amount, _ := strconv.Atoi(c.FormValue("amount"))

	if(utils.Verify(publickey) == false){
		return c.JSON(http.StatusCreated, "error publickey, please input again or register now")
	}else{
		regulatorpub, _ = utils.GenerateRegKey()
		cm_and_r        = utils.CreateCM_v(regulatorpub, amount)
		elgamal         = utils.CreateElgamalC(regulatorpub, amount, publickey)
		signature       = utils.CreateSign(publisherpriv, amount)
		//TODO: sendTranscation
	}

	return c.JSON(http.StatusCreated, publickey)
}