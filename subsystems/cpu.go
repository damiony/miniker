package subsystems

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

type CpuSubsystem struct{}

func (c *CpuSubsystem) Name() string {
	return "cpu"
}

func (c *CpuSubsystem) Set(cgroup string, cfg *SubsystemConfig) error {
	subsystemCgroupRoot, err := getCgroupPath(c.Name(), cgroup, true)
	if err != nil {
		return err
	}

	if cfg.CpuShare == "" {
		return nil
	}

	// cpu.shares被用于设置能使用的cpu相对值，作用于所有内核，默认是1024。
	// 假设cgroup A被设置为1024，cgroup B被设置为512,
	// 则A能使用66.66%的cpu资源，B能使用33.33%的cpu资源。
	limitFileName := path.Join(subsystemCgroupRoot, "cpu.shares")
	if err := os.WriteFile(limitFileName, []byte(cfg.MemLimit), 0644); err != nil {
		return fmt.Errorf("error set cgroup %v", err)
	}
	return nil
}

func (c *CpuSubsystem) Apply(cgroup string, pid int) error {
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

func (c *CpuSubsystem) Remove(cgroup string) error {
	subsystemCgroupRoot, err := getCgroupPath(c.Name(), cgroup, false)
	if err != nil {
		return err
	}

	if err := os.Remove(subsystemCgroupRoot); err != nil {
		return fmt.Errorf("error remove directory %v", err)
	}
	return nil
}
