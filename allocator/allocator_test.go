package allocator

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
	"os"
	"sort"
	"testing"
	"time"
)

type subC struct {
	id int
}

func (*subC) UpdateAddresses([]resolver.Address) {

}
func (s *subC) Connect() {
	fmt.Printf("select subConn id: %d\n", s.id)
}

func TestLoadConfig(t *testing.T) {
	cis := []connInfo{
		{addr: "1.0.0.1:1"},
		{addr: "1.0.0.1:2"},
		{addr: "1.0.0.1:3"},
		{addr: "1.0.0.1:4"},
		{addr: "1.0.0.1:5"},
		{addr: "1.0.0.1:6"},
		{addr: "1.0.0.1:7"},
	}
	s, err := loadConfig("./example_config.json", cis, "exam_svc")
	if err != nil {

	}
	fmt.Println(len(s.Group["group1"].Weight))
	fmt.Printf("config: %+v\n", s)
}

// 刚好符合 selector，此时选择相关分组
func TestRightMatchedSelector(t *testing.T) {

	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc"}}
	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc"}}
	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc"}}
	sci8 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.8:1", ServerName: "exam_svc"}}
	sci9 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.9:1", ServerName: "exam_svc"}}

	rdCs[&subC{id: 1}] = sci1
	rdCs[&subC{id: 2}] = sci2
	rdCs[&subC{id: 3}] = sci3
	rdCs[&subC{id: 4}] = sci4
	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7
	rdCs[&subC{id: 8}] = sci8
	rdCs[&subC{id: 9}] = sci9

	pb := allocatorPickerBuilder{"./example_config.json", 10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
	p = pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	fmt.Println("==================================test right v1==================================")
	md := metadata.Pairs("request-type", "v1", "method-type", "v1")
	ctx := metadata.NewOutgoingContext(context.Background(), md)
	ctx = context.WithValue(ctx, rpcIDKey, uint64(1))

	pickInfo := balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}

	fmt.Println("==================================test right v2==================================")
	md = metadata.Pairs("request-type", "v2", "method-type", "v2")
	ctx = metadata.NewOutgoingContext(context.Background(), md)
	ctx = context.WithValue(ctx, rpcIDKey, uint64(1))

	pickInfo = balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}

	fmt.Println("==================================test right v3==================================")
	md = metadata.Pairs("request-type3", "v3", "method-type3", "v3")
	ctx = metadata.NewOutgoingContext(context.Background(), md)
	ctx = context.WithValue(ctx, rpcIDKey, uint64(1))

	pickInfo = balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}
}

// 条件符合 selector（但会多 metadata），此时仍选择相关分组
func TestMatchedSelector(t *testing.T) {

	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc"}}
	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc"}}
	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc"}}
	sci8 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.8:1", ServerName: "exam_svc"}}
	sci9 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.9:1", ServerName: "exam_svc"}}

	rdCs[&subC{id: 1}] = sci1
	rdCs[&subC{id: 2}] = sci2
	rdCs[&subC{id: 3}] = sci3
	rdCs[&subC{id: 4}] = sci4
	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7
	rdCs[&subC{id: 8}] = sci8
	rdCs[&subC{id: 9}] = sci9

	pb := allocatorPickerBuilder{"./example_config.json", 10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	fmt.Println("==================================test v1==================================")
	md := metadata.Pairs("request-type", "v1", "method-type", "v1", "over-type", "v1")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	pickInfo := balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}
}

