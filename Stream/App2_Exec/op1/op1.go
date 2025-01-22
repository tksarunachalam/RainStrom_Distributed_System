package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Output represents the key-value format for op1
type Output struct {
	Key   string `json:"Key"`
	Value int    `json:"Value"`
}

func main() {
	// Validate input arguments
	if len(os.Args) != 3 {
		//example: ././app2-op1 "Punched Telespar" 'key1:-9828662.01808626,4879483.90715145,3,No Outlet,"30"" X 30""", ,Punched Telespar,2010,Warning, ,W14-2,Champaign,3,,AERIAL,E,,3,'
		fmt.Println("Usage: ./app2-op1 <X> <content>")
		return
	}

	// Extract the inputs
	//fmt.Println("All args: ", os.Args)
	signPostType := strings.TrimSpace(os.Args[1]) // <X> (Sign Post to match)
	inputLine := strings.TrimSpace(os.Args[2])    // <content>
	//fmt.Println("Input Line: ", inputLine)

	// Split into key and line content using the colon delimiter
	parts := strings.SplitN(inputLine, ":", 2)
	if len(parts) != 2 {
		fmt.Println("Malformed input, expected a key:value format")
		return
	}
	//fmt.Println("Parts: ", parts)
	//key := strings.TrimSpace(parts[0])
	lineContent := strings.TrimSpace(parts[1])
	//fmt.Println("line content: ", lineContent)

	// Split lineContent into fields (comma-separated)
	fields := strings.Split(lineContent, ",")
	//fmt.Println("Fields: ", fields)
	if len(fields) < 9 {
		fmt.Println("Malformed line content, expected at least 9 fields")
		return
	}

	// Extract the Sign_Post (7th field, index 6) and Category (9th field, index 8)
	signPost := strings.TrimSpace(fields[7])
	category := strings.TrimSpace(fields[9])
	//fmt.Println("Signpost arg: ", signPostType, "SignPost input: ", signPost)
	// Check if the Sign_Post matches the provided Sign Post Type
	if signPost == signPostType {
		// Prepare output as key: category and value: 1
		output := Output{
			Key:   category,
			Value: 1,
		}

		//fmt.Println("Output tuple: ", output)

		// Convert the output to JSON
		jsonOutput, err := json.Marshal(output)
		if err != nil {
			fmt.Printf("Error marshaling output: %v\n", err)
			return
		}

		// Print the JSON output
		fmt.Println(string(jsonOutput))
	}
}
