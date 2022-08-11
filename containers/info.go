package containers

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
)

// 容器信息
type ContainerInfo struct {
	Pid        string `json:"pid"`
	Id         string `json:"id"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	CreateTime string `json:"createTime"`
	Status     string `json:"status"`
}

func recordContainerInfo(pid int, containerName string, cmds []string) string {
	cInfo := &ContainerInfo{}
	cInfo.Pid = strconv.Itoa(pid)
	logger.Sugar().Infof("pid : %s", os.Getpid())
	cInfo.Id = generateId()
	cInfo.CreateTime = time.Now().Format("2006-01-02 15:04:05")
	if containerName == "" {
		containerName = cInfo.Id
	}
	cInfo.Name = containerName
	cInfo.Command = strings.Join(cmds, " ")
	cInfo.Status = RUNNING

	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		logger.Sugar().Errorf("mkdir %s err %v", dirUrl, err)
		return ""
	}

	b, err := json.Marshal(cInfo)
	if err != nil {
		logger.Sugar().Errorf("marshal json %v err %v", *cInfo, err)
		return ""
	}

	fileName := dirUrl + ConfigName
	if err := os.WriteFile(fileName, b, 0644); err != nil {
		logger.Sugar().Errorf("write %s to file %s, err %v", b, fileName, err)
		return ""
	}
	return containerName
}

func generateId() string {
	chars := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	rand.Seed(time.Now().UnixNano())
	id := make([]byte, 10)
	for i := range id {
		j := rand.Intn(len(chars))
		id[i] = chars[j]
	}
	return string(id)
}

func deleteContainerInfo(containerName string) {
	if containerName == "" {
		return
	}
	dirUrl := fmt.Sprintf(DefaultInfoLocation, containerName)
	if err := os.RemoveAll(dirUrl); err != nil {
		logger.Sugar().Errorf("remove dir %s err %v", dirUrl, err)
	}
}

func listContainers() {
	dirUrl := fmt.Sprintf(DefaultInfoLocation, "")
	dirUrl = dirUrl[:len(dirUrl)-1]
	dirs, err := os.ReadDir(dirUrl)
	if err != nil {
		logger.Sugar().Errorf("Read dir %s, err %v", dirUrl, err)
		return
	}

	var containerInfos []*ContainerInfo
	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		cInfo := getContainerInfo(dir.Name())
		if cInfo != nil {
			containerInfos = append(containerInfos, cInfo)
		}
	}
	printContainers(containerInfos)
}

// 根据容器名称获取容器信息
func getContainerInfo(containerName string) *ContainerInfo {
	containerFileDir := fmt.Sprintf(DefaultInfoLocation, containerName)
	containerFileName := containerFileDir + ConfigName
	file, err := os.Open(containerFileName)
	if err != nil {
		logger.Sugar().Errorf("Open file %s err %v", containerFileName, err)
		return nil
	}
	defer file.Close()

	res, err := io.ReadAll(file)
	if err != nil {
		logger.Sugar().Errorf("Read file %s err %v", containerFileName, err)
		return nil
	}

	var cInfo ContainerInfo
	err = json.Unmarshal(res, &cInfo)
	if err != nil {
		logger.Sugar().Errorf("Unmarshal json err %v", err)
		return nil
	}

	return &cInfo
}

func printContainers(containerInfos []*ContainerInfo) {
	maxLen := map[string]int{}
	for _, info := range containerInfos {
		if maxLen["pid"] < len(info.Pid) {
			maxLen["pid"] = len(info.Pid)
		}
		if maxLen["id"] < len(info.Id) {
			maxLen["id"] = len(info.Id)
		}
		if maxLen["name"] < len(info.Name) {
			maxLen["name"] = len(info.Name)
		}
		if maxLen["cmd"] < len(info.Command) {
			maxLen["cmd"] = len(info.Command)
		}
		if maxLen["ct"] < len(info.CreateTime) {
			maxLen["ct"] = len(info.CreateTime)
		}
		if maxLen["status"] < len(info.Status) {
			maxLen["status"] = len(info.Status)
		}
	}
	infoFormat := "%-" + strconv.Itoa(maxLen["pid"]) + "s\t" +
		"%-" + strconv.Itoa(maxLen["id"]) + "s\t" +
		"%-" + strconv.Itoa(maxLen["name"]) + "s\t" +
		"%-" + strconv.Itoa(maxLen["status"]) + "s\t" +
		"%-" + strconv.Itoa(maxLen["ct"]) + "s\t" +
		"%-" + strconv.Itoa(maxLen["cmd"]) + "s\n"
	fmt.Printf(infoFormat, "Pid", "Id", "Name", "Status", "CreateTime", "Cmd")
	for _, info := range containerInfos {
		fmt.Printf(infoFormat, info.Pid, info.Id, info.Name, info.Status, info.CreateTime, info.Command)
	}
}
