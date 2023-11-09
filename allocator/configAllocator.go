/* A library that facilitates the interaction between the allocator and configuration files
 *
 * Multiple services are defined, and the destination service for a request is determined by extracting it
 * from the method name, which involves grouping.
 *
 * The "group" are defined to specify the number of replicas for each group.
 *
 * The "select" mechanism decides which requests match certain metadata criteria. Requests will be directed to a group
 * only if all criteria are met. If multiple groups meet the criteria, one is randomly selected.
 * If no group matches the criteria, one is randomly chosen from all available groups.
 * In any scenario, a connection is always established.
 *
 * The "group-addr" configuration maintains a record of the mapping between each group and its associated addresses.
 * Multiple addresses can be associated with a single group.
 */

package allocator

import (
	"encoding/json"
	"github.com/rs/zerolog/log"
	"os"
	"sort"
)

const defaultGroupName string = "notGrouped"

// json 解析，相关字段必须大写，需要导出至 json 解析器

type serviceConfig struct {
	Group map[string]groupInfo `json:"group"`
}
type groupInfo struct {
	Number   int               `json:"number"`
	Selector map[string]string `json:"selector"`
	Weight   []float64         `json:"weight"`
}
type groupAddresses struct {
	Addresses []string           `json:"addresses"`
	WeightMap map[string]float64 `json:"weight"`
}

type allocatorConfig map[string]serviceConfig
type groupsAddresses map[string]groupAddresses

// loadConfig 读取配置并将 connInfo 中连接分配到相关的分组
func loadConfig(configPath string, cis []connInfo, serviceName string) (*serviceConfig, error) {
	// 配置文件不存在
	_, err := os.Stat(configPath)
	if os.IsNotExist(err) {
		log.Error().Msgf("cannot find file: %s", configPath)
		return nil, err
	}

	// 初始化 map
	config := make(allocatorConfig)

	// 返回相关服务的配置
	var svcConfig serviceConfig

	// 读取JSON配置文件并解析
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Error().Msgf("read %s error: %v", configPath, err)
		return &svcConfig, err
	}
	if err := json.Unmarshal(data, &config); err != nil {
		log.Error().Msgf("json unmarshal %s error: %v", configPath, err)
		return &svcConfig, err
	}
	// 提取相关服务配置
	svcConfig = config[serviceName]

	if err = parseAddr(cis, &svcConfig, serviceName); err != nil {
		log.Error().Msgf("parseAddr error: %v", err)
		return &svcConfig, err
	}

	return &svcConfig, nil
}

