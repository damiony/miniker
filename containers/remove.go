package containers

import (
	"fmt"
	"os"
)

func removeContainer(containerName string) {
	// 获取容器信息
	containerInfo := getContainerInfo(containerName)
	if containerInfo == nil {
		logger.Sugar().Errorf("Cannot get container by name %s", containerName)
		return
	}
	// 检查容器是否停止
	if containerInfo.Status != EXIT {
		logger.Sugar().Errorf("Container status is not exit")
		return
	}
	// 删除mntUrl
	if err := os.RemoveAll(mntUrl); err != nil {
		logger.Sugar().Errorf("remove %s err %v", mntUrl, err)
		return
	}
	// 删除容器信息
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirUrl); err != nil {
		logger.Sugar().Errorf("remove %s err %v", dirUrl, err)
	}
}
