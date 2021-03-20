package main

import (
	"context"
	"log"
	"net"
	"os"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
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

func (s *grpcServer) Run(mode string, network string, address string) {
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(logGRPC),
	}
	server := grpc.NewServer(opts...)
	s.server = server

	csi.RegisterIdentityServer(server, &IdentityServer{})

	if mode == "node" {
		csi.RegisterNodeServer(server, NewNodeServer())
	}

	if mode == "controller" {
		driver, err := NewDriver()
		if err != nil {
			log.Fatalf("Failed to create driver: %v", err)
		}
		csi.RegisterControllerServer(server, driver)
	}

	_, err := os.Stat(address)
	if err == nil {
		os.Remove(address)
	}
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

func logGRPC(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	klog.V(3).Infof("GRPC call: %s", info.FullMethod)
	// TODO: klog.V(5).Infof("GRPC request: %s", protosanitizer.StripSecrets(req))
	klog.V(5).Infof("GRPC request: %s", req)
	resp, err := handler(ctx, req)
	if err != nil {
		klog.Errorf("GRPC error: %v", err)
	} else {
		// TODO: klog.V(5).Infof("GRPC response: %s", protosanitizer.StripSecrets(resp))
		klog.V(5).Infof("GRPC response: %s", resp)
	}
	return resp, err
}
