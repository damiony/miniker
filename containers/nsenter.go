package containers

/*
#define _GNU_SOURCE
#include <unistd.h>
#include <errno.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <fcntl.h>
__attribute__((constructor)) void enter_namespace(void) {
	char *mydocker_pid;
	mydocker_pid = getenv("miniker_pid");
	if (mydocker_pid) {
		fprintf(stdout, "got mydocker_pid=%s\n", mydocker_pid);
	} else {
		fprintf(stdout, "missing mydocker_pid env skip nsenter");
		return;
	}
	char *mydocker_cmd;
	mydocker_cmd = getenv("miniker_cmd");
	if (mydocker_cmd) {
		//fprintf(stdout, "got mydocker_cmd=%s\n", mydocker_cmd);
	} else {
		//fprintf(stdout, "missing mydocker_cmd env skip nsenter");
		return;
	}
	int i;
	char nspath[1024];
	char *namespaces[] = { "ipc", "uts", "net", "pid", "mnt" };
	for (i=0; i<5; i++) {
		sprintf(nspath, "/proc/%s/ns/%s", mydocker_pid, namespaces[i]);
		int fd = open(nspath, O_RDONLY);
		if (setns(fd, 0) == -1) {
			//fprintf(stderr, "setns on %s namespace failed: %s\n", namespaces[i], strerror(errno));
		} else {
			//fprintf(stdout, "setns on %s namespace succeeded\n", namespaces[i]);
		}
		close(fd);
	}
	int res = system(mydocker_cmd);
	exit(0);
	return;
}
*/
import "C"
import (
	"os"
	"os/exec"
	"strings"
)

func execCommands(containerName string, commands []string) {
	containerInfo := getContainerInfo(containerName)
	if containerInfo == nil {
		logger.Sugar().Errorf("Cannot get container info for name %s", containerName)
		return
	}

	cmdStr := strings.Join(commands, " ")
	logger.Sugar().Infof("container pid %s", containerInfo.Pid)
	logger.Sugar().Infof("command %s", cmdStr)

	cmd := exec.Command("/proc/self/exe", "exec")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	os.Setenv(ENV_EXEC_PID, containerInfo.Pid)
	os.Setenv(ENV_EXEC_CMD, cmdStr)

	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("Exec container %s error %v", containerName, err)
	}
}
