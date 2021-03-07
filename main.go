package main

import (
	"flag"
	"log"
	"os"
)

func main() {
	flag.Set("logtostderr", "true")

	if len(os.Args) < 2 {
		log.Fatalln(`you must specify "node" or "controller"`)
	}

	println(os.Args[1])

	var mode string
	switch os.Args[1] {
	case "node", "controller":
		mode = os.Args[1]
	default:
		log.Fatalln(`you must specify "node" or "controller"`)
	}

	fs := flag.NewFlagSet("restvirt-csi", flag.ExitOnError)
	var address string
	fs.StringVar(&address, "csi-address", "", "test me bro")
	var apiEndpoint string
	fs.StringVar(&apiEndpoint, "api-endpoint", "http://localhost:8090", "test me bro")
	fs.Parse(os.Args[2:])

	if address == "" {
		log.Fatalln("csi-address must be set")
	}

	server := NewGRPCServer()
	server.Run(mode, "unix", address)
}
