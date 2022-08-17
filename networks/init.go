package networks

import (
	"io/fs"
	"os"
	"path/filepath"
)

var networks = map[string]*Network{}
var drivers = map[string]NetworkDriver{}

func init() {
	if err := Init(); err != nil {
		logger.Sugar().Error(err)
		os.Exit(1)
	}
}

// 初始化网络相关的信息
func Init() error {
	// 创建bridge驱动
	bridgeDriver := &BridgeNetworkDriver{}
	drivers[bridgeDriver.Name()] = bridgeDriver

	// 检查网络的存储目录是否存在，如果不存在就创建
	if _, err := os.Stat(DefaultNetworkPath); err != nil {
		if os.IsNotExist(err) {
			os.MkdirAll(DefaultNetworkPath, 0644)
		} else {
			return err
		}
	}

	// 扫描目录，并对每个结果执行一次函数
	filepath.WalkDir(DefaultNetworkPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		nw := &Network{
			// 使用文件名做为网络名
			Name: d.Name(),
		}
		if err := nw.load(path); err != nil {
			logger.Sugar().Errorf("Error load network: %v", err)
			return nil
		}
		networks[nw.Name] = nw
		return nil
	})

	return nil
}
