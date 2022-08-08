package containers

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
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

	setUpMount()
	// 重新挂载`/proc`
	// defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	// syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	// 查找命令的绝对路径
	path, err := exec.LookPath(commands[0])
	if err != nil {
		// logger.Sugar().Error(err.Error())
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

func setUpMount() {
	pwd, err := os.Getwd()
	if err != nil {
		logger.Sugar().Errorf("Error get current location. %v", err)
		return
	}

	logger.Sugar().Infof("Current location is %s", pwd)
	pivotRoot(pwd)

	// 挂载/proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	syscall.Mount("proc", "/proc", "proc", uintptr(defaultMountFlags), "")

	// 挂载tmpfs
	syscall.Mount("tempfs", "/dev", "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
}

// 修改rootfs
func pivotRoot(newRoot string) error {
	// 重新挂载newRoot
	if err := syscall.Mount(newRoot, newRoot, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}
	// 创建oldRoot，用于存放旧的rootfs
	oldRoot := path.Join(newRoot, ".privot_root")
	if err := os.Mkdir(oldRoot, 0777); err != nil {
		return err
	}
	// 挂载rootfs到新的文件系统
	if err := syscall.PivotRoot(newRoot, oldRoot); err != nil {
		return err
	}
	// 修改工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		return err
	}
	// 删除挂载
	oldRoot = path.Join(newRoot, ".")
	if err := syscall.Unmount(oldRoot, syscall.MNT_DETACH); err != nil {
		return err
	}
	// 删除oldRoot
	return os.Remove(oldRoot)
}
