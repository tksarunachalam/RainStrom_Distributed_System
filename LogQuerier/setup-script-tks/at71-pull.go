package main

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

var vms = []string{"fa24-cs425-7801.cs.illinois.edu", "fa24-cs425-7802.cs.illinois.edu", "fa24-cs425-7803.cs.illinois.edu", "fa24-cs425-7804.cs.illinois.edu", "fa24-cs425-7805.cs.illinois.edu", "fa24-cs425-7806.cs.illinois.edu", "fa24-cs425-7807.cs.illinois.edu", "fa24-cs425-7808.cs.illinois.edu", "fa24-cs425-7809.cs.illinois.edu", "fa24-cs425-7810.cs.illinois.edu"}

// Command to run populate-logs.go on a VM

func pullCode() {
	var wg sync.WaitGroup
	wg.Add(len(vms))

	for _, vm := range vms {
		go func(vm string) {
			defer wg.Done()
			// SSH into each VM and run the populate command
			cmd := exec.Command("ssh", fmt.Sprintf("at71@%s", vm), "cd cs425 && git pull https://at71:@gitlab.engr.illinois.edu/muktaj2/cs425.git")

			op, err := cmd.CombinedOutput()
			fmt.Println(string(op))
			if err != nil {
				log.Printf("Failed to do git pull %s: %v", vm, err)
			} else {
				log.Printf("Git pull completed on %s", vm)
			}
		}(vm)
	}

	// Wait for all VM log populations to complete
	wg.Wait()
	log.Println("All git pulls are completed.")
}

func main() {
	pullCode()
}
