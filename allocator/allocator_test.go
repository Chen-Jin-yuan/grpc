package allocator

import (
	"context"
	"fmt"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
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

func TestLoadConfigAndPickOneConn(t *testing.T) {
	configMap["exam_svc"] = []ServiceEndpoint{
		{IP: "1.0.0.1:1", Weight: 0.2},
		{IP: "1.0.0.3:1", Weight: 0.8},
	}
	configMap["exam_svc2"] = []ServiceEndpoint{
		{IP: "1.0.0.2:1", Weight: 1},
	}

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

	pb := allocatorPickerBuilder{10001}
	p := pb.Build(base.PickerBuildInfo{ReadySCs: rdCs})

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

}

func TestHttp(t *testing.T) {
	go httpServerStart(10001)
	fmt.Printf("configMap: %+v\n", configMap)

	// 执行 server_test.py
	time.Sleep(5e9)

	fmt.Printf("configMap: %+v\n", configMap)
}
