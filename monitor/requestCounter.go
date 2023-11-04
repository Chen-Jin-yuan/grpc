package monitor

import (
	"fmt"
	"strconv"
	"sync"
)

type RequestCounters struct {
	// 读写互斥锁
	mu       sync.RWMutex
	counters map[interface{}]Counter
}

// RequestCounter : *RequestCounter 实现 Counter 接口。这样获取一个 Counter 返回的就是引用
type RequestCounter struct {
	// 读写互斥锁
	mu     sync.RWMutex
	counts map[interface{}]int
}

func NewCounter() Counter {
	return &RequestCounter{
		counts: make(map[interface{}]int),
	}
}

func (rc *RequestCounter) IncrementOfValue(RequestValue interface{}) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.counts[RequestValue]++
}

func (rc *RequestCounter) GetCountOfValue(RequestValue interface{}) int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	return rc.counts[RequestValue]
}

func (rc *RequestCounter) GetData() map[interface{}]int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	data := make(map[interface{}]int)
	for key, value := range rc.counts {
		data[key] = value
	}
	return data
}

func NewRequestCounters() *RequestCounters {
	return &RequestCounters{
		counters: make(map[interface{}]Counter),
	}
}

func (rcs *RequestCounters) GetCounter(RequestKey interface{}) Counter {
	// 加写锁防止同时 NewCounter
	rcs.mu.Lock()
	defer rcs.mu.Unlock()
	if rcs.counters[RequestKey] == nil {
		rcs.counters[RequestKey] = NewCounter()
	}

	return rcs.counters[RequestKey]
}

func (rcs *RequestCounters) GetCountersData() map[interface{}]map[interface{}]int {
	rcs.mu.RLock()
	defer rcs.mu.RUnlock()

	data := make(map[interface{}]map[interface{}]int)

	// 遍历每个计数器
	for key, counter := range rcs.counters {
		data[key] = counter.GetData()
	}

	return data
}

func (rcs *RequestCounters) ToJSON() (map[string]map[string]int, error) {
	// 不用加锁，获取到数据后都是剩余的一些处理
	data := rcs.GetCountersData()
	jsonMap := make(map[string]map[string]int)

	for counterKey, counterData := range data {
		var keyStr string
		var valueMap map[string]int

		switch k := counterKey.(type) {
		case string:
			keyStr = k
		case int:
			keyStr = strconv.Itoa(k)
		default:
			return nil, fmt.Errorf("unsupported counter key type: %T", counterKey)
		}

		valueMap = make(map[string]int)
		for key, value := range counterData {
			switch k := key.(type) {
			case string:
				valueMap[k] = value
			case int:
				valueMap[strconv.Itoa(k)] = value
			default:
				return nil, fmt.Errorf("unsupported value key type: %T", key)
			}
		}

		jsonMap[keyStr] = valueMap
	}

	return jsonMap, nil
}
