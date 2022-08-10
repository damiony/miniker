package containers

import (
	"miniker/subsystems"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"
)

// run命令的主要执行逻辑
func Run(tty bool, args []string, cfg *subsystems.SubsystemConfig, volume string) {
	parent, writePipe := NewParentProcess(tty, volume)
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

	// // 删除工作目录
	rootUrl := "/root/software/"
	mntUrl := "/root/software/mnt/"
	deleteWorkSpace(rootUrl, mntUrl, volume)
	os.Exit(0)
}

// 创建子进程，执行init命令
func NewParentProcess(tty bool, volume string) (*exec.Cmd, *os.File) {
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
	// 将`readPipe`传递给新进程，用于读取父进程传递给它的消息
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	cmd.ExtraFiles = []*os.File{readPipe}

	rootUrl := "/root/software/"
	mntUrl := "/root/software/mnt/"
	NewWorkSpace(rootUrl, mntUrl, volume)
	cmd.Dir = mntUrl
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

// 为容器创建工作目录
func NewWorkSpace(rootUrl string, mntUrl string, volume string) error {
	createReadOnlyLayer(rootUrl)
	createWriteLayer(rootUrl)
	createMountPoint(rootUrl, mntUrl)

	// 挂载volume
	if volume != "" {
		volumeUrls := volumeUrlExtract(volume)
		if len(volumeUrls) != 2 || volumeUrls[0] == "" || volumeUrls[1] == "" {
			logger.Sugar().Errorf("Wrong volume parameters %s", volume)
		} else {
			volumeMount(mntUrl, volumeUrls)
			logger.Sugar().Info(volume)
		}
	}

	return nil
}

// 创建只读层
func createReadOnlyLayer(rootUrl string) error {
	busyBoxUrl := path.Join(rootUrl, "busybox")
	busyBoxTar := path.Join(rootUrl, "busybox.tar")
	if err := os.MkdirAll(busyBoxUrl, 0777); err != nil {
		logger.Sugar().Errorf("error create read only layer. %v", err)
		return err
	}

	if _, err := exec.Command("tar", "-xvf", busyBoxTar, "-C", busyBoxUrl).CombinedOutput(); err != nil {
		logger.Sugar().Errorf("error tar busybox. %v", err)
		return err
	}
	logger.Sugar().Infof("create readonly dir %s", busyBoxUrl)

	return nil
}

// 创建读写层
func createWriteLayer(rootUrl string) error {
	writeUrl := path.Join(rootUrl, "writeLayer")
	if exist, _ := pathExists(writeUrl); exist {
		return nil
	}
	if err := os.Mkdir(writeUrl, 0777); err != nil {
		logger.Sugar().Errorf("error mkdir %s. %v", writeUrl, err)
		return err
	}
	return nil
}

// 将读写层挂载为aufs
func createMountPoint(rootUrl, mntUrl string) error {
	if exist, _ := pathExists(mntUrl); !exist {
		if err := os.MkdirAll(mntUrl, 0777); err != nil {
			logger.Sugar().Errorf("error mkdir %s. %v", mntUrl, err)
			return err
		}
	}
	// 将只读层和可写层挂载到mntUrl
	dirs := "dirs=" + rootUrl + "writeLayer:" + rootUrl + "busybox"
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

// 提取volume参数
func volumeUrlExtract(volume string) []string {
	return strings.Split(volume, ":")
}

// 挂载volume
func volumeMount(mntUrl string, volumes []string) {
	hostUrl := volumes[0]
	if err := os.MkdirAll(hostUrl, 0777); err != nil {
		logger.Sugar().Errorf("error mkdir %s. %v", hostUrl, err)
		return
	}

	guestUrl := path.Join(mntUrl, volumes[1])
	if exist, _ := pathExists(guestUrl); !exist {
		if err := os.MkdirAll(guestUrl, 0777); err != nil {
			logger.Sugar().Errorf("error mkdir %s. %v", guestUrl, err)
			return
		}
	}

	dirs := "dirs=" + hostUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", guestUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("error mount aufs. %v", err)
	}
}

// 删除容器的工作目录
func deleteWorkSpace(rootUrl string, mntUrl string, volume string) {
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
	cmd := exec.Command("umount", volUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("error umount %s. %v", volUrl, err)
		return err
	}
	// 卸载整个容器的挂载点
	cmd = exec.Command("umount", mntUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("error umount %s. %v", mntUrl, err)
		return err
	}
	// 删除整个容器的挂载点
	if err := os.RemoveAll(mntUrl); err != nil {
		logger.Sugar().Errorf("error remove %s. %v", mntUrl, err)
		return err
	}
	return nil
}
