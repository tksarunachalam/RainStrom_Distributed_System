package utils

import (
	config "cs425/Config"
	"cs425/HyDfs/models"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type FilesInfo struct {
	FileName string
	FileId   int
}

func CreateNewFile(fileName string, fileContents []byte) {

	fmt.Println("Filename to search", fileName)
	if fileExists(fileName) {
		log.Println("File already present")
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filepath.Join(FILE_BASE_PATH, filename))
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		log.Println("file not present", err)
		return false
	}
	return false
}

func GetServerAddrListFromNodeId(nodeIds []int) []string {

	var addrList []string
	for i, nodeId := range nodeIds {
		if i == nodeId-1 {
			addrList = append(addrList, config.Conf.Server[i])
		}
	}
	return addrList
}

func ClearFilesOnJoin(dirPath string) {
	err := filepath.Walk(dirPath, DeleteFilesInDir)
	if err != nil {
		fmt.Println("Error clearing log directory:", err)
	}
}

func DeleteFilesInDir(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Println("Error accessing path:", err)
		return err
	}
	if !info.IsDir() {
		err = os.Remove(path)
		if err != nil {
			fmt.Println("Error deleting file:", err)
		}
	}
	return nil
}

func GetFilesInRange(startID, endID int) ([]string, error) {
	// Directory path
	dirPath := HDFS_FILE_PATH
	var filesInRange []string

	// Read the directory
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if !file.IsDir() {
			fileName := filepath.Join(dirPath, file.Name())
			// Get the hash ID for each file
			hashID := config.GetFileHashId(fileName)
			//sentFiles := alreadySentFiles[receiver]

			// Check if hashID is in range and if the file has already been sent
			if hashID >= startID && hashID <= endID {
				log.Println(" Inside else if block .File in range:", fileName)
				filesInRange = append(filesInRange, fileName)
				//alreadySentFiles[receiver] = append(sentFiles, fileName) // Mark this file as sent
			} else if hashID > endID && hashID > startID || hashID <= startID && hashID < endID {
				log.Println("End of ring. Inside else if block .File in range:", fileName)
				filesInRange = append(filesInRange, fileName)
				//alreadySentFiles[receiver] = append(sentFiles, fileName)
			}
		}
	}

	return filesInRange, nil
}

// Helper function to check if a file has already been sent
// func isFileSent(fileName string, sentFiles []string) bool {
// 	for _, sentFile := range sentFiles {
// 		if sentFile == fileName {
// 			return true // File has been sent
// 		}
// 	}
// 	return false
// }

func GetFilesUsingHash() []FilesInfo {
	files, err := ioutil.ReadDir(HDFS_FILE_PATH)
	if err != nil {
		fmt.Println("Error reading HDFS file path:", err)
		return nil
	}
	var fileInfos []FilesInfo
	for _, file := range files {
		if !file.IsDir() { // Only display files, not directories
			filePath := filepath.Join(HDFS_FILE_PATH, file.Name()) // Get the full file path
			fileID := config.GetFileHashId(filePath)               // Get the file ID
			fileInfos = append(fileInfos, FilesInfo{
				FileName: file.Name(),
				FileId:   fileID,
			})
		}
	}
	return fileInfos
}

func ReadFileFromLocal(command models.CreateCommand) ([]byte, error) {
	// Open the source HDFS file for reading
	sourceFile, err := os.Open(command.HydfsFileName)
	if err != nil {
		return nil, fmt.Errorf("error opening Hydfs file: %w", err)
	}
	defer sourceFile.Close()

	// Read the contents of the HDFS file
	content, err := ioutil.ReadAll(sourceFile)
	if err != nil {
		return nil, fmt.Errorf("error reading Hydfs file: %w", err)
	}

	// Open or create the local file for writing
	localFile, err := os.OpenFile(command.LocalFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening or creating local file: %w", err)
	}
	defer localFile.Close()

	// Write the contents to the local file
	_, err = localFile.Write(content)
	if err != nil {
		return nil, fmt.Errorf("error writing to local file: %w", err)
	}

	fmt.Printf("Successfully copied from %s to %s\n", command.HydfsFileName, command.LocalFileName)

	// Return the file contents
	return content, nil
}

func ExtractLineCountFromFilename(fileName string) (int, error) {
	// Split the filename by the underscore character
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	parts := strings.Split(baseName, "_")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid filename format: %s", fileName)
	}

	// Get the last part and convert it to an integer
	lineCountStr := parts[len(parts)-1]
	lineCount, err := strconv.Atoi(lineCountStr)
	if err != nil {
		return 0, fmt.Errorf("error converting line count to integer: %w", err)
	}

	return lineCount, nil
}
