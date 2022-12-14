package main

import (
	"log"
	"miniker/containers"
	"miniker/networks"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "miniker",
		Usage: "Simple docker runtime",
		Commands: []*cli.Command{
			containers.NewRunCommand(),
			containers.NewInitCommand(),
			containers.NewCommitCommand(),
			containers.NewPsCommand(),
			containers.NewLogsCommand(),
			containers.NewExecCommand(),
			containers.NewStopCommand(),
			containers.NewRemoveCommand(),
			networks.NewNetworkCommand(),
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
