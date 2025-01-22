package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	pb "cs425/LogQuerier/log"

	"google.golang.org/grpc"
)

var vms = []string{"fa24-cs425-7801.cs.illinois.edu", "fa24-cs425-7802.cs.illinois.edu", "fa24-cs425-7803.cs.illinois.edu", "fa24-cs425-7804.cs.illinois.edu", "fa24-cs425-7805.cs.illinois.edu", "fa24-cs425-7806.cs.illinois.edu", "fa24-cs425-7807.cs.illinois.edu", "fa24-cs425-7808.cs.illinois.edu", "fa24-cs425-7809.cs.illinois.edu", "fa24-cs425-7810.cs.illinois.edu"}

// Command to run populate-logs.go on a VM
var populateCommand = "go run ./cs425/populate-logs.go"

func populateLogsOnVMs() {
	var wg sync.WaitGroup
	wg.Add(len(vms))

	for _, vm := range vms {
		go func(vm string) {
			defer wg.Done()
			// SSH into each VM and run the populate command
			cmd := exec.Command("ssh", fmt.Sprintf("muktaj2@%s", vm), populateCommand)

			_, err := cmd.CombinedOutput()
			if err != nil {
				log.Printf("Failed to run log population on %s: %v", vm, err)
			} else {
				log.Printf("Log population completed on %s", vm)
			}
		}(vm)
	}

	// Wait for all VM log populations to complete
	wg.Wait()
	log.Println("All log populations are completed.")
}

func happyPathTest() {
	expectedResponse := "10"
	// List of VM addresses with gRPC ports
	hosts := readConfig()
	flags := []string{"-c"}
	pattern := "Illinois"
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fileName := "test" + ".log"

	for _, host := range hosts {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			// Connect to the VM's gRPC server
			connection, err := grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock())
			if err != nil {
				log.Printf("Failed to connect to %s, Error = %v", host, err)
				return
			}
			defer connection.Close()

			client := pb.NewQueryServiceClient(connection)
			req := &pb.GrepRequest{Flag: flags, Pattern: pattern, FileName: fileName}
			resp, err := client.ExecuteGrep(context.Background(), req)
			if err != nil {
				log.Printf("FAILED on host %s", host)
				return
			}

			if resp != nil {
				if strings.TrimSpace(resp.GetResponse()) == expectedResponse {
					log.Printf("PASSED on host %s", host)
				} else {
					log.Printf("FAILED on host %s", host)
				}
			}
		}(host)
	}

	wg.Wait()
}

func readConfig() []string {

	config, err := os.ReadFile("../client/servers.config")
	if err != nil {
		log.Fatal("config file not present or error opening file", err)
	}
	servers := strings.Split(string(config), "\n")
	return servers

}

func main() {
	populateLogsOnVMs()
	happyPathTest()
}
