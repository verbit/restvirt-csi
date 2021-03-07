package main

import (
	"flag"
	"log"
)

func main() {
	var address string
	flag.StringVar(&address, "csi-address", "", "test me bro")
	var apiEndpoint string
	flag.StringVar(&apiEndpoint, "api-endpoint", "http://localhost:8090", "test me bro")

	flag.Set("logtostderr", "true")

	flag.Parse()

	if address == "" {
		log.Fatalln("csi-address must be set")
	}

	if flag.NArg() < 1 {
		log.Fatalln(`you must specify "node" or "controller"`)
	}

	mode := flag.Arg(0)

	server := NewGRPCServer()
	server.Run(mode, "unix", address)
}
