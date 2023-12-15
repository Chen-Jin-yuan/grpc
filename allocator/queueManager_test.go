package allocator

import (
	"context"
	"fmt"
	pb "github.com/Chen-Jin-yuan/GRPC-MyHelloWorld/helloworld"
	"github.com/Chen-Jin-yuan/grpc/consul"
	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCount(t *testing.T) {
	go initServer(5e9, 2)
	time.Sleep(1e9)
	go initClient(5e8)
	time.Sleep(5e8)

	for {
		jsonData, err := getHandlerJSONData()
		if err != nil {
			fmt.Printf("err: %v\n", err)
		} else {
			fmt.Printf("jsonData: %s\n", jsonData)
		}
		fmt.Println()
		time.Sleep(1e9)
	}
}

func initClient(rate int) {
	// Set up a connection to the server
	consuladdr := "127.0.0.1:8500"
	svcname := "helloServer"
	consulClient, err := consul.NewClient(consuladdr)

	conn, err := Dial(
		svcname,
		// 路径从项目根路径开始，
		WithBalancerBF(consulClient, "configBF.json", 10001),
		WithStatsHandlerBF(),
	)
	//conn, err := grpc.Dial(*addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewGreeterClient(conn)
	time.Sleep(1e9)
	id := 0
	for {
		md := metadata.Pairs("request-type", "v1")
		ctx := metadata.NewOutgoingContext(context.Background(), md)
		go hello(&c, ctx, id)
		md = metadata.Pairs("request-type", "v1")
		ctx = metadata.NewOutgoingContext(context.Background(), md)
		go helloAgain(&c, ctx, id)
		id += 1
		time.Sleep(time.Duration(rate))
	}
}
func hello(c *pb.GreeterClient, ctx context.Context, id int) {
	r1, err := (*c).SayHello(ctx, &pb.HelloRequest{Name: strconv.Itoa(id)})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r1.GetMessage())
}

func helloAgain(c *pb.GreeterClient, ctx context.Context, id int) {
	r2, err := (*c).SayHelloAgain(ctx, &pb.HelloAgainRequest{Name: strconv.Itoa(id), Number: 1})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	log.Printf("Greeting: %s", r2.GetMessage())
}

func initServer(rate int, maxStreams uint32) {
	consulAddr := "127.0.0.1:8500"
	client, err := consul.NewClient(consulAddr)
	if err != nil {
		log.Fatalf("Got error while initializing Consul agent: %v", err)
	}
	var wg sync.WaitGroup
	wg.Add(2)
	startServer(client, 50000, "50000", &wg, maxStreams, rate)
	startServer(client, 50001, "50001", &wg, maxStreams, rate)
	startServer(client, 50002, "50002", &wg, maxStreams, rate)
	startServer(client, 50003, "50003", &wg, maxStreams, rate)
	wg.Wait()
}
func startServer(client *consul.Client, port int, sid string, wg *sync.WaitGroup, maxStreams uint32, r int) {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	serverIns := server{c: client, id: sid, rate: r}

	s := grpc.NewServer(grpc.MaxConcurrentStreams(maxStreams))
	pb.RegisterGreeterServer(s, &serverIns)

	err = client.Register("helloServer", sid, "127.0.0.1", port)
	if err != nil {
		log.Fatalf("Got error while register service: %v", err)
	}
	log.Printf("server listening at %v", lis.Addr())

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
		wg.Done()
	}()

}

// server is used to implement helloworld.GreeterServer.
type server struct {
	pb.UnimplementedGreeterServer
	c    *consul.Client
	id   string
	rate int
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	time.Sleep(time.Duration(s.rate))
	return &pb.HelloReply{Message: "Hello " + s.id + " " + in.GetName()}, nil
}

// SayHelloAgain implements helloworld.GreeterServer
func (s *server) SayHelloAgain(ctx context.Context, in *pb.HelloAgainRequest) (*pb.HelloAgainReply, error) {
	//time.Sleep(time.Duration(s.rate))
	return &pb.HelloAgainReply{Message: "Hello again " + s.id + " " + in.GetName(), DoubleNumber: in.GetNumber() * 2}, nil
}

