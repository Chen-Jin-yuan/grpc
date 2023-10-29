package allocator

import (
	"context"
	"fmt"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
	"testing"
)

type subC struct {
	id int
}

func (*subC) UpdateAddresses([]resolver.Address) {

}
func (s *subC) Connect() {
	fmt.Printf("select subConn id: %d\n", s.id)
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

	pb := allocatorPickerBuilder{"./example_config.json"}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

	fmt.Println("==================================test right v1==================================")
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

	fmt.Println("==================================test right v2==================================")
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

	fmt.Println("==================================test right v3==================================")
	md = metadata.Pairs("request-type3", "v3", "method-type3", "v3")
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

	pb := allocatorPickerBuilder{"./example_config.json"}
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

	pb := allocatorPickerBuilder{"./example_config.json"}
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

	pb := allocatorPickerBuilder{"./example_config.json"}
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

	pb := allocatorPickerBuilder{"./example_config.json"}
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

	pb := allocatorPickerBuilder{"./example_config.json"}
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

	pb := allocatorPickerBuilder{"./example_config.json"}
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