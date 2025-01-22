package stream

import (
	"bufio"
	"context"
	groupMembership "cs425/GroupMembership"
	"cs425/HyDfs/models"
	"cs425/HyDfs/utils"
	m "cs425/Stream/models"
	"cs425/Stream/network"
	pb "cs425/protos"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

// ReadFilesToLines reads the file and returns the content as a list of key-value pairs
func (ws *WorkerService) PerformTask(task *pb.Task) {
	fmt.Println("In perform task")
	//add the task to the worker's task queue
	ws.mu.Lock()
	// fmt.Println("In perform task lock")
	ws.outputMap = make(map[string]int)
	ws.UnackedTuples = make([]*pb.Tuple, 0)
	ws.RetryCounts = make(map[string]int)
	ws.OutputPartitionMap = task.ParentJob.OutputPartitions
	fmt.Println("output partition map", task.ParentJob.OutputPartitions)
	ws.Worker.TaskQueue = append(ws.Worker.TaskQueue, task)
	log.Println("Adding tasks to queue", ws.Worker.TaskQueue)

	ws.mu.Unlock()
	// fmt.Println("after perform task unlock")

	sequenceNumber := -1
	filePath := task.ParentJob.HydfsSrcFile
	// Strip the filename from the path
	fileName := filepath.Base(filePath)

	if !checkIfFileExists(fileName) {
		//file not present, read it from HDFS
		fmt.Println("file not present, read it from HDFS: fileName", fileName)
		fileFetched, err := ws.getHyDfsFile(fileName)
		if fileFetched {

			filePath = "HyDfs/Files/" + fileName
			fmt.Println("updating file path")
			task.ParentJob.HydfsSrcFile = filePath
		} else {
			log.Printf("Error fetching file from HDFS: %v", err)
			return
		}
	} else {
		filePath = filepath.Join(utils.HDFS_FILE_PATH, fileName)
	}
	fmt.Println("final file path:", filePath)
	partitionId := task.Partition
	log.Println("At working partitionId:", partitionId)
	start := int64(task.ParentJob.InputPartitions[partitionId].Start)
	end := int64(task.ParentJob.InputPartitions[partitionId].End)
	log.Println("At working start and end:", start, end)
	var cmd string
	cmd = ""

	var lines map[string]string
	fmt.Println("task type:", task.TaskType)
	if task.TaskType == pb.TaskType_TASK_1 {
		//create connection to targer worker
		StartTasks()
		log.Println("sleep released for stage1")
		cmd = task.ParentJob.Op1Exe
		log.Println("exec is", cmd)

		log.Println("opening file Path", filePath)
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Println("Error opening file: ", err)
		}
		defer file.Close()
		fmt.Println("file:", file)

		lines = make(map[string]string)
		scanner := bufio.NewScanner(file)
		fmt.Println("scanner:", scanner)
		if scanner == nil {
			log.Println("Failed to initialize scanner")
		}
		lineNumber := -1
		fmt.Println("start and end:", start, end)
		for scanner.Scan() {
			//log.Println("Entering the for loop heree and line number: ", lineNumber)
			lineNumber++
			if int64(lineNumber) < start {
				//log.Println("continue loop", int64(lineNumber), start)
				continue
			}
			if int64(lineNumber) > end {
				log.Println("break loop", int64(lineNumber), start)
				break
			}
			//log.Println("should be starting the streaming and linenumber: ", lineNumber)
			//<file.txt:1, "hello world...">
			key := fmt.Sprintf("%s:%d", fileName, lineNumber)
			lines[key] = scanner.Text()
			//get the key value pair and store as a string
			tuple := fmt.Sprintf("'%s,%s'", key, lines[key])
			fmt.Println("Sending tuple to op1:", tuple)
			log.Println("Sending tuple to op1:", tuple)

			arguments := ws.getArguments(task, "")
			output, err := ExecuteTasks(cmd, tuple, ExecDir, arguments)
			fmt.Println("output from op1:", (string(output)))
			if err != nil || output == nil {
				fmt.Printf("Error executing task for tuple:%v. Error: %v", tuple, err)
				log.Printf("Error executing task for tuple:%v. Error: %v", tuple, err)

				continue
			}
			// fmt.Println(string(output))
			stage1OutputTuple := ConvertOutputToTuple(output)
			//check if the stage1OutputTuple is null
			if stage1OutputTuple.Key == "" {
				fmt.Println("Did not match criteria, skipping tuple")
				log.Println("Did not match criteria, skipping tuple")

				continue
			}

			stage1OutputTuple.TupleId = createTupleId(task.ParentJob.JobId, int(task.TaskId), sequenceNumber)
			fmt.Println("tuple:", tuple)
			intermidateTuple, receiverWorkerConn := ws.getTupleAndReceiver(stage1OutputTuple, task)
			intermidateTuple.TaskId = task.TaskId
			fmt.Println("stage tasks map", task.ParentJob.StageTasksWorkerMap)
			// intermidateTuple.ReceiverTaskId = task.ParentJob.StageTasksMap[task.TaskId]

			//failed tuples
			if ackReceived, exists := ws.TupleAckMap[stage1OutputTuple.TupleId]; exists && !ackReceived {
				ackReceived, err := network.SendTuple(intermidateTuple, receiverWorkerConn)
				fmt.Println("Send Tuple response:", ackReceived)
				log.Println("Send Tuple response:", ackReceived)

				if err != nil {
					log.Printf("Error sending tuple to worker: %v", err)
				}
				ws.TupleAckMap[stage1OutputTuple.TupleId] = ackReceived
			} else if !exists {
				// If the tupleId does not exist in the map, send the tuple and add it to the map
				ackReceived, err := network.SendTuple(intermidateTuple, receiverWorkerConn)
				fmt.Println("Send Tuple response 2:", ackReceived)
				log.Println("Send Tuple response 2 for tuple:", intermidateTuple.TupleId, ackReceived)

				if err != nil && ackReceived {
					log.Printf("Error but ackReceived: %v", err)
					ackReceived = true
				} else if err != nil {
					log.Printf("Error sending tuple to worker: %v", err)
					ackReceived = false
				}
				ws.TupleAckMap[stage1OutputTuple.TupleId] = ackReceived
				log.Printf("final ack received for tuple %s: %v", intermidateTuple.TupleId, ackReceived)
				if !ackReceived {
					ws.UnackedTuples = append(ws.UnackedTuples, &intermidateTuple)
					ws.RetryCounts[intermidateTuple.TupleId] = 0 // Initialize retry count
				}
			}

			sequenceNumber++

			//hash the output key
			//add the tuple and the worker id to a local map with ack status (ack tiemout will be 10s)

			//tupleId and pb.ACk

			//send the output to the connection established worker
		}

		if err := scanner.Err(); err != nil {
			fmt.Printf("Error reading file for split 1:  %v", err)
		}
		//  Check if all tuples are acknowledged and send ack to leader after completion
		// for key, ack := range ws.TupleAckMap {
		// 	log.Print("AckMap: ", key, ", ", ack)
		// }
		ws.mu.Lock()
		allAcknowledged := true
		for _, ack := range ws.TupleAckMap {
			if !ack {
				allAcknowledged = false
				break
			}
		}
		if len(ws.TupleAckMap) == 0 {
			allAcknowledged = false
		}
		ws.mu.Unlock()

		if allAcknowledged {
			// Send acknowledgment to the leader
			ws.finishTask(task)
		} else {
			ws.ResendUnackedTuples()
		}
	}
	if task.TaskType == pb.TaskType_TASK_2 {
		err := ws.PrepareTask(task)
		if err != nil {
			log.Printf("Error preparing task of task Id: %d %v", task.TaskId, err)
		}
	}
}

