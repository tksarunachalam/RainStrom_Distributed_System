package stream

import (
	"context"
	config "cs425/Config"
	hydfs "cs425/HyDfs"
	"cs425/HyDfs/utils"
	m "cs425/Stream/models"
	pb "cs425/protos"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"
)

//TODO: Reschedule, send to worker, act on acks (clean up queues etc)

//hash partitioning - based of number of lines % num of partitions
//scheduling
//send the tasks to the respective worker servers
//check for fault tolerance and reschedule tasks

// Leader represents the leader node responsible for scheduling tasks and managing workers
type Leader struct {
	Workers       map[int]*m.Worker // Worker nodes indexed by their IDs
	TaskMap       map[int]*pb.Task  // Map of taskID to Task details
	TaskQueue     []*pb.Task        // Queue of unassigned tasks waiting for an available worker
	WorkerService *WorkerService    //Leader's worker service
	WorkerConns   map[int]*grpc.ClientConn
	CompletedTask []*pb.Task
	JobTaskMap    map[string][]*pb.Task
	assignedTasks []*pb.Task
	mu            sync.Mutex // Mutex for synchronization
	hy            *hydfs.HyDfsService
	failedNodes   []int
}

// NewLeader initializes a new leader with workers and job queue
func InitLeaderService(hy *hydfs.HyDfsService) *Leader {
	workers := make(map[int]*m.Worker)
	for i := 0; i < WorkerCount; i++ {
		workers[i] = &m.Worker{
			ID:        i,
			IsAlive:   false,
			TaskQueue: make([]*pb.Task, 0), // Each worker has a task queue with buffer size 10
		}
	}
	leader := &Leader{
		Workers:       workers,
		TaskMap:       make(map[int]*pb.Task),
		hy:            hy,
		WorkerConns:   make(map[int]*grpc.ClientConn),
		JobTaskMap:    make(map[string][]*pb.Task),
		assignedTasks: make([]*pb.Task, 0),
		CompletedTask: make([]*pb.Task, 0),
		failedNodes:   make([]int, 0),
	}

	return leader
}
func RunLeader(leader *Leader) {
	go leader.getFailedNodeSignal()
	go leader.getJoinedNodeSignal()
}

// Hash function for partitioning based on key (e.g., word or filename:linenumber)
func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))

	return int(h.Sum32())
}

// PartitionInputData partitions input data into contiguous chunks for Stage 1 (Split)
func (l *Leader) PartitionInputData(job *pb.Job) {
	fmt.Println("Partitioning input data for Stage 1...")

	// Number of lines in the file (can be obtained from metadata)
	var lineCount int
	if job.MetaData == nil || job.MetaData["lineCount"] == "" {
		job.MetaData = make(map[string]string)
		job.MetaData["lineCount"] = "1000" // Default value if not provided
		lineCount, _ = strconv.Atoi(job.MetaData["lineCount"])
	} else {
		lineCount, _ = strconv.Atoi(job.MetaData["lineCount"])
	}

	// Calculate number of lines per task (partition)
	linesPerTask := int(math.Ceil(float64(lineCount) / float64(job.NumTasks)))

	partitions := make([]*pb.Partition, job.NumTasks)
	job.OutputPartitions = map[int32]int32{}
	//initialize the stage tasks map for workers

	job.StageTasksWorkerMap = make(map[int32]*pb.PrevWorkerList)

	for i := 0; i < int(job.NumTasks); i++ {
		startLine := i * linesPerTask
		endLine := startLine + linesPerTask - 1

		if endLine >= lineCount {
			endLine = lineCount - 1 // Ensure we don't go beyond total number of lines
		}

		partitions[i] = &pb.Partition{
			Id:       int32(i),
			FileName: job.HydfsSrcFile,
			Start:    int32(startLine),
			End:      int32(endLine),
		}
		job.OutputPartitions[int32(i)] = -1
	}

	job.InputPartitions = partitions

	fmt.Println("Input data partitioned:", partitions)
}

