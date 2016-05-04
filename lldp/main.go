package main

import (
	"flag"
	"fmt"
	"l2/lldp/rpc"
	"l2/lldp/server"
	"utils/keepalive"
	"utils/logging"
)

func main() {
	fmt.Println("Starting lldp daemon")
	paramsDir := flag.String("params", "./params", "Params directory")
	flag.Parse()
	fileName := *paramsDir
	if fileName[len(fileName)-1] != '/' {
		fileName = fileName + "/"
	}

	fmt.Println("Start logger")
	logger, err := logging.NewLogger("lldpd", "LLDP", true)
	if err != nil {
		fmt.Println("Failed to start the logger. Nothing will be logged...")
	}
	logger.Info("Started the logger successfully.")

	logger.Info("Starting LLDP server....")
	// Create lldp server handler
	lldpSvr := lldpServer.LLDPNewServer(logger)
	// Until Server is connected to clients do not start with RPC
	lldpSvr.LLDPStartServer(*paramsDir)

	// Start keepalive routine
	go keepalive.InitKeepAlive("lldpd", fileName)

	// Create lldp rpc handler
	lldpHdl := lldpRpc.LLDPNewHandler(lldpSvr, logger)
	logger.Info("Starting LLDP RPC listener....")
	err = lldpRpc.LLDPRPCStartServer(logger, lldpHdl, *paramsDir)
	if err != nil {
		logger.Err(fmt.Sprintln("Cannot start lldp server", err))
		return
	}
}
