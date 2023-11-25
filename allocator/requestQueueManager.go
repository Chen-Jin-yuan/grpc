package allocator

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"sync"
)

const defaultRequestType = "default"
const defaultTarget = "unknown"
const mdRequestTypeKey = "request-type"
const rpcIDKey = "rpc-id"
const blockLen = 2

type requestsCounter struct {
	TypeCounter map[string]uint64 `json:"type_counter"`
	Total       uint64            `json:"total"`
}

//TODO: 如果某些请求超时失败了，会导致 waiting 和 running 不会执行递减，导致数量卡在那里

// ClientStatsHandler 客户端 grpc 拦截器
type ClientStatsHandler struct {
	// handleRPC 无法知道是发给哪个 IP 的，因此使用 rpcID 去表示，需要加锁
	rpcID            uint64
	idToIp           map[uint64]string
	outReadyRequests map[string]*requestsCounter
	// target -> counter : requestType & total
	// 是一个实时的数据
	waitingRequests map[string]*requestsCounter
	runningRequests map[string]*requestsCounter
	// 互斥锁
	rpcIDMu            sync.Mutex
	idToIpMu           sync.Mutex
	outReadyRequestsMu sync.Mutex
	waitingRequestsMu  sync.Mutex
	runningRequestsMu  sync.Mutex
}

var clientStatsHandler = &ClientStatsHandler{idToIp: make(map[uint64]string),
	outReadyRequests: make(map[string]*requestsCounter), waitingRequests: make(map[string]*requestsCounter),
	runningRequests: make(map[string]*requestsCounter)}

func GetClientStatsHandler() *ClientStatsHandler {
	return clientStatsHandler
}

func (h *ClientStatsHandler) getIdToIp(rpcID uint64) string {
	h.idToIpMu.Lock()
	defer h.idToIpMu.Unlock()
	target, ok := h.idToIp[rpcID]
	if !ok {
		target = defaultTarget
	}
	return target
}
func (h *ClientStatsHandler) getIdToIpAndDelete(rpcID uint64) string {
	h.idToIpMu.Lock()
	defer h.idToIpMu.Unlock()
	target, ok := h.idToIp[rpcID]
	if !ok {
		target = defaultTarget
	} else {
		delete(h.idToIp, rpcID)
	}
	return target
}
func (h *ClientStatsHandler) setIdToIp(rpcID uint64, ip string) {
	h.idToIpMu.Lock()
	h.idToIp[rpcID] = ip
	h.idToIpMu.Unlock()
}

func (h *ClientStatsHandler) incOutReadyRequests(target string, requestType string) {
	h.outReadyRequestsMu.Lock()
	if counter, ok := h.outReadyRequests[target]; ok {
		counter.Total++
		counter.TypeCounter[requestType]++
	} else {
		// 如果不存在，创建内层 counter
		counter = &requestsCounter{TypeCounter: make(map[string]uint64), Total: 0}
		h.outReadyRequests[target] = counter
		counter.Total++
		counter.TypeCounter[requestType]++
	}
	h.outReadyRequestsMu.Unlock()
}

func (h *ClientStatsHandler) getTotalWaitingRequests(target string) uint64 {
	var count uint64 = 0
	h.waitingRequestsMu.Lock()
	counter, ok := h.waitingRequests[target]
	if ok {
		count = counter.Total
	}
	h.waitingRequestsMu.Unlock()
	return count
}
func (h *ClientStatsHandler) getWaitingRequests(target string, requestType string) uint64 {
	var count uint64 = 0
	h.waitingRequestsMu.Lock()
	// 如果 key 不存在，说明还没有请求，返回0
	if counter, ok := h.waitingRequests[target]; ok {
		if value, exists := counter.TypeCounter[requestType]; exists {
			count = value
		}
	}
	h.waitingRequestsMu.Unlock()
	return count
}
func (h *ClientStatsHandler) incWaitingRequests(target string, requestType string) {
	h.waitingRequestsMu.Lock()
	if counter, ok := h.waitingRequests[target]; ok {
		counter.Total++
		counter.TypeCounter[requestType]++
	} else {
		// 如果不存在，创建内层 counter
		counter = &requestsCounter{TypeCounter: make(map[string]uint64), Total: 0}
		h.waitingRequests[target] = counter
		counter.Total++
		counter.TypeCounter[requestType]++
	}
	h.waitingRequestsMu.Unlock()
}
func (h *ClientStatsHandler) decWaitingRequests(target string, requestType string) {
	h.waitingRequestsMu.Lock()
	if counter, ok := h.waitingRequests[target]; ok {
		counter.Total--
		counter.TypeCounter[requestType]--
	} else {
		// 不应该在还没有创建 map 时，就需要减少计数
		log.Error().Msg("decWaitingRequests error")
	}
	h.waitingRequestsMu.Unlock()
}

