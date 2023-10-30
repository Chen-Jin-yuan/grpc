package allocator

import (
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
)

// Name is the name of allocator.
const Name = "allocator"

var firstStart = true

// NewBuilder creates a new weight balancer builder.
// HealthCheck 会使用服务端的健康检查来判断服务是否可用，如果服务端没有实现健康检查，则该配置不起作用
func newBuilder(configPath string, httpServerPort int) balancer.Builder {
	return base.NewBalancerBuilderV2(Name, &allocatorPickerBuilder{allocatorConfigPath: configPath,
		allocatorPort: httpServerPort}, base.Config{HealthCheck: true})
}

func Init(configPath string, httpServerPort int) {
	balancer.Register(newBuilder(configPath, httpServerPort))
}

type allocatorPickerBuilder struct {
	allocatorConfigPath string
	allocatorPort       int
}

func (pb *allocatorPickerBuilder) Build(info base.PickerBuildInfo) balancer.V2Picker {
	grpclog.Infof("allocatorPicker: newPicker called with info: %v", info)
	if len(info.ReadySCs) == 0 {
		return base.NewErrPickerV2(balancer.ErrNoSubConnAvailable)
	}

	// 连接信息
	var cis []connInfo
	var serviceName string
	// 注：依赖于服务发现 resolver 把 ServiceName 写入 ServerName
	index := 0
	for subConn, subConnInfo := range info.ReadySCs {
		serviceName = subConnInfo.Address.ServerName
		cis = append(cis, connInfo{
			sc:     subConn,
			group:  "",
			addr:   subConnInfo.Address.Addr,
			load:   0,
			weight: -1,
			index:  index,
		})
		index++
	}

	log.Info().Msgf("allocatorPicker connInfo list: %+v", cis)
	// 加载配置，一并把地址配置好
	svcConfig, err := loadConfig(pb.allocatorConfigPath, cis, serviceName)
	if err != nil {
		log.Error().Msgf("allocatorPicker loadConfig error: %v", err)
	}
	log.Info().Msgf("allocatorPicker load [%s] config: %+v", serviceName, svcConfig)

	// 启动 http 服务器
	if firstStart {
		firstStart = false
		go getSvcConfigByHttp(pb.allocatorPort)
	}

	return &allocatorPicker{
		connInfos: cis,
		config:    svcConfig,
	}
}

type connInfo struct {
	sc    balancer.SubConn
	group string
	addr  string

	/* load 用于负载均衡，每次选择最小 load 连接。weight 指定负载比例。一个连接使用一次后，load 更新为 load + 1 / weight
	 * weight 是一个百分比小数，同一组的 weight 之和是1
	 * weight 默认是0，此时同分组的连接增长速度相同（初始化为-1，标记为未处理，但处理后默认是1）
	 * 根据计算公式，假如一个分组两个连接，weight 分别是 0.2 和 0.8，那么 load 分别加 5 和 1.25，load 再次相同时发送的比例是 1：4
	 * weight 不能大于1，需要归一化
	 *
	 * load 不需要写入文件，绑定到一个 ip 上。因为负载均衡是基于一段时间内已有副本来均衡，而不是基于整个历史的均衡
	 * weight 需要写入文件，绑定到 ip，一个分组，固定好比例后可能会给不同副本不同资源，这些副本不应该改变比例
	 */
	load   float64
	weight float64
	index  int
}

type allocatorPicker struct {
	// subConns is the snapshot of the allocator when this picker was
	// created. The slice is immutable. Each Get() will do a round robin
	// selection from it and return the selected SubConn.
	// here use connInfos instead of subConns

	mu sync.Mutex

	connInfos []connInfo

	config *serviceConfig
}

func (p *allocatorPicker) Pick(pickInfo balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	var groupingField = make(map[string][]string)
	// 从 gRPC context 中提取 metadata
	md, ok := metadata.FromOutgoingContext(pickInfo.Ctx)
	if ok {
		for key, values := range md {
			if len(values) > 0 {
				groupingField[key] = append(groupingField[key], values...)
			}
		}
	}

	// 获取所有候选者连接
	candidates := p.selectConn(groupingField)
	// 从候选者连接中，选择一个连接
	sc := p.pickOneConn(candidates)

	p.mu.Unlock()
	return balancer.PickResult{SubConn: sc}, nil
}

// selectConn 返回一组可选择的连接
func (p *allocatorPicker) selectConn(groupingField map[string][]string) []connInfo {
	var candidates []connInfo
	var targetGroup []string
	for groupName, info := range p.config.Group {
		selector := true
		// 只要 metadata 中设置的字段，能匹配完分组 selector 中的字段，就允许路由到这个分组
		for selectorName, selectorValue := range info.Selector {
			values, exists := groupingField[selectorName]
			// 如果该字段不存在,则选择下一个分组
			if !exists || len(values) == 0 {
				selector = false
				break
			}
			// 只要 selector 在 metadata 一个 key 的 values 中的一项，该字段就通过
			selectorPass := false
			for _, v := range values {
				if v == selectorValue {
					selectorPass = true
					break
				}
			}
			if !selectorPass {
				selector = false
				break
			}
		}
		if selector {
			targetGroup = append(targetGroup, groupName)
		}
	}
	// 如果没有匹配，优先使用未分组的副本，未分组的副本服务未匹配的请求
	// 这里也可以修改为：未分组的副本总会被加入选择列表，主要还是看需要怎么配置，是服务未匹配的请求，还是服务所有请求
	if len(targetGroup) == 0 {
		targetGroup = append(targetGroup, defaultGroupName)
	}
	for _, groupName := range targetGroup {
		candidates = append(candidates, p.getGroupConn(groupName)...)
	}
	// 如果没有相关的连接，从所有连接里选择
	if len(candidates) == 0 {
		return p.connInfos
	}

	return candidates
}

// getGroupConn 指定 groupName，获取分组的连接
func (p *allocatorPicker) getGroupConn(groupName string) []connInfo {
	var candidates []connInfo
	for _, ci := range p.connInfos {
		if ci.group == groupName {
			candidates = append(candidates, ci)
		}
	}
	return candidates
}

// pickOneConn 选择一个连接，挑选 load 最小的
// 用轮询算法可能有问题，因为遍历 map 每次都是无序的，没有固定的顺序。因此同一种请求，返回的 candidates 列表也可能顺序不同
func (p *allocatorPicker) pickOneConn(candidates []connInfo) balancer.SubConn {
	// 初始化最小 load 和对应的元素下标
	minLoad := candidates[0].load
	minLoadIndex := 0

	// 找到最小 load 和对应的下标
	for i, info := range candidates {
		if info.load < minLoad {
			minLoad = info.load
			minLoadIndex = i
		}
	}

	// 更新 load，load += 1 / weight
	// 获取目标在 connInfos 中的下标
	index := candidates[minLoadIndex].index

	// 多一层判断，如果未初始化则默认为1。如果走到这层逻辑，则前面可能有错误
	w := p.connInfos[index].weight
	if w == -1 || w == 0 {
		w = 1.0
	}
	// 这里用 0.1 / w，防止 load 增长太快溢出，但 float64 不太可能溢出
	p.connInfos[index].load += 0.1 / w

	return p.connInfos[index].sc
}
