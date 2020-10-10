package utils
import(
	"github.com/urfave/cli"
)
var(
	PortFlag = cli.StringFlag{
		Name:  "port, p",
		Usage: "the port of this server",
		Value:  "1323",
	}

)
