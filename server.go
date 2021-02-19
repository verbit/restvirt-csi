package main

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"log"
	"net"
	"sync"
)

type grpcServer struct {
	wg     sync.WaitGroup
	server *grpc.Server
}

func NewGRPCServer() *grpcServer {
	return &grpcServer{}
}

func (s *grpcServer) Start(apiHost string, network string, address string) {

	s.wg.Add(1)

	go s.Run(apiHost, network, address)

	return
}

func (s *grpcServer) Run(apiHost string, network string, address string) {

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)
	s.server = server

	driver, err := NewDriver(apiHost)
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}
	csi.RegisterIdentityServer(server, driver)
	csi.RegisterControllerServer(server, driver)
	csi.RegisterNodeServer(server, driver)

	listener, err := net.Listen(network, address)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	server.Serve(listener)
}

func (s *grpcServer) Wait() {
	s.wg.Wait()
}

func (s *grpcServer) Stop() {
	s.server.GracefulStop()
}

func (s *grpcServer) ForceStop() {
	s.server.Stop()
}
