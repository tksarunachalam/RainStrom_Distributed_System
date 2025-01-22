package utils

import (
	"log"
	"os"
	"strings"
)

func ReadConfig() []string {

	//TODO: read from a single location
	config, err := os.ReadFile("servers.config")
	if err != nil {
		log.Fatal("In utils config file not present or error opening file", err)
	}
	servers := strings.Split(string(config), "\n")
	return servers

}

func ConvertListToSet(list []int) []int {

	mySet := make(map[int]bool)
	for _, v := range list {
		mySet[v] = true
	}

	// Convert the set back to a list (if needed)
	uniqueList := []int{}
	for k := range mySet {
		uniqueList = append(uniqueList, k)
	}
	return uniqueList

}
