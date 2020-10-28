package main

import (
	"exchange/crypto"
	"exchange/utils"
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
		utils.EthAccountFlag,
		utils.EthKeyFlag,
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
	ea := ctx.String("ethaccount")
	ek := ctx.String("ethkey")
	publisherpub, publisherpriv, _ = utils.GenerateKey(gk)
	regulatorpub, _ = utils.GenerateRegKey()
	if(utils.UnlockAccount(ea, ek) == true) {
		startNetwork(ctx)
	}else{
		fmt.Println("erro unlock exchanger eth_account")
		return
	}
}

func startNetwork(ctx *cli.Context) error {
	e := echo.New()
	port := ctx.String("port")

	// Root level middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.POST("/buy", buy)
	e.GET("/pubpub",pubpub)

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

	if(utils.Verify(publickey) == false){
		return c.JSON(http.StatusCreated, "error publickey, please check again or registe now")
	}else{
		cm_and_r        = utils.CreateCM_v(regulatorpub, amount)
		elgamal         = utils.CreateElgamalC(regulatorpub, amount, publickey)
		signature       = utils.CreateSign(publisherpriv, amount)
		//TODO: sendTranscation
	}

	return c.JSON(http.StatusCreated, publickey)
}

func pubpub(c echo.Context) error {
	return c.JSON(http.StatusCreated, publisherpub)
}