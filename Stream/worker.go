package stream

import (
	"context"
	config "cs425/Config"
	groupMembership "cs425/GroupMembership"
	hydfs "cs425/HyDfs"
	filePb "cs425/HyDfs/protos"
	m "cs425/Stream/models"
	pb "cs425/protos"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
)

//TODO: receives a task apply transformation, add output to hydfs file only for counts, if split then also do hash partitioning and send to leader ack about completion
// send to worker, send an ack to leader when a task is received by another worker for count
//If processing count then display results on console and also log the output to a hydfs file

// Worker struct represents a worker node
var lastTupleId int32

type WorkerService struct {
	Worker             *m.Worker
	WorkerConns        map[int]*grpc.ClientConn
	processedIDs       map[string]bool // Map to track processed unique IDs (for duplicate detection)
	outputMap          map[string]int
	OutputPartitionMap map[int32]int32
	TupleAckMap        map[string]bool
	aol                map[string]struct{}
	UnackedTuples      []*pb.Tuple
	RetryCounts        map[string]int
	prepareTasksDone   sync.WaitGroup
	mu                 sync.Mutex
	hy                 *hydfs.HyDfsService
}

type RainStromServer struct {
	pb.UnimplementedRainStromServiceServer
	Ws     *WorkerService
	Leader *Leader
}

// SendTask handles receiving tasks from the leader over gRPC
func (s *RainStromServer) SendTask(ctx context.Context, task *pb.Task) (*pb.Ack, error) {
	fmt.Printf("Received Task %d for File %v \n", task.TaskId, task.Files)
	fmt.Println("adding task to worker queue", task.TaskId)

	go s.Ws.PerformTask(task)

	return &pb.Ack{Message: fmt.Sprintf("Task %d recevied successfully!", task.TaskId), ResponseType: pb.ResponseType_SUCCESS}, nil
}
func (s *RainStromServer) UpdateTask(ctx context.Context, req *pb.UpdateTaskRequest) (*pb.Ack, error) {
	// Process the updated task information
	fmt.Printf("Received updated task from leader: %v\n", req.Task)
	log.Printf("Received updated task from leader: %v\n", req.Task)

	log.Println("curr node task queue", s.Ws.Worker.TaskQueue)
	// Add logic to handle the updated task information
	//check if received task is in the task queue of the worker
	found := false

	for index, task := range s.Ws.Worker.TaskQueue {
		if task.TaskId == req.Task.TaskId {
			found = true
			s.Ws.OutputPartitionMap = req.Task.ParentJob.OutputPartitions
			s.Ws.Worker.TaskQueue[index] = req.Task
			log.Println("Receiver task updated for TaskId: with new map", req.Task.TaskId, s.Ws.OutputPartitionMap)
		}
	}

	if found {
		return &pb.Ack{
			Message: "Task updated successfully",
		}, nil
	}

	log.Printf("Worker not processing tasks with TaskId %d", req.Task.TaskId)
	return &pb.Ack{
		Message: "Task not found in the queue",
	}, nil
}

func InitWorkerService(hy *hydfs.HyDfsService, nodeId int) *WorkerService {

	worker := m.Worker{
		ID:        nodeId - 1,
		IsAlive:   true,
		TaskQueue: make([]*pb.Task, 0), // Each worker has a task queue with buffer size 10
	}
	workerService := &WorkerService{Worker: &worker,
		hy:                 hy,
		WorkerConns:        make(map[int]*grpc.ClientConn),
		processedIDs:       make(map[string]bool),
		outputMap:          make(map[string]int),
		OutputPartitionMap: make(map[int32]int32),
		TupleAckMap:        make(map[string]bool),
		RetryCounts:        make(map[string]int),
		UnackedTuples:      make([]*pb.Tuple, 0),
	}

	// InitHydfsService(ms)
	// JoinNode := <-hy.JoinNodeSignalChan
	// log.Println("From Hydfs: new node join", JoinNode)
	fmt.Println("ws- hy", workerService.hy)
	return workerService
}