func (ws *WorkerService) PrepareTask(task *pb.Task) error {

	// 1. Initialize the word count map from the Hydfs output file
	//Get the hydfs file
	//get only the filename from the path
	log.Println("In prepare task()")
	ws.prepareTasksDone.Add(1)
	defer ws.prepareTasksDone.Done()
	ws.processedIDs = make(map[string]bool)
	ws.outputMap = make(map[string]int)
	ws.TupleAckMap = make(map[string]bool)
	ws.aol = make(map[string]struct{})

	//wait for 10 seconds to get the file
	time.Sleep(15 * time.Second)

	destHydfsFileName := filepath.Base(task.ParentJob.HydfsDestFile)

	fileFetched, err := ws.getHyDfsFile(destHydfsFileName)
	if fileFetched && err == nil {
		//filePath := filepath.Join("HyDfs/Files", destHydfsFile) //assuming that destHydfs is just a filename and not the whole path
		file, err := os.Open(filepath.Join(utils.FILE_BASE_PATH, destHydfsFileName))
		if err != nil {
			return fmt.Errorf("error opening local output file: %v", err)
		}
		log.Println("Opening local output file: ", file, filepath.Join(utils.FILE_BASE_PATH, destHydfsFileName))
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry m.Output
			err = json.Unmarshal([]byte(scanner.Text()), &entry) //Assuming file contains data in json format
			if err != nil {
				return fmt.Errorf("error parsing JSON from Hydfs file: %v", err)
			}
			// 	if the word already exists in the map, its count is updated.
			// If the word is not in the map, it gets initialized with the value from the file.
			//If entry.Key (the word) is new to outputMap, it will initialize its value in the map to 0 , and entry.Value will be added to it.
			log.Println("The key, value, word count initialization is: ", entry.Key, entry, entry.Value, ws.outputMap[entry.Key])
			ws.outputMap[entry.Key] = entry.Value
		}
		//log the final output map
		log.Println("The final output map being initialized: ", ws.outputMap)

		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading Hydfs file: %v", err)
		}
	} else {
		return fmt.Errorf("error opening Hydfs file: %v", err)
	}

	fmt.Println("Loading AOL on aol map to check for duplicates")
	log.Println("Loading AOL on aol map to check for duplicates")

	// aol := make(map[string]struct{})
	if checkIfAOLExists(filepath.Join("Stream/", AOL_FILE_FOR_TUPLE)) {
		file, err := os.Open(filepath.Join("Stream/", AOL_FILE_FOR_TUPLE))
		if err != nil {
			return fmt.Errorf("error opening AOL file: %v", err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			ws.aol[scanner.Text()] = struct{}{}
		}
		log.Println("AOL MAP retrived:", ws.aol)
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading AOL file: %v", err)
		}

	}

	log.Println("Prepare task done in the new worker")
	return nil
}
func (ws *WorkerService) sendAckToLeader(task *pb.Task) error { //TODO
	// Use the connection to the leader (assuming leader is at WorkerConn[2])
	conn, ok := ws.WorkerConns[groupMembership.IntroducerNodeId-1]
	if !ok {
		return fmt.Errorf("connection to leader not found")
	}

	client := pb.NewRainStromServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ack := &pb.Ack{
		Message:          fmt.Sprintf("Acknowledgment for task %d", task.TaskId),
		ResponseType:     pb.ResponseType_COMPLETED,
		TaskId:           task.TaskId,
		NotificationType: pb.NotificationType_LEADER,
	}

	_, err := client.SendNotification(ctx, ack)
	if err != nil {
		return fmt.Errorf("failed to send acknowledgment to leader: %v", err)
	}

	log.Printf("Acknowledgment sent to leader for task %d", task.TaskId)
	return nil

}

