package consul

import (
	"github.com/hashicorp/consul/api"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/resolver"
	"net"
	"strconv"
	"sync"
)

var (
	client *Client
)

// InitResolver 初始化注册 resolver
func InitResolver(c *Client) {
	log.Info().Msg("consul init\n")
	client = c
	resolver.Register(NewBuilder())
}

type consulBuilder struct {
}

type consulResolver struct {
	wg                   sync.WaitGroup
	cc                   resolver.ClientConn
	svcName              string
	svcTag               string
	disableServiceConfig bool
	lastIndex            uint64
	quitC                chan struct{}
}

func NewBuilder() resolver.Builder {
	return &consulBuilder{}
}

// Build 构建 resolver
// 约定: 给 Dial 接口传递的 target 形式为 consul://consul/svcName，在 DialContext 中被解析，consul 为 Scheme，svcName 为 Endpoint
// 注：给定的 target 需要满足形式为：scheme://authority/endpoint，最终传到该函数的是 target 是 endpoint
func (cb *consulBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {

	log.Info().Msg("calling consul build\n")
	log.Info().Msgf("target: %v\n", target)
	svcName := target.Endpoint

	cr := &consulResolver{
		svcName:              svcName,
		cc:                   cc,
		disableServiceConfig: opts.DisableServiceConfig,
		lastIndex:            0,
		quitC:                make(chan struct{}),
	}

	cr.wg.Add(1)
	go cr.watcher()
	return cr, nil

}

func (cr *consulResolver) watcher() {
	log.Info().Msg("calling consul watcher\n")
	for {
		select {
		// 如果接收到退出信号（r.quitC），则跳出循环，终止更新过程。
		case <-cr.quitC:
			break
		default:
			newAddrs, lastIndex, err := cr.getInstances(cr.lastIndex)
			if err != nil {
				log.Error().Msgf("error retrieving instances from Consul: %v\n", err)
			}

			// 更新索引
			cr.lastIndex = lastIndex
			log.Info().Msgf("newAddrs: %v\n", newAddrs)
			cr.cc.NewAddress(newAddrs)
			cr.cc.NewServiceConfig(cr.svcName)
		}
	}

}

// getInstances 从 Consul 检索服务的新实例集合。
// 参数 lastIndex 用于指定等待索引，以便在检索实例时获得最新更改。
func (cr *consulResolver) getInstances(lastIndex uint64) ([]resolver.Address, uint64, error) {
	// 使用 Consul 客户端的 Health() 函数检索服务的健康信息。
	// 传递的参数包括服务名称、标签、是否仅匹配健康实例和查询选项。
	services, meta, err := client.Health().Service(cr.svcName, cr.svcTag, true,
		&api.QueryOptions{WaitIndex: cr.lastIndex})

	if err != nil {
		// 如果在检索期间发生错误，返回一个 nil 实例切片、传入的 lastIndex 和错误信息。
		return nil, lastIndex, err
	}

	// 存储实例地址。
	var newAddrs []resolver.Address

	// 遍历健康服务信息，提取实例的 IP 地址和端口，并将它们连接成完整的地址。
	for _, service := range services {
		s := service.Service.Address
		// 如果服务地址为空，使用节点地址作为替代。
		if len(s) == 0 {
			s = service.Node.Address
		}
		// 使用 net.JoinHostPort() 函数将 IP 地址和端口号连接成完整的实例地址。
		addr := net.JoinHostPort(s, strconv.Itoa(service.Service.Port))
		// 将地址添加到实例切片中。
		newAddrs = append(newAddrs, resolver.Address{Addr: addr})
	}

	// 返回包含实例地址、Consul 返回的最新等待索引和无错误的结果。
	return newAddrs, meta.LastIndex, nil
}

func (cb *consulBuilder) Scheme() string {
	return "consul"
}

// ResolveNow 上层（如果认为需要更新）可以通过该方法主动刷新服务信息
// 如果 watcher 使用了某种阻塞方式，应该在此实现 watcher 更新
func (cr *consulResolver) ResolveNow(opt resolver.ResolveNowOptions) {
}

// Close 关闭观察者。
func (cr *consulResolver) Close() {
	close(cr.quitC)
	cr.wg.Wait()
}
