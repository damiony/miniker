package containers

import (
	"errors"
	"fmt"
	"miniker/subsystems"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var logger *zap.Logger
var rootUrl string = "/root/software"
var mntUrl string = "/root/software/mnt"

func init() {
	logger, _ = zap.NewProduction()
}

func NewRunCommand() *cli.Command {
	return &cli.Command{
		Name:  "run",
		Usage: `Create a container. miniker run -it [command]`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "it",
				Usage: "enable tty",
			},
			&cli.StringFlag{
				Name:  "m",
				Usage: "memory",
			},
			&cli.StringFlag{
				Name:  "cpushare",
				Usage: "CPU shares (relative weight)",
			},
			&cli.StringFlag{
				Name:  "cpuset",
				Usage: "Cpus in which to allow execution",
			},
			&cli.StringFlag{
				Name:  "v",
				Usage: "Bind mount a volume",
			},
			&cli.StringFlag{
				Name:  "d",
				Usage: "Run container in background",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Assign a name to the container",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("wrong container command")
			}
			createTty := ctx.Bool("it")
			detach := ctx.String("d")
			if createTty && detach != "" {
				return errors.New("-it and -d cannot exist at the same time")
			}
			if !createTty && detach == "" {
				return errors.New("at least one of the '-it' and '-d' must exist")
			}

			volume := ctx.String("v")
			subsystemConfig := &subsystems.SubsystemConfig{
				MemLimit: ctx.String("m"),
				CpuSet:   ctx.String("cpuset"),
				CpuShare: ctx.String("cpushare"),
			}
			cmds := ctx.Args().Slice()
			containerName := ctx.String("name")
			Run(createTty, cmds, subsystemConfig, volume, containerName)
			return nil
		},
	}
}

func NewInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Init container process",
		Action: func(ctx *cli.Context) error {
			logger.Sugar().Info("init come on")
			err := RunContainerInitProcess()
			return err
		},
	}
}

func NewCommitCommand() *cli.Command {
	return &cli.Command{
		Name:  "commit",
		Usage: "Create a new image from a container",
		Action: func(ctx *cli.Context) error {
			logger.Sugar().Info("Commit a new image")
			if ctx.Args().Len() < 1 {
				return fmt.Errorf("wrong parameters")
			}
			imangeName := ctx.Args().Get(0)
			commitImage(imangeName)
			return nil
		},
	}
}

func NewPsCommand() *cli.Command {
	return &cli.Command{
		Name:  "ps",
		Usage: "List all containers info",
		Action: func(ctx *cli.Context) error {
			logger.Sugar().Info("List container info")
			listContainers()
			return nil
		},
	}
}
