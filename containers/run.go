package containers

import (
	"miniker/subsystems"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func Run(tty bool, args []string, cfg *subsystems.SubsystemConfig) {
	parent, writePipe := NewParentProcess(tty)
	if parent == nil {
		logger.Sugar().Error("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		logger.Sugar().Error(err)
		return
	}

	// 创建cgroup管理器
	cgroupManager := subsystems.NewCgroupManager("miniker", cfg)
	defer cgroupManager.Destroy()
	// 设置资源限制
	cgroupManager.Set()
	// 将容器进程加入到cgroup
	cgroupManager.Apply(parent.Process.Pid)

	// 将父进程的命令参数传递给子进程
	sendCommandsToPipe(writePipe, args)
	parent.Wait()
}

func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
	// 创建管道，用于进程间通信
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logger.Sugar().Error("error create pipe :", err)
		return nil, nil
	}
	// 调用进程自身，进而创建出一个新进程，新进程位于隔离环境中
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWNS,
	}
	// 将`readPipe`传递给新进程，用于读取父进程传递给它的消息
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.ExtraFiles = []*os.File{readPipe}
	return cmd, writePipe
}

func sendCommandsToPipe(pipe *os.File, commands []string) {
	msgs := strings.Join(commands, " ")
	logger.Sugar().Infof("send msg %s\n", msgs)
	defer pipe.Close()

	_, err := pipe.WriteString(msgs)
	if err != nil {
		logger.Sugar().Error("error send commands :", err)
	}
}