func (ws *WorkerService) ResendUnackedTuples() {
	for {
		time.Sleep(RETRY_TIMEOUT * time.Second)
		if len(ws.Worker.TaskQueue) == 0 {
			log.Println("No tasks in the queue")
			return
		}
		task := ws.Worker.TaskQueue[len(ws.Worker.TaskQueue)-1]

		if len(ws.UnackedTuples) == 0 {
			log.Println("All failed tuples acknowledged")
			ws.finishTask(task) // Finish task if all tuples are acknowledged
			return
		}
		log.Println("%d unacknowledged tuples", len(ws.UnackedTuples))

		allMaxRetriesReached := true

		ws.mu.Lock() // Lock to ensure thread-safe access to UnackedTuples and RetryCounts
		for i := 0; i < len(ws.UnackedTuples); {
			tuple := ws.UnackedTuples[i]
			log.Printf("Retrying the tuple %s", tuple.TupleId)

			receiverWorkerConn := ws.getReceiverByTuple(*tuple, task)
			ackReceived, err := network.SendTuple(*tuple, receiverWorkerConn)
			fmt.Println("Resend Tuple response:", ackReceived)

			if err != nil {
				log.Printf("Error resending tuple to worker: %v", err)
			}

			// Update the acknowledgment status
			ws.TupleAckMap[tuple.TupleId] = ackReceived

			if ackReceived {
				// Remove from unacknowledged list if acknowledged
				ws.UnackedTuples = append(ws.UnackedTuples[:i], ws.UnackedTuples[i+1:]...)
				delete(ws.RetryCounts, tuple.TupleId) // Remove retry count for this tuple
				log.Printf("\nTuple :%s acknowledged after retry", tuple.TupleId)
				continue // Skip incrementing i since we removed this element
			} else {
				// Increment the retry count
				if _, exists := ws.RetryCounts[tuple.TupleId]; !exists {
					ws.RetryCounts[tuple.TupleId] = 0
				}
				ws.RetryCounts[tuple.TupleId]++

				if ws.RetryCounts[tuple.TupleId] >= MAX_RETRY_COUNT+1 {
					log.Printf("Max retries reached for tuple %s", tuple.TupleId)
					log.Printf("Failed to send tuple %s after %d retries", tuple.TupleId, ws.RetryCounts[tuple.TupleId])
					ws.UnackedTuples = append(ws.UnackedTuples[:i], ws.UnackedTuples[i+1:]...) // Remove from unacknowledged list
					delete(ws.RetryCounts, tuple.TupleId)                                      // Remove retry count for this tuple
					continue                                                                   // Skip incrementing i since we removed this element
				} else {
					allMaxRetriesReached = false
				}
			}
			i++ // Increment i only if no element was removed
		}
		ws.mu.Unlock() // Unlock after updating data structures

		if allMaxRetriesReached {
			log.Println("Max retries reached for all unacknowledged tuples")
			ws.finishTask(task) // Finish task if max retries reached for all tuples
			return
		}
	}
}