// 条件符合多个 selector，选择多个分组中的连接
func TestOverMatchedSelector(t *testing.T) {

	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc"}}
	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc"}}
	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc"}}
	sci8 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.8:1", ServerName: "exam_svc"}}
	sci9 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.9:1", ServerName: "exam_svc"}}

	rdCs[&subC{id: 1}] = sci1
	rdCs[&subC{id: 2}] = sci2
	rdCs[&subC{id: 3}] = sci3
	rdCs[&subC{id: 4}] = sci4
	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7
	rdCs[&subC{id: 8}] = sci8
	rdCs[&subC{id: 9}] = sci9

	pb := allocatorPickerBuilder{"./example_config.json", 10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	fmt.Println("==================================test over diff field v1==================================")
	md := metadata.Pairs("request-type", "v1", "method-type", "v1",
		"request-type3", "v3", "method-type3", "v3", "over-type", "v1")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	pickInfo := balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 20; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}

	fmt.Println("==================================test over same field v1==================================")
	p = pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
	md = metadata.Pairs("request-type", "v1", "method-type", "v1",
		"request-type", "v2", "method-type", "v2", "request-type", "v3", "over-type", "v1")
	ctx = metadata.NewOutgoingContext(context.Background(), md)

	pickInfo = balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 20; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}
}

// 不符合 selector 但有冗余连接，此时选择冗余连接
func TestNotMatchedSelectorWithNotGrouped(t *testing.T) {

	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc"}}
	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc"}}
	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc"}}
	sci8 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.8:1", ServerName: "exam_svc"}}
	sci9 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.9:1", ServerName: "exam_svc"}}

	rdCs[&subC{id: 1}] = sci1
	rdCs[&subC{id: 2}] = sci2
	rdCs[&subC{id: 3}] = sci3
	rdCs[&subC{id: 4}] = sci4
	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7
	rdCs[&subC{id: 8}] = sci8
	rdCs[&subC{id: 9}] = sci9

	pb := allocatorPickerBuilder{"./example_config.json", 10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	fmt.Println("==================================test none==================================")
	md := metadata.Pairs("request-type", "none", "method-type", "none")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	pickInfo := balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}
}

// 不符合 selector 但没有冗余连接，此时随机选一个
func TestNotMatchedSelectorWithoutNotGrouped(t *testing.T) {

	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc"}}
	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc"}}
	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc"}}

	rdCs[&subC{id: 1}] = sci1
	rdCs[&subC{id: 2}] = sci2
	rdCs[&subC{id: 3}] = sci3
	rdCs[&subC{id: 4}] = sci4
	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7

	pb := allocatorPickerBuilder{"./example_config.json", 10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	fmt.Println("==================================test none==================================")
	md := metadata.Pairs("request-type", "none", "method-type", "none")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	pickInfo := balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 30; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}
}

// 没有 Metadata，此时随机选一个（或从 notGrouped（如果有）里选）
func TestWithoutMetadata(t *testing.T) {

	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc"}}
	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc"}}
	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc"}}
	sci8 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.8:1", ServerName: "exam_svc"}}
	sci9 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.9:1", ServerName: "exam_svc"}}

	rdCs[&subC{id: 1}] = sci1
	rdCs[&subC{id: 2}] = sci2
	rdCs[&subC{id: 3}] = sci3
	rdCs[&subC{id: 4}] = sci4
	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7
	rdCs[&subC{id: 8}] = sci8
	rdCs[&subC{id: 9}] = sci9

	pb := allocatorPickerBuilder{"./example_config.json", 10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	fmt.Println("==================================test without metadata==================================")

	pickInfo := balancer.PickInfo{FullMethodName: "hello", Ctx: context.Background()}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}
}

// 测试没有 selector 的服务
// 没有 selector 的组，总会被加入连接池。如果不这样做就永远不会被选择
func TestWithoutSelector(t *testing.T) {

	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc2"}}
	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc2"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc2"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc2"}}
	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc2"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc2"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc2"}}
	sci8 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.8:1", ServerName: "exam_svc2"}}
	sci9 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.9:1", ServerName: "exam_svc2"}}

	rdCs[&subC{id: 1}] = sci1
	rdCs[&subC{id: 2}] = sci2
	rdCs[&subC{id: 3}] = sci3
	rdCs[&subC{id: 4}] = sci4
	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7
	rdCs[&subC{id: 8}] = sci8
	rdCs[&subC{id: 9}] = sci9

	pb := allocatorPickerBuilder{"./example_config.json", 10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	fmt.Println("==================================test without selector v1==================================")
	md := metadata.Pairs("request-type", "v1", "method-type", "v1")
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	pickInfo := balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}

	fmt.Println("==================================test without selector v2==================================")
	p = pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
	md = metadata.Pairs("request-type", "v2", "method-type", "v2")
	ctx = metadata.NewOutgoingContext(context.Background(), md)

	pickInfo = balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}

	fmt.Println("==================================test without selector v3==================================")
	p = pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
	md = metadata.Pairs("request-type", "v3", "method-type", "v3")
	ctx = metadata.NewOutgoingContext(context.Background(), md)

	pickInfo = balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}

	for i := 0; i < 10; i++ {
		res, err := p.Pick(pickInfo)
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		res.SubConn.Connect()
	}
}

