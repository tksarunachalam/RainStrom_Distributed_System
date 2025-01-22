package models

import (
	"encoding/json"
	"log"
	"time"
)

type FileWMetaData struct {
	FileName     string
	ClientPeerId int
	TimeStamp    time.Time
	RequestType  string
}

type AolEntry struct {
	FileData     []byte
	FileMetadata FileWMetaData
}

func GetAolEntry(fileData []byte, fileMetaData FileWMetaData) AolEntry {

	aolEntry := AolEntry{
		FileData:     fileData,
		FileMetadata: fileMetaData,
	}
	return aolEntry
}

func ConvertAolEntryToBytes(aolEntry AolEntry) []byte {

	aolEntryBytes, err := json.Marshal(aolEntry)
	if err != nil {
		log.Println("error marshelling AolEntry to bytes", err)
		return nil
	}
	return aolEntryBytes
}

func ConvertBytesToAolEntry(data []byte) (AolEntry, error) {
	var aolEntry AolEntry

	err := json.Unmarshal(data, &aolEntry)
	if err != nil {
		log.Println("Error unmarshalling bytes to AolEntry:", err)
		return AolEntry{}, err
	}

	return aolEntry, nil
}
