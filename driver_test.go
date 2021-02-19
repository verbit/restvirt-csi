package main

import (
	"fmt"
	"github.com/kubernetes-csi/csi-test/v4/pkg/sanity"
)

import (
	"testing"
)

func TestMeBro(t *testing.T) {
	network := "unix"
	address := "/tmp/csi.sock"
	endpoint := fmt.Sprintf("%s://%s", network, address)

	server := NewGRPCServer()
	server.Start("http://localhost:8090", network, address)
	defer server.Stop()

	config := sanity.NewTestConfig()
	config.Address = endpoint

	sanity.Test(t, config)
}
