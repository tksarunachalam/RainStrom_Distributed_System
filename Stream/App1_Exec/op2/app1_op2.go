package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Input represents the input JSON structure
type Input struct {
	Key   string `json:"Key"`
	Value int    `json:"Value"`
}

// Output represents the output JSON structure
type Output struct {
	Key   string `json:"Key"`
	Value int    `json:"Value"`
}

func main() {
	// Check if the required arguments are provided
	// fmt.Println("all the args", os.Args)
	if len(os.Args) < 2 {
		fmt.Println("Usage: ./program '{\"Key\":\"<csv_line>\", \"Value\":<int>}'")
		return
	}

	// Parse the input JSON string from command-line argument
	inputJSON := os.Args[1]
	var input Input
	// fmt.Println("in exec ip", inputJSON)
	err := json.Unmarshal([]byte(inputJSON), &input)
	if err != nil {
		fmt.Printf("Error parsing input JSON: %s\n", err)
		return
	}
	// tuple := os.Args[1]
	// tupleParts := strings.SplitN(tuple, ":", 2)
	// if len(tupleParts) != 2 {
	// 	fmt.Printf("Invalid tuple format: %s\n", tuple)
	// 	return
	// }

	// Extract the CSV line from Key
	csvLine := input.Key

	// Split the CSV line by commas
	csvParts := strings.Split(csvLine, ",")
	if len(csvParts) < 4 {
		fmt.Println("Invalid CSV format: not enough commas")
		return
	}

	// Extract the value between the 2nd and 3rd commas
	extractedKey := strings.TrimSpace(csvParts[3] + "," + csvParts[4])

	// Create the output structure
	output := Output{
		Key:   extractedKey,
		Value: 1,
	}

	// Marshal the output to JSON format
	outputJSON, err := json.Marshal(output)
	if err != nil {
		fmt.Printf("Error marshaling output JSON: %s\n", err)
		return
	}

	// Print the output JSON
	fmt.Println(string(outputJSON))
}
