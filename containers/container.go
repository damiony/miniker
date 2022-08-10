package containers

import (
	"errors"
	"miniker/subsystems"

	"github.com/urfave/cli/v2"
	"go.uber.org/zap"
)

var logger *zap.Logger

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
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("wrong container command")
			}
			cmd := ctx.Args().Slice()
			tty := ctx.Bool("it")
			volume := ctx.String("v")
			subsystemConfig := &subsystems.SubsystemConfig{
				MemLimit: ctx.String("m"),
				CpuSet:   ctx.String("cpuset"),
				CpuShare: ctx.String("cpushare"),
			}
			Run(tty, cmd, subsystemConfig, volume)
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
			return nil
		},
	}
}