func (ws *WorkerService) removeUnackedTuple(tupleId string) {
	for i, tuple := range ws.UnackedTuples {
		if tuple.TupleId == tupleId {
			ws.UnackedTuples = append(ws.UnackedTuples[:i], ws.UnackedTuples[i+1:]...)
			log.Println("Tuple removed from unacked list: ", tupleId)
			break
		}
	}
}

func (ws *WorkerService) finishTask(task *pb.Task) {

	fmt.Println("In all acks")
	err := ws.sendAckToLeader(task) //TODO
	if err != nil {
		log.Printf("Failed to send acknowledgment to leader for task %d: %v", task.TaskId, err)
	} else {

		log.Printf("Acknowledgment sent to leader for task %d", task.TaskId)
		taskIndex := -1
		for i, task2 := range ws.Worker.TaskQueue {
			fmt.Println("In worker task queue")
			if task2.TaskId == task.TaskId { // Assuming Task has an Id field
				taskIndex = i
				ws.Worker.TaskQueue = append(ws.Worker.TaskQueue[:taskIndex], ws.Worker.TaskQueue[taskIndex+1:]...)
				fmt.Printf("TaskID %d successfully removed from the task queue for sender.\n", task.TaskId)
				break
			}
		}
		if taskIndex == -1 {
			fmt.Printf("TaskID %d not found in the task queue.\n", task.TaskId)
		}
	}
}

func checkIfFileExists(fileName string) bool {
	filePath := filepath.Join(utils.HDFS_FILE_PATH, fileName)
	fmt.Println("checking file exists:", filePath)
	//check if the file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fmt.Println("file does not exist")
		return false
	}
	return true
}

func checkIfAOLExists(fileName string) bool {

	fmt.Println("checking file exists:", fileName)
	//check if the file exists
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		fmt.Println("file does not exist")
		return false
	}
	return true
}

func (worker *WorkerService) getHyDfsFile(fileName string) (bool, error) {

	//add HDFS file path xto the file name
	filePath := filepath.Join(utils.HDFS_FILE_PATH, fileName)

	readCommand := models.CreateCommand{
		LocalFileName: filepath.Join(utils.FILE_BASE_PATH, fileName),
		HydfsFileName: filePath,
	}
	fmt.Println("worker hyds:", worker.hy)
	resp := worker.hy.DoReadOperationClient(readCommand)
	fmt.Println("resp of read:", resp)
	if resp.IsSuccess {
		return true, nil

	}

	return false, fmt.Errorf("Error reading file: %v", resp.Message)

}

