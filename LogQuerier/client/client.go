package main

import (
	"context"
	pb "cs425/LogQuerier/log"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

var totalCount int = 0
var mapCount = make(map[int]int)

func main() {

	//read server IP from config file
	hosts := readConfig()

	//args passed in terminal
	query := os.Args[1:]
	pattern, flags := getUserQuery(query)

	var wg sync.WaitGroup
	start := time.Now() // Start measuring time
	for index, host := range hosts {
		wg.Add(1)
		go SendGrepRequest(index, host, pattern, flags, &wg)
	}
	wg.Wait()
	latency := time.Since(start)

	fmt.Println("Latency for the query", latency)

	if !slices.Contains(flags, "-c") {
		for key, value := range mapCount {
			fmt.Println("\nNo of matching lines in server", key+1, ":", value)
		}
		fmt.Println("\ntotal matched lines:", totalCount)
	}

}

func getUserQuery(query []string) (pattern string, flags []string) {

	//query: hello -n -f
	//query to words
	for _, word := range query {
		if strings.HasPrefix(word, "-") {
			flags = append(flags, word)
		} else {
			pattern = word
		}
	}
	return pattern, flags
}

func readConfig() []string {

	config, err := os.ReadFile("servers.config")
	if err != nil {
		log.Fatal("config file not present or error opening file", err)
	}
	servers := strings.Split(string(config), "\n")
	return servers

}

func SendGrepRequest(index int, host string, pattern string, flags []string, wg *sync.WaitGroup) {
	defer wg.Done()

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	connection, err := grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(60*1024*1024), // Set to 10MB, adjust as necessary
	))
	if err != nil {
		log.Printf("did not connect to %s, Error = %v", host, err)
		return
	}

	defer connection.Close()
	client := pb.NewQueryServiceClient(connection)

	//log file name
	fileName := "vm" + strconv.Itoa(index+1) + ".log"

	req := &pb.GrepRequest{Flag: flags, Pattern: pattern, FileName: fileName}
	resp, err := client.ExecuteGrep(context.Background(), req)
	if err != nil {
		log.Printf("could not connect to: %v server. Error:%v", index+1, err.Error())
		return
	}
	if len(resp.GetResponse()) == 0 {
		log.Printf("\nNo matching pattern found in server%v:", index+1)
		return
	}
	fmt.Printf("\nResponse from server%v:\n%v", index+1, resp.GetResponse())
	noOfLines := len(strings.Split(resp.GetResponse(), "\n")) - 1

	totalCount += noOfLines
	mapCount[index] = noOfLines

}