// AssignTasks assigns tasks to workers based on the partitioning strategy for Stage 1 (Split)
func (l *Leader) AssignTasks(job *pb.Job) {
	fmt.Println("Assigning tasks to workers...")
	var prevWorkers []int32
	assigned := true
	for i := 0; i < int(job.NumTasks); i++ {

		// First, attempt to assign a new task to a worker
		l.mu.Lock()
		//If split task:
		task := &pb.Task{
			TaskId:    int32(len(l.TaskMap)) + 1,
			Partition: int32(i),
			TaskType:  pb.TaskType_TASK_1,
		}
		taskID := task.TaskId
		l.JobTaskMap[job.JobId] = append(l.JobTaskMap[job.JobId], task)
		//elase if for count task:
		l.mu.Unlock()

		workerFiles := utils.GetNodeIdsOfFile(config.GetFileHashId(job.HydfsSrcFile), l.hy.Ms.MemberShipList)

		// Try to assign the task to a preferred worker with data locality (i.e., has the file)
		if !l.assignToPreferredWorker(task, workerFiles) {
			// If no preferred workers are available, try assigning to any other available worker
			if !l.assignToAvailableWorker(task) {
				// If no workers are available, queue the task at the leader for later assignment
				l.mu.Lock()
				l.TaskQueue = append(l.TaskQueue, task)
				l.mu.Unlock()
				fmt.Printf("No available workers. Queuing Task %d.\n", task.TaskId)
				assigned = false
				continue
			}

		}
		l.mu.Lock()
		l.TaskMap[int(taskID)] = task
		// job.StageTasksWokerMap[taskID] = -1
		l.mu.Unlock()
		l.assignedTasks = append(l.assignedTasks, task)
		prevWorkers = append(prevWorkers, task.WorkerId)
	}

	if assigned && len(l.assignedTasks) > 0 {
		for i := 0; i < int(job.NumTasks); i++ {
			stage2Tasks := &pb.Task{
				TaskId:   int32(len(l.TaskMap) + 1),
				TaskType: pb.TaskType_TASK_2,
			}
			if !l.assignToAvailableWorker(stage2Tasks) {
				l.mu.Lock()
				l.TaskQueue = append(l.TaskQueue, stage2Tasks)
				assigned = false
				l.mu.Unlock()
				fmt.Printf("No available workers. Queuing Task %d.\n", stage2Tasks.TaskId)
			}

			// job.OutputPartitions[assignedTasks[i].TaskId] = stage2Tasks.WorkerId
			l.mu.Lock()
			l.TaskMap[int(stage2Tasks.TaskId)] = stage2Tasks
			job.OutputPartitions[int32(i)] = stage2Tasks.WorkerId

			if _, exists := job.StageTasksWorkerMap[stage2Tasks.WorkerId]; !exists {
				job.StageTasksWorkerMap[stage2Tasks.WorkerId] = &pb.PrevWorkerList{PrevWorkers: []int32{}}
			}
			if job.StageTasksWorkerMap[stage2Tasks.WorkerId].PrevWorkers == nil {
				job.StageTasksWorkerMap[stage2Tasks.WorkerId].PrevWorkers = []int32{}
			}

			// value is list of stage 1 workers
			job.StageTasksWorkerMap[stage2Tasks.WorkerId].PrevWorkers = prevWorkers
			// job.StageTasksMap[assignedTasks[i].TaskId] = stage2Tasks.TaskId
			fmt.Println("Receiver worker is: : Sender workers: ", stage2Tasks.WorkerId, job.StageTasksWorkerMap[stage2Tasks.WorkerId].PrevWorkers)
			l.JobTaskMap[job.JobId] = append(l.JobTaskMap[job.JobId], stage2Tasks)
			l.mu.Unlock()
			l.assignedTasks = append(l.assignedTasks, stage2Tasks)
		}
		//iterate through assignedTasks and send the tasks to the respective worker servers
		log.Println("Job stage tasks worker map after assignment", job.StageTasksWorkerMap)
		for _, task := range l.assignedTasks {
			task.ParentJob = job

			fmt.Println("Task : assigned to worker", task.TaskId, task.WorkerId)
			log.Println("Task : assigned to worker", task.TaskId, task.WorkerId)
			go l.SendTaskToWorker(task)
		}
	}

}

