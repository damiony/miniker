package containers

import (
	"errors"
	"fmt"
	"miniker/networks"
	"miniker/subsystems"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

// run命令的主要执行逻辑
func Run(tty bool, args []string, cfg *subsystems.SubsystemConfig, vol, cName, iName, netName string, portM []string) {
	if cName == "" {
		cName = generateId()
	}

	parent, writePipe := NewParentProcess(tty, vol, cName, iName)
	if parent == nil {
		logger.Sugar().Error("Failed to create container process")
		return
	}
	if err := parent.Start(); err != nil {
		logger.Sugar().Error(err)
		return
	}

	cName = recordContainerInfo(parent.Process.Pid, cName, args)
	// 创建cgroup管理器
	cgroupManager := subsystems.NewCgroupManager("miniker", cfg)
	// 设置资源限制
	cgroupManager.Set()
	// 将容器进程加入到cgroup
	cgroupManager.Apply(parent.Process.Pid)

	// 将容器连接到指定网络
	if err := networks.Connect(netName, cName, []string{}, parent.Process.Pid); err != nil {
		logger.Sugar().Error(err)
	}
	// 将父进程的命令参数传递给子进程
	sendCommandsToPipe(writePipe, args)

	if tty {
		parent.Wait()
		// 删除工作目录
		deleteWorkSpace(cName, vol)
		// 删除容器信息
		deleteContainerInfo(cName)
		// 释放cgroup资源
		cgroupManager.Destroy()
	}
	logger.Sugar().Info(tty)
	// os.Exit(0)
}

// 创建子进程，执行init命令
func NewParentProcess(createTty bool, volume, containerName, imageName string) (*exec.Cmd, *os.File) {
	// 创建管道，用于进程间通信
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		logger.Sugar().Error("error create pipe :", err)
		return nil, nil
	}
	// 调用进程自身，进而创建出一个新进程，新进程位于隔离环境中
	cmd := exec.Command("/proc/self/exe", "init")
	// 需要配置CLONE_NEWUSER，否则执行pivot_root时会一直提示参数错误
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWIPC |
			syscall.CLONE_NEWPID |
			syscall.CLONE_NEWUTS |
			syscall.CLONE_NEWNET |
			syscall.CLONE_NEWUSER |
			syscall.CLONE_NEWNS,
		UidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getuid(),
				Size:        1,
			},
		},
		GidMappings: []syscall.SysProcIDMap{
			{
				ContainerID: 0,
				HostID:      os.Getgid(),
				Size:        1,
			},
		},
	}

	// 重定向标准输入、标准输出和标准错误
	if createTty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		logFile, err := createLogFile(containerName)
		if err != nil {
			logger.Sugar().Errorf("create log file err %v", err)
			return nil, nil
		}
		// cmd.Stdin = os.Stdin
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	// 将`readPipe`传递给新进程，用于读取父进程传递给它的消息
	cmd.ExtraFiles = []*os.File{readPipe}
	// 创建工作目录
	if err := NewWorkSpace(imageName, containerName, volume); err != nil {
		return nil, nil
	}

	cmd.Dir = fmt.Sprintf(MntUrl, os.Getenv("HOME"), containerName)
	return cmd, writePipe
}

func sendCommandsToPipe(pipe *os.File, commands []string) {
	msgs := strings.Join(commands, " ")
	logger.Sugar().Infof("Send msg %s\n", msgs)
	defer pipe.Close()

	_, err := pipe.WriteString(msgs)
	if err != nil {
		logger.Sugar().Error("error send commands :", err)
	}
}

// 为容器创建工作目录
func NewWorkSpace(imageName, containerName, volume string) error {
	if err := createReadOnlyLayer(imageName); err != nil {
		return err
	}
	if err := createWriteLayer(containerName); err != nil {
		return err
	}
	if err := createMountPoint(imageName, containerName); err != nil {
		return err
	}
	if err := mountVolume(containerName, volume); err != nil {
		return err
	}
	return nil
}

// 创建只读层
func createReadOnlyLayer(imageName string) error {
	cur, err := os.Getwd()
	if err != nil {
		logger.Sugar().Errorf("cannot get pwd %v", err)
		return err
	}

	unTarUrl := path.Join(cur, "resources", imageName) + ".tar"
	if file, err := os.Stat(unTarUrl); err != nil || file.IsDir() {
		errMsg := fmt.Sprintf("%s is not a directory", unTarUrl)
		logger.Sugar().Error(errMsg)
		return errors.New(errMsg)
	}

	imageUrl := fmt.Sprintf(ImageUrl, os.Getenv("HOME"), imageName)
	if err := os.MkdirAll(imageUrl, 0777); err != nil {
		logger.Sugar().Errorf("error create read only layer. %v", err)
		return err
	}

	if _, err := exec.Command("tar", "-xf", unTarUrl, "-C", imageUrl).CombinedOutput(); err != nil {
		logger.Sugar().Errorf("error tar %s. %v", err, unTarUrl)
		return err
	}
	logger.Sugar().Infof("Create readonly dir %s", imageUrl)

	return nil
}

// 创建读写层
func createWriteLayer(containerName string) error {
	writeUrl := fmt.Sprintf(WriteLayer, os.Getenv("HOME"), containerName)
	if exist, _ := pathExists(writeUrl); exist {
		return nil
	}
	if err := os.MkdirAll(writeUrl, 0777); err != nil {
		logger.Sugar().Errorf("error mkdir %s. %v", writeUrl, err)
		return err
	}
	return nil
}

// 将读写层挂载为aufs
func createMountPoint(imageName, containerName string) error {
	mntUrl := fmt.Sprintf(MntUrl, os.Getenv("HOME"), containerName)
	if err := os.MkdirAll(mntUrl, 0777); err != nil {
		logger.Sugar().Errorf("Create dir %s err %v", mntUrl, err)
		return err
	}

	imageUrl := fmt.Sprintf(ImageUrl, os.Getenv("HOME"), imageName)
	writeUrl := fmt.Sprintf(WriteLayer, os.Getenv("HOME"), containerName)
	// 将只读层和可写层挂载到mntUrl
	dirs := "dirs=" + writeUrl + ":" + imageUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("error mount aufs. %v", err)
		return err
	}
	return nil
}

// 检查文件路径是否存在
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// 删除容器的工作目录
func deleteWorkSpace(containerName, volume string) {
	mntUrl := fmt.Sprintf(MntUrl, os.Getenv("HOME"), containerName)
	if volume != "" {
		volumes := volumeUrlExtract(volume)
		// 如果volumes参数无效
		if len(volumes) != 2 || volumes[0] == "" || volumes[1] == "" {
			deleteMountPoint(mntUrl)
		} else {
			volUrl := path.Join(mntUrl, volumes[1])
			deleteMountPointWithVolume(mntUrl, volUrl)
		}
	} else {
		deleteMountPoint(mntUrl)
	}
}

// 删除挂载点
func deleteMountPoint(mntUrl string) error {
	// 卸载
	cmd := exec.Command("umount", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("error umount %s. %v", mntUrl, err)
		return err
	}
	// 删除
	if err := os.RemoveAll(mntUrl); err != nil {
		logger.Sugar().Errorf("error remove %s. %v", mntUrl, err)
		return err
	}
	return nil
}

// 删除volume和容器挂载点
func deleteMountPointWithVolume(mntUrl string, volUrl string) error {
	// 卸载volume
	if err := umountVolume(volUrl); err != nil {
		return err
	}
	if err := deleteMountPoint(mntUrl); err != nil {
		return err
	}
	return nil
}