func (h *ClientStatsHandler) getTotalRunningRequests(target string) uint64 {
	var count uint64 = 0
	h.runningRequestsMu.Lock()
	counter, ok := h.runningRequests[target]
	if ok {
		count = counter.Total
	}
	h.runningRequestsMu.Unlock()
	return count
}
func (h *ClientStatsHandler) getRunningRequests(target string, requestType string) uint64 {
	var count uint64 = 0
	h.runningRequestsMu.Lock()
	// 如果 key 不存在，说明还没有请求，返回0
	if counter, ok := h.runningRequests[target]; ok {
		if value, exists := counter.TypeCounter[requestType]; exists {
			count = value
		}
	}
	h.runningRequestsMu.Unlock()
	return count
}
func (h *ClientStatsHandler) incRunningRequests(target string, requestType string) {
	h.runningRequestsMu.Lock()
	if counter, ok := h.runningRequests[target]; ok {
		counter.Total++
		counter.TypeCounter[requestType]++
	} else {
		// 如果不存在，创建内层 counter
		counter = &requestsCounter{TypeCounter: make(map[string]uint64), Total: 0}
		h.runningRequests[target] = counter
		counter.Total++
		counter.TypeCounter[requestType]++
	}
	h.runningRequestsMu.Unlock()
}
func (h *ClientStatsHandler) decRunningRequests(target string, requestType string) {
	h.runningRequestsMu.Lock()
	if counter, ok := h.runningRequests[target]; ok {
		counter.Total--
		counter.TypeCounter[requestType]--
	} else {
		// 不应该在还没有创建 map 时，就需要减少计数
		log.Error().Msg("decRunningRequests error")
	}
	h.runningRequestsMu.Unlock()
}

func (h *ClientStatsHandler) TagConn(ctx context.Context, _ *stats.ConnTagInfo) context.Context {
	return ctx
}

func (h *ClientStatsHandler) HandleConn(_ context.Context, _ stats.ConnStats) {
	return
}

func (h *ClientStatsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context {
	// handleRPC 无法知道是发给哪个 IP 的，因此使用 rpc-id 去表示，需要加锁

	// 取值+1
	h.rpcIDMu.Lock()
	ctx = context.WithValue(ctx, rpcIDKey, h.rpcID)
	h.rpcID++
	h.rpcIDMu.Unlock()
	return ctx
}

func (h *ClientStatsHandler) HandleRPC(ctx context.Context, s stats.RPCStats) {
	// Begin是一个 RPC 的开始，OutPayload 就发出去了，发出去后到服务端会起一个协程
	// 如果服务端到达了最大并发数，那么会在 OutPayload 前阻塞（排队）

	switch s.(type) {
	//case *stats.Begin:

	//case *stats.End:

	//case *stats.InHeader:

	case *stats.InPayload:
		requestType := defaultRequestType
		// 从 gRPC context 中提取 metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			reqType := md.Get(mdRequestTypeKey)
			if len(reqType) > 0 {
				requestType = reqType[0]
			}
		}
		rpcID := ctx.Value(rpcIDKey).(uint64)
		target := h.getIdToIpAndDelete(rpcID)
		h.decRunningRequests(target, requestType)

	//case *stats.InTrailer:
	//	fmt.Println("handlerRPC InTrailer...")
	//case *stats.OutHeader:
	//	fmt.Println("handlerRPC OutHeader...")
	case *stats.OutPayload:
		requestType := defaultRequestType
		// 从 gRPC context 中提取 metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if ok {
			reqType := md.Get(mdRequestTypeKey)
			if len(reqType) > 0 {
				requestType = reqType[0]
			}
		}

		rpcID := ctx.Value(rpcIDKey).(uint64)
		target := h.getIdToIp(rpcID)
		h.decWaitingRequests(target, requestType)
		h.incRunningRequests(target, requestType)
	}
}

// 获取一个连接的排队数
func getWaitingCount(target string) uint64 {
	return GetClientStatsHandler().getTotalWaitingRequests(target)
}

// 获取一个连接正在运行的请求数
func getRunningCount(target string) uint64 {
	return GetClientStatsHandler().getTotalRunningRequests(target)
}

// 是否达到最大并发度，使用排队数才能知道是否阻塞
func isBlocked(target string) bool {
	// 需要有个阈值，避免 pick 到 out 之间花费的一些时间影响判断
	if getWaitingCount(target) > blockLen {
		return true
	}
	return false
}

func (h *ClientStatsHandler) getJSONData() ([]byte, error) {
	// 加锁
	h.waitingRequestsMu.Lock()
	h.runningRequestsMu.Lock()
	h.outReadyRequestsMu.Lock()
	defer h.waitingRequestsMu.Unlock()
	defer h.runningRequestsMu.Unlock()
	defer h.outReadyRequestsMu.Unlock()
	// 将数据转换为 JSON
	data := struct {
		WaitingRequests  map[string]*requestsCounter `json:"waiting_requests"`
		RunningRequests  map[string]*requestsCounter `json:"running_requests"`
		OutReadyRequests map[string]*requestsCounter `json:"out_ready_requests"`
	}{
		WaitingRequests:  h.waitingRequests,
		RunningRequests:  h.runningRequests,
		OutReadyRequests: h.outReadyRequests,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return jsonData, nil
}

func getHandlerJSONData() ([]byte, error) {
	return GetClientStatsHandler().getJSONData()
}
