package models

import "time"

type CreateCommand struct {
	LocalFileName   string
	HydfsFileName   string
	VmId            int
	TimeStamp       time.Time
	ReplicaServers  []int
	IsCreate        bool
	FileData        []byte
	IsStreamResults bool
}

type CommandResponse struct {
	IsSuccess bool
	Message   string
}

type MultiAppendCommand struct {
	HydfsFileName string
	Vms           []int
	LocalFiles    []string
}

type MergeCommand struct {
	HydfsFileName string
	Vms           []int
}
