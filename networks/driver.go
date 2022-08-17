package networks

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netns"
)

// 网络驱动接口
type NetworkDriver interface {
	// 驱动名
	Name() string
	// 创建网络
	Create(subnet string, name string) (*Network, error)
	// 删除网络
	Delete(network *Network) error
	// 连接网络端点和网络
	Connect(network *Network, endpoint *EndPoint) error
	// 断开连接
	Disconnect(network *Network, endpoint *EndPoint) error
}

// 网络驱动的具体实现
type BridgeNetworkDriver struct{}

func (b *BridgeNetworkDriver) Name() string {
	return "bridge"
}

// 创建网络
func (b *BridgeNetworkDriver) Create(subnet string, name string) (*Network, error) {
	ip, ipRange, _ := net.ParseCIDR(subnet)
	ipRange.IP = ip

	// 初始化网络对象
	nw := &Network{
		Name:    name,
		IpRange: ipRange,
		Driver:  b.Name(),
	}

	// 初始化bridge
	err := b.initBridge(nw)
	if err != nil {
		logger.Sugar().Errorf("Error init bridge: %v", err)
	}

	return nw, err
}

// 初始化bridge
func (b *BridgeNetworkDriver) initBridge(nw *Network) error {
	//  将网络名字用作bridge的名字
	bridgeName := nw.Name
	// 创建bridge
	if err := createBridgeInterface(bridgeName); err != nil {
		return err
	}
	// 设置bridge的地址和路由
	gatewayIp := *nw.IpRange
	if err := setInterfaceIP(bridgeName, gatewayIp.String()); err != nil {
		return err
	}
	logger.Sugar().Info("set interface ip")
	// 启动bridge
	if err := setInterfaceUP(bridgeName); err != nil {
		return err
	}
	logger.Sugar().Info("set interface up")
	// 设置iptables的SNAT规则
	if err := setupIPTables(bridgeName, nw.IpRange); err != nil {
		return err
	}
	logger.Sugar().Info("set interface iptables")
	return nil
}

// 创建bridge
func createBridgeInterface(bridgeName string) error {
	// 检查bridge是否已经存在
	_, err := net.InterfaceByName(bridgeName)
	if err == nil || !strings.Contains(err.Error(), "no such network interface") {
		return err
	}

	// 初始化一个Link对象
	la := netlink.NewLinkAttrs()
	la.Name = bridgeName

	// 使用Link创建Bridge对象
	// LinkAdd相当于 ip link add ...
	br := &netlink.Bridge{
		LinkAttrs: la,
	}
	if err := netlink.LinkAdd(br); err != nil {
		return err
	}

	return nil
}

// 为网络接口设置ip地址
func setInterfaceIP(name string, rawIP string) error {
	// 查找指定的网络接口
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	// 解析rawIP
	ipNet, err := netlink.ParseIPNet(rawIP)
	if err != nil {
		return err
	}
	// 给网络接口配置地址，并且设置路由
	// AddrAdd 相当于 ip addr add ...
	addr := &netlink.Addr{
		IPNet: ipNet,
	}
	return netlink.AddrAdd(iface, addr)
}

// 将网络接口设置为UP状态
func setInterfaceUP(interfaceName string) error {
	// 获取网络接口
	iface, err := netlink.LinkByName(interfaceName)
	if err != nil {
		return err
	}

	// 将网络接口设置为 UP 状态
	// LinkSetUp 相当于 ip link set ...
	if err := netlink.LinkSetUp(iface); err != nil {
		return err
	}

	return nil
}

func setupIPTables(bridgeName string, subnet *net.IPNet) error {
	// 设置iptables的MASQUERADE规则
	// iptables -t nat -A POSTROUTING -s <bridgeName> ! -o <bridgeName> -j MASQUERADE
	iptablesCmd := fmt.Sprintf("-t nat -A POSTROUTING -s %s ! -o %s -j MASQUERADE", subnet.String(), bridgeName)
	cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
	output, err := cmd.Output()
	if err != nil {
		logger.Sugar().Errorf("iptables Output, %v", output)
	}
	return err
}

// 删除网络
func (b *BridgeNetworkDriver) Delete(network *Network) error {
	br, err := netlink.LinkByName(network.Name)
	if err != nil {
		return err
	}
	return netlink.LinkDel(br)
}

