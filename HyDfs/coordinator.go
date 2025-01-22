package hydfs

import (
	"context"
	c "cs425/Config"
	config "cs425/Config"
	"cs425/HyDfs/models"
	pb "cs425/HyDfs/protos"
	"cs425/HyDfs/utils"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
)

// var fileCache = utils.NewCache(100)

func RunCoordinator(host string) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(60*1024*1024), // Set to 10MB, adjust as necessary
	))
	if err != nil {
		log.Printf("did not connect to %s, Error = %v", host, err)
		return
	}

	defer conn.Close()

	c := pb.NewHyDfsServiceClient(conn)

	r, err := c.ExecuteFileSystem(ctx, &pb.FileRequest{
		TypeReq:       pb.RequestType_CREATE,
		HydfsFileName: "sample1.txt"})
	if err != nil {
		log.Println("could not execute request: %v", err)
		return
	}

	log.Printf("old Response from server: %s", r.GetAck())

	// if err := uploadFile(c, "HyDfs/Files/sample1.txt"); err != nil {
	// 	log.Printf("failed to upload file: %v", err)
	// }
}

func (hs *HyDfsService) DoWriteOperationClient(createCommand models.CreateCommand) error {
	fileHashId := c.GetFileHashId(createCommand.HydfsFileName)

	destPeerIds := utils.FileRouting(fileHashId, hs.Ms.MemberShipList)
	destPeerId := destPeerIds[0]
	destNodeId := hs.Ms.MemberShipList[hs.Ms.MemberShipList.GetIndexOfPeer(destPeerId)].NodeId
	var writeRequest pb.FileTransferRequest
	if createCommand.IsCreate {
		log.Println("IsCreate is true")

		writeRequest = CreateFileRequest(createCommand.LocalFileName, createCommand.HydfsFileName, destNodeId, destPeerId, nil, pb.TransferRequestType_CREATE_TransferFileType)
	} else {
		//update file
		writeRequest = CreateFileRequest(createCommand.LocalFileName, createCommand.HydfsFileName, destNodeId, destPeerId, nil, pb.TransferRequestType_APPEND_TransferFileType)
	}
	if createCommand.IsStreamResults {
		writeRequest = CreateFileRequest(createCommand.LocalFileName, createCommand.HydfsFileName, destNodeId, destPeerId, createCommand.FileData, pb.TransferRequestType_STREAM_Results)
	}
	return hs.TransferFile(writeRequest)
}
func (hs *HyDfsService) DoReadOperationClient(readCommand models.CreateCommand) models.CommandResponse {
	fileHashId := c.GetFileHashId(readCommand.HydfsFileName)
	log.Println("File to get hash id: ", fileHashId)

	destPeerIds := utils.FileRouting(fileHashId, hs.Ms.MemberShipList)
	selfFilesIds := utils.GetFilesUsingHash()

	log.Println("Files present in the same node: ", selfFilesIds)
	var self bool
	for _, file := range selfFilesIds {
		if fileHashId == file.FileId {
			log.Println("File present in the same node")
			//self = true
			//get the extension of the file
			ext := filepath.Ext(readCommand.HydfsFileName)
			log.Println("Extension of the file: ", ext)
			if ext == ".csv" {
				self = true
			}

			break
		}
	}
	if self {
		//file present in the same node
		_, err := utils.ReadFileFromLocal(readCommand)
		if err == nil {
			return models.CommandResponse{
				IsSuccess: true,
				Message:   fmt.Sprintf("File read successfully to localFileName: %s", readCommand.LocalFileName)}
		} else {
			return models.CommandResponse{
				IsSuccess: false,
				Message:   fmt.Sprintf("Error while reading file: %s", err.Error())}
		}

	} else {
		log.Println("File not present in the same node")
		var destPeerId, destNodeId int
		//Read from particular VM ID if provided
		if readCommand.VmId != 0 {
			destNodeId = readCommand.VmId
			destPeerId = hs.Ms.MemberShipList[destNodeId-1].PeerId
		} else {
			//read from master of the file
			destPeerId = destPeerIds[0]
			destNodeId = hs.Ms.MemberShipList[hs.Ms.MemberShipList.GetIndexOfPeer(destPeerId)].NodeId
		}
		readRequest := CreateReadRequest(readCommand.LocalFileName, readCommand.HydfsFileName, destNodeId, destPeerId, nil, pb.TransferRequestType_READ_Transfer)
		err := hs.TransferFile(readRequest)
		if err != nil {
			return models.CommandResponse{
				IsSuccess: false,
				Message:   fmt.Sprintf("Error while reading file: %s", err.Error())}
		}
	}
	return models.CommandResponse{
		IsSuccess: true,
		Message:   fmt.Sprintf("File read successfully to localFileName: %s", readCommand.LocalFileName)}
}

