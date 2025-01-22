package appendLogs

import (
	"context"
	config "cs425/Config"
	groupMembership "cs425/GroupMembership"
	"cs425/HyDfs/models"
	pb "cs425/HyDfs/protos"
	"cs425/HyDfs/utils"
	"encoding/json"
	"log"
	"time"

	"google.golang.org/grpc"
)

func ReplicateLogEntryToNeighbors(aolEntry models.AolEntry, ms *groupMembership.MembershipService) {

	fileHashId := config.GetFileHashId(aolEntry.FileMetadata.FileName)
	destPeerIds := utils.FileRouting(fileHashId, ms.MemberShipList)

	var destNodeIds []int
	for index, destPeerId := range destPeerIds {
		if index != 0 {
			destNodeIds = append(destNodeIds, ms.MemberShipList[ms.MemberShipList.GetIndexOfPeer(destPeerId)].NodeId)
		}
	}

	log.Println("Replicating log entry of file: to nodes:", aolEntry.FileMetadata.FileName, destNodeIds)
	// List of servers to replicate to
	var servers []string
	for _, destNode := range destNodeIds {
		targetNodeAddr := config.Conf.Server[destNode-1]
		servers = append(servers, targetNodeAddr)

	}

	for _, serverAddr := range servers {
		conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
		if err != nil {
			log.Printf("Failed to connect to %s: %v", serverAddr, err)
			return
		}
		defer conn.Close()

		client := pb.NewHyDfsServiceClient(conn)
		aolEntryBytes, err := json.Marshal(aolEntry)
		if err != nil {
			log.Println("Error converting aolEntry to bytes")
			return
		}
		req := &pb.ReplicateRequest{
			AolEntry: aolEntryBytes,
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		resp, err := client.ReplicateLogEntry(ctx, req)
		if err != nil {
			log.Printf("Error replicating log entry to %s: %v", serverAddr, err)
			continue
		}
		if resp.Success {
			log.Printf("Successfully replicated log entry to %s", serverAddr)
		} else {
			log.Printf("Failed to replicate log entry to %s", serverAddr)
		}
	}
}
