package allocator

import "math"

// cis、newAddr 均已分好组，需要配置权重
func appendWeightFirst(cis []connInfo, newAddr *groupsAddresses, sc *serviceConfig) {
	weights := make(map[string][]float64)
	// 配置 weightMap
	groupAddrToWeight := make(map[string]map[string]float64)

	for groupName, info := range sc.Group {
		w := normalizeWeight(info.Weight, info.Number)
		weights[groupName] = w
		groupAddrToWeight[groupName] = make(map[string]float64)
	}

	for i := range cis {
		// 未分组的连接，权重为1
		if cis[i].group == defaultGroupName {
			cis[i].weight = 1
		}
		// 遍历该组的 weight，直接分配
		for j, weight := range weights[cis[i].group] {
			if weight == -1 {
				continue
			}
			cis[i].weight = weight
			// -1 标记为该权重已分配
			weights[cis[i].group][j] = -1
			groupAddrToWeight[cis[i].group][cis[i].addr] = weight
			break
		}
	}

	for groupName := range *newAddr {
		a := (*newAddr)[groupName].Addresses
		(*newAddr)[groupName] = groupAddresses{Addresses: a, WeightMap: groupAddrToWeight[groupName]}
	}
}

// 对新连接进行分组
func appendWeightForNewConn(cis []connInfo, newAddr *groupsAddresses, sc *serviceConfig) {
	weights := make(map[string][]float64)
	// 配置 weightMap
	groupAddrToWeight := make(map[string]map[string]float64)

	for groupName, info := range sc.Group {
		w := normalizeWeight(info.Weight, info.Number)
		weights[groupName] = w
		groupAddrToWeight[groupName] = make(map[string]float64)
	}

	// 先遍历一遍 cis，把已经分配权重的那些连接对应的权重列表过滤
	for i := range cis {
		if cis[i].weight == -1 {
			continue
		}
		// 把分组权重列表相同的一项删去
		for j, weight := range weights[cis[i].group] {
			if cis[i].weight == weight {
				groupAddrToWeight[cis[i].group][cis[i].addr] = weight
				weights[cis[i].group][j] = -1
				break
			}
		}
	}

	for i := range cis {
		// 未分组的连接，权重为1
		if cis[i].group == defaultGroupName {
			cis[i].weight = 1
		}
		// 已分配权重的地址跳过
		if cis[i].weight != -1 {
			continue
		}
		// 遍历该组的 weight，直接分配
		for j, weight := range weights[cis[i].group] {
			if weight == -1 {
				continue
			}
			cis[i].weight = weight
			// -1 标记为该权重已分配
			weights[cis[i].group][j] = -1
			groupAddrToWeight[cis[i].group][cis[i].addr] = weight
			break
		}
	}

	for groupName := range *newAddr {
		a := (*newAddr)[groupName].Addresses
		(*newAddr)[groupName] = groupAddresses{Addresses: a, WeightMap: groupAddrToWeight[groupName]}
	}
}

func normalizeWeight(input []float64, number int) []float64 {
	length := len(input)

	if length == 0 {
		// 情况 1: 输入切片长度为0，返回均衡的切片
		input = make([]float64, number)
		for i := 0; i < number; i++ {
			input[i] = 1.0
		}
	} else if length == number {
		// 情况 2: 输入切片长度等于number，进行归一化

	} else if length < number {
		// 情况 3: 输入切片长度小于number，补齐1并归一化
		for i := length; i < number; i++ {
			input = append(input, 1.0)
		}
	} else {
		// 情况 4: 输入切片长度大于number，删除超出部分并归一化
		input = input[:number]
	}
	absSlice(input)
	normalizeSlice(input)
	return input
}

// 对一个切片进行归一化
func normalizeSlice(input []float64) {
	total := 0.0
	for _, value := range input {
		total += value
	}
	for i := range input {
		input[i] = input[i] / total
	}
}

// 切片绝对值化
func absSlice(input []float64) {
	for i := range input {
		input[i] = math.Abs(input[i])
	}
}
