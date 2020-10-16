package regdb

import (
	"fmt"
	"github.com/urfave/cli"
	"regulator/utils"
)

var (
	InitCommand = cli.Command{
		Action:    utils.MigrateFlags(InitDB),
		Name:      "init",
		Usage:     "Initialize a new publickey pool",
		ArgsUsage: "<genesisPath>",
		Flags: []cli.Flag{
			utils.ChainIDFlag,
			utils.DatabaseFlag,
			utils.DataportFlag,
			utils.DbPasswdPortFlag,
		},
		Category: "BASE COMMANDS",
		Description: `
The init command initializes a new Redis database for the server.
This is a destructive action and changes the network in which you will be
participating.

It expects the chainID as argument.`,
	}
)

// InitDB will initialise the given chainID and writes it into
// Redis as chain's mark or will fail hard if it can't succeed.
func InitDB(ctx *cli.Context) error {
	chainID := ctx.String("chainID")
	regDb := ConnectToDB(ctx.String("dataport"), ctx.String("passwd"), ctx.Int("database"))
	if Exists(regDb, "chainConfig") {
		chainConfig := Get(regDb, "chainConfig").(*Identity)
		if chainConfig.ID == chainID {
			fmt.Println("Database has been initialised by chainID", chainID, "sometimes before")
		} else {
			utils.Fatalf("Database has been initialised by chainID " + chainConfig.ID)
		}
	} else {
		err := Set(regDb, "chainConfig", &Identity{
			Name:    "",
			ID:      chainID,
			Hashky:  "",
			ExtInfo: "",
		})
		if err != nil {
			utils.Fatalf("Failed to initialise database: %v", err)
		}
	}
	// 判断db有无公私钥，无则生成，有则什么都不干
	if !Exists(regDb, "key") {
		_, priv, err := utils.GenElgKeys()
		if err != nil {
			utils.Fatalf("%v", err)
		}
		if err := Set(regDb, "key", priv); err != nil {
			utils.Fatalf("Failed to set : %v", err)
		}
		//fmt.Printf("公钥：P:%x\nG1:%x\nG2:%x\nH:%x\n私钥：\nX:%x\n", pub.P, pub.G1, pub.G2, pub.H, priv.X)
	}
	return nil
}
