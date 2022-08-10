package containers

import (
	"os/exec"
	"path"
)

func commitImage(name string) {
	fileName := path.Join(rootUrl, name+".tar")
	logger.Sugar().Infof("tar %s to %s", mntUrl, fileName)
	if _, err := exec.Command("tar", "-cf", fileName, "-C", mntUrl, ".").CombinedOutput(); err != nil {
		logger.Sugar().Errorf("error tar image %s, %v", name, err)
	}
}
