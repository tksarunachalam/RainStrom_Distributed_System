package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Output struct format
type Output struct {
	Key   string `json:"Key"`
	Value int    `json:"Value"`
}

func main() {
	// Check if the required arguments are provided
	if len(os.Args) < 3 {
		fmt.Println("Usage: ./program <pattern> <key:value>")
		return
	}

	// Extract pattern and input tuple from command-line arguments
	pattern := os.Args[1]
	tuple := os.Args[2]

	// Compile the provided pattern into a regular expression
	regex, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Printf("Invalid pattern: %s\n", err)
		return
	}

	// Parse the input tuple into key and value
	tupleParts := strings.SplitN(tuple, ":", 2)
	if len(tupleParts) != 2 {
		fmt.Printf("Invalid tuple format: %s\n", tuple)
		return
	}
	// key, value := tupleParts[0]
	value := strings.Trim(tupleParts[1], `'`) // Remove surrounding quotes from value, if any
	// Check if the value contains the provided pattern
	if regex.MatchString(value) {
		output := Output{
			Key:   value,
			Value: 1, // Example: using length of value as an integer representation
		}

		// Marshal output to JSON format
		jsonOutput, err := json.Marshal(output)
		if err != nil {
			fmt.Printf("Error marshaling output: %s\n", err)
			return
		}
		fmt.Println(string(jsonOutput))
	} else {
		// nullOutput := Output{
		// 	Key:   "",
		// 	Value: 0,
		// }
		// jsonOutput, _ := json.Marshal(nullOutput)
		// fmt.Println(string(jsonOutput))
		fmt.Println("")
	}
}
