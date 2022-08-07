package subsystems

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

type CpuSetSubsystem struct{}

func (c *CpuSetSubsystem) Name() string {
	return "cpuset"
}

func (c *CpuSetSubsystem) Set(cgroup string, cfg *SubsystemConfig) error {
	subsystemCgroupRoot, err := getCgroupPath(c.Name(), cgroup, true)
	if err != nil {
		return err
	}

	if cfg.CpuSet != "" {
		// cpuset.cpus可以指定容器使用的cpu内核
		limitFileName := path.Join(subsystemCgroupRoot, "cpuset.cpus")
		if err := os.WriteFile(limitFileName, []byte(cfg.MemLimit), 0644); err != nil {
			return fmt.Errorf("error set cgroup %v", err)
		}
	}
	return nil
}

func (c *CpuSetSubsystem) Apply(cgroup string, pid int) error {
	subsystemCgroupRoot, err := getCgroupPath(c.Name(), cgroup, false)
	if err != nil {
		return err
	}

	limitFileName := path.Join(subsystemCgroupRoot, "tasks")
	if err := os.WriteFile(limitFileName, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("error apply cgroup %v", err)
	}
	return nil
}

func (c *CpuSetSubsystem) Remove(cgroup string) error {
	subsystemCgroupRoot, err := getCgroupPath(c.Name(), cgroup, false)
	if err != nil {
		return err
	}

	if err := os.Remove(subsystemCgroupRoot); err != nil {
		return fmt.Errorf("error remove directory %v", err)
	}
	return nil
}