// SendTaskToWorker sends a task to the appropriate worker's task queue for execution.
func (l *Leader) SendTaskToWorker(task *pb.Task) {

	worderAddr := config.Conf.Server[task.WorkerId]
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, worderAddr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(60*1024*1024), // Set to 60MB, adjust as necessary
	))

	if err != nil {
		log.Printf("Failed to connect to worker %d at %s: %v", task.WorkerId, worderAddr, err)
		return
	}
	defer conn.Close()

	client := pb.NewRainStromServiceClient(conn)

	fmt.Print("worker id", task.WorkerId)
	// workerTask := pb.GetWorkerTask(task)
	fmt.Println("Sending tasks over grpc")
	response, err := client.SendTask(ctx, task)
	if err != nil {
		log.Println("Failed to send task %d to worker %d: %v", task.TaskId, task.WorkerId, err)
	}

	fmt.Printf("Received acknowledgment from Worker %d: %s\n", task.WorkerId, response)
	log.Printf("Received acknowledgment from Worker %d: %s\n", task.WorkerId, response)
}

// RescheduleSplitTask reschedules a failed/unassigned split task to another available worker.
func (l *Leader) RescheduleSplitTask(task *pb.Task) {
	//Then call his commented code for re-assigning the task
	// Lock the mutex to safely access shared resources
	failedWorkerId := task.WorkerId
	log.Println("In reschedule task: failed worker id", failedWorkerId)
	log.Println("the failed tasks is", task)
	log.Println("the failed tasks parent", task.ParentJob)
	log.Println("the failed tasks parent", task.ParentJob.StageTasksWorkerMap)

	//create a copy of task and store in failed tasks
	var prevSenderWorkers []int32
	if task.ParentJob != nil && task.ParentJob.StageTasksWorkerMap != nil {
		if task.ParentJob.StageTasksWorkerMap[task.WorkerId] != nil {
			prevSenderWorkers = task.ParentJob.StageTasksWorkerMap[task.WorkerId].PrevWorkers
		} else {
			log.Println("Worker ID not found in StageTasksWorkerMap")
		}
	} else {
		log.Println("ParentJob or StageTasksWorkerMap is nil")
	}

	task.WorkerId = -1 // Reset worker ID to indicate unassigned task

	// Identify workers with the file corresponding to the task's partition
	workerFiles := utils.GetNodeIdsOfFile(int(task.Partition), l.hy.Ms.MemberShipList)
	fmt.Println("WorkerFiles", workerFiles)
	assigned := true
	// Try to reassign the task to a worker with data locality
	if !l.assignToPreferredWorker(task, workerFiles) {
		fmt.Println("preferred worker if")
		// If no preferred workers are available, try assigning to any other available worker
		if !l.assignToAvailableWorker(task) {
			// If no workers are available, requeue the task for later reassignment
			l.TaskQueue = append(l.TaskQueue, task)
			assigned = false
			fmt.Printf("Failed to reschedule Task %d. Requeued.\n", task.TaskId)
		} else {
			fmt.Printf("Task %d reassigned to an available worker.\n", task.TaskId)
		}
	} else {
		fmt.Printf("Task %d reassigned to a preferred worker with data locality.\n", task.TaskId)
	}
	if task.TaskType == pb.TaskType_TASK_1 && assigned {

		go l.SendTaskToWorker(task)
		return
	}
	log.Println("Job stage tasks worker map before assignment for failed workers", task.ParentJob)

	if task.TaskType == pb.TaskType_TASK_2 && assigned {
		log.Println("updating the senders about stage 2 failure")
		if task.ParentJob != nil && task.ParentJob.StageTasksWorkerMap != nil {
			log.Println("the new rescheduled task", task)
			if task.ParentJob.StageTasksWorkerMap[failedWorkerId] != nil {
				task.ParentJob.StageTasksWorkerMap[failedWorkerId].PrevWorkers = prevSenderWorkers
				log.Println("Job stage tasks worker map after assignment for failed workers", task.ParentJob.StageTasksWorkerMap)
			} else {
				log.Println("Failed worker ID not found in StageTasksWorkerMap")
			}
		} else {
			log.Println("ParentJob or StageTasksWorkerMap is nil")
		}
		// task.ParentJob.StageTasksWorkerMap[failedWorkerId].PrevWorkers = prevSenderWorkers
		log.Println("Job stage tasks worker map after assignment for failed workers", task.ParentJob.StageTasksWorkerMap)
		l.updateSenderWorkersForFailedNode(task, int(failedWorkerId))
		// senderTasks := l.updateSenderTasksForFailedNode(task)
		//Multicast this information to all the workers. use ls.WorkerConns[workerId] and send the updated task
		// l.MulticastUpdatedTask(senderTasks)
	}

	if assigned {
		go l.SendTaskToWorker(task)
	}

}

