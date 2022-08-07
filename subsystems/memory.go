package subsystems

import (
	"fmt"
	"os"
	"path"
	"strconv"
)

type MemorySubsystem struct{}

func (mem *MemorySubsystem) Name() string {
	return "memory"
}

func (mem *MemorySubsystem) Set(cgroup string, cfg *SubsystemConfig) error {
	subsystemCgroupRoot, err := getCgroupPath(mem.Name(), cgroup, true)
	if err != nil {
		return err
	}

	if cfg.MemLimit != "" {
		limitFileName := path.Join(subsystemCgroupRoot, "memory.limit_in_bytes")
		if err := os.WriteFile(limitFileName, []byte(cfg.MemLimit), 0644); err != nil {
			return fmt.Errorf("error set cgroup %v", err)
		}
	}
	return nil
}

func (mem *MemorySubsystem) Apply(cgroup string, pid int) error {
	subsystemCgroupRoot, err := getCgroupPath(mem.Name(), cgroup, false)
	if err != nil {
		return err
	}

	limitFileName := path.Join(subsystemCgroupRoot, "tasks")
	if err := os.WriteFile(limitFileName, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("error apply cgroup %v", err)
	}
	return nil
}

func (mem *MemorySubsystem) Remove(cgroup string) error {
	subsystemCgroupRoot, err := getCgroupPath(mem.Name(), cgroup, false)
	if err != nil {
		return err
	}

	if err := os.Remove(subsystemCgroupRoot); err != nil {
		return fmt.Errorf("error remove directory %v", err)
	}
	return nil
}
