package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Count represents the final output format
type Count struct {
	Key   string `json:"Key"`
	Value int    `json:"Value"`
}

func main() {
	// Check if the correct number of arguments are passed
	if len(os.Args) != 3 {
		fmt.Println("Usage: ./app2-op2 <initial_count> <{\"Key\": \"category\", \"Value\": 1}>")
		return
	}

	// Parse the second argument (key, value)
	keyValue := os.Args[2]
	// Expecting the format {"Key": "category", "Value": 1}
	keyValue = strings.Trim(keyValue, "{}") // Remove the surrounding curly braces
	parts := strings.Split(keyValue, ",")
	if len(parts) != 2 {
		fmt.Println("Invalid format. Expected format: {\"Key\": \"category\", \"Value\": 1}")
		return
	}

	// Extract key-value pair from input
	var key, valueStr string
	for _, part := range parts {
		if strings.HasPrefix(part, "\"Key\":") {
			key = strings.Trim(strings.Split(part, ":")[1], "\" ,")
		} else if strings.HasPrefix(part, "\"Value\":") {
			valueStr = strings.Trim(strings.Split(part, ":")[1], "\" ,")
		}
	}

	// Convert value to integer
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		fmt.Println("Invalid value. It should be an integer.")
		return
	}

	// Parse the first argument (initial_count)
	initialCountStr := os.Args[1]
	initialCount, err := strconv.Atoi(initialCountStr)
	if err != nil {
		fmt.Println("Invalid initial count. It should be an integer.")
		return
	}

	// Calculate the final count
	currentCount := initialCount + value

	// Create the output tuple
	output := Count{
		Key:   key,
		Value: currentCount,
	}

	// Marshal the output to JSON and print it
	jsonOutput, err := json.Marshal(output)
	if err != nil {
		fmt.Printf("Error marshaling output: %s\n", err)
		return
	}
	fmt.Println(string(jsonOutput))
}
