package regdb

import (
	"encoding/json"
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
		result := Get(regDb, "chainConfig")
		chainConfig := new(Identity)
		if err := json.Unmarshal([]byte(result), &chainConfig); err != nil {
			utils.Fatalf("Failed to initialise database: %v", err)
		}
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
	return nil
}
