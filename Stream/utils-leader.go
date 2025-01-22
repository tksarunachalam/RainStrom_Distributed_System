package stream

import (
	"context"
	config "cs425/Config"
	groupMembership "cs425/GroupMembership"
	m "cs425/Stream/models"
	pb "cs425/protos"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
)

// assignToPreferredWorker tries to assign a task to a worker that has data locality (i.e., has the file)
func (l *Leader) assignToPreferredWorker(task *pb.Task, workerFiles []int) bool {
	for index, worker := range l.Workers {
		if worker.IsAlive && len(worker.TaskQueue) < 1 && containsFile(workerFiles, index) {
			log.Println("before lock")
			l.mu.Lock()
			log.Println("after lock")

			worker.TaskQueue = append(worker.TaskQueue, task)
			task.WorkerId = int32(worker.ID)
			//go l.SendTaskToWorker(task)
			l.mu.Unlock()
			log.Printf("Assigned Task %d to Worker %d (data locality).\n", task.TaskId, worker.ID)
			fmt.Println("Assigned to preferred worker", worker.ID)
			return true // Successfully assigned the task based on data locality
		}
	}
	return false // No preferred workers were available
}

// assignToAvailableWorker tries to assign a task to any other available worker without jobs in its queue
func (l *Leader) assignToAvailableWorker(task *pb.Task) bool {
	assigned := false
	for _, worker := range l.Workers {
		log.Println("Worker ID: and is alive, quee len", worker.ID, worker.IsAlive, len(worker.TaskQueue))
		if worker.IsAlive && len(worker.TaskQueue) == 0 { // Prefer idle workers with no Tasks assigned yet
			l.mu.Lock()
			worker.TaskQueue = append(worker.TaskQueue, task)
			task.WorkerId = int32(worker.ID)
			//go l.SendTaskToWorker(task)
			assigned = true
			fmt.Println("Assigned2 to availble worker", worker.ID)

			l.mu.Unlock()
			log.Printf("Assigned Task %d to Worker %d (no current jobs).\n", task.TaskId, worker.ID)
			return true // Successfully assigned the task to an idle worker
		}
	}
	//iterate through the workers again to assign to a worker with less jobs
	if !assigned {
		for _, worker := range l.Workers {
			if worker.IsAlive && len(worker.TaskQueue) < MAX_TASK_PER_NODE {
				l.mu.Lock()
				worker.TaskQueue = append(worker.TaskQueue, task)
				task.WorkerId = int32(worker.ID)
				fmt.Println("Assigned2 to availble worker", worker.ID)

				// go l.SendTaskToWorker(task)
				assigned = true
				l.mu.Unlock()

				log.Printf("Assigned Task %d to Worker %d (no current jobs).\n", task.TaskId, worker.ID)
				return true // Successfully assigned the task to an idle worker
			}
		}
	}

	return false // No available idle workers found
}

// CheckForAvailableWorkers checks if there are any available workers and assigns queued tasks if possible.
func (l *Leader) CheckForAvailableWorkers() {
	// l.mu.Lock()
	// defer l.mu.Unlock()

	// for i := 0; i < len(l.TaskQueue); i++ {
	// 	task := l.TaskQueue[i]

	// 	if l.assignToPreferredWorker(&task) || l.assignToAvailableWorker(&task) {
	// 		fmt.Printf("Assigned queued Task %d.\n", task.ID)
	// 		l.TaskQueue = append(l.TaskQueue[:i], l.TaskQueue[i+1:]...) // Remove from queue after assignment.
	// 		i--                                                         // Adjust index after removal.
	// 	}
	// }
}

// contains checks if a peer ID is in the list of routedFile
func containsFile(slice []int, item int) bool {
	for _, v := range slice {
		if v == item+1 {
			return true
		}
	}
	return false
}

// func GetWorkerTask(task m.Task) pb.Task {
// 	workerTask := pb.Task{
// 		TaskId:    int32(task.TaskID),
// 		Partition: int32(task.Partition),
// 		WorkerId:  int32(task.WorkerID),
// 		Files:     task.Files,
// 	}
// 	workerJob := m.GetJobForWorker(task.ParentJob)

// 	workerTask.ParentJob = workerJob

// 	return workerTask
// }

