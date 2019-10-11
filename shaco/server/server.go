package shaco

import (
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type server struct {
	Server      *grpc.Server
	serviceName string
	target      string
	host        string
	port        int
	weight      int
}

func NewServer(target string, serviceName string, weight, port int, opts ...grpc.ServerOption) *server {
	var host string
	iface, err := net.InterfaceByName("eth0")
	var addrs []net.Addr
	if err != nil {
		fmt.Errorf("get interface address by nane err: %s", err)
		addrs, err = net.InterfaceAddrs()
	} else {
		addrs, err = iface.Addrs()
	}
	if err != nil {
		panic(fmt.Sprintf("get interface address err: %s", err))
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				host = ipnet.IP.String()
			}
		}
	}

	if host == "" {
		panic("get local ip empty")
	}
	s := grpc.NewServer(opts...)
	return &server{
		Server:      s,
		target:      target,
		serviceName: serviceName,
		host:        host,
		port:        port,
		weight:      weight,
	}
}

func (s *server) Run() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	err = register(s.target, s.serviceName, s.host, s.port, s.weight, time.Second*10, 15)
	if err != nil {
		return err
	}

	reflection.Register(s.Server)
	if err := s.Server.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %v", err)
	}
	return nil
}

func (s *server) Close() {
	s.Server.Stop()
	unRegister()
}
