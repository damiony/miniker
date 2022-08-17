package networks

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
)

type IPAM struct {
	SubnetAllocatorPath string
	Subnets             map[string]string
}

// 创建默认的ipam
var ipAllocator = &IPAM{
	SubnetAllocatorPath: DefaultIpamAllocatorPath,
}

// 从文件中加载ip地址的分配信息
func (ipam *IPAM) load() error {
	// 检查文件是否存在
	if _, err := os.Stat(ipam.SubnetAllocatorPath); err != nil {
		if os.IsNotExist(err) {
			logger.Sugar().Errorf("no such file: %s", ipam.SubnetAllocatorPath)
			return nil
		}
		return err
	}

	// 打开文件
	ipamFile, err := os.Open(ipam.SubnetAllocatorPath)
	if err != nil {
		return err
	}
	defer ipamFile.Close()

	// 读取文件内容
	var content []byte
	scanner := bufio.NewScanner(ipamFile)
	for scanner.Scan() {
		b := scanner.Bytes()
		content = append(content, b...)
	}
	err = scanner.Err()
	if err != nil && err != io.EOF {
		return err
	}

	// 解析文件内容
	err = json.Unmarshal(content, &ipam.Subnets)
	if err != nil {
		return err
	}
	return nil
}

// 将ip地址的分配信息存储到文件
func (ipam *IPAM) dump() error {
	allocateDir, _ := path.Split(ipam.SubnetAllocatorPath)
	if _, err := os.Stat(allocateDir); err != nil {
		// 如果目录不存在，则创建
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(allocateDir, 0622); err != nil {
			logger.Sugar().Error(err)
			return err
		}
	}

	allocateFile, err := os.OpenFile(ipam.SubnetAllocatorPath, os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	defer allocateFile.Close()

	b, err := json.Marshal(ipam.Subnets)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}

	_, err = allocateFile.Write(b)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	return nil
}

// 分配IP
func (ipam *IPAM) Allocate(subnet *net.IPNet) (net.IP, error) {
	if ipam.Subnets == nil {
		ipam.Subnets = map[string]string{}
		if err := ipam.load(); err != nil {
			ipam.Subnets = nil
			return nil, err
		}
	}

	// ones是网络号位数，bits是ip的总位数
	ones, bits := subnet.Mask.Size()
	if _, exist := ipam.Subnets[subnet.String()]; !exist {
		ipam.Subnets[subnet.String()] = strings.Repeat("0", 1<<(bits-ones))
	}

	// 分配ip
	var ip net.IP
	for i := range ipam.Subnets[subnet.String()] {
		// 寻找第一个非1位
		if ipam.Subnets[subnet.String()][i] == '0' {
			ipalloc := []byte(ipam.Subnets[subnet.String()])
			ipalloc[i] = '1'
			ipam.Subnets[subnet.String()] = string(ipalloc)

			cnt := i + 1
			// 复制ip值，防止修改原值
			ip = append([]byte(nil), subnet.IP.To4()...)
			var max byte = 255
			for j := 1; j <= 4; j++ {
				rem := int(max - []byte(ip)[4-j])
				if cnt > rem {
					cnt -= rem
					[]byte(ip)[4-j] = max
				} else {
					[]byte(ip)[4-j] += byte(cnt)
					cnt = 0
				}
			}
			break
		}
	}

	logger.Sugar().Info("allocate ip ", ip)
	// 存储ip的分配信息
	if err := ipam.dump(); err != nil {
		return nil, err
	}
	return ip, nil
}

// 释放IP
func (ipam *IPAM) Release(subnet *net.IPNet, ip net.IP) error {
	// 加载ip的分配信息
	if ipam.Subnets == nil {
		ipam.Subnets = map[string]string{}
		if err := ipam.load(); err != nil {
			ipam.Subnets = nil
			return err
		}
	}

	// 获取subnet的网络分段信息，用于查询ip分配
	_, cidr, _ := net.ParseCIDR(subnet.String())
	if ipam.Subnets[cidr.String()] == "" {
		return fmt.Errorf("cannot get %s info", cidr.String())
	}

	// 计算待释放ip相当于起始ip的偏移量
	startIp := subnet.IP
	i := 0
	for j := 1; j <= 4; j++ {
		// ip增量
		diff := int([]byte(ip)[4-j] - []byte(startIp)[4-j])
		i += diff
	}

	// 修改ip的分配信息
	ipalloc := []byte(ipam.Subnets[cidr.String()])
	ipalloc[i] = '0'
	ipam.Subnets[cidr.String()] = string(ipalloc)

	// 存储数据到文件
	if err := ipam.dump(); err != nil {
		return err
	}
	return nil
}
