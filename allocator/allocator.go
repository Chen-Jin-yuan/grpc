package allocator

import (
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/metadata"
	"sync"
)

// Name is the name of allocator.
const Name = "allocator"

var firstStart = true

// NewBuilder creates a new weight balancer builder.
// HealthCheck 会使用服务端的健康检查来判断服务是否可用，如果服务端没有实现健康检查，则该配置不起作用
func newBuilder(httpServerPort int) balancer.Builder {
	return base.NewBalancerBuilderV2(Name, &allocatorPickerBuilder{
		allocatorPort: httpServerPort}, base.Config{HealthCheck: true})
}

func Init(httpServerPort int) {
	balancer.Register(newBuilder(httpServerPort))
}

type allocatorPickerBuilder struct {
	allocatorPort int
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
			weight: 1,
		})
	}

	for i := range cis {
		cis[i].index = i
	}

	log.Info().Msgf("allocatorPicker connInfo list: %+v", cis)
	// 加载配置，一并把地址配置好
	loadConfig(cis, serviceName)

	log.Info().Msgf("allocatorPicker load [%s] config", serviceName)

	// 启动 http 服务器
	if firstStart {
		firstStart = false
		go httpServerStart(pb.allocatorPort)
	}

	return &allocatorPicker{connInfos: cis}
}

type connInfo struct {
	sc    balancer.SubConn
	group string // 目标组或非目标组
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
}

func (p *allocatorPicker) Pick(pickInfo balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()

	// 从 gRPC context 中提取 metadata
	requestType := defaultRequestType
	md, ok := metadata.FromOutgoingContext(pickInfo.Ctx)
	if ok {
		reqType := md.Get(mdRequestTypeKey)
		if len(reqType) > 0 {
			requestType = reqType[0]
		}
	}

	candidates := p.selectConn()
	// 从候选者连接中，选择一个连接
	sc, ip := p.pickOneConn(candidates)

	rpcID := pickInfo.Ctx.Value(rpcIDKey).(uint64)
	GetClientStatsHandler().setIdToIp(rpcID, ip)
	//GetClientStatsHandler().incWaitingRequests(ip, requestType)
	GetClientStatsHandler().incOutReadyRequests(ip, requestType)
	// requestType 总数计数，key 为 all
	GetClientStatsHandler().incOutReadyRequests("all", requestType)

	p.mu.Unlock()
	return balancer.PickResult{SubConn: sc}, nil
}

// selectConn 返回一组可选择的连接
func (p *allocatorPicker) selectConn() []connInfo {
	// 筛选出 group 为 targetName 的候选连接
	var candidates []connInfo
	for _, info := range p.connInfos {
		if info.group == targetName {
			candidates = append(candidates, info)
		}
	}

	// 如果没有符合条件的连接，将 candidates 实例化为整个 cis
	if len(candidates) == 0 {
		candidates = p.connInfos
	}

	return candidates
}

// pickOneConn 选择一个连接，挑选 load 最小的
// 用轮询算法可能有问题，因为遍历 map 每次都是无序的，没有固定的顺序。因此同一种请求，返回的 candidates 列表也可能顺序不同
func (p *allocatorPicker) pickOneConn(candidates []connInfo) (balancer.SubConn, string) {
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

	// 更新 load，load += 1
	// 获取目标在 connInfos 中的下标
	index := candidates[minLoadIndex].index

	// 多一层判断，如果未初始化则默认为1。如果走到这层逻辑，则前面可能有错误
	w := p.connInfos[index].weight
	if w == -1 || w == 0 {
		w = 1.0
	}
	// 这里用 0.1 / w，防止 load 增长太快溢出，但 float64 不太可能溢出
	p.connInfos[index].load += 0.1 / w

	return p.connInfos[index].sc, p.connInfos[index].addr
}

/*
 ************************************************* allocatorByFunc *************************************************
 ************ allocator 按照请求进行分组；对同种请求调用同一个下游微服务时，allocatorByFunc 按调用的 function 分组 *************
 */