func (worker *WorkerService) getOutputFile(fileName string) (bool, error) {

	//add HDFS file path to the file name
	//filePath := filepath.Join(utils.HDFS_FILE_PATH, fileName)

	readCommand := models.CreateCommand{
		LocalFileName: "HyDfs/Files/" + fileName,
	}
	fmt.Println("worker hyds:", worker.hy)
	resp := worker.hy.DoReadOperationClient(readCommand)
	fmt.Println("resp of read:", resp)
	if resp.IsSuccess {
		return true, nil

	}

	return false, fmt.Errorf("Error reading file: %v", resp.Message)

}

func (ws *WorkerService) getTupleAndReceiver(output m.Output, task *pb.Task) (pb.Tuple, *grpc.ClientConn) {
	tuple := getTuple(output)
	receiverWorkerConn := ws.getReceiverByTuple(tuple, task)
	// receiverWorkerId := int(task.ParentJob.OutputPartitions[task.TaskId])
	// keyHash := int32(m.HashKey(output.Key, int(task.ParentJob.NumTasks)))
	// receiverWorkerId := int(ws.OutputPartitionMap[keyHash])
	// //get receiverWorkerAddress from the worker id from the worker map
	// fmt.Println("tupleid being sent to receiver worker id:", tuple.TupleId, receiverWorkerId)
	// log.Println("worker conns mao:", ws.WorkerConns)
	// receiverWorkerConn := ws.WorkerConns[receiverWorkerId]
	// // tuple.ReceiverTaskId = task.ParentJob.StageTasksMap[task.TaskId]
	// log.Println("receiver worker conn:", receiverWorkerConn)
	return tuple, receiverWorkerConn

}
func (ws *WorkerService) getReceiverByTuple(tuple pb.Tuple, task *pb.Task) *grpc.ClientConn {

	keyHash := int32(m.HashKey(tuple.Key, int(task.ParentJob.NumTasks)))
	receiverWorkerId := int(ws.OutputPartitionMap[keyHash])
	fmt.Println("tupleid being sent to receiver worker id:", tuple.TupleId, receiverWorkerId)
	log.Println("tupleid being sent to receiver worker id:", tuple.TupleId, receiverWorkerId)

	log.Println("worker conns mao:", ws.WorkerConns)
	receiverWorkerConn := ws.WorkerConns[receiverWorkerId]
	// tuple.ReceiverTaskId = task.ParentJob.StageTasksMap[task.TaskId]
	log.Println("receiver worker conn:", receiverWorkerConn)
	return receiverWorkerConn

}

