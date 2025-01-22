package appendLogs

import (
	"cs425/HyDfs/models"
	"cs425/HyDfs/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/arriqaaq/aol" //credits - append only logs for handling concurrent writes
)

// create a global Mapper
var (
	Mapper      = make(map[string]*models.AoLMetaDataMap)
	mapperMutex sync.Mutex
	logMutex    sync.Mutex
)

func init() {

	// Initialize the Mapper with default entries if required
	Mapper = make(map[string]*models.AoLMetaDataMap)

}

func WriteLog(aolEntry models.AolEntry) {
	//call func MapToAOLLog(fileMetadata) and use the file returned here and open it below
	logFilePath, _, err := MapToAOLLog(aolEntry.FileMetadata)
	if err != nil {
		fmt.Println("Error mapping file:", err)
		return
	}
	var DefaultOptions = &aol.Options{
		NoSync:      false,    // Fsync after every write
		SegmentSize: 20971520, // 20 MB log segment files.
		DirPerms:    0750,
		FilePerms:   0640,
	}
	logMutex.Lock()
	defer logMutex.Unlock()
	// Open the mapped log file
	alog, err := aol.Open(logFilePath, DefaultOptions)
	if err != nil {
		fmt.Println("Error opening log at path: :", logFilePath, err)
		return
	}
	defer alog.Close()

	log.Println("wrting to aol - once with data ", string(aolEntry.FileData))
	err = alog.Write(models.ConvertAolEntryToBytes(aolEntry))
	//set is Read to false as soon as write
	// Mapper[mapperKey].IsRead = false
	if err != nil {
		fmt.Println("Error writing to log:", err)
		log.Println("write error", err)
	}

}

// a func MapToAOLLog(fileMetadata) to check if fileMetadata.fileName is present in teh Mapper then return the log file path for it else create a key and create a new log file for it in log file
func MapToAOLLog(fileMetadata models.FileWMetaData) (string, string, error) {
	// mapperMutex.Lock()
	// defer mapperMutex.Unlock()

	// Get just the base file name from the full path
	fileName := filepath.Base(fileMetadata.FileName)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]
	logFilePath, exists := Mapper[fileNameWithoutExt]

	if exists {
		// Return existing log file path if it exists
		return logFilePath.LogPath, fileNameWithoutExt, nil
	}

	// Generate a new log file path if the file name is not present in Mapper
	newLogFilePath := fmt.Sprintf("%s/%s", utils.AOL_PATH, fileNameWithoutExt)
	// newLogFilePath := fmt.Sprintf("HyDfs/Files/Logs/%s", fileNameWithoutExt)

	// Update Mapper with the new entry

	Mapper[fileNameWithoutExt] = &models.AoLMetaDataMap{
		LogPath:      newLogFilePath,
		IsRead:       false,
		IsReplicated: false,
	}
	log.Println("Created aol log file. key is ", fileNameWithoutExt, Mapper[fileNameWithoutExt])

	return newLogFilePath, fileNameWithoutExt, nil
}

func ReadAllLogs() {
	logsDir := utils.AOL_PATH
	log.Println("in read all")

	// Read the directory contents
	files, err := os.ReadDir(logsDir)
	if err != nil {
		fmt.Println("Error reading logs directory:", err)
		return
	}
	if len(files) == 0 {
		log.Println("No log files found in the directory.")
		return
	}

	// Iterate over the files and read each log file
	for _, file := range files {

		log.Println("Going inside read log", file.Name())
		ReadLog(file.Name()) // Pass the log file name to the ReadLog function

	}
}

func ReadLog(logFileName string) {
	logFilePath := fmt.Sprintf("%s/%s", utils.AOL_PATH, logFileName)

	_, exists := Mapper[logFileName]
	if !exists {
		return
	}
	log.Println("Mapper value for logfile name", logFileName, Mapper[logFileName])
	// if Mapper[logFileName].IsRead {
	// 	//delete the file
	// }

	// Lock the log for reading
	logMutex.Lock()
	defer logMutex.Unlock()

	alog, err := aol.Open(logFilePath, nil)
	if err != nil {
		fmt.Println("Error opening log:", err)
		return
	}
	defer alog.Close()

	// Iterate over all segments in the AOL log
	totalSegments := alog.Segments()
	for segmentIndex := 0; segmentIndex < totalSegments; segmentIndex++ {
		entryIndex := 0
		for {
			// Read each entry in the current segment
			data, err := alog.Read(uint64(segmentIndex+1), uint64(entryIndex))
			if err == aol.ErrEOF {
				break // End of segment
			} else if err != nil {
				fmt.Println("Error reading entry:", err)
				return
			}

			// Convert the data back to an AOLEntry to get the destination file name
			aolEntry, err := models.ConvertBytesToAolEntry(data)
			if err != nil {
				fmt.Println("Error converting entry:", err)
				return
			}
			if aolEntry.FileMetadata.RequestType == "REPLICATE_Transfer_TransferFileType" {
				// check if the file is present
				//if present just continue
				log.Println("failure replica aol write", aolEntry.FileMetadata.FileName)
				if _, err := os.Stat(aolEntry.FileMetadata.FileName); err == nil {
					// File is present, continue to the next iteration
					entryIndex++
					continue
				}
			}

			// Open the target file in append mode
			filePath := fmt.Sprintf("%s", aolEntry.FileMetadata.FileName)
			file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Println("Error opening file:", err)
				return
			}

			// Write the entry data to the target file
			_, err = file.Write(aolEntry.FileData)
			if err != nil {
				fmt.Println("Error writing to file:", err)
			}
			file.Close() // Close the file after writing each entry

			entryIndex++

		}
	}
	Mapper[logFileName].IsRead = true

	// Clear the log file after reading all entries

	if Mapper[logFileName].IsRead {
		err := filepath.Walk(logFilePath, utils.DeleteFilesInDir)
		if err != nil {
			fmt.Println("Error clearing log directory:", err)
		}
		log.Println("Deleting read and replicated log file:", logFilePath)
		log.Println("the Mapper value", Mapper[logFileName])

	} else {
		log.Println("Log file not deleted; IsRead or IsReplicated is false for:", logFilePath)
		Mapper[logFileName].IsRead = false
	}

}

func MergeAoLOnRead(hyfdsFileName string) {

	fileName := filepath.Base(hyfdsFileName)
	fileNameWithoutExt := fileName[:len(fileName)-len(filepath.Ext(fileName))]

	mapValue, exists := Mapper[fileNameWithoutExt]
	if exists && !mapValue.IsRead {
		//Aol needs to be merged before response
		ReadLog(fileNameWithoutExt)
		log.Println("Merged Aol for file:", fileNameWithoutExt)
	}
}