func (ws *WorkerService) RunWorker(addr string, isLeader bool, leader *Leader) {

	fmt.Println("Worker Service starting at", addr)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	s := grpc.NewServer()

	filePb.RegisterHyDfsServiceServer(s, &hydfs.Server{Hs: ws.hy})
	if isLeader && leader != nil {
		pb.RegisterRainStromServiceServer(s, &RainStromServer{Ws: ws, Leader: leader})
	} else {
		pb.RegisterRainStromServiceServer(s, &RainStromServer{Ws: ws})
	}
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
		return
	}
	fmt.Println("Worker Service started at", addr)
}

// ExchangeTuples handles bi-directional streaming of tuples and acknowledgments
func (s *RainStromServer) ExchangeTuples(stream pb.RainStromService_ExchangeTuplesServer) error {
	log.Printf("ExchangeTuples waiting for preareTasks to complete")
	s.Ws.prepareTasksDone.Wait()
	log.Printf("ExchangeTuples - preareTasks to completed")

	for {

		// Receive a tuple from the stream
		tuple, err := stream.Recv()
		fmt.Println("tuple, err: ", tuple, err)
		if err == io.EOF {
			fmt.Println("Entering EOF line")
			return nil // End of stream
		}
		if err != nil {
			log.Printf("Error receiving tuple: %v", err)
			return err
		}
		log.Println("Received tuple: with id", tuple, tuple.TupleId)

		s.Ws.mu.Lock()
		// if s.Ws.processedIDs[tuple.TupleId] {
		// 	// Duplicate detected, skip processing
		// 	log.Printf("Duplicate tuple detected: %s", tuple.TupleId)
		// 	s.Ws.mu.Unlock()
		// 	continue
		// }
		var destFile string
		log.Println("worker queue in recevier: ", s.Ws.Worker.TaskQueue)
		if len(s.Ws.Worker.TaskQueue) == 0 {
			log.Println("No tasks in the queue")
			destFile = "output.txt"
		} else {
			destFile = s.Ws.Worker.TaskQueue[len(s.Ws.Worker.TaskQueue)-1].ParentJob.HydfsDestFile
		}
		log.Println("dest file at the receiver worker is ", destFile)
		fmt.Printf("Going to call ProcessReceivedTuples func ")

		processedTupleId, err := s.Ws.ProcessReceivedTuples(tuple, AOL_FILE_FOR_TUPLE, destFile)
		if err != nil || processedTupleId == "" {
			log.Printf("Error processing tuple: %v", err)
			if err.Error() == pb.ResponseType_COMPLETED.String() {
				log.Printf("Duplicate tuple discarding found in aol. Discarding it: %s", tuple.TupleId)
				err = stream.Send(&pb.Ack{
					TupleId:      tuple.TupleId,
					Message:      "tuple failed to process" + err.Error(),
					ResponseType: pb.ResponseType_COMPLETED,
				})
				s.Ws.mu.Unlock()
				continue
			}

			err = stream.Send(&pb.Ack{
				TupleId:      tuple.TupleId,
				Message:      "tuple failed to process" + err.Error(),
				ResponseType: pb.ResponseType_FAIL,
			})
			if err != nil {
				log.Printf("Error sending failure acknowledgment: %v", err)
			}
			s.Ws.mu.Unlock()
			continue
		}

		// Mark this tuple as processed//
		//Process tuples here
		//Add to the processed map. create entry if not present
		s.Ws.processedIDs[tuple.TupleId] = true
		fmt.Println("Processed tuple: ", tuple.TupleId)
		s.Ws.mu.Unlock()

		// Send acknowledgment back to sender
		err = stream.Send(&pb.Ack{
			TupleId:      tuple.TupleId,
			Message:      "Processed successfully",
			ResponseType: pb.ResponseType_SUCCESS,
		})
		if err != nil {
			log.Printf("Error sending acknowledgment: %v", err)
			return err
		}
	}
}

func (s *RainStromServer) TriggerConnectToWorkers(ctx context.Context, req *pb.Empty) (*pb.Ack, error) {
	go s.Ws.ConnectToWorkers()
	clearFiles()

	return &pb.Ack{
		Message:      "Triggered ConnectToWorkers",
		ResponseType: pb.ResponseType_SUCCESS,
	}, nil
}

func (ws *WorkerService) SendCreateJobRequestToLeader(req m.RainStromRequest) error {

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
