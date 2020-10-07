package utils

import (
	"github.com/urfave/cli"
	"os"
)

var (
	InitCommand = cli.Command{
		Action:    MigrateFlags(InitDB),
		Name:      "init",
		Usage:     "Bootstrap and initialize a new publickey pool",
		ArgsUsage: "<genesisPath>",
		Flags: []cli.Flag{
			DataDirFlag,
		},
		Category: "BLOCKCHAIN COMMANDS",
		Description: `
The init command initializes a new genesis block and definition for the network.
This is a destructive action and changes the network in which you will be
participating.

It expects the genesis file as argument.`,
	}
)

// initGenesis will initialise the given JSON format genesis file and writes it as
// the zero'd block (i.e. genesis) or will fail hard if it can't succeed.
func InitDB(ctx *cli.Context) error {
	// Make sure we have a valid genesis JSON
	genesisPath := ctx.Args().First()
	if len(genesisPath) == 0 {
		Fatalf("Must supply path to genesis JSON file")
	}
	_, err := os.Open(genesisPath)
	return err
}
