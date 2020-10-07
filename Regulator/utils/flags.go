package utils

import "github.com/urfave/cli"

var (
	DataportFlag = cli.IntFlag{
		Name:  "dataport, dp",
		Usage: "Data port for Redis",
		Value: 6379,
	}
	DatabaseFlag = cli.IntFlag{
		Name:  "database, db",
		Usage: "Number of database for Redis",
		Value: 0,
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "port, p",
		Usage: "Network listening port",
		Value: 1323,
	}
)