// ReceiveJob receives an incoming job request from a source process and schedules it.
func (l *Leader) CreateJob(req m.RainStromRequest) {

	//nodes := utils.FileRouting(config.GetFileHashId(utils.HDFS_FILE_PATH+"/"+"output.txt"), l.hy.Ms.MemberShipList)

	// Clear the output.txt and aol.txt files
	filesToClear := []string{"Stream/aol.txt"} //recheck path
	for _, filePath := range filesToClear {
		err := os.WriteFile(filePath, []byte{}, 0644) // Truncate or create the file
		if err != nil {
			fmt.Printf("Failed to clear file %s: %v\n", filePath, err)
			return
		}
	}

	job := pb.Job{
		JobId:         m.GenerateJobID(),
		Op1Exe:        req.Op1Exe,
		Op2Exe:        req.Op2Exe,
		HydfsSrcFile:  req.HyDFSSrcFile,
		HydfsDestFile: req.HyDFSDestFile,
		NumTasks:      int32(req.NumTasks),
		SrcWorker:     int32(req.SrcWorkerId),
		MetaData:      req.MetaData,
	}
	l.TaskMap = make(map[int]*pb.Task)
	l.JobTaskMap = make(map[string][]*pb.Task)
	l.assignedTasks = make([]*pb.Task, 0)
	l.CompletedTask = make([]*pb.Task, 0)
	if l.Workers != nil {
		for i, worker := range l.Workers {
			if worker != nil {
				l.Workers[i].TaskQueue = make([]*pb.Task, 0)
			}
		}
	}
	l.assignedTasks = make([]*pb.Task, 0)
	l.CompletedTask = make([]*pb.Task, 0)
	l.TaskQueue = make([]*pb.Task, 0)

	fmt.Println("Created new job:", job)

	l.PartitionInputData(&job) // Partition input data for Stage 1 (Split).
	l.AssignTasks(&job)        // Assign tasks for Stage 1.

	go l.MonitorJobCompletion(job) // Monitor job completion asynchronously.
}

