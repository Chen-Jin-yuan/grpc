package dialer

import (
	"fmt"
	"github.com/Chen-Jin-yuan/grpc/allocator"
	"google.golang.org/grpc/balancer/roundrobin"
	"os"
	"strings"
	"time"

	"github.com/Chen-Jin-yuan/grpc/consul"
	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"

	"github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// DialOption 允许配置拨号器的可选参数
type DialOption func(name string) (grpc.DialOption, error)

// WithTracer 启用追踪RPC调用
func WithTracer(tracer opentracing.Tracer) DialOption {
	return func(name string) (grpc.DialOption, error) {
		return grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(tracer)), nil
	}
}

// WithBalancer 启用客户端负载均衡，如果配置文件没有问题则
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
		allocator.Init(configPath, allocatorPort)
		return grpc.WithBalancerName(allocator.Name), nil
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
	dialopts = append(dialopts, grpc.WithStatsHandler(allocator.GetClientStatsHandler()))

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