func (l *Leader) MulticastUpdatedTask(task *pb.Task) {
	log.Printf("Multicasting updated task to workers", l.WorkerConns)
	for workerId, conn := range l.WorkerConns {
		log.Printf("Multicasting updated task to worker %d", workerId)
		client := pb.NewRainStromServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := &pb.UpdateTaskRequest{
			Task: task,
		}

		_, err := client.UpdateTask(ctx, req)
		if err != nil {
			log.Printf("Failed to update task on worker %d: %v", workerId, err)
		} else {
			log.Printf("Updated task on worker %d", workerId)
		}
	}
}

func (l *Leader) getSenderWorkersForFailedNode(task *pb.Task, failedWorkerId int) []int32 {

	// var senderTasks []*pb.Task

	//Iterate through  the task.ParentJob.OutputPartitions
	log.Println("getSenderWorkersForFailedNode() Stage Tasks Worker map", task.ParentJob.StageTasksWorkerMap)
	for receiverWorker, senderWorkers := range task.ParentJob.StageTasksWorkerMap {
		log.Println("sendeer worker reciver worker : ", receiverWorker, int32(failedWorkerId), " is alive: ", l.Workers[int(receiverWorker)].IsAlive)
		if receiverWorker == int32(failedWorkerId) {
			//get the sender task
			// sender := task.ParentJob.StageTasksWorkerMap[index]
			log.Println("Sender task found for failed Task 2 Worker is:", senderWorkers)

			return senderWorkers.PrevWorkers
			// return l.TaskMap[int(sender)]
		}
	}

	return nil
}
func (l *Leader) getPartitionForFailedNode(failedTasks *pb.Task, failedWorkerId int) int {

	//Iterate through  the task.ParentJob.OutputPartitions
	fmt.Println("current task output partitions", failedTasks.ParentJob.OutputPartitions)
	for hash, receiverWorker := range failedTasks.ParentJob.OutputPartitions {
		fmt.Println("Receiver worker,failedWorker and hash", receiverWorker, failedWorkerId, hash)
		if receiverWorker == int32(failedWorkerId) {
			return int(hash)
		}
	}
	return -1
}

func (l *Leader) updateSenderWorkersForFailedNode(rescheduledTask *pb.Task, failedWorkerId int) {

	senderWokers := l.getSenderWorkersForFailedNode(rescheduledTask, failedWorkerId)
	fmt.Println("Sender workers for rescheduled tasks are", rescheduledTask.TaskId, senderWokers)
	log.Println("Sender workers for rescheduled tasks are", senderWokers)
	hashKey := l.getPartitionForFailedNode(rescheduledTask, failedWorkerId)
	fmt.Println("Hash key for failed Task 2 Worker is:", hashKey)
	for _, sender := range senderWokers {
		senderWorker := l.Workers[int(sender)]
		for index, senderTask := range senderWorker.TaskQueue {
			senderWorker.TaskQueue[index].ParentJob.OutputPartitions[int32(hashKey)] = rescheduledTask.WorkerId
			senderWorker.TaskQueue[index].ParentJob.StageTasksWorkerMap = rescheduledTask.ParentJob.StageTasksWorkerMap
			fmt.Println("Sender task updated for failed Task 2 Worker is:", senderWorker.TaskQueue[index].ParentJob.OutputPartitions)
			log.Println("Sender task updated for failed Task 2 Worker is:", senderWorker.TaskQueue[index].ParentJob.OutputPartitions)
			l.MulticastUpdatedTask(senderTask)
		}

	}

}
func (l *Leader) RemoveTaskFromWorkerQueue(workerId int, taskId int32) {
	l.mu.Lock()
	defer l.mu.Unlock()

	worker, ok := l.Workers[workerId]
	if !ok {
		log.Printf("Worker %d not found", workerId)
		return
	}

	for i, task := range worker.TaskQueue {
		if task.TaskId == taskId {
			// Remove the task from the TaskQueue
			worker.TaskQueue = append(worker.TaskQueue[:i], worker.TaskQueue[i+1:]...)
			fmt.Printf("Removed task %d from worker %d's TaskQueue\n", taskId, workerId)
			return
		}
	}

	log.Printf("Task %d not found in worker %d's TaskQueue", taskId, workerId)
}

func (l *Leader) checkIfTaskCompleted(task *pb.Task) bool {
	// Check if all the tasks in the job are completed
	log.Println("checkIfTaskCompleted():", l.CompletedTask)
	for _, completedTask := range l.CompletedTask {
		if completedTask.TaskId == task.TaskId {
			return true
		}
	}
	return false
}