// -------------------------------------------------------------------------------------------------------------------
// DialOption 允许配置拨号器的可选参数
type DialOption func(name string) (grpc.DialOption, error)

// WithBalancer 启用客户端负载均衡
func WithBalancer(client *consul.Client, configPath string, allocatorPort int) DialOption {
	return func(name string) (grpc.DialOption, error) {
		// 借助 consul 的服务注册与服务发现机制，执行负载均衡
		consul.InitResolver(client)
		// 如果文件不存在，使用轮询策略
		_, err := os.Stat(configPath)
		if os.IsNotExist(err) {
			return grpc.WithBalancerName(roundrobin.Name), nil
		}
		// 使用 allocator
		Init(configPath, allocatorPort)
		return grpc.WithBalancerName(Name), nil
	}
}

func WithBalancerBF(client *consul.Client, configPath string, allocatorPort int) DialOption {
	return func(name string) (grpc.DialOption, error) {
		// 借助 consul 的服务注册与服务发现机制，执行负载均衡
		consul.InitResolver(client)
		// 如果文件不存在，使用轮询策略
		_, err := os.Stat(configPath)
		if os.IsNotExist(err) {
			return grpc.WithBalancerName(roundrobin.Name), nil
		}
		// 使用 allocator
		InitBF(configPath, allocatorPort)
		return grpc.WithBalancerName(NameBF), nil
	}
}

// WithBalancerRR 启用客户端负载均衡
func WithBalancerRR(client *consul.Client) DialOption {
	return func(name string) (grpc.DialOption, error) {
		// 借助 consul 的服务注册与服务发现机制，执行负载均衡
		consul.InitResolver(client)
		return grpc.WithBalancerName(roundrobin.Name), nil
	}
}

// WithStatsHandler 返回客户端拦截器
func WithStatsHandler() DialOption {
	return func(name string) (grpc.DialOption, error) {
		return grpc.WithStatsHandler(GetClientStatsHandler()), nil
	}
}

// WithStatsHandlerBF 返回客户端拦截器
func WithStatsHandlerBF() DialOption {
	return func(name string) (grpc.DialOption, error) {
		GetClientStatsHandler().SetIsCountFunc()
		return grpc.WithStatsHandler(GetClientStatsHandler()), nil
	}
}

// Dial 返回带有追踪拦截器的负载平衡的gRPC客户端连接
// 传入的 name，可以是单独是目标服务名 svcName，也可以是 consul://consul/svcName 的格式
func Dial(name string, opts ...DialOption) (*grpc.ClientConn, error) {
	name = addSchemeIfNeeded(name, "consul")

	dialopts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Timeout:             120 * time.Second, // 保持活动连接的超时时间
			PermitWithoutStream: true,              // 允许在没有流的情况下执行保持活动操作
		}),
	}

	//设置非安全连接
	dialopts = append(dialopts, grpc.WithInsecure())
	// 应用可选配置参数
	for _, fn := range opts {
		opt, err := fn(name)
		if err != nil {
			return nil, fmt.Errorf("options setting err: %v", err)
		}
		dialopts = append(dialopts, opt)
	}

	// 使用gRPC.Dial创建客户端连接
	conn, err := grpc.Dial(name, dialopts...)
	if err != nil {
		return nil, fmt.Errorf("dial fail %s: %v", name, err)
	}

	return conn, nil
}

func addSchemeIfNeeded(target string, scheme string) string {
	// 标准字格式：scheme://authority/endpoint
	// 检查字符串是否已经以 scheme://scheme 开头
	prefix := scheme + "://" + scheme + "/"
	if !strings.HasPrefix(target, prefix) {
		// 如果不是，添加 scheme://scheme 前缀
		target = prefix + target
	}
	return target
}
