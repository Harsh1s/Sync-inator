package main

import (
	"cse224/proj5/pkg/syncinator"
	"flag"
	"io"
	"log"
)

func main() {
	serverId := flag.Int64("i", -1, "(required) Server ID")
	configFile := flag.String("f", "", "(required) Config file, absolute path")
	debug := flag.Bool("d", false, "Output log statements")
	flag.Parse()

	config := syncinator.LoadRaftConfigFile(*configFile)

	// Disable log outputs if debug flag is missing
	if !(*debug) {
		log.SetFlags(0)
		log.SetOutput(io.Discard)
	}

	log.Fatal(startServer(*serverId, config))
}

func startServer(id int64, config syncinator.RaftConfig) error {
	raftServer, err := syncinator.NewRaftServer(id, config)
	if err != nil {
		log.Fatal("Error creating servers")
	}
	return syncinator.ServeRaftServer(raftServer)
}
