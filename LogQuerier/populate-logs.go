package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

// Log levels and sample log messages
var logLevels = []string{"INFO", "ERROR", "WARN"}
var logMessages = []string{
	"Starting server...",
	"Server connection failed",
	"Server connected successfully",
	"New client connected",
	"Timed out: Faulty connection",
	"Server is disconnected",
}

var illinoisMessage = "Illinois network detected"

func generateRandomLogMessage() string {
	// Randomizing the log messages and levels
	level := logLevels[rand.Intn(len(logLevels))]
	message := logMessages[rand.Intn(len(logMessages))]
	return fmt.Sprintf("%s %s %s", level, time.Now().Format("2024-09-07 15:04:05"), message)
}

func generateIllinoisLogMessage() string {
	// Randomize the log level but always use the Illinois message
	level := logLevels[rand.Intn(len(logLevels))]
	return fmt.Sprintf("%s %s %s", level, time.Now().Format("2024-09-07 15:04:05"), illinoisMessage)
}

func main() {
	fileName := "./cs425/server/test.log"
	// Open the log file in trunc mode, create it if it doesn't exist
	file, err := os.OpenFile(fileName,os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer file.Close()

	// Set the logger to write to the log file
	logger := log.New(file, "", 0)

	// Seed the random generator
	rand.Seed(time.Now().UnixNano())

	// Counter for logs containing "Illinois"
	illinoisCount := 0
	totalLogs := 300 // total number of logs to generate (you can adjust this)

	for i := 0; i < totalLogs; i++ {
		// If we have fewer than 10 Illinois messages, randomly decide to log an Illinois message
		var logMessage string
		if illinoisCount < 10 && rand.Float32() < 0.1 { // 10% chance for an Illinois message
			logMessage = generateIllinoisLogMessage()
			illinoisCount++
		} else {
			logMessage = generateRandomLogMessage() // Log a random message
		}

		// Write the log message to the file
		logger.Println(logMessage)
	}

	// Ensure exactly 10 Illinois logs are printed by adding any missing ones
	for illinoisCount < 10 {
		logger.Println(generateIllinoisLogMessage())
		illinoisCount++
	}
}