//// NameBF is the name of allocator.
//const NameBF = "allocatorByFunc"
//
//// NewBuilder creates a new weight balancer builder.
//// HealthCheck 会使用服务端的健康检查来判断服务是否可用，如果服务端没有实现健康检查，则该配置不起作用
//func newBuilderBF(configPath string, httpServerPort int) balancer.Builder {
//	return base.NewBalancerBuilderV2(NameBF, &allocatorBFPickerBuilder{allocatorBFConfigPath: configPath,
//		allocatorBFPort: httpServerPort}, base.Config{HealthCheck: true})
//}
//
//func InitBF(configPath string, httpServerPort int) {
//	balancer.Register(newBuilderBF(configPath, httpServerPort))
//}
//
//type allocatorBFPickerBuilder struct {
//	allocatorBFConfigPath string
//	allocatorBFPort       int
//}
//
//func (pb *allocatorBFPickerBuilder) Build(info base.PickerBuildInfo) balancer.V2Picker {
//	grpclog.Infof("allocatorBFPicker: newPicker called with info: %v", info)
//	if len(info.ReadySCs) == 0 {
//		return base.NewErrPickerV2(balancer.ErrNoSubConnAvailable)
//	}
//
//	// 连接信息
//	var cis []connInfo
//	var serviceName string
//	// 注：依赖于服务发现 resolver 把 ServiceName 写入 ServerName
//	for subConn, subConnInfo := range info.ReadySCs {
//		serviceName = subConnInfo.Address.ServerName
//		cis = append(cis, connInfo{
//			sc:     subConn,
//			group:  "",
//			addr:   subConnInfo.Address.Addr,
//			load:   0,
//			weight: 1, // 这个版本不使用权重
//		})
//	}
//
//	// range 遍历 ReadySCs 是无序的，对于相同连接，cis 每次都不一样
//	// 为了让不同副本有一个相同的结果，需要对 cis 按 addr 排序
//	sort.Slice(cis, func(i, j int) bool {
//		return cis[i].addr < cis[j].addr
//	})
//	// addr 从小到大，index 也是从小到大的
//	for i := range cis {
//		cis[i].index = i
//	}
//
//	log.Info().Msgf("allocatorBFPicker connInfo list: %+v", cis)
//	// 加载配置，一并把地址配置好
//	svcConfig, err := loadConfig(pb.allocatorBFConfigPath, cis, serviceName)
//	if err != nil {
//		log.Error().Msgf("allocatorBFPicker loadConfig error: %v", err)
//	}
//	log.Info().Msgf("allocatorBFPicker load [%s] config: %+v", serviceName, svcConfig)
//
//	// 启动 http 服务器
//	if firstStart {
//		firstStart = false
//		go httpServerStart(pb.allocatorBFPort)
//	}
//
//	return &allocatorBFPicker{
//		connInfos: cis,
//		config:    svcConfig,
//	}
//}
//
//type allocatorBFPicker struct {
//	// subConns is the snapshot of the allocator when this picker was
//	// created. The slice is immutable. Each Get() will do a round robin
//	// selection from it and return the selected SubConn.
//	// here use connInfos instead of subConns
//
//	mu sync.Mutex
//
//	connInfos []connInfo
//
//	config *serviceConfig
//}
//
//func (p *allocatorBFPicker) Pick(pickInfo balancer.PickInfo) (balancer.PickResult, error) {
//	p.mu.Lock()
//	var groupingField = make(map[string][]string)
//	// 解析 functionName
//	_, functionName := extractServiceAndMethod(pickInfo.FullMethodName)
//	if functionName == "" {
//		functionName = defaultFunctionName
//	}
//	groupingField[functionTypeKey] = append(groupingField[functionTypeKey], functionName)
//
//	// 获取所有候选者连接
//	candidates, _ := p.selectConn(groupingField)
//	// 从候选者连接中，选择一个连接
//	sc, ip := p.pickOneConn(candidates)
//
//	rpcID := pickInfo.Ctx.Value(rpcIDKey).(uint64)
//	GetClientStatsHandler().setIdToIp(rpcID, ip)
//	GetClientStatsHandler().setIdToFuncName(rpcID, functionName)
//	GetClientStatsHandler().incWaitingRequests(ip, functionName)
//	GetClientStatsHandler().incOutReadyRequests(ip, functionName)
//
//	p.mu.Unlock()
//	return balancer.PickResult{SubConn: sc}, nil
//}
//
//// selectConn 返回一组可选择的连接
//func (p *allocatorBFPicker) selectConn(groupingField map[string][]string) ([]connInfo, []connInfo) {
//	var candidates []connInfo
//	var otherCandidates []connInfo
//	// targetGroup 和 otherGroup 内的 group 都是乱序的
//	var targetGroup []string
//	var otherGroup []string
//	for groupName, info := range p.config.Group {
//		selector := true
//		// 匹配 function name
//		for selectorName, selectorValue := range info.Selector {
//			values, exists := groupingField[selectorName]
//			// 如果该字段不存在,则选择下一个分组
//			if !exists || len(values) == 0 {
//				selector = false
//				break
//			}
//			selectorPass := false
//			for _, v := range values {
//				if v == selectorValue {
//					selectorPass = true
//					break
//				}
//			}
//			if !selectorPass {
//				selector = false
//				break
//			}
//		}
//		if selector {
//			targetGroup = append(targetGroup, groupName)
//		} else {
//			otherGroup = append(otherGroup, groupName)
//		}
//	}
//	// 如果没有匹配，优先使用未分组的副本，未分组的副本服务未匹配的请求
//	// 这里也可以修改为：未分组的副本总会被加入选择列表，主要还是看需要怎么配置，是服务未匹配的请求，还是服务所有请求
//	if len(targetGroup) == 0 {
//		targetGroup = append(targetGroup, defaultGroupName)
//	} else {
//		otherGroup = append(otherGroup, defaultGroupName)
//	}
//	// 由于 targetGroup 是乱序的，所以 candidates 不是严格按序的
//	for _, groupName := range targetGroup {
//		candidates = append(candidates, p.getGroupConn(groupName)...)
//	}
//	// 为了借用资源，有机会选择其他分组的一个副本
//	for _, groupName := range otherGroup {
//		otherCandidates = append(otherCandidates, p.getGroupConn(groupName)...)
//	}
//	// 如果没有相关的连接，从所有连接里选择
//	if len(candidates) == 0 {
//		return p.connInfos, nil
//	}
//
//	return candidates, otherCandidates
//}
//
//// getGroupConn 指定 groupName，获取分组的连接
//// 遍历切片，同组的连接是按序的（即 addr、index 从小到大）
//func (p *allocatorBFPicker) getGroupConn(groupName string) []connInfo {
//	var candidates []connInfo
//	for _, ci := range p.connInfos {
//		if ci.group == groupName {
//			candidates = append(candidates, ci)
//		}
//	}
//	return candidates
//}
//
//// pickOneConn 选择一个连接，挑选 load 最小的
//// 用轮询算法可能有问题，因为遍历 map 每次都是无序的，没有固定的顺序。因此同一种请求，返回的 candidates 列表也可能顺序不同
//func (p *allocatorBFPicker) pickOneConn(candidates []connInfo) (balancer.SubConn, string) {
//	// 初始化最小 load 和对应的元素下标
//	minLoad := candidates[0].load
//	minLoadIndex := 0
//
//	// 找到最小 load 和对应的下标
//	for i, info := range candidates {
//		if info.load < minLoad {
//			minLoad = info.load
//			minLoadIndex = i
//		}
//	}
//
//	// 更新 load，load += 1
//	// 获取目标在 connInfos 中的下标
//	index := candidates[minLoadIndex].index
//
//	p.connInfos[index].load += 1
//
//	return p.connInfos[index].sc, p.connInfos[index].addr
//}
//
//// input: /helloworld.Greeter/SayHello
//// helloworld 是包名，Greeter 是服务名，SayHello 是 function
//func extractServiceAndMethod(input string) (string, string) {
//	parts := strings.Split(input, "/")
//	if len(parts) == 3 {
//		// 解析包名和服务名
//		//parts1 := strings.Split(parts[1], ".")
//		//if len(parts1) == 2 {
//		//	return parts1[1], parts[2]
//		//}
//		return parts[1], parts[2]
//	}
//	return "", ""
//}

