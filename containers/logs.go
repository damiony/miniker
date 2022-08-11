package containers

import (
	"errors"
	"fmt"
	"io"
	"os"
)

// 创建日志文件，路径为/var/run/miniker/{containerName}/container.log
func createLogFile(containerName string) (*os.File, error) {
	if containerName == "" {
		return nil, errors.New("containerName cannot be empty")
	}
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		return nil, err
	}

	fileName := dirUrl + LogName
	file, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	return file, err
}

// 将容器的日志打印到控制台
func printLogs(containerName string) {
	if containerName == "" {
		logger.Sugar().Error("containerName cannot be empty")
		return
	}
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	fileName := dirUrl + LogName

	file, err := os.Open(fileName)
	if err != nil {
		logger.Sugar().Errorf("Cannot open file %s err %v", fileName, err)
		return
	}
	defer file.Close()

	res, err := io.ReadAll(file)
	if err != nil {
		logger.Sugar().Errorf("Read file %s err %v", fileName, err)
		return
	}

	_, err = fmt.Fprint(os.Stdout, string(res))
	if err != nil {
		logger.Sugar().Errorf("print file %s err %v", fileName, err)
	}
}
