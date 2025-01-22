package cli

import (
	"cs425/HyDfs/utils"
	m "cs425/Stream/models"
	"log"
	"strconv"
)

func createRainRequest(op1 string, op2 string, srcFile string, destFile string, pattern string) m.RainStromRequest {
	req := m.RainStromRequest{
		Op1Exe:        "./" + op1,
		Op2Exe:        "./" + op2,
		HyDFSSrcFile:  utils.HDFS_FILE_PATH + "/" + srcFile,
		HyDFSDestFile: utils.HDFS_FILE_PATH + "/" + destFile,
		NumTasks:      2,
		MetaData:      make(map[string]string),
	}
	lineCount, err := utils.ExtractLineCountFromFilename(srcFile)
	if err != nil {
		log.Println("Error extracting line count from file")
		req.MetaData["lineCount"] = "1000"
	} else {
		req.MetaData["lineCount"] = strconv.Itoa(lineCount)
	}
	req.MetaData["pattern"] = pattern
	return req

}
