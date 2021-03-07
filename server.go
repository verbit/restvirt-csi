package main

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/golang/glog"
	"google.golang.org/grpc"
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
		csi.RegisterNodeServer(server, &NodeServer{})
	}

	if mode == "controller" {
		driver, err := NewDriver()
		if err != nil {
			log.Fatalf("Failed to create driver: %v", err)
		}
		csi.RegisterControllerServer(server, driver)
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
	glog.V(3).Infof("GRPC call: %s", info.FullMethod)
	// TODO: glog.V(5).Infof("GRPC request: %s", protosanitizer.StripSecrets(req))
	glog.V(5).Infof("GRPC request: %s", req)
	resp, err := handler(ctx, req)
	if err != nil {
		glog.Errorf("GRPC error: %v", err)
	} else {
		// TODO: glog.V(5).Infof("GRPC response: %s", protosanitizer.StripSecrets(resp))
		glog.V(5).Infof("GRPC response: %s", resp)
	}
	return resp, err
}
