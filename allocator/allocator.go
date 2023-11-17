package allocator

import (
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"math"
	"sort"
	"sync"
)

// Name is the name of allocator.
const Name = "allocator"

var firstStart = true
var firstAllocateAll = true

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
	for subConn, subConnInfo := range info.ReadySCs {
		serviceName = subConnInfo.Address.ServerName
		cis = append(cis, connInfo{
			sc:     subConn,
			group:  "",
			addr:   subConnInfo.Address.Addr,
			load:   0,
			weight: 1, // 这个版本不使用权重
		})
	}

	// range 遍历 ReadySCs 是无序的，对于相同连接，cis 每次都不一样
	// 为了让不同副本有一个相同的结果，需要对 cis 按 addr 排序
	sort.Slice(cis, func(i, j int) bool {
		return cis[i].addr < cis[j].addr
	})
	// addr 从小到大，index 也是从小到大的
	for i := range cis {
		cis[i].index = i
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
		go httpServerStart(pb.allocatorPort)
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

	// 标记该连接在全局 cis 的下标，在最终选出候选者连接中的某一个时，要修改 cis 相应元素的 load
	index int
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
	requestType := defaultRequestType
	md, ok := metadata.FromOutgoingContext(pickInfo.Ctx)
	if ok {
		for key, values := range md {
			if len(values) > 0 {
				groupingField[key] = append(groupingField[key], values...)
			}
		}
		reqType := md.Get(mdRequestTypeKey)
		if len(reqType) > 0 {
			requestType = reqType[0]
		}
	}

	// 计数器记录该次请求，现在先不在这里实现计数
	//countRequest(groupingField)

	// 获取所有候选者连接
	candidates, otherCandidates := p.selectConn(groupingField)
	// 从候选者连接中，选择一个连接
	sc, ip := p.pickOneConn(candidates, otherCandidates)

	rpcID := pickInfo.Ctx.Value(rpcIDKey).(uint64)
	GetClientStatsHandler().setIdToIp(rpcID, ip)
	GetClientStatsHandler().incWaitingRequests(ip, requestType)

	p.mu.Unlock()
	return balancer.PickResult{SubConn: sc}, nil
}

// selectConn 返回一组可选择的连接
func (p *allocatorPicker) selectConn(groupingField map[string][]string) ([]connInfo, []connInfo) {
	var candidates []connInfo
	var otherCandidates []connInfo
	// targetGroup 和 otherGroup 内的 group 都是乱序的
	var targetGroup []string
	var otherGroup []string
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
		} else {
			otherGroup = append(otherGroup, groupName)
		}
	}
	// 如果没有匹配，优先使用未分组的副本，未分组的副本服务未匹配的请求
	// 这里也可以修改为：未分组的副本总会被加入选择列表，主要还是看需要怎么配置，是服务未匹配的请求，还是服务所有请求
	if len(targetGroup) == 0 {
		targetGroup = append(targetGroup, defaultGroupName)
	}
	// 由于 targetGroup 是乱序的，所以 candidates 不是严格按序的
	for _, groupName := range targetGroup {
		candidates = append(candidates, p.getGroupConn(groupName)...)
	}
	// 为了借用资源，有机会选择其他分组的一个副本
	for _, groupName := range otherGroup {
		otherCandidates = append(otherCandidates, p.getGroupConn(groupName)...)
	}
	// 如果没有相关的连接，从所有连接里选择
	if len(candidates) == 0 {
		return p.connInfos, nil
	}

	return candidates, otherCandidates
}

// getGroupConn 指定 groupName，获取分组的连接
// 遍历切片，同组的连接是按序的（即 addr、index 从小到大）
func (p *allocatorPicker) getGroupConn(groupName string) []connInfo {
	var candidates []connInfo
	for _, ci := range p.connInfos {
		if ci.group == groupName {
			candidates = append(candidates, ci)
		}
	}
	return candidates
}

// pickOneConn 选择一个连接，先排序，然后优先发满一个目标，再发满下一个
// 优先发满定义：没有排队，且运行数最大
func (p *allocatorPicker) pickOneConn(candidates []connInfo, otherCandidates []connInfo) (balancer.SubConn, string) {
	var index int
	// 1.如果本组有容器没到最大并发度，优先发满
	index = findMaxRunningCountIndex(candidates)
	// index != -1 说明有非阻塞的容器
	if index != -1 {
		return p.connInfos[index].sc, p.connInfos[index].addr
	}

	// 2.如果本组的满了，其他组的没满，从其他组里选一个发送，借用资源。选择一个非阻塞且 running 数最少的
	index = findMinRunningCountIndex(otherCandidates)
	// index != -1 说明有非阻塞的容器
	if index != -1 {
		return p.connInfos[index].sc, p.connInfos[index].addr
	}

	// 3.如果都满了，在本组内选择排队数最少的目标发送
	// 每个容器最大并发度可能不同，因此满载时 running 数可能不同，不能根据 running 数的大小来选择，可能会挤爆最大并发度最小的容器
	index = findMinWaitingCountIndex(candidates)
	return p.connInfos[index].sc, p.connInfos[index].addr
}

func findMinWaitingCountIndex(cis []connInfo) int {
	// 初始化一个足够大的值作为初始最小值
	minQueueCount := uint64(math.MaxUint64)
	minIndex := 0

	// 遍历 connInfo 切片
	for i, ci := range cis {
		// 获取当前元素的 waiting 数
		queueCount := getWaitingCount(ci.addr)

		// 更新最小值和最小值对应的 index
		if queueCount < minQueueCount {
			minQueueCount = queueCount
			minIndex = cis[i].index
		}
	}

	return minIndex
}

// 借用其他组副本的资源时，选择非阻塞的且 running 数最小的那个
func findMinRunningCountIndex(cis []connInfo) int {
	// 初始化一个足够大的值作为初始最小值
	minRunningCount := uint64(math.MaxUint64)
	minIndex := -1

	// 遍历 connInfo 切片
	for i, ci := range cis {
		// 阻塞了就跳过
		if isBlocked(cis[i].addr) {
			continue
		}

		// 获取当前元素的 running 数
		runningCount := getRunningCount(ci.addr)

		// 更新最小值和最小值对应的 index
		if runningCount < minRunningCount {
			minRunningCount = runningCount
			minIndex = cis[i].index
		}
	}

	return minIndex
}

// 找到最大运行数且非阻塞的目标
func findMaxRunningCountIndex(cis []connInfo) int {
	var maxRunningCount uint64
	maxIndex := -1

	// 遍历 connInfo 切片
	for i, ci := range cis {
		// 阻塞了就跳过
		if isBlocked(cis[i].addr) {
			continue
		}

		// 获取当前元素的 running 数
		runningCount := getRunningCount(ci.addr)

		// 数据初始化为第一个非阻塞的目标
		if maxIndex == -1 {
			maxRunningCount = runningCount
			maxIndex = cis[i].index
		} else if runningCount > maxRunningCount {
			maxRunningCount = runningCount
			maxIndex = cis[i].index
		}
	}
	return maxIndex
}
