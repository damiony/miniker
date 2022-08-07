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
		Name: "run",
		Usage: `create a container
				miniker run -it [command]`,
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
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("wrong container command")
			}
			cmd := ctx.Args().Slice()
			tty := ctx.Bool("it")
			subsystemConfig := &subsystems.SubsystemConfig{
				MemLimit: ctx.String("m"),
				CpuSet:   ctx.String("cpuset"),
				CpuShare: ctx.String("cpushare"),
			}
			Run(tty, cmd, subsystemConfig)
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
