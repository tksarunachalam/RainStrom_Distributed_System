package stream

import (
	"crypto/sha1"
	"cs425/HyDfs/utils"
	"cs425/Stream/models"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func ExecuteTasks(command string, input string, dir string, arg []string) ([]byte, error) {

	//go to the directory of the executable
	//log current directory
	// log.Println("Current directory: ", dir)
	arg = append(arg, input)
	fmt.Println("arg to cmd", arg)

	cmd := exec.Command(command, arg...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing command: %v", err)
		return nil, err
	}
	return out, nil

}

func ConvertOutputToTuple(output []byte) models.Output {

	var outputTuple models.Output
	log.Println("converitng output to tuple", string(output))
	err := json.Unmarshal(output, &outputTuple)
	if err != nil {
		log.Println("Output: ", string(output))
		log.Printf("Error unmarshalling output: %v", err)
	}
	return outputTuple
}

func StartTasks() {
	time.Sleep(STAGE_1_TASK_WAIT * time.Second)
}

func clearFiles() {
	// Path to files
	aolFilePath := "Stream/aol.txt"
	outputFilePath := utils.HDFS_FILE_PATH + "/output.txt" // Recheck path if needed

	// Clear the content of aol.txt (truncate or create the file)
	err := os.WriteFile(aolFilePath, []byte{}, 0644)
	if err != nil {
		fmt.Printf("Failed to clear file %s: %v\n", aolFilePath, err)
	} else {
		fmt.Printf("Successfully cleared content of %s\n", aolFilePath)
	}

	// Delete output.txt
	err = os.Remove(outputFilePath)
	if err != nil {
		fmt.Printf("Failed to delete file %s: %v\n", outputFilePath, err)
	} else {
		fmt.Printf("Successfully deleted %s\n", outputFilePath)
	}

	// Delete output.txt
	err = os.Remove(utils.FILE_BASE_PATH + "/output.txt")
	if err != nil {
		fmt.Printf("Failed to delete local output file %s: %v\n", utils.FILE_BASE_PATH+"/output.txt", err)
	} else {
		fmt.Printf("Successfully deleted local output %s\n", utils.FILE_BASE_PATH+"/output.txt")
	}
}

func createTupleId(jobId string, TaskId int, sequenceNumber int) string {

	input := fmt.Sprintf("%s-%d-%d", jobId, string(TaskId), sequenceNumber)

	// Apply consistent hashing using SHA-1
	hash := sha1.New()
	hash.Write([]byte(input))
	hashBytes := hash.Sum(nil)

	// Convert the hash to a hexadecimal string
	tupleId := fmt.Sprintf("%x", hashBytes)

	return tupleId
}