func parseAddr(cis []connInfo, sc *serviceConfig, svcName string) error {
	fileName := svcName + ".json"
	var oldAddr *groupsAddresses
	var newAddr *groupsAddresses
	var err error

	needConnNums := 0
	for _, info := range sc.Group {
		needConnNums += info.Number
	}

	// 如果文件不存在，说明是第一次配置，不需要与原来的分组匹配解析
	// 为了使不同副本之间分组一致，第一次检测到连接数是预定个数时，强制按序分组一次
	// 这是因为 grpc 检测到的连接数可能由于时序等原因，第一次可能只获得若干个下游可用连接，在排序条件下也会出现分配不一致
	/*
	 * 例如，有两个上游副本，四个下游副本，ip 为 1，2，3，4；分两组 group1 和 group2，每组两个连接
	 * 	由于时序原因，两个上游副本第一次可能只检测到两个连接，并且可能不一样
	 * 	上游副本1检测到 ip 1和2，分配到 group1；上游副本2检测到 ip 2和3，分配到group1。
	 * 	因此在排序的条件下，也会出现分组不一致。
	 * 这里的解决方式是，在第一次下游四个连接都 ready 了，强制重新分配一次。
	 */
	_, err = os.Stat(fileName)
	if os.IsNotExist(err) || ((len(cis) == needConnNums) && firstAllocateAll) {
		if (len(cis) == needConnNums) && firstAllocateAll {
			firstAllocateAll = false
		}
		newAddr = addrAllocate(cis, sc)
		appendWeightFirst(cis, newAddr, sc)
		// 写入新数据
		if err = writeGroupAddr(fileName, newAddr); err != nil {
			log.Error().Msgf("writeGroupAddr %s, data: %v, error: %v", fileName, newAddr, err)
		}
		log.Info().Msgf("addrAllocate firstAddr: %+v", newAddr)
		return err
	}
	// 需要读取旧数据做匹配分析
	// 读取旧数据
	oldAddr, err = readGroupAddr(fileName)
	if err != nil {
		log.Error().Msgf("readGroupAddr %s error: %v", fileName, err)
		return err
	}
	log.Info().Msgf("matchAddr oldAddr: %+v", oldAddr)
	// 匹配分析
	newAddr = matchAddr(cis, oldAddr, sc)
	log.Info().Msgf("matchAddr newAddr: %+v", newAddr)
	// 添加权重
	appendWeightForNewConn(cis, newAddr, sc)
	log.Info().Msgf("after append weight newAddr: %+v", newAddr)
	// 写回更新数据
	if err = writeGroupAddr(fileName, newAddr); err != nil {
		log.Error().Msgf("writeGroupAddr %s, data: %v, error: %v", fileName, newAddr, err)
	}
	return err
}

// addrAllocate 当连接还没有分配的时候，分配连接到各分组中：配置 connInfo 信息以及需要写入文件的 groupsAddresses 信息
func addrAllocate(cis []connInfo, sc *serviceConfig) *groupsAddresses {
	// 初始化 map
	groupsAddr := make(groupsAddresses)
	counter := 0

	// 对 group 排序遍历 sc.Group，保证分配不是随机的
	// cis 已经是排序过的
	groupNames := make([]string, 0, len(sc.Group))
	for groupName := range sc.Group {
		groupNames = append(groupNames, groupName)
	}

	// 对键进行排序
	sort.Strings(groupNames)

	// 解析 group
	for _, groupName := range groupNames {
		groupINfo := sc.Group[groupName]

		// 一个 group 的地址放入 addr
		var addr []string
		for i := 0; i < groupINfo.Number; i++ {
			// 设置 connInfo 所属分组
			cis[counter].group = groupName
			addr = append(addr, cis[counter].addr)

			counter++
			// 配置分组的副本总和大于或等于可用连接数，后续分组不分配
			if counter >= len(cis) {
				groupsAddr[groupName] = groupAddresses{Addresses: addr}
				return &groupsAddr
			}

			// 该分组分配完毕
			if i == groupINfo.Number-1 {
				groupsAddr[groupName] = groupAddresses{Addresses: addr}
			}
		}
	}

	// 如果有多余的连接，这些连接没有所属的分组，默认为 notGrouped
	var addr []string
	for i := counter; i < len(cis); i++ {
		// 设置 connInfo 所属分组
		cis[counter].group = defaultGroupName
		addr = append(addr, cis[counter].addr)
		// 该分组分配完毕
		if i == len(cis)-1 {
			groupsAddr[defaultGroupName] = groupAddresses{Addresses: addr}
		}
	}

	return &groupsAddr
}