func (l *Leader) MarkTaskCompleted(TaskId int32) {
	fmt.Println("Checking for task : ", TaskId)
	// taskIndex := -1
	for _, task := range l.TaskQueue {
		fmt.Println("Leader queue values", task.TaskId)
		if task.TaskId == TaskId { // Assuming Task has an Id field
			// taskIndex = i
			break
		}
	}

	// Remove the task from the TaskQueue
	// completedTask := l.TaskQueue[taskIndex]
	// l.TaskQueue = append(l.TaskQueue[:taskIndex], l.TaskQueue[taskIndex+1:]...)
	completedTask := l.TaskMap[int(TaskId)]
	l.TaskMap[int(TaskId)] = nil
	if completedTask == nil {
		log.Printf("Task %d not found in the task queue\n", TaskId)
		return
	}
	l.RemoveTaskFromWorkerQueue(int(completedTask.WorkerId), TaskId)
	l.CompletedTask = append(l.CompletedTask, completedTask)
	fmt.Printf("TaskID %d successfully removed from the task queue for leader\n", TaskId)
}

func (l *Leader) sendAckToReceiver() error { //TODO
	// Use the connection to the leader (assuming leader is at WorkerConn[2])
	log.Println("In send ack to receiver and assigned task is: ", l.assignedTasks)
	for i := NUM_OF_TASKS_PER_JOB; i < 2*NUM_OF_TASKS_PER_JOB; i++ {
		receiverTaskId := l.assignedTasks[i].TaskId
		receiverTask := l.assignedTasks[i]

		fmt.Println("Marking receiver task as completed:", receiverTaskId)
		l.MarkTaskCompleted(receiverTaskId)

		receiverWorker := int(receiverTask.WorkerId)
		conn, ok := l.WorkerConns[receiverWorker]
		if !ok {
			return fmt.Errorf("connection to receiver not found")
		}

		log.Println("Leader connection to receiver worker established:", receiverWorker, conn)

		client := pb.NewRainStromServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ack := &pb.Ack{
			Message:          fmt.Sprintf("Acknowledgment for task %d", receiverTaskId),
			ResponseType:     pb.ResponseType_COMPLETED,
			TaskId:           int32(receiverTaskId),
			NotificationType: pb.NotificationType_RECEIVER,
		}

		_, err := client.SendNotification(ctx, ack)
		if err != nil {
			log.Println("failed to send acknowledgment to receiver: %v", err)
			continue
		}

		fmt.Printf("Leader sent ack  to receiver for task %d", receiverTaskId)
	}
	return nil

}

func (l *Leader) checkIfAllSenderCompleted() bool {
	fmt.Println("In send task completed", l.CompletedTask)
	if len(l.CompletedTask) != NUM_OF_TASKS_PER_JOB {
		return false
	}
	if len(l.assignedTasks) < NUM_OF_TASKS_PER_JOB {
		log.Printf("Error: assignedTasks has fewer elements than expected: %d < %d\n", len(l.assignedTasks), NUM_OF_TASKS_PER_JOB)
		return false
	}

	for i := 0; i < NUM_OF_TASKS_PER_JOB; i++ {
		//check if l.assignedTasks[i].TaskId contians in l.CompletedTask
		taskCompleted := false
		for _, completedTask := range l.CompletedTask {
			if l.assignedTasks[i].TaskId == completedTask.TaskId {
				taskCompleted = true
				fmt.Println("Task completed: ", l.assignedTasks[i].TaskId)
				break
			}
		}
		if !taskCompleted {
			return false
		}
	}
	fmt.Println("checkIfAll sender return true")
	return true
}

func (l *Leader) ClearFileRequest(req m.RainStromRequest) error {

	// Connect to all workers
	conn, err := grpc.Dial(config.Conf.Server[groupMembership.IntroducerNodeId-1], grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(10*time.Second))
	if err != nil {
		log.Printf("Failed to connect to server: %v", err)
		return err
	}
	defer conn.Close()
	client := pb.NewRainStromServiceClient(conn)
	data, err := json.Marshal(req)
	command := &pb.Command{
		Payload:     data,
		CommandType: pb.CommandType_RainStromRequest,
		SenderId:    int32(config.Conf.NodeId),
		ReceiverId:  int32(groupMembership.IntroducerNodeId),
	}
	resp, err := client.TransferCommand(context.Background(), command)
	if err != nil {
		log.Printf("Failed to send command: %v", err)
		return err
	}
	log.Println("Received response from leader: ", resp)
	return nil
}