func TestNormalizeWeight(t *testing.T) {
	input1 := []float64{}
	input2 := []float64{-1.0, 2.0, 3.0}
	input3 := []float64{0.2, 0.3}
	input4 := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	input5 := []float64{-0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7}

	number := 5

	output1 := normalizeWeight(input1, number)
	output2 := normalizeWeight(input2, number)
	output3 := normalizeWeight(input3, number)
	output4 := normalizeWeight(input4, number)
	output5 := normalizeWeight(input5, number)

	fmt.Println(output1)
	fmt.Println(output2)
	fmt.Println(output3)
	fmt.Println(output4)
	fmt.Println(output5)
}

func TestAppendWeight(t *testing.T) {
	cis := []connInfo{
		{addr: "1.0.0.1:1", weight: -1},
		{addr: "1.0.0.1:2", weight: -1},
		{addr: "1.0.0.1:3", weight: -1},
		{addr: "1.0.0.1:4", weight: -1},
		{addr: "1.0.0.1:5", weight: -1},
		{addr: "1.0.0.1:6", weight: -1},
		{addr: "1.0.0.1:7", weight: -1},
		{addr: "1.0.0.1:8", weight: -1},
	}
	// 初始化 map
	config := make(allocatorConfig)

	// 返回相关服务的配置
	var svcConfig serviceConfig
	configPath := "./example_config.json"
	svcName := "exam_svc"
	// 读取JSON配置文件并解析
	data, err := os.ReadFile(configPath)
	if err != nil {
	}
	if err = json.Unmarshal(data, &config); err != nil {
	}
	// 提取相关服务配置
	svcConfig = config[svcName]

	newAddr := addrAllocate(cis, &svcConfig)
	fmt.Printf("before append weight, newAddr: %+v, cis: %+v\n", newAddr, cis)
	appendWeightFirst(cis, newAddr, &svcConfig)
	fmt.Printf("after append weight, newAddr: %+v, cis: %+v\n", newAddr, cis)
	if err = writeGroupAddr(svcName+".json", newAddr); err != nil {
	}

}

func TestAppendWeighForNewConn(t *testing.T) {
	cis := []connInfo{
		{addr: "1.0.0.1:1", weight: -1},
		{addr: "1.0.0.2:2", weight: -1},
		{addr: "1.0.0.2:3", weight: -1},
		{addr: "1.0.0.1:4", weight: -1},
		{addr: "1.0.0.2:5", weight: -1},
		{addr: "1.0.0.1:6", weight: -1},
		{addr: "1.0.0.1:8", weight: -1},
		{addr: "1.0.0.1:9", weight: -1},
	}
	// 初始化 map
	config := make(allocatorConfig)

	// 返回相关服务的配置
	var svcConfig serviceConfig
	configPath := "./example_config.json"
	svcName := "exam_svc"
	// 读取JSON配置文件并解析
	data, err := os.ReadFile(configPath)
	if err != nil {
	}
	if err = json.Unmarshal(data, &config); err != nil {
	}
	// 提取相关服务配置
	svcConfig = config[svcName]

	if err = parseAddr(cis, &svcConfig, svcName); err != nil {
	}
	fmt.Printf("parseAddr, cis: %+v\n", cis)
}

