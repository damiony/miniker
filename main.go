package main

import (
	"log"
	"miniker/containers"
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
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
