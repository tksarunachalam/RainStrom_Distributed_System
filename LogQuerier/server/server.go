package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"

	pb "cs425/LogQuerier/log"

	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedQueryServiceServer
}

func (s *server) ExecuteGrep(ctx context.Context, req *pb.GrepRequest) (*pb.GrepResponse, error) {

	flag := req.Flag
	pattern := req.Pattern
	fileName := req.FileName

	args := append(flag, pattern, fileName)

	cmd := exec.Command("grep", args...)
	output, err := cmd.CombinedOutput()
	//prints the op of grep on the server
	if !checkIfFilePresent(fileName) {
		return nil, errors.New("log file not present on server")
	}
	if err != nil && len(output) != 0 {
		fmt.Print("Error in executing command: ", err)
		return nil, err
	}
	if output == nil || len(output) == 0 {
		fmt.Println("no matching patters found")
		return nil, nil
	}
	return &pb.GrepResponse{Response: string(output)}, nil

}

func checkIfFilePresent(fileName string) bool {
	_, err := os.Open(fileName)
	if os.IsNotExist(err) {
		log.Println(fileName + "not present on")
		return false
	}
	return true
}

func main() {

	//1st is the server id
	//  go run server.go server1

	serverId := os.Args[1]
	address, err := getAddress(serverId)
	if err != nil {
		log.Println("Error getting the server details from server.conf", err)
		return
	}

	runServer(address, serverId)

}

func getAddress(serverId string) (string, error) {

	config, err := os.ReadFile("../client/servers.config")
	if err != nil {
		log.Fatal("config file not present or error opening file", err)
	}
	servers := strings.Split(string(config), "\n")

	numStr := strings.TrimPrefix(serverId, "server")
	serverNo, err := strconv.Atoi(numStr)
	if err != nil {
		log.Fatalln("Error reading server Id mentioned in argument:", err)
		return "", err
	}
	fmt.Println("Server Address", servers[serverNo-1])
	return servers[serverNo-1], nil
}

func runServer(address string, name string) {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer(
		grpc.MaxSendMsgSize(60*1024*1024), // Set to 10MB, adjust as necessary
		grpc.MaxRecvMsgSize(60*1024*1024), // Increase the maximum receive size as well
	)
	pb.RegisterQueryServiceServer(s, &server{})
	log.Printf("%s listening at %v", name, lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