///*
// ************************************************* allocatorRR *************************************************
// ****************************************** 装配了计数器的简单负载均衡策略 *******************************************
//************************************ 只需要记一个总数即可，负载均衡不需要借用等策略 *************************************
//*/
//
//// NameRR is the name of allocatorRR balancer.
//const NameRR = "allocatorRR"
//
//// newBuilderRR creates a new roundrobin balancer builder.
//func newBuilderRR() balancer.Builder {
//	return base.NewBalancerBuilderV2(NameRR, &rrPickerBuilder{}, base.Config{HealthCheck: true})
//}
//
//func InitRR() {
//	balancer.Register(newBuilderRR())
//}
//
//type rrPickerBuilder struct{}
//
//func (*rrPickerBuilder) Build(info base.PickerBuildInfo) balancer.V2Picker {
//	grpclog.Infof("allocatorRRPicker: newPicker called with info: %v", info)
//	if len(info.ReadySCs) == 0 {
//		return base.NewErrPickerV2(balancer.ErrNoSubConnAvailable)
//	}
//	var scs []balancer.SubConn
//
//	for sc := range info.ReadySCs {
//		scs = append(scs, sc)
//	}
//	return &rrPicker{
//		subConns: scs,
//		// Start at a random index, as the same RR balancer rebuilds a new
//		// picker when SubConn states change, and we don't want to apply excess
//		// load to the first server in the list.
//		next: Intn(len(scs)),
//	}
//}
//
//type rrPicker struct {
//	// subConns is the snapshot of the roundrobin balancer when this picker was
//	// created. The slice is immutable. Each Get() will do a round robin
//	// selection from it and return the selected SubConn.
//	subConns []balancer.SubConn
//
//	mu   sync.Mutex
//	next int
//}
//
//func (p *rrPicker) Pick(balancer.PickInfo) (balancer.PickResult, error) {
//	p.mu.Lock()
//	sc := p.subConns[p.next]
//	p.next = (p.next + 1) % len(p.subConns)
//	p.mu.Unlock()
//	return balancer.PickResult{SubConn: sc}, nil
//}