func (worker *WorkerService) ProcessReceivedTuples(tuple *pb.Tuple, aolFile string, destHydfsFile string) (string, error) {
	fmt.Println("Inside ProcessReceivedTuples and destFile, aolfile  is: ", destHydfsFile, aolFile)

	aolFilePath := filepath.Join("Stream/", aolFile)
	aolFileHandle, err := os.OpenFile(aolFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", fmt.Errorf("error opening AOL file for writing: %v", err)
	}
	defer aolFileHandle.Close()

	fmt.Println("if tuple exists skip else do operation")
	log.Println("Check if tuple exists for tuple id: ", tuple.TupleId)
	if _, exists := worker.aol[tuple.TupleId]; exists {
		// Discard duplicates; do nothing
		fmt.Println("Tuple exists, discarding duplicates", tuple.TupleId)
		log.Println("Tuple exists, discarding duplicates", tuple.TupleId)

		return "", fmt.Errorf(pb.ResponseType_COMPLETED.String())
	}
	if worker.aol == nil {
		log.Println("AOL map is nil, initializing")
		worker.aol = make(map[string]struct{})
	}

	if _, exists := worker.aol[tuple.TupleId]; !exists {
		// Add to AOL
		fmt.Println("Tuple does not exist, add to AOL", tuple.TupleId)
		log.Println("Tuple does not exist, add to AOL", tuple.TupleId)
		worker.aol[tuple.TupleId] = struct{}{}
		log.Println("The  aol map  updated after adding new tuple is")
		log.Println("the len of worker queue is:", len(worker.Worker.TaskQueue))
		if len(worker.Worker.TaskQueue) == 0 {
			log.Println("No tasks in the queue")
			return "", fmt.Errorf("no receiver tasks in the queue")
		}
		task := worker.Worker.TaskQueue[len(worker.Worker.TaskQueue)-1]
		cmd := task.ParentJob.Op2Exe

		log.Println("exec is", cmd)

		arguments := worker.getArguments(task, tuple.Key)
		inputTuple := pb.Tuple{
			Key:   tuple.Key,
			Value: tuple.Value,
		}
		inputJson, _ := json.Marshal(inputTuple)
		stringInput := string(inputJson)
		// stringInput = fmt.Sprintf("'%s'", stringInput)
		log.Println("Input tuple is after '': ", stringInput)
		type Input struct {
			Key   string `json:"Key"`
			Value int    `json:"Value"`
		}
		var testInput Input

		err := json.Unmarshal([]byte(stringInput), &testInput)
		if err != nil {
			fmt.Println("Error unmarshalling input: ", err)
		}
		fmt.Println("test input:", testInput)

		output, err := ExecuteTasks(cmd, stringInput, ExecDir, arguments)
		fmt.Println("Executed task output is: ", string(output))
		var outputTuple m.Output
		fmt.Println("output from op1:", (string(output)))
		if err != nil || output == nil {
			fmt.Printf("Error executing task for tuple:%v. Error: %v", tuple, err)
			return "", fmt.Errorf("error executing task for tuple: %v", err)
		}
		outputTuple = ConvertOutputToTuple(output)
		log.Println("Output tuple is: ", outputTuple)

		// Update word count
		fmt.Println("Word count map", worker.outputMap)

		// worker.mu.Lock()
		worker.outputMap[tuple.Key] = int(outputTuple.Value)
		// worker.mu.Unlock()
		tuple.Value = int32(worker.outputMap[tuple.Key])
		fmt.Println("Updated word count is for key, value: ", worker.outputMap[tuple.Key], tuple.Key, tuple.Value)
		log.Println("Updated word count is for key, value: ", worker.outputMap[tuple.Key], tuple.Key, tuple.Value)

		// Write to Hydfs file: TO BE CHANGED
		outputJson, err := json.Marshal(outputTuple)
		if err != nil {
			return "", fmt.Errorf("error serializing tuple to JSON: %v", err)
		}
		outputJsonWithNewline := append(outputJson, '\n')

		err = worker.PerformWriteData(outputJsonWithNewline, destHydfsFile, false)
		if err != nil {
			return "", fmt.Errorf("error writing to Hydfs file: %v", err)
		}
		fmt.Println("DOne writing to file")
		_, err = aolFileHandle.WriteString(tuple.TupleId + "\n")
		if err != nil {
			return "", fmt.Errorf("error writing to AOL file: %v", err)
		}

	}

	return tuple.TupleId, nil
}

func getTuple(tuple m.Output) pb.Tuple {
	return pb.Tuple{
		TupleId: tuple.TupleId,
		Key:     tuple.Key,
		Value:   int32(tuple.Value),
	}

}

func (worker *WorkerService) PerformWriteData(data []byte, hydfsFileName string, isCreate bool) error {
	createCommand := models.CreateCommand{
		FileData:        data,
		HydfsFileName:   hydfsFileName,
		IsCreate:        isCreate,
		IsStreamResults: true,
	}
	err := worker.hy.DoWriteOperationClient(createCommand)
	if err != nil {
		log.Println("Error while writing file: %s", err.Error())
		return err
	} else {
		log.Println("File written successfully")
	}
	return nil
}

func (ws WorkerService) getArguments(task *pb.Task, key string) []string {
	arguments := make([]string, 0)

	fmt.Println("task type:", task.TaskType, task.TaskId, task.ParentJob.Op1Exe, task.ParentJob.Op2Exe)

	if task.TaskType == pb.TaskType_TASK_1 && (task.ParentJob.Op1Exe == App1_op1 || task.ParentJob.Op1Exe == App2_op1) {
		pattern := task.ParentJob.MetaData["pattern"]
		log.Printf("Pattern to be matched: %s\n", pattern)
		arguments = append(arguments, pattern)
		return arguments

	}

	if task.TaskType == pb.TaskType_TASK_2 && task.ParentJob.Op2Exe == App2_op2 {
		//get the count for the key from op1 putput
		count := ws.outputMap[key]
		log.Println("count for task", task.TaskId, task.TaskType)
		arguments = append(arguments, strconv.Itoa(count))
		return arguments
	}

	return arguments
}
