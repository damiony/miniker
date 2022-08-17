package containers

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

func mountVolume(containerName, volume string) error {
	logger.Sugar().Info("Volume is", volume)
	// volume可以为空
	if volume == "" {
		return nil
	}
	// 挂载volume
	volumeUrls := volumeUrlExtract(volume)
	if len(volumeUrls) != 2 || volumeUrls[0] == "" || volumeUrls[1] == "" {
		logger.Sugar().Errorf("Wrong volume parameters %s", volume)
		return errors.New("wrong volume parameters")
	}

	hostUrl := volumeUrls[0]
	if err := os.MkdirAll(hostUrl, 0777); err != nil {
		logger.Sugar().Errorf("error mkdir %s. %v", hostUrl, err)
		return err
	}

	mntUrl := fmt.Sprintf(MntUrl, os.Getenv("HOME"), containerName)
	guestUrl := path.Join(mntUrl, volumeUrls[1])
	if err := os.MkdirAll(guestUrl, 0777); err != nil {
		logger.Sugar().Errorf("error mkdir %s. %v", guestUrl, err)
		return err
	}

	dirs := "dirs=" + hostUrl
	cmd := exec.Command("mount", "-t", "aufs", "-o", dirs, "none", guestUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("error mount aufs. %v", err)
	}

	return nil
}

// 提取volume参数
func volumeUrlExtract(volume string) []string {
	return strings.Split(volume, ":")
}

func umountVolume(volUrl string) error {
	// 卸载volume
	cmd := exec.Command("umount", volUrl)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logger.Sugar().Errorf("error umount %s. %v", volUrl, err)
		return err
	}
	return nil
}
