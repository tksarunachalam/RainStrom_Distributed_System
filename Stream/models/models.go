package models

import (
	pb "cs425/protos"

	"github.com/google/uuid"
)

// Task represents a task assigned to a worker
// type Task struct {
// 	TaskID    int
// 	Partition int
// 	WorkerID  int
// 	Files     []string
// 	ParentJob Job
// 	TaskType  int
// }

// Job represents a job request sent to the leader
// GenerateJobID generates a unique job ID using UUID
func GenerateJobID() string {
	return uuid.New().String()
}

// type Job struct {
// 	JobID            string
// 	Op1Exe           string
// 	Op2Exe           string
// 	HyDFSSrcFile     string
// 	HyDFSDestFile    string
// 	NumTasks         int
// 	InputPartitions  []*Partition // Partitions of input data (for Stage 1)
// 	OutputPartitions []*Partition // Partitions of output data (for Stage 2)
// 	MetaData         map[string]string
// }

// Partition represents a partition of input or output data
// type Partition struct {
// 	ID       int
// 	FileName string
// 	Start    int // Start line number or block index
// 	End      int // End line number or block index
// }

// Worker represents a worker node in the system
type Worker struct {
	ID        int
	IsAlive   bool
	TaskQueue []*pb.Task
	MaxJobs   int
}

type FileClearRequestFromLeader struct {
	FilePath string
}

type RainStromRequest struct {
	Op1Exe        string
	Op2Exe        string
	HyDFSSrcFile  string
	HyDFSDestFile string
	NumTasks      int
	MetaData      map[string]string
	SrcWorkerId   int
}

func GetWorkerPartition(partition *pb.Partition) *pb.Partition {
	workerPartition := &pb.Partition{
		Id:       int32(partition.Id),
		FileName: partition.FileName,
		Start:    int32(partition.Start),
		End:      int32(partition.End),
	}

	return workerPartition
}

type Output struct {
	TupleId string
	Key     string
	Value   int
	TaskId  int
}

// func GetJobForWorker(job pb.Job) *pb.Job {
// 	workerJob := &pb.Job{
// 		JobId:         job.JobId,
// 		Op1Exe:        job.Op1Exe,
// 		Op2Exe:        job.Op2Exe,
// 		HydfsSrcFile:  job.HydfsSrcFile,
// 		HydfsDestFile: job.HydfsDestFile,
// 		NumTasks:      int32(job.NumTasks),
// 		MetaData:      job.MetaData,
// 	}
// 	for _, partition := range job.InputPartitions {
// 		workerJob.InputPartitions = append(workerJob.InputPartitions, GetWorkerPartition(partition))
// 	}
// 	for _, partition := range job.OutputPartitions {
// 		workerJob.OutputPartitions = append(workerJob.OutputPartitions, GetWorkerPartition(partition))
// 	}

// 	return workerJob
// }

// func GetWorkerTask(task *Task) *pb.Task {
// 	workerTask := &pb.Task{
// 		TaskId:    int32(task.TaskID),
// 		Partition: int32(task.Partition),
// 		WorkerId:  int32(task.WorkerID),
// 		Files:     task.Files,
// 	}
// 	workerJob := GetJobForWorker(task.ParentJob)

// 	workerTask.ParentJob = workerJob

// 	return workerTask
// }