func TestHttp(t *testing.T) {
	httpServerStart(10001)
	time.Sleep(1e12)
}

// 测试相同配置的不同副本，初次分配的连接是否一致（未添加排序遍历时，是不一致的）
func TestOrderFirst(t *testing.T) {
	cis := []connInfo{
		{addr: "1.0.0.1:1", weight: -1},
		{addr: "1.0.0.1:2", weight: -1},
		{addr: "1.0.0.1:3", weight: -1},
		{addr: "1.0.0.1:4", weight: -1},
		{addr: "1.0.0.1:5", weight: -1},
		{addr: "1.0.0.1:6", weight: -1},
		{addr: "1.0.0.1:7", weight: -1},
		{addr: "1.0.0.1:8", weight: -1},
	}
	for i := 0; i < 10; i++ {
		// 初始化 map
		config := make(allocatorConfig)

		// 返回相关服务的配置
		var svcConfig serviceConfig
		configPath := "./example_config.json"
		svcName := "exam_svc"
		// 读取JSON配置文件并解析
		data, err := os.ReadFile(configPath)
		if err != nil {
		}
		if err = json.Unmarshal(data, &config); err != nil {
		}
		// 提取相关服务配置
		svcConfig = config[svcName]

		newAddr := addrAllocate(cis, &svcConfig)
		appendWeightFirst(cis, newAddr, &svcConfig)
		fmt.Printf("newAddr: %+v, cis: %+v\n", newAddr, cis)
		if err = writeGroupAddr(svcName+".json", newAddr); err != nil {
		}
	}

}

// 测试相同配置的不同副本，再次分配的连接是否一致（未添加排序遍历时，是不一致的）
func TestOrderForNewConn(t *testing.T) {

	// 初始化 map
	config := make(allocatorConfig)

	// 返回相关服务的配置
	var svcConfig serviceConfig
	configPath := "./example_config.json"
	svcName := "exam_svc"

	for i := 0; i < 10; i++ {
		// 读取JSON配置文件并解析
		data, err := os.ReadFile(configPath)
		if err != nil {
		}
		if err = json.Unmarshal(data, &config); err != nil {
		}
		// 提取相关服务配置
		svcConfig = config[svcName]
		fmt.Printf("==============================================================\n")
		var oldAddr *groupsAddresses
		var newAddr *groupsAddresses

		cis := []connInfo{
			{addr: "1.0.0.1:1", weight: -1},
			{addr: "1.0.0.1:2", weight: -1},
			{addr: "1.0.0.1:13", weight: -1},
			{addr: "1.0.0.1:14", weight: -1},
			{addr: "1.0.0.1:15", weight: -1},
			{addr: "1.0.0.1:16", weight: -1},
			{addr: "1.0.0.1:8", weight: -1},
			{addr: "1.0.0.1:9", weight: -1},
		}

		// 需要读取旧数据做匹配分析
		// 读取旧数据
		fileName := svcName + ".json"
		oldAddr, err = readGroupAddr(fileName)
		if err != nil {
			log.Error().Msgf("readGroupAddr %s error: %v", fileName, err)
		}
		//log.Info().Msgf("matchAddr oldAddr: %+v", oldAddr)
		// 匹配分析
		newAddr = matchAddr(cis, oldAddr, &svcConfig)
		// 添加权重
		appendWeightForNewConn(cis, newAddr, &svcConfig)
		log.Info().Msgf("after append weight newAddr: %+v, cis: %+v", newAddr, cis)
	}
}