// MonitorJobCompletion monitors the completion of all tasks in a job and proceeds with further stages.
func (l *Leader) MonitorJobCompletion(job pb.Job) {
	fmt.Println("Monitoring Job Completion....")
	// time.Sleep(20 * time.Second)              //Simulate monitoring logic
	ticker := time.NewTicker(RETRY_TIMEOUT * time.Second) // Check every 5 seconds
	defer ticker.Stop()

	//Check if all tasks in the JobTaskMap are completed
	for {
		select {
		case <-ticker.C:
			log.Println("leader locking monitor job completion")
			l.mu.Lock()
			completed := false
			// defer l.mu.Unlock()

			if len(l.assignedTasks) != 0 {
				for i := 0; i < len(l.JobTaskMap[job.JobId]); {
					task := l.JobTaskMap[job.JobId][i]

					log.Println("Checking if task is completed", task.TaskId)
					if l.checkIfTaskCompleted(task) {
						log.Println("Task ", task.TaskId, "is completed")
						l.JobTaskMap[job.JobId] = append(l.JobTaskMap[job.JobId][:i], l.JobTaskMap[job.JobId][i+1:]...)

						//remove the completed task from the Job task map

					} else {
						i++
						completed = false
					}
				}
				if l.JobTaskMap != nil && len(l.JobTaskMap[job.JobId]) == 0 {
					completed = true
				}
				// l.mu.Unlock()
			}
			if completed {
				l.assignedTasks = make([]*pb.Task, 0)
				l.CompletedTask = make([]*pb.Task, 0)
				l.JobTaskMap[job.JobId] = make([]*pb.Task, 0)
				fmt.Println("All tasks in the job completed successfully.")
				l.mu.Unlock()
				log.Println("leader unlocking monitor job completion")

				return
				//Notify source worker that the job is completed
			}
			log.Println("Leader releasing the lock")
			l.mu.Unlock()
		}
	}

}

func (l *Leader) getFailedNodeSignal() {

	for {
		if failedNodeId, ok := <-l.hy.FailNodeSignalChan; ok {

			log.Println("Leader detected failed node: ", failedNodeId)
			l.Workers[failedNodeId-1].IsAlive = false
			//clear workers task queue
			l.Workers[failedNodeId-1].TaskQueue = make([]*pb.Task, 0)
			timeout := time.After(15 * time.Second)
			l.failedNodes = append(l.failedNodes, failedNodeId)
			// Wait for the timeout to complete before proceeding
			select {
			case <-timeout:
				// Proceed with handling the worker failure
				log.Println("Timeout reached, proceeding to handle worker failure.")
				log.Println("The list of failed workers is: ", l.failedNodes)
				for _, node := range l.failedNodes {
					go l.handleWorkerFailure(node)
					//failedNodes = append(failedNodes[:i], failedNodes[i+1:]...)
				}
				l.failedNodes = []int{}
			}

		}
		runtime.Gosched()
	}
}

func (l *Leader) getJoinedNodeSignal() {

	for {
		if joinedNodeId, ok := <-l.hy.JoinNodeSignalChan; ok {

			fmt.Println("Leader detected joined node: ", joinedNodeId)
			l.Workers[joinedNodeId-1].IsAlive = true
			fmt.Println(joinedNodeId, " marked as alive: ", l.Workers[joinedNodeId-1].IsAlive)
			// l.connectLeaderToWorker(joinedNodeId-1, )
		}
		runtime.Gosched()
	}
}

