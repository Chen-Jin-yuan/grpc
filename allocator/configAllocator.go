package allocator

import (
	"github.com/rs/zerolog/log"
	"sync"
)

const targetName string = "target"

// ServiceConfigMap 表示服务配置映射
type ServiceConfigMap map[string][]ServiceEndpoint

// ServiceEndpoint 表示服务的IP和权重
type ServiceEndpoint struct {
	IP     string
	Weight float64
}

var configMap = make(ServiceConfigMap)

// 通过 api 修改 config，读写需要加锁。http server 会修改 config，多个建立不同下游服务的均衡器会读取 config
var configMapMutex sync.RWMutex

// loadConfig 读取配置并将 connInfo 中连接分配到相关的分组
func loadConfig(cis []connInfo, serviceName string) {
	configMapMutex.Lock()
	defer configMapMutex.Unlock()

	// 从 configMap 中获取 serviceName 对应的列表
	if endpoints, ok := configMap[serviceName]; ok {
		// 如果有相关的 value，检索每个元素
		for _, endpoint := range endpoints {
			// 匹配 cis 中的元素的 addr 和 value 列表的 ip
			for i := range cis {
				if cis[i].addr == endpoint.IP {
					// 更新 cis 中元素的 group 和权重
					cis[i].group = targetName
					cis[i].weight = endpoint.Weight
					break
				}
			}
		}
	} else {
		// 如果没有相关的 value，将 cis 列表所有元素的 group 都设置为 targetName
		for i := range cis {
			cis[i].group = targetName
			// 在这里可以设置默认权重，这里设置为1.0
			cis[i].weight = 1.0
		}
	}

	// 打印日志
	log.Info().Msgf("Service: %s", serviceName)
	for _, info := range cis {
		log.Info().Msgf("IP: %s, Group: %s, Weight: %f", info.addr, info.group, info.weight)
	}
}
