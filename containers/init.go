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

	err := setUpMount()
	if err != nil {
		return err
	}
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

func setUpMount() error {
	pwd, err := os.Getwd()
	if err != nil {
		logger.Sugar().Errorf("Error get current location. %v", err)
		return err
	}

	logger.Sugar().Infof("Current location is %s", pwd)

	// Todo：挂载/proc和tmpfs需要在privotRoot之前执行，否则会提示无权限

	// 挂载/proc
	defaultMountFlags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV
	err = syscall.Mount("proc", path.Join(pwd, "proc"), "proc", uintptr(defaultMountFlags), "")
	if err != nil {
		logger.Sugar().Errorf("error mount proc. %v", err)
		return err
	}

	// 挂载tmpfs
	err = syscall.Mount("tempfs", path.Join(pwd, "dev"), "tmpfs", syscall.MS_NOSUID|syscall.MS_STRICTATIME, "mode=755")
	if err != nil {
		logger.Sugar().Errorf("error mount /dev. %v", err)
		return err
	}

	err = pivotRoot(pwd)
	if err != nil {
		logger.Sugar().Errorf("error privot_root %v", err)
		return err
	}
	return nil
}

// 修改rootfs
func pivotRoot(newRoot string) error {
	// 重新挂载newRoot
	if err := syscall.Mount(newRoot, newRoot, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		logger.Sugar().Error(err)
		return err
	}
	// 创建oldRoot，用于存放旧的rootfs
	oldRoot := path.Join(newRoot, ".privot_root")
	if err := os.MkdirAll(oldRoot, 0777); err != nil {
		logger.Sugar().Error(err)
		return err
	}
	logger.Sugar().Infof("create old root %s", oldRoot)
	// 挂载rootfs到新的文件系统
	if err := syscall.PivotRoot(newRoot, oldRoot); err != nil {
		logger.Sugar().Error(err)
		return err
	}
	// 修改工作目录到根目录
	if err := syscall.Chdir("/"); err != nil {
		logger.Sugar().Error(err)
		return err
	}
	// 删除挂载
	oldRoot = path.Join("/", ".privot_root")
	if err := syscall.Unmount(oldRoot, syscall.MNT_DETACH); err != nil {
		logger.Sugar().Error(err)
		return err
	}
	// 删除oldRoot
	return os.Remove(oldRoot)
}
