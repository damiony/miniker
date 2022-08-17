package containers

import (
	"errors"
	"fmt"
	"miniker/subsystems"
	"os"

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
			&cli.BoolFlag{
				Name:  "d",
				Usage: "Run container in background",
			},
			&cli.StringFlag{
				Name:  "name",
				Usage: "Assign a name to the container",
			},
			&cli.StringFlag{
				Name:  "network",
				Usage: "Connect a container to a network",
			},
			&cli.StringSliceFlag{
				Name:  "p",
				Usage: "Publish a container's ports to the host",
			},
		},
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("wrong container command")
			}
			createTty := ctx.Bool("it")
			detach := ctx.Bool("d")
			if createTty && detach {
				return errors.New("-it and -d cannot exist at the same time")
			}
			if !createTty && !detach {
				return errors.New("at least one of the '-it' and '-d' must exist")
			}

			volume := ctx.String("v")
			subsystemConfig := &subsystems.SubsystemConfig{
				MemLimit: ctx.String("m"),
				CpuSet:   ctx.String("cpuset"),
				CpuShare: ctx.String("cpushare"),
			}
			imageName := ctx.Args().Get(0)
			cmds := ctx.Args().Slice()[1:]
			containerName := ctx.String("name")
			networkName := ctx.String("network")
			portMapping := ctx.StringSlice("p")
			Run(createTty, cmds, subsystemConfig, volume, containerName, imageName, networkName, portMapping)
			return nil
		},
	}
}

func NewInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Init container process",
		Action: func(ctx *cli.Context) error {
			logger.Sugar().Info("Init come on")
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
			if ctx.Args().Len() < 2 {
				return fmt.Errorf("please input container name and image name")
			}
			containerName := ctx.Args().Get(0)
			imangeName := ctx.Args().Get(1)
			commitImage(containerName, imangeName)
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

func NewLogsCommand() *cli.Command {
	return &cli.Command{
		Name:  "logs",
		Usage: "Fetch the logs of a container. miniker logs [containerName]",
		Action: func(ctx *cli.Context) error {
			logger.Sugar().Info("print logs of container")
			if ctx.Args().Len() == 0 {
				return errors.New("please input your container name")
			}
			containerName := ctx.Args().Get(0)
			printLogs(containerName)
			return nil
		},
	}
}

func NewExecCommand() *cli.Command {
	return &cli.Command{
		Name:  "exec",
		Usage: "Run a command in a running container",
		Action: func(ctx *cli.Context) error {
			if os.Getenv(ENV_EXEC_PID) != "" {
				logger.Sugar().Infof("enter containers, pid %d", os.Getpid())
				return nil
			}
			if ctx.Args().Len() < 2 {
				return errors.New("please input container name and commands")
			}
			containerName := ctx.Args().Get(0)
			commands := ctx.Args().Slice()[1:]
			execCommands(containerName, commands)
			return nil
		},
	}
}

func NewStopCommand() *cli.Command {
	return &cli.Command{
		Name:  "stop",
		Usage: "Stop running container",
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("please input container name")
			}
			containerName := ctx.Args().Get(0)
			stopContainer(containerName)
			return nil
		},
	}
}

func NewRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:  "remove",
		Usage: "Remove a container",
		Action: func(ctx *cli.Context) error {
			if ctx.Args().Len() < 1 {
				return errors.New("please input container name")
			}
			containerName := ctx.Args().Get(0)
			removeContainer(containerName)
			return nil
		},
	}
}