// 测试计数器
//func TestCounter(t *testing.T) {
//
//	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
//	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc2"}}
//	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc2"}}
//	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc2"}}
//	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc2"}}
//	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc2"}}
//	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc2"}}
//	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc2"}}
//	sci8 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.8:1", ServerName: "exam_svc2"}}
//	sci9 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.9:1", ServerName: "exam_svc2"}}
//
//	rdCs[&subC{id: 1}] = sci1
//	rdCs[&subC{id: 2}] = sci2
//	rdCs[&subC{id: 3}] = sci3
//	rdCs[&subC{id: 4}] = sci4
//	rdCs[&subC{id: 5}] = sci5
//	rdCs[&subC{id: 6}] = sci6
//	rdCs[&subC{id: 7}] = sci7
//	rdCs[&subC{id: 8}] = sci8
//	rdCs[&subC{id: 9}] = sci9
//
//	pb := allocatorPickerBuilder{"./example_config.json", 10001}
//	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
//
//	fmt.Println("==================================v1==================================")
//	md := metadata.Pairs("request-type", "v1", "method-type", "v1")
//	ctx := metadata.NewOutgoingContext(context.Background(), md)
//
//	pickInfo := balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}
//
//	for i := 0; i < 20; i++ {
//		res, err := p.Pick(pickInfo)
//		if err != nil {
//			fmt.Printf("Pick err: %v\n", err)
//			return
//		}
//		res.SubConn.Connect()
//	}
//
//	fmt.Println("==================================v2==================================")
//	p = pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
//	md = metadata.Pairs("request-type", "v2", "method-type", "v2")
//	ctx = metadata.NewOutgoingContext(context.Background(), md)
//
//	pickInfo = balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}
//
//	for i := 0; i < 5; i++ {
//		res, err := p.Pick(pickInfo)
//		if err != nil {
//			fmt.Printf("Pick err: %v\n", err)
//			return
//		}
//		res.SubConn.Connect()
//	}
//
//	fmt.Println("==================================v3==================================")
//	p = pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
//	md = metadata.Pairs("request-type", "v1", "method-type", "v3")
//	ctx = metadata.NewOutgoingContext(context.Background(), md)
//
//	pickInfo = balancer.PickInfo{FullMethodName: "hello", Ctx: ctx}
//
//	for i := 0; i < 10; i++ {
//		res, err := p.Pick(pickInfo)
//		if err != nil {
//			fmt.Printf("Pick err: %v\n", err)
//			return
//		}
//		res.SubConn.Connect()
//	}
//	if rcs == nil {
//		return
//	}
//	jsonData, err := rcs.ToJSON()
//	if err != nil {
//		fmt.Printf("err: %v\n", err)
//		return
//	}
//	fmt.Printf("counter json data: %v\n", jsonData)
//
//	// http查看
//	time.Sleep(1e12)
//}

// 测试第一次获得所有连接时，强制重新分组
func TestFirstAllocateAll(t *testing.T) {

	rdCs := make(map[balancer.SubConn]base.SubConnInfo)

	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "exam_svc"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "exam_svc"}}

	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "exam_svc"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "exam_svc"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "exam_svc"}}

	subC3 := &subC{id: 3}
	rdCs[&subC{id: 2}] = sci2
	rdCs[subC3] = sci3

	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7

	pb := allocatorPickerBuilder{"./example_config.json", 10001}
	pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
	fmt.Println()

	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "exam_svc"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "exam_svc"}}

	subC4 := &subC{id: 4}

	rdCs[&subC{id: 1}] = sci1
	pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
	fmt.Println()

	rdCs[subC4] = sci4
	pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
	fmt.Println()

	delete(rdCs, subC4)
	delete(rdCs, subC3)
	pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
	fmt.Println()

	rdCs[subC3] = sci3
	rdCs[subC4] = sci4

	pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})
}

