package utils

import (
	"github.com/urfave/cli"
)

var (
	PortFlag = cli.StringFlag{
		Name:  "port, p",
		Usage: "the port of this server",
		Value: "1323",
	}
	KeyFlag = cli.StringFlag{
		Name:  "generatekey, gk",
		Usage: "the string that you generate your pub/pri key",
		Value: "",
	}
	EthAccountFlag = cli.StringFlag{
		Name:  "ethaccount, ea",
		Usage: "the eth_account of you",
		Value: "",
	}
	EthKeyFlag = cli.StringFlag{
		Name:  "ethkey, ek",
		Usage: "the key that you unlock your eth_account",
		Value: "",
	}
)
