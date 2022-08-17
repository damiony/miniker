package containers

import (
	"fmt"
	"os"
	"os/exec"
	"path"
)

// 提取镜像，存储格式为.tar
func commitImage(containerName, imageName string) {
	cur, err := os.Getwd()
	if err != nil {
		logger.Sugar().Errorf("get pwd err %v", err)
		return
	}

	fileName := path.Join(cur, "resources", imageName+".tar")
	mntUrl := fmt.Sprintf(MntUrl, os.Getenv("HOME"), containerName)
	logger.Sugar().Infof("tar %s to %s", mntUrl, fileName)
	if _, err := exec.Command("tar", "-cf", fileName, "-C", mntUrl, ".").CombinedOutput(); err != nil {
		logger.Sugar().Errorf("error tar image %s, %v", imageName, err)
	}
}
