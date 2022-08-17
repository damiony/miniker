package networks

import (
	"errors"

	"github.com/urfave/cli/v2"
)

func NewNetworkCommand() *cli.Command {
	return &cli.Command{
		Name:  "network",
		Usage: "miniker network COMMAND",
		Subcommands: []*cli.Command{
			NewListCommand(),
			NewCreateCommand(),
			NewRemoveCommand(),
		},
	}
}

func NewListCommand() *cli.Command {
	return &cli.Command{
		Name:  "ls",
		Usage: "List networks",
		Action: func(ctx *cli.Context) error {
			listNetworks()
			return nil
		},
	}
}

func NewCreateCommand() *cli.Command {
	return &cli.Command{
		Name:  "create",
		Usage: "Create a network",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "subnet",
				Usage: "network segment",
			},
			&cli.StringFlag{
				Name:    "driver",
				Aliases: []string{"d"},
				Usage:   "Driver to manage the Network",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("pllease input network name")
			}
			subnet := ctx.String("subnet")
			driver := ctx.String("driver")
			name := ctx.Args().Get(0)
			return CreateNetwork(driver, subnet, name)
		},
	}
}

func NewRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:  "rm",
		Usage: "Remove one network",
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("please input network name")
			}
			networkName := ctx.Args().Get(0)
			return DeleteNetwork(networkName)
		},
	}
}
