package network

import (
	"context"
	pb "cs425/protos"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
)

func SendTuple(tuple pb.Tuple, conn *grpc.ClientConn) (bool, error) {

	if conn == nil {
		log.Println("Target receiver connection is nil for tuple: ", tuple.TupleId, tuple.Key)
		return false, nil
	}
	client := pb.NewRainStromServiceClient(conn)
	return TransferTuple(client, &tuple)

}

func TransferTuple(client pb.RainStromServiceClient, tuple *pb.Tuple) (bool, error) {

	stream, err := client.ExchangeTuples(context.Background())
	if err != nil {
		log.Printf("Failed to create stream: %v", err)
		return false, err
	}

	err = stream.Send(tuple)
	if err != nil {
		log.Printf("Error sending tuple: . Error:%v", tuple, err)
		return false, err
	}
	fmt.Println("Sent tuple: ", tuple.TupleId, tuple.Key)

	// var ackReceived bool
	ackReceived := make(chan bool)

	// var wg sync.WaitGroup

	// wg.Add(1)
	go func(expectedID string) {
		// defer wg.Done()
		for {
			resp, recvErr := stream.Recv()
			if recvErr != nil {
				fmt.Printf("Error receiving acknowledgment for ID %s: %v", expectedID, recvErr)
				log.Printf("Error receiving acknowledgment for ID %s: %v", expectedID, recvErr)
				ackReceived <- false
				return
			}

			if resp.TupleId == expectedID && (resp.ResponseType == pb.ResponseType_SUCCESS || resp.ResponseType == pb.ResponseType_COMPLETED) {

				fmt.Printf("Acknowledgment received for tuple ID %s\n", expectedID)
				log.Printf("Acknowledgment received for tuple ID %s\n", expectedID, resp.ResponseType)
				ackReceived <- true
				return
			}
			if resp.TupleId == expectedID && resp.ResponseType == pb.ResponseType_FAIL {
				log.Printf("Failed to process tuple ID %s: %s", expectedID, resp.Message)
				ackReceived <- false
				return
			}
		}

	}(tuple.TupleId)

	select {
	case ackProcessed := <-ackReceived:
		fmt.Println("Ack Processed: ", ackProcessed)
		return ackProcessed, nil
	case <-time.After(7 * time.Second): // Timeout for acknowledgment
		fmt.Println("Timeout waiting for acknowledgment")
		return false, fmt.Errorf("timeout waiting for acknowledgment")
	}

	return false, nil
}
