package containers

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func RunContainerInitProcess() error {
	// 从管道读取父进程的消息
	commands := readParentCommands()
	if len(commands) == 0 {
		return fmt.Errorf("wrong commands")
	}
	logger.Sugar().Infof("Commands is %v\n", commands)

	// 重新挂载`/proc`
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	// 查找命令的绝对路径
	path, err := exec.LookPath(commands[0])
	if err != nil {
		logger.Sugar().Error(err.Error())
		return err
	}
	logger.Sugar().Infof("find path %s\n", path)

	if err := syscall.Exec(path, commands, os.Environ()); err != nil {
		logger.Sugar().Error(err.Error())
		return err
	}
	return nil
}

func readParentCommands() []string {
	pipe := os.NewFile(uintptr(3), "pipe")
	res, err := ioutil.ReadAll(pipe)
	if err != nil {
		logger.Sugar().Error(err)
		return nil
	}
	defer pipe.Close()

	return strings.Split(string(res), " ")
}