func TestGroupData(t *testing.T) {

	// 构造数据
	group := map[string][]int{
		"exam_svc":  {1, 2, 3},
		"exam_svc3": {4, 5, 6},
	}

	// 将数据编码为 JSON 字符串
	jsonData, err := json.Marshal(group)
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}

	// 上面是 post 端做的事情
	// 下面是 server 做的事情

	// 解析 JSON 数据
	var groupData map[string][]int
	if err := json.Unmarshal(jsonData, &groupData); err != nil {
		fmt.Println("Error Unmarshal JSON:", err)
	}
	fmt.Printf("data: %s\n", jsonData)
	fmt.Printf("map: %v\n", groupData)

	cis := []connInfo{
		{addr: "1.0.0.1:1"},
		{addr: "1.0.0.1:2"},
		{addr: "1.0.0.1:3"},
		{addr: "1.0.0.1:4"},
		{addr: "1.0.0.1:5"},
		{addr: "1.0.0.1:6"},
		{addr: "1.0.0.1:7"},
	}
	s, err := loadConfig("./example_config.json", cis, "exam_svc")
	if err != nil {

	}
	fmt.Printf("config: %+v\n", s)

	// 处理接收到的数据
	configMu.Lock()
	for svcName, numbers := range groupData {
		_, ok := config[svcName]
		if !ok {
			// 这个服务不存在
			continue
		}
		groupNames := make([]string, 0, len(config[svcName].Group))
		for groupName := range config[svcName].Group {
			groupNames = append(groupNames, groupName)
		}
		// 对 group 进行排序
		sort.Strings(groupNames)

		// 如果 numbers 长度小于组数，后面的组不会修改；反之多余的 numbers 不会被用到
		minLen := min(len(numbers), len(groupNames))
		for i := 0; i < minLen; i++ {
			// config[svcName].Group[groupNames[i]].Number = numbers[i]
			// serviceConfig 结构体是值类型，无法直接赋值，需要创建一个新的；groupInfo 也是这样
			sc := config[svcName]
			gi := sc.Group[groupNames[i]]
			gi.Number = numbers[i]
			// 重新赋值
			sc.Group[groupNames[i]] = gi
			config[svcName] = sc
		}
	}
	configMu.Unlock()
	cis = []connInfo{
		{addr: "1.0.0.1:1"},
		{addr: "1.0.0.1:4"},
		{addr: "1.0.0.1:5"},
		{addr: "1.0.0.1:6"},
		{addr: "1.0.0.1:7"},
		{addr: "1.0.0.1:8"},
		{addr: "1.0.0.1:9"},
	}
	s, err = loadConfig("./example_config.json", cis, "exam_svc")
	if err != nil {

	}
	fmt.Printf("config: %+v\n", s)
}

func TestBFSelector(t *testing.T) {
	rdCs := make(map[balancer.SubConn]base.SubConnInfo)
	sci1 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.1:1", ServerName: "helloServer"}}
	sci2 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.2:1", ServerName: "helloServer"}}
	sci3 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.3:1", ServerName: "helloServer"}}
	sci4 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.4:1", ServerName: "helloServer"}}
	sci5 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.5:1", ServerName: "helloServer"}}
	sci6 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.6:1", ServerName: "helloServer"}}
	sci7 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.7:1", ServerName: "helloServer"}}
	sci8 := base.SubConnInfo{Address: resolver.Address{Addr: "1.0.0.8:1", ServerName: "helloServer"}}

	rdCs[&subC{id: 1}] = sci1
	rdCs[&subC{id: 2}] = sci2
	rdCs[&subC{id: 3}] = sci3
	rdCs[&subC{id: 4}] = sci4
	rdCs[&subC{id: 5}] = sci5
	rdCs[&subC{id: 6}] = sci6
	rdCs[&subC{id: 7}] = sci7
	rdCs[&subC{id: 8}] = sci8

	pb := allocatorBFPickerBuilder{"./configBF.json", 10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	ctx := context.WithValue(context.Background(), rpcIDKey, uint64(1))

	pickInfo := []balancer.PickInfo{{FullMethodName: "/helloworld.Greeter/SayHello", Ctx: ctx},
		{FullMethodName: "/helloworld.Greeter/SayHelloAgain", Ctx: ctx}}

	for i := 0; i < 24; i++ {
		res, err := p.Pick(pickInfo[i%2])
		if err != nil {
			fmt.Printf("Pick err: %v\n", err)
			return
		}
		jsonData, err := getHandlerJSONData()
		if err != nil {
			fmt.Printf("err: %v\n", err)
		} else {
			fmt.Printf("jsonData: %s\n", jsonData)
		}
		res.SubConn.Connect()
		fmt.Println()
	}
}