func (hs *HyDfsService) TransferFilesOnFailure(filesInRange []string, fromPeerId int, toPeerId int) {

	log.Println("Inside() TransferFilesOnFailure")
	toPeerIndex := hs.Ms.MemberShipList.GetIndexOfPeer(toPeerId)
	if toPeerIndex < 0 || toPeerIndex >= len(hs.Ms.MemberShipList) {
		log.Println("Invalid toPeerId")
		return
	}
	toNodeId := hs.Ms.MemberShipList[hs.Ms.MemberShipList.GetIndexOfPeer(toPeerId)].NodeId
	log.Print("To Node ID: ", toNodeId)

	for _, files := range filesInRange {
		if strings.HasSuffix(files, ".csv") {
			log.Println("Skipping CSV file for failure replicate: ", files)
			continue
		}

		log.Println("Starting file replication on failure for files: ", files)

		replicateRequest := CreateFileRequest(files, files, toNodeId, toPeerId, nil, pb.TransferRequestType_REPLICATE_Transfer_TransferFileType)
		log.Printf("\nRequest created to replicate file: %s from Node: %d to Node: %d", files, replicateRequest.SrcNodeId, replicateRequest.DestNodeId)
		err := hs.TransferFile(replicateRequest)
		if err != nil {
			log.Printf("Error while replicating file: %s", err.Error())
		}
	}

}

func (hs *HyDfsService) InitiateMultiAppend(multiAppend models.MultiAppendCommand) error {

	fmt.Println("to vms :", multiAppend.Vms)
	fmt.Println("local files :", multiAppend.LocalFiles)
	return hs.Multiappend(multiAppend.HydfsFileName, multiAppend.Vms, multiAppend.LocalFiles)

}
func (hs *HyDfsService) Multiappend(hydfsFileName string, vms []int, localFiles []string) error {
	if len(vms) != len(localFiles) {
		return fmt.Errorf("the number of VMs and local files must be equal")
	}
	fmt.Println("Multiappend - vms", vms)
	fmt.Println("Multiappend - localFiles", localFiles)
	fmt.Println("Multiappend - hydfsFileName", hydfsFileName)
	fileHashId := config.GetFileHashId(hydfsFileName)
	destPeerIds := utils.FileRouting(fileHashId, hs.Ms.MemberShipList)
	destPeerId := destPeerIds[0]
	destNodeId := hs.Ms.MemberShipList[hs.Ms.MemberShipList.GetIndexOfPeer(destPeerId)].NodeId
	fmt.Println("Multiappend - destNode and peerId", destNodeId, destPeerId)

	var wg sync.WaitGroup
	errChan := make(chan error, len(vms))

	//start time
	for i := range vms {
		wg.Add(1)
		go func(vmId int, localFileName string, hydfsFileName string, destNodeId int) {
			defer wg.Done()

			// Create a gRPC connection to the target VM
			conn, err := grpc.Dial(config.Conf.Server[vmId-1], grpc.WithInsecure())
			if err != nil {
				errChan <- fmt.Errorf("failed to connect to node %d: %v", vmId, err)
				return
			}
			defer conn.Close()

			client := pb.NewHyDfsServiceClient(conn)
			multiAppendRequest := CreateFileRequest(localFileName, hydfsFileName, destNodeId, destPeerId, nil, pb.TransferRequestType_MULTI_APPEND_Transfer)
			multiAppendRequest.TargetAddr = config.Conf.Server[destNodeId-1]

			fmt.Println("sending write request to  with object ", vmId, multiAppendRequest)
			// Call the uploadFile function to append data from local file
			resp, err := client.MultiAppend(context.Background(), &multiAppendRequest)
			fmt.Println("Multiappend response from ", vmId, resp)
			if err != nil {
				fmt.Println("Error in multiappend", err)
				errChan <- fmt.Errorf("failed to send multiappend request to node %d: %v", vmId, err)
				return
			} else if resp.Response == pb.ResponseType_SUCCESS {
				fmt.Printf("Multiappend request success at to node %d", vmId)

				log.Printf("Multiappend request success at to node %d", vmId)
			}

		}(vms[i], localFiles[i], hydfsFileName, destNodeId)
	}

	// Wait for all goroutines to finish
	wg.Wait()

	// Check if any errors occurred
	close(errChan)
	if len(errChan) > 0 {
		return <-errChan // Return the first error encountered
	}

	fmt.Println("ALL Multi appended success")
	//end time
	return nil
}

