package containers

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// 停止正在运行的容器
func stopContainer(containerName string) {
	// 获取容器信息
	containerInfo := getContainerInfo(containerName)
	if containerInfo == nil {
		logger.Sugar().Errorf("Cannot get container info by name %s", containerName)
		return
	}

	// 获取pid
	pid, err := strconv.Atoi(containerInfo.Pid)
	if err != nil {
		logger.Sugar().Errorf("convert string %s err %v", containerInfo.Pid, err)
		return
	}

	// 发送SIGTERM信号
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil {
		logger.Sugar().Errorf("kill pid %d err %v", pid, err)
		// return
	}

	// 修改容器状态
	containerInfo.Status = EXIT
	containerInfo.Pid = ""
	updateContainerInfo(containerInfo)

	// 卸载mntUrl
	mntUrl := fmt.Sprintf(MntUrl, os.Getenv("HOME"), containerName)
	cmd := exec.Command("umount", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("umount %s err %v", mntUrl, err)
		// return
	}
}
