package main

import (
	cli "cs425/CLI"
	config "cs425/Config"
	groupMembership "cs425/GroupMembership"
	hydfs "cs425/HyDfs"
	stream "cs425/Stream"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {

	_, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to read file: %v", err)
		os.Exit(-1)
	}
	// serverId is the vm no running. to be passed via terminal
	serverId := os.Args[1]
	servers := getAllServers()
	serverid, _ := strconv.Atoi(serverId)
	ms := groupMembership.InitProcess()
	go ms.RunProcess(servers, serverid)
	fmt.Println("ms :", ms)

	config.Conf.NodeId = serverid
	config.Conf.PeerId = config.GetServerIdInRing(config.Conf.Server[serverid-1])
	fmt.Println(config.Conf.Server)
	hy, err := hydfs.InitHydfsService(ms)
	// go hy.RunNode(config.Conf.Server[serverid-1])

	var leader *stream.Leader
	isLeader := false
	if serverid == groupMembership.IntroducerNodeId {
		leader = stream.InitLeaderService(hy)
		leader.Workers[serverid-1].IsAlive = true
		isLeader = true
		go stream.RunLeader(leader)

	}

	ws := stream.InitWorkerService(hy, serverid)
	fmt.Println("ws", ws)
	go ws.RunWorker(config.Conf.Server[serverid-1], isLeader, leader)
	// if err := ws.ConnectToWorkers(); err != nil {
	// 	log.Fatalf("Failed to connect to workers: %v", err)
	// }
	// defer ws.CloseConnections()

	time.Sleep(2 * time.Second)
	cli.CliUtility(serverid, ms, hy, leader, ws)

}

func getAllServers() []string {
	config, err := os.ReadFile("servers.config")
	if err != nil {
		log.Fatal("config file not present or error opening file", err)
	}
	return strings.Split(string(config), "\n")
}