func (hs *HyDfsService) InitateMerge(mergeRequest models.MergeCommand) error {
	// Get file hash ID and determine peers responsible for this file
	fileHashId := config.GetFileHashId(mergeRequest.HydfsFileName)
	destPeerIds := utils.FileRouting(fileHashId, hs.Ms.MemberShipList)

	// Check if current node is not primary
	if config.Conf.PeerId != destPeerIds[0] {
		// Not primary, need to send request to primary
		primaryNodeId := hs.Ms.MemberShipList[hs.Ms.MemberShipList.GetIndexOfPeer(destPeerIds[0])].NodeId

		fmt.Println("Current node is not primary. Forwarding merge request to primary node ", primaryNodeId)
		log.Printf("Current node is not primary. Forwarding merge request to primary node %d", primaryNodeId)

		// Create gRPC connection to primary
		conn, err := grpc.Dial(config.Conf.Server[primaryNodeId-1], grpc.WithInsecure())
		if err != nil {
			return fmt.Errorf("failed to connect to primary node %d: %v", primaryNodeId, err)
		}
		defer conn.Close()

		client := pb.NewHyDfsServiceClient(conn)

		// Prepare merge request
		mergeReq := &pb.MergeRequest{
			HydfsFileName: mergeRequest.HydfsFileName,
		}

		// Call NotifyPrimaryToMerge RPC method on primary
		log.Printf("Sending merge notification for file %s to primary node %d", mergeRequest.HydfsFileName, primaryNodeId)
		resp, err := client.NotifyPrimaryToMerge(context.Background(), mergeReq)
		if err != nil || resp.Message != "Merge triggered successfully" {
			return fmt.Errorf("failed to notify primary for merge: %v", err)
		}

		log.Printf("Primary node %d successfully triggered merge", primaryNodeId)
		fmt.Println("Primary node successfully triggered merge:", resp)
		return nil
	}

	// If current node is primary, proceed with triggering merge on replicas
	log.Println("Current node is primary. Proceeding with merge operation.")

	// Loop over destination peer IDs and get list of replica nodes (excluding first one which is primary)
	var replicaNodes []int
	for _, destPeerId := range destPeerIds[1:] {
		destNodeId := hs.Ms.MemberShipList[hs.Ms.MemberShipList.GetIndexOfPeer(destPeerId)].NodeId
		replicaNodes = append(replicaNodes, destNodeId)
	}

	return hs.TriggerMerge(mergeRequest.HydfsFileName, replicaNodes)
}

// TriggerMerge sends a merge request from primary VM to all replicas.
func (hs *HyDfsService) TriggerMerge(hydfsFileName string, replicas []int) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(replicas))

	// Iterate over each replica and send a merge request
	for _, replicaId := range replicas {
		wg.Add(1)
		go func(replicaId int) {
			defer wg.Done()

			// Create gRPC connection to replica
			conn, err := grpc.Dial(config.Conf.Server[replicaId-1], grpc.WithInsecure())
			if err != nil {
				errChan <- fmt.Errorf("failed to connect to replica %d: %v", replicaId, err)
				return
			}
			defer conn.Close()

			client := pb.NewHyDfsServiceClient(conn)

			// Prepare file transfer request (read local file first)
			srcPath := hydfsFileName
			srcData, err := os.ReadFile(srcPath)
			if err != nil {
				fmt.Println("Error reading source file", err)
				errChan <- fmt.Errorf("failed to read source file: %v", err)
				return
			}

			transferReq := &pb.FileTransferRequest{
				SrcFileName:         hydfsFileName,
				DestFileName:        hydfsFileName,
				FileData:            srcData,
				TransferRequestType: pb.TransferRequestType_MERGE_Transfer,
			}
			fmt.Println("Sending merge request to replica", replicaId)
			log.Println("Sending merge request to replica", replicaId)

			// Call MergeFile RPC method on replica
			fmt.Println("MergeFile() start for ", replicaId)
			resp, err := client.MergeFile(context.Background(), transferReq)
			if err != nil {
				fmt.Println("Error merging file on replica", err, resp)
				errChan <- fmt.Errorf("failed to merge on replica %d: %v", replicaId, err)
				return
			}
			fmt.Println("MergeFile() finish for ", replicaId)

		}(replicaId)
	}

	// Wait for all goroutines (merge operations) to finish
	wg.Wait()

	// Check if any errors occurred during merging process
	close(errChan)
	if len(errChan) > 0 {
		return <-errChan // Return the first error encountered
	}
	fmt.Println("All replicas merged successfully")
	log.Println("All replicas merged successfully")
	return nil
}

func CreateFileRequest(srcFileName string, hydfsFileName string, destNodeId int, destPeerId int, fileData []byte, reqType pb.TransferRequestType) pb.FileTransferRequest {

	fileWriteRequest := pb.FileTransferRequest{
		SrcNodeId:           int32(c.Conf.NodeId),
		SrcPeerd:            int32(c.GetServerIdInRing(c.Conf.Server[config.Conf.NodeId-1])),
		SrcFileName:         srcFileName,
		DestFileName:        hydfsFileName,
		DestNodeId:          int32(destNodeId),
		DestPeerId:          int32(destPeerId),
		FileData:            fileData,
		TransferRequestType: reqType,
	}
	return fileWriteRequest
}

func CreateReadRequest(srcFileName string, hydfsFileName string, destNodeId int, destPeerId int, fileData []byte, reqType pb.TransferRequestType) pb.FileTransferRequest {

	readFileRequest := CreateFileRequest(srcFileName, hydfsFileName, destNodeId, destPeerId, fileData, reqType)
	readFileRequest.TransferRequestType = pb.TransferRequestType_READ_Transfer
	return readFileRequest
}