// matchAddr 从新连接和老数据中匹配解析新数据出来，保持原有地址不变，分配新地址
// 注：不应该有重复的地址
func matchAddr(cis []connInfo, oldAddr *groupsAddresses, sc *serviceConfig) *groupsAddresses {
	// 记录每个 group 还缺少多少个地址
	missingCount := make(map[string]int)
	// 放置更新的地址数据
	newAddr := make(groupsAddresses)

	// 1.匹配原有地址，原有地址的 weight 不变
	// 遍历 Addresses 字段
	// 按序遍历 group，这里按不按序都没影响，只是匹配而不会分配
	groupNames := make([]string, 0, len(sc.Group))
	for groupName := range sc.Group {
		groupNames = append(groupNames, groupName)
	}

	// 对键进行排序
	sort.Strings(groupNames)

	for _, groupName := range groupNames {
		groupData := (*oldAddr)[groupName]
		// notGrouped 不匹配，等后面有剩余连接再分配
		if groupName == defaultGroupName {
			continue
		}
		weightMap := make(map[string]float64)
		a := newAddr[groupName].Addresses
		matched := 0

		for _, address := range groupData.Addresses {
			for i := 0; i < len(cis); i++ {
				// 如果一个连接已分组，不再进行匹配，防止当地址重复时出现混乱
				if cis[i].group != "" {
					continue
				}
				// 在未分组的连接中匹配到一个地址，该地址按原来的配置即可
				if cis[i].addr == address {
					// 更新地址数据
					a = append(a, address)
					// 更新 connInfo
					cis[i].group = groupName
					cis[i].weight = groupData.WeightMap[address]
					weightMap[address] = cis[i].weight
					// 匹配数加一
					matched++
					break
				}
			}
		}
		newAddr[groupName] = groupAddresses{Addresses: a, WeightMap: weightMap}
		missingCount[groupName] = 0 - matched
	}

	// 根据配置分析出每组还缺少的地址数
	for groupName, groupINfo := range sc.Group {
		missingCount[groupName] += groupINfo.Number
	}

	// 2.进行剩余连接分配，这些连接可能由于重启、崩溃、新加入等原因，使连接池地址改变。
	// 这些连接只需要随机分配，如果剩余连接数不够，最后一个组（map 并不会排序）不会被满足
	// 新分配的地址，weight 还未分配，需要后续处理
	// 按序遍历，这里不按序会导致地址分配不确定
	for _, groupName := range groupNames {
		miss := missingCount[groupName]
		a := newAddr[groupName].Addresses
		for j := 0; j < miss; j++ {
			for i := 0; i < len(cis); i++ {
				// 寻找还未分组的连接，加入到该分组中
				if cis[i].group != "" {
					continue
				}
				// 更新地址数据
				a = append(a, cis[i].addr)
				// 更新 connInfo
				cis[i].group = groupName
				break
			}
		}
		weightMap := newAddr[groupName].WeightMap
		newAddr[groupName] = groupAddresses{Addresses: a, WeightMap: weightMap}
	}

	// 3.如果有多余的连接，这些连接没有所属的分组，默认为 notGrouped
	var defaultAddr []string
	for i := 0; i < len(cis); i++ {
		if cis[i].group != "" {
			continue
		}
		// 更新地址数据
		defaultAddr = append(defaultAddr, cis[i].addr)
		// 更新 connInfo
		cis[i].group = defaultGroupName
	}
	// 该分组分配完毕
	if defaultAddr != nil {
		newAddr[defaultGroupName] = groupAddresses{Addresses: defaultAddr}
	}

	return &newAddr
}

// readGroupAddr 读取 groupAddr 数据
func readGroupAddr(fileName string) (*groupsAddresses, error) {
	var groupsAddrConfig groupsAddresses

	// 读取文件内容
	data, err := os.ReadFile(fileName)
	if err != nil {
		return &groupsAddrConfig, err
	}

	// 解析 JSON 数据
	if err := json.Unmarshal(data, &groupsAddrConfig); err != nil {
		return &groupsAddrConfig, err
	}

	return &groupsAddrConfig, nil
}

// writeGroupAddr 将 groupAddr 数据写入 JSON 文件。
func writeGroupAddr(fileName string, g *groupsAddresses) error {
	// 编码数据为 JSON
	jsonData, err := json.Marshal(g)
	if err != nil {
		return err
	}

	err = os.WriteFile(fileName, jsonData, 0644)
	if err != nil {
		return err
	}

	return nil
}