func (l *Leader) handleWorkerFailure(failedNodeId int) {
	//check if the node has failed, if it has failed then check what task it was assigned to and the re-assign that task to another apt worker

	//go leader.getFailedNodeSignal() in NewLeader() can give a list of failed nodes. For every failed node, reassign the tasks

	//Get the correct task that would have to be re-assigned from the failed node (can also be after timing out for an ack)

	// Set the worker is alive value to be false

	//print all keys and values of the task map
	l.PrintTaskMap()

	log.Println("Is alive in handle worker failure", l.Workers[failedNodeId-1], l.Workers[failedNodeId-1].IsAlive)
	//reschedule the tasks
	// Iterate through tasks and find those assigned to the failed worker

	for _, task := range l.TaskMap {
		if task == nil {
			continue
		}
		log.Println("Task worker id", task.WorkerId)
		log.Println("failed worker id", failedNodeId-1)
		if task.WorkerId == int32(failedNodeId)-1 {
			fmt.Printf("Task %d assigned to Worker %d needs reassignment.\n", task.TaskId, failedNodeId-1)
			log.Printf("Task %d assigned to Worker %d needs reassignment.\n", task.TaskId, failedNodeId-1)

			// Reschedule the task to another worker

			l.RescheduleSplitTask(task) // just for testing split task
			fmt.Printf("REASSIGNMENT:: Task : %d rescheduled at worker %d\n", task.TaskId, task.WorkerId)
			log.Printf("REASSIGNMENT:: Task : %d rescheduled at worker %d\n", task.TaskId, task.WorkerId)

			break
			//TODO: also add for count tasks
		}
	}

}

func (s *RainStromServer) SendNotification(ctx context.Context, req *pb.Ack) (*pb.Ack, error) {
	// Process the notification
	fmt.Println("In send notification()", req.TaskId)
	if req.NotificationType == pb.NotificationType_LEADER {
		fmt.Printf("Received notification: %s\n", req.Message, req.TaskId, req.ResponseType)

		stage1Task := s.Leader.TaskMap[int(req.TaskId)]
		fmt.Println("stage1Task", stage1Task)

		s.Leader.MarkTaskCompleted(req.TaskId)
		// s.Leader.CompletedTask = append(s.Leader.CompletedTask, stage1Task)
		//check if all tasks completed
		if s.Leader.checkIfAllSenderCompleted() {
			fmt.Println("All sender tasks completed")
			s.Leader.sendAckToReceiver()
			//clear the completed tasks
			// s.Leader.CompletedTask = make([]*pb.Task, 0)
			// s.Leader.assignedTasks = make([]*pb.Task, 0)

		}

	} else if req.NotificationType == pb.NotificationType_RECEIVER {
		taskIndex := -1
		fmt.Println("In receiver notification:", req.TaskId)
		for i, task := range s.Ws.Worker.TaskQueue {
			fmt.Println("In worker task queue")
			if task.TaskId == req.TaskId {
				taskIndex = i
				s.Ws.Worker.TaskQueue = append(s.Ws.Worker.TaskQueue[:taskIndex], s.Ws.Worker.TaskQueue[taskIndex+1:]...)
				fmt.Printf("TaskID %d successfully removed from the task queue for reciver\n", req.TaskId)
				break
			}
		}

		if taskIndex == -1 {
			fmt.Printf("TaskID %d not found in the task queue.\n", req.TaskId)
		}
	}

	return &pb.Ack{}, nil
	//
}

func (s *RainStromServer) TransferCommand(ctx context.Context, req *pb.Command) (*pb.Ack, error) {
	// Handle the received command
	log.Printf("Received Command: %+v\n", req)

	// Example: Deserialize the payload (assuming it's JSON)
	if req.CommandType == pb.CommandType_RainStromRequest {
		var data m.RainStromRequest
		if err := json.Unmarshal(req.Payload, &data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
		log.Printf("Deserialized Payload: %+v\n", data)

		// Implement the logic to transfer the command to other workers or leaders
		if s.Leader != nil {
			s.Leader.CreateJob(data)
		} else {
			log.Println("Leader is nil. Cannot process the command.")
		}
		return &pb.Ack{Message: "Command transferred successfully"}, nil
	}
	return &pb.Ack{Message: "Command not recognized"}, nil
}

func (l *Leader) PrintTaskMap() {

	for k, v := range l.TaskMap {
		log.Printf("TaskMap[%d] = %v\n", k, v)
	}
}
func (l *Leader) PrintWorkerTaskQueue() {

	for k, v := range l.Workers {
		for _, task := range v.TaskQueue {
			log.Printf("Worker[%d] = %v\n", k, task)
		}
	}
}
