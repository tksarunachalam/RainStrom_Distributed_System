package stream

import (
	"context"
	config "cs425/Config"
	pb "cs425/protos"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
)

func (ws *WorkerService) ConnectToWorkers() {
	for workerId, addr := range config.Conf.Server {
		fmt.Println("workerId: and Addr", workerId, addr)
		if workerId == ws.Worker.ID {
			// Skip connecting to itself
			// continue
		}
		ws.connectToWorker(workerId, addr)
	}

}
func (ls *Leader) ConnectLeaderToWorkers() {
	for workerId, addr := range config.Conf.Server {
		fmt.Println("Leader to workerId: and Addr Connection", workerId, addr)

		ls.connectLeaderToWorker(workerId, addr)
	}
}

func (ws *WorkerService) connectToWorker(workerId int, addr string) {
	for {
		conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(TimeOut)*time.Second), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(60*1024*1024)))
		if err != nil {
			log.Printf("Failed to connect to worker %d at %s: %v. Retrying...", workerId, addr, err)
			// time.Sleep(1 * time.Minute) // Wait before retrying
			return
		}
		// ws.mu.Lock()
		ws.WorkerConns[workerId] = conn
		// ws.mu.Unlock()
		fmt.Printf("Connected to worker %d at %s", workerId, addr)
		return
	}
}
func (ls *Leader) connectLeaderToWorker(workerId int, addr string) {
	for {
		conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(time.Duration(TimeOut)*time.Second), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(60*1024*1024)))
		if err != nil {
			log.Printf("Failed to connect to worker %d at %s: %v. Retrying...", workerId, addr, err)
			// time.Sleep(1 * time.Minute) // Wait before retrying
			return
		}
		// ls.mu.Lock()
		ls.WorkerConns[workerId] = conn
		// ls.mu.Unlock()
		fmt.Printf("Connected Leader to worker %d at %s", workerId, addr)
		return
	}
}

func (ws *WorkerService) TriggerConnectToWorkers() {
	fmt.Println("Enter TriggerConnectToWorkers(). worker conn map: ", ws.WorkerConns)
	for workerId, conn := range ws.WorkerConns {
		client := pb.NewRainStromServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(TimeOut)*time.Second)
		defer cancel()

		ack, err := client.TriggerConnectToWorkers(ctx, &pb.Empty{})
		fmt.Println("TriggerConnectToWorkers() ack: ", ack)
		if err != nil {
			log.Printf("Failed to trigger ConnectToWorkers on worker %d: %v", workerId, err)
		} else {
			fmt.Printf("Triggered ConnectToWorkers on worker %d", workerId)
		}
	}
}
func (ws *WorkerService) CreatePersistentConnections() {

	ws.ConnectToWorkers()
	ws.TriggerConnectToWorkers()
}

func (ws *WorkerService) CloseConnections() {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	for workerId, conn := range ws.WorkerConns {
		conn.Close()
		log.Printf("Closed connection to worker %d", workerId)
	}
}
