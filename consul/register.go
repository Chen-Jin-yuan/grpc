package consul

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"github.com/rs/zerolog/log"
	"net"
	"os"
	"strconv"
)

// Register a service with registry
func (c *Client) Register(name string, id string, ip string, port int) error {
	// 如果 ip 为空，则获取本地主机地址
	if ip == "" {
		var err error
		ip, err = getLocalIP()
		if err != nil {
			return err
		}
	}
	// tag 可以为空
	reg := &consul.AgentServiceRegistration{
		ID:      id,
		Name:    name,
		Port:    port,
		Address: ip,
	}
	log.Info().Msgf("Trying to register service [ name: %s, id: %s, address: %s:%d ]", name, id, ip, port)
	return c.Agent().ServiceRegister(reg)
}

// RegisterWithCheck Register and Check a service with registry
func (c *Client) RegisterWithCheck(name string, id string, ip string, port int,
	timeout int, interval int, deregisterAfter int) error {
	// 如果 ip 为空，则获取本地主机地址
	if ip == "" {
		var err error
		ip, err = getLocalIP()
		if err != nil {
			return err
		}
	}

	// 增加consul健康检查回调函数
	// TODO：传递默认值，需要更改为更优雅的函数选项模式
	if timeout == 0 {
		timeout = 5
	}
	if interval == 0 {
		interval = 5
	}
	if deregisterAfter == 0 {
		deregisterAfter = 30
	}
	check := new(consul.AgentServiceCheck)
	check.HTTP = fmt.Sprintf("http://%s:%d/actuator/health", ip, port)         // HTTP 检查的地址
	check.Timeout = strconv.Itoa(timeout) + "s"                                // 健康检查超时时间
	check.Interval = strconv.Itoa(interval) + "s"                              // 健康检查间隔
	check.DeregisterCriticalServiceAfter = strconv.Itoa(deregisterAfter) + "s" // 故障检查失败30s后 consul自动将注册服务删除

	// tag 可以为空
	reg := &consul.AgentServiceRegistration{
		ID:      id,
		Name:    name,
		Port:    port,
		Address: ip,
		Check:   check,
	}
	log.Info().Msgf("Trying to register service with check [ name: %s, id: %s, address: %s:%d ]",
		name, id, ip, port)
	return c.Agent().ServiceRegister(reg)
}

// Deregister removes the service address from registry
func (c *Client) Deregister(id string) error {
	return c.Agent().ServiceDeregister(id)
}

// Look for the network device being dedicated for gRPC traffic.
// The network CDIR should be specified in os environment
// "DSB_HOTELRESERV_GRPC_NETWORK".
// If not found, return the first non loopback IP address.
// 获取本地主机的 ip 地址，如果有多个网卡，返回满足要求的 ip
func getLocalIP() (string, error) {
	var ipGrpc string
	var ips []net.IP

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				ips = append(ips, ipnet.IP)
			}
		}
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("registry: can not find local ip")
	} else if len(ips) > 1 {
		// by default, return the first network IP address found.
		ipGrpc = ips[0].String()

		grpcNet := os.Getenv("DSB_GRPC_NETWORK")
		_, ipNetGrpc, err := net.ParseCIDR(grpcNet)
		if err != nil {
			log.Error().Msgf("An invalid network CIDR is set in environment DSB_HOTELRESERV_GRPC_NETWORK: %v", grpcNet)
		} else {
			for _, ip := range ips {
				if ipNetGrpc.Contains(ip) {
					ipGrpc = ip.String()
					log.Info().Msgf("gRPC traffic is routed to the dedicated network %s", ipGrpc)
					break
				}
			}
		}
	} else {
		// only one network device existed
		ipGrpc = ips[0].String()
	}

	return ipGrpc, nil
}
