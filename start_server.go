// package main

// import (
// 	"fmt"
// 	"log"
// 	"os/exec"
// 	"strconv"
// 	"sync"
// )

// var vms = []string{"fa24-cs425-7801.cs.illinois.edu", "fa24-cs425-7802.cs.illinois.edu", "fa24-cs425-7803.cs.illinois.edu", "fa24-cs425-7804.cs.illinois.edu", "fa24-cs425-7805.cs.illinois.edu", "fa24-cs425-7806.cs.illinois.edu", "fa24-cs425-7807.cs.illinois.edu", "fa24-cs425-7808.cs.illinois.edu", "fa24-cs425-7809.cs.illinois.edu", "fa24-cs425-7810.cs.illinois.edu"}

// // Command to run populate-logs.go on a VM

// func StartServerOnVMs() {
// 	var wg sync.WaitGroup
// 	wg.Add(len(vms))

// 	for index, vm := range vms {
// 		go func(index int, vm string) {
// 			defer wg.Done()
// 			// SSH into each VM and run the populate command
// 			server_num := "server" + strconv.Itoa(index+1)
// 			fmt.Println(string(server_num))
// 			cmd1 := " cd cs425/server && go run server.go " + server_num
// 			cmd := exec.Command("ssh", fmt.Sprintf("muktaj2@%s", vm), cmd1)

// 			output, err := cmd.CombinedOutput()
// 			fmt.Println(string(output))
// 			if err != nil {
// 				log.Printf("Failed to start server %s: %v", vm, err)
// 			} else {
// 				log.Printf("start server completed on %s", vm)
// 			}
// 		}(index, vm)
// 	}

// 	// Wait for all VM log populations to complete
// 	wg.Wait()
// 	log.Println("All Server starts are completed.")
// }

// func main1() {
// 	StartServerOnVMs()
// }
