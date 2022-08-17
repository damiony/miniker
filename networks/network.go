package networks

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path"

	"github.com/vishvananda/netlink"
	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	logger, _ = zap.NewProduction()
}

type Network struct {
	// 结构体实例的名称
	Name string
	// ip地址段
	IpRange *net.IPNet
	// 网络驱动名
	Driver string
}

// 将网络的配置信息存储到文件
func (nw *Network) dump(dumpPath string) error {
	if _, err := os.Stat(dumpPath); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(dumpPath, 0644); err != nil {
			logger.Sugar().Errorf("create dir %s err %v", dumpPath, err)
			return err
		}
	}

	nwPath := path.Join(dumpPath, nw.Name)
	nwFile, err := os.OpenFile(nwPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Sugar().Errorf("open file %s error %v", nwPath, err)
		return err
	}
	defer nwFile.Close()

	infos, err := json.Marshal(nw)
	if err != nil {
		logger.Sugar().Errorf("marshal json err %v", err)
		return err
	}

	_, err = nwFile.Write(infos)
	if err != nil {
		logger.Sugar().Errorf("write network err %v", err)
		return err
	}
	return nil
}

// 从文件中加载网络的配置信息
func (nw *Network) load(loadPath string) error {
	configFile, err := os.Open(loadPath)
	if err != nil {
		logger.Sugar().Errorf("open file %s err %v", loadPath, err)
		return err
	}
	defer configFile.Close()

	content, err := io.ReadAll(configFile)
	if err != nil {
		logger.Sugar().Errorf("read file %s err %v", configFile, err)
		return err
	}

	err = json.Unmarshal(content, nw)
	if err != nil {
		logger.Sugar().Errorf("unmarshal err %v", err)
		return err
	}

	return nil
}

// 删除网络的配置文件
func (nw *Network) remove(remPath string) error {
	filePath := path.Join(remPath, nw.Name)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return os.Remove(filePath)
}

// 网络端点
type EndPoint struct {
	Id          string           `json:"id"`
	Device      netlink.Veth     `json:"dev"`
	IPAddress   net.IP           `json:"ip"`
	MacAddress  net.HardwareAddr `json:"mac"`
	PortMapping []string         `json:"portmapping"`
	NetWork     *Network
}

// 创建网络
func CreateNetwork(driver, subnet, name string) error {
	_, cidr, err := net.ParseCIDR(subnet)
	if err != nil {
		logger.Sugar().Errorf("parse cidr %s err %v", subnet, err)
		return err
	}
	// 给网段分配网关ip
	gatewayIp, err := ipAllocator.Allocate(cidr)
	if err != nil {
		return err
	}
	cidr.IP = gatewayIp
	logger.Info(cidr.String())

	// 使用指定的驱动创建网络
	nw, err := drivers[driver].Create(cidr.String(), name)
	if err != nil {
		return err
	}

	// 保存创建的网络信息
	return nw.dump(DefaultNetworkPath)
}

// 运行容器时连接到指定网络
func Connect(networkName string, name string, portMapping []string, pid int) error {
	// 获取指定网络的信息
	network, ok := networks[networkName]
	if !ok {
		return fmt.Errorf("no such network %s", networkName)
	}

	// 使用IPAM获取网段内的某个ip地址
	ip, err := ipAllocator.Allocate(network.IpRange)
	if err != nil {
		return err
	}

	// 创建网络端点
	endpoint := &EndPoint{
		Id:          fmt.Sprintf("%s-%s", name, networkName),
		IPAddress:   ip,
		NetWork:     network,
		PortMapping: portMapping,
	}

	// 使用驱动连接网络端点和网络
	if err := drivers[network.Driver].Connect(network, endpoint); err != nil {
		return err
	}

	// 进入容器的network namespace，配置ip地址和路由信息
	if err := configEndpointIpAndRoute(endpoint, pid); err != nil {
		return err
	}

	// 配置容器的端口映射
	return configPortMapping(endpoint)
}

// 删除网络
func DeleteNetwork(networkName string) error {
	logger.Sugar().Info(networkName)
	logger.Sugar().Info(networks)
	nw := networks[networkName]
	if nw == nil {
		return fmt.Errorf("no such network: %s", networkName)
	}

	// 释放网络的网关ip
	if err := ipAllocator.Release(nw.IpRange, nw.IpRange.IP); err != nil {
		return err
	}

	// 使用驱动删除网络
	if err := drivers[nw.Driver].Delete(nw); err != nil {
		return err
	}

	// 删除网络的配置信息
	return nw.remove(DefaultNetworkPath)
}
