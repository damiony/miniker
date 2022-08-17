package subsystems

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"

	"go.uber.org/zap"
)

type SubsystemConfig struct {
	MemLimit string // 内存限制
	CpuSet   string // CPU时间片权重
	CpuShare string // CPU核心数限制
}

type Subsystem interface {
	// 返回subsystem的名称，如memory、cpu等等
	Name() string
	// 设置指定cgroup的subsystem限制
	// cgroup在文件系统中是以相对路径的形式表示的
	Set(cgroup string, cfg *SubsystemConfig) error
	// 将进程id加入cgroup
	Apply(cgroup string, pid int) error
	// 删除指定cgroup
	Remove(cgroup string) error
}

var logger *zap.Logger

func init() {
	logger, _ = zap.NewProduction()
}

var SubsystemsIns = []Subsystem{
	&MemorySubsystem{},
	&CpuSetSubsystem{},
	&CpuSubsystem{},
}

type CgroupManager struct {
	Path   string
	Config *SubsystemConfig
}

func NewCgroupManager(path string, cfg *SubsystemConfig) *CgroupManager {
	return &CgroupManager{
		Path:   path,
		Config: cfg,
	}
}

func (s *CgroupManager) Set() error {
	logger.Sugar().Info("Set cgroup")
	for _, subSysIns := range SubsystemsIns {
		subSysIns.Set(s.Path, s.Config)
	}
	return nil
}

func (s *CgroupManager) Apply(pid int) error {
	logger.Sugar().Info("Apply pid")
	for _, subSysIns := range SubsystemsIns {
		subSysIns.Apply(s.Path, pid)
	}
	return nil
}

func (s *CgroupManager) Destroy() error {
	logger.Sugar().Info("destroy cgroup")
	for _, subSysIns := range SubsystemsIns {
		subSysIns.Remove(s.Path)
	}
	return nil
}

// 获取`cgroup`的绝对路径
func getCgroupPath(subsystem string, cgroup string, autoCreate bool) (string, error) {
	root := findCgroupMountPoint(subsystem)
	if _, err := os.Stat(path.Join(root, cgroup)); err == nil || (autoCreate && os.IsNotExist(err)) {
		if os.IsNotExist(err) {
			if err := os.Mkdir(path.Join(root, cgroup), 0755); err != nil {
				return "", fmt.Errorf("error create cgroup %v", err)
			}
		}
		return path.Join(root, cgroup), nil
	} else {
		return "", fmt.Errorf("error get cgroup path %v", err)
	}
}

// 获取`subsystem`挂载点的根目录
func findCgroupMountPoint(subsystem string) string {
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		logger.Sugar().Error(err)
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		msg := scanner.Text()
		fields := strings.Split(msg, " ")
		for _, t := range strings.Split(fields[len(fields)-1], ",") {
			if t == subsystem {
				return fields[4]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Sugar().Error(err)
		return ""
	}

	return ""
}