// 连接网络端点和网络
func (b *BridgeNetworkDriver) Connect(network *Network, endpoint *EndPoint) error {
	bridgeName := network.Name
	br, err := netlink.LinkByName(bridgeName)
	if err != nil {
		return err
	}

	// 对veth进行配置
	la := netlink.NewLinkAttrs()
	// veth的名字
	la.Name = endpoint.Id[:5]
	// 将veth的一端连接到bridge
	la.MasterIndex = br.Attrs().Index
	// 创建veth
	endpoint.Device = netlink.Veth{
		LinkAttrs: la,
		PeerName:  "cif-" + endpoint.Id[:5],
	}
	if err := netlink.LinkAdd(&endpoint.Device); err != nil {
		return fmt.Errorf("error add endpoint device: %v", err)
	}

	// 将veth设置为UP
	if err := netlink.LinkSetUp(&endpoint.Device); err != nil {
		return fmt.Errorf("error set endpoint device up: %v", err)
	}
	return nil
}

// 配置容器中网络端点的地址和路由
func configEndpointIpAndRoute(endpoint *EndPoint, pid int) error {
	// 获取veth网络接口的另一端
	peerLink, err := netlink.LinkByName(endpoint.Device.PeerName)
	if err != nil {
		return fmt.Errorf("error get veth iface: %v", err)
	}

	// 将veth的另一端加入到容器的网络空间中
	// 当前函数执行完，需要从容器的网络空间中回到之前的网络空间
	defer enterContainerNetns(&peerLink, pid)()

	// 获取容器网络的ip地址和网段
	interfaceIP := *endpoint.NetWork.IpRange
	interfaceIP.IP = endpoint.IPAddress
	// 配置veth的网络
	if err := setInterfaceIP(endpoint.Device.PeerName, interfaceIP.String()); err != nil {
		return err
	}
	// 启动veth端点
	if err := setInterfaceUP(endpoint.Device.PeerName); err != nil {
		return err
	}
	// 开启"lo"网络接口
	if err := setInterfaceUP("lo"); err != nil {
		return err
	}
	// 设置容器的路由
	// 0.0.0.0/0 表示所有的ip地址
	_, cidr, _ := net.ParseCIDR("0.0.0.0/0")
	defaultRoute := &netlink.Route{
		LinkIndex: peerLink.Attrs().Index,
		Gw:        endpoint.NetWork.IpRange.IP,
		Dst:       cidr,
	}
	if err := netlink.RouteAdd(defaultRoute); err != nil {
		logger.Sugar().Errorf("error set route, %v", err)
		return err
	}

	return nil
}

func enterContainerNetns(nwLink *netlink.Link, pid int) func() {
	// 访问 /proc/[pid]/ns/net 文件
	f, err := os.Open("/proc/" + fmt.Sprintf("%d", pid) + "/ns/net")
	if err != nil {
		logger.Sugar().Errorf("error get container net namespace, %v", err)
	}

	// 获取文件描述符
	nsFD := f.Fd()

	// 锁定程序现场，否则无法保证一直处于正确的网络空间
	runtime.LockOSThread()

	// 将网络接口挂在到容器的net namespace
	if err := netlink.LinkSetNsFd(*nwLink, int(nsFD)); err != nil {
		logger.Sugar().Errorf("error set link netns: %v", err)
	}

	// 获取当前网络的namespace，便于后续退回
	oringinNs, err := netns.Get()
	if err != nil {
		logger.Sugar().Errorf("error get current netns, %v", err)
	}

	// 将当前进程加入容器的net namespace
	if err := netns.Set(netns.NsHandle(nsFD)); err != nil {
		logger.Sugar().Errorf("error set netns, %v", err)
	}

	return func() {
		// 退回之前的net namespace
		netns.Set(oringinNs)
		// 关闭namespace文件
		oringinNs.Close()
		// 取消对线程的锁定
		runtime.UnlockOSThread()
		// 关闭namespace文件
		f.Close()
	}
}

// 端口映射
func configPortMapping(endpoint *EndPoint) error {
	for _, pm := range endpoint.PortMapping {
		ports := strings.Split(pm, ":")
		if len(ports) != 2 {
			logger.Sugar().Errorf("port mapping format error, %v", pm)
			continue
		}

		iptablesCmd := fmt.Sprintf("-t nat -A PREROUTING -p tcp -m tcp --dport %s -j DNAT --to-destination %s:%s", ports[0], endpoint.IPAddress.String(), ports[1])
		cmd := exec.Command("iptables", strings.Split(iptablesCmd, " ")...)
		output, err := cmd.Output()
		if err != nil {
			logger.Sugar().Errorf("iptables err %v, output %v", err, output)
			continue
		}
	}
	return nil
}

// 断开连接
func (b *BridgeNetworkDriver) Disconnect(network *Network, endpoint *EndPoint) error {
	return nil
}
