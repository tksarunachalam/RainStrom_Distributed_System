package hydfs

import (
	"context"
	config "cs425/Config"
	appendLogs "cs425/HyDfs/AOL"
	"cs425/HyDfs/models"
	pb "cs425/HyDfs/protos"
	"cs425/HyDfs/utils"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
)

func (Hs *HyDfsService) TransferFile(fileTransReq pb.FileTransferRequest) error {

	// targetAddr := utils.GetServerAddrListFromNodeId(targetNodes)
	targetNodeAddr := config.Conf.Server[fileTransReq.DestNodeId-1]

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, targetNodeAddr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(60*1024*1024), // Set to 60MB, adjust as necessary
	))

	if err != nil {
		log.Printf("did not connect to %s: %v", targetNodeAddr, err)
		return err
	}
	defer conn.Close()

	client := pb.NewHyDfsServiceClient(conn)

	if fileTransReq.TransferRequestType == pb.TransferRequestType_READ_Transfer {
		//read request
		err := downloadFile(client, fileTransReq.DestFileName, fileTransReq.SrcFileName)
		if err != nil {
			log.Printf("failed to download file from %s: %v", targetNodeAddr, err)
			return err
		}
		return nil
	}
	if fileTransReq.TransferRequestType == pb.TransferRequestType_STREAM_Results {
		//stream request
		res, err := client.WriteStreamResults(context.Background(), &fileTransReq)
		if err != nil {
			log.Printf("failed to stream results from %s: %v", targetNodeAddr, err)
			return err
		}
		fmt.Println("Stream results response: ", res)
		return nil
	}

	if err := uploadFile(client, fileTransReq); err != nil {
		log.Printf("failed to upload file to %s: %v", targetNodeAddr, err)
		return err
	}
	return nil
}

// Method to send the file at each process
func uploadFile(client pb.HyDfsServiceClient, fileTransferRequest pb.FileTransferRequest) error {

	log.Println("Client sending a file to Server id", fileTransferRequest.DestNodeId)
	log.Println("target address mentioned:", fileTransferRequest.TargetAddr)
	//getFromCacheIfPresent(fileTransferRequest.SrcFileName)
	stream, err := client.SendFile(context.Background())
	if err != nil {
		return fmt.Errorf("could not open stream: %v", err)
	}

	file, err := os.Open(fileTransferRequest.SrcFileName)
	if err != nil {

		return fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	//send file in chunks - GRPC streaming
	buf := make([]byte, utils.FILE_TRANSFER_CHUNK_SIZE)
	for {
		n, err := file.Read(buf)
		log.Println(string(buf[:n]))

		if err == io.EOF {
			log.Println("breaking here eof")
			break
		}
		if err != nil {
			return fmt.Errorf("could not read chunk: %v", err)
		}

		//stream the chunk
		if err := stream.Send(&pb.FileSendRequest{
			FileTransferRequest: &fileTransferRequest,
			Chunk:               buf[:n],
		}); err != nil {
			return fmt.Errorf("could not send chunk: %v", err)
		}
	}
	//fmt.Println("file contents read into stream")
	_, err = stream.CloseAndRecv()
	if err != nil {
		fmt.Println("Error in receiving response", err)
		return fmt.Errorf("could not receive response: %v", err)
	}

	log.Printf("File uploaded successfully")
	return nil
}

// Method to receive the file at each process
func (s *Server) SendFile(stream pb.HyDfsService_SendFileServer) error {
	var fileName string
	var fileData []byte

	var fileTransferRequest pb.FileTransferRequest

	log.Println("Server recevied a file")
	for {
		// Receive each chunk from the client
		req, err := stream.Recv()
		if err == io.EOF {
			// End of stream indicates all data has been received
			break
		}
		if err != nil {
			return fmt.Errorf("failed to receive chunk: %v", err)
		}
		fileTransferRequest = *req.FileTransferRequest

		log.Println("Request Object", fileTransferRequest)
		// Extract file name from the first message if not already set
		if fileName == "" {
			fileName = req.FileTransferRequest.DestFileName
		}
		if fileTransferRequest.TargetAddr != "" {
			fmt.Println("Received file from target Server:", fileTransferRequest.TargetAddr)
		}

		// Append received chunk to file data buffer
		fileData = append(fileData, req.GetChunk()...)
	}
	// Ensure the directory exists
	if err := os.MkdirAll(utils.HDFS_FILE_PATH, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}

	// Save reconstructed file data to disk

	fmt.Println("the destination filename", fileName)
	//file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)'
	err := s.Hs.HandleRequestTypeOnReceive(fileTransferRequest, fileData)

	// err := os.WriteFile(fileName, fileData, 0644)
	if err != nil {
		return fmt.Errorf("Write operation failed: %v", err)
	}

	// Send confirmation back to client
	return stream.SendAndClose(&pb.FileSendResponse{Message: "File uploaded successfully"})
}

// READ - Server side
func (s *Server) ReadFile(req *pb.FileReadRequest, stream pb.HyDfsService_ReadFileServer) error {

	log.Println("Server received a read request for file: ", req.HydfsFileName)
	appendLogs.MergeAoLOnRead(req.HydfsFileName)

	//check if aol entry has been mergerd

	file, err := os.Open(req.HydfsFileName)
	if err != nil {
		return fmt.Errorf("could not open file: %v", err)
	}
	defer file.Close()

	// Read and send file in chunks
	buf := make([]byte, utils.FILE_TRANSFER_CHUNK_SIZE)
	for {
		n, err := file.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("could not read chunk: %v", err)
		}

		// Send chunk over gRPC stream
		if err := stream.Send(&pb.FileReadResponse{Chunk: buf[:n]}); err != nil {
			return fmt.Errorf("could not send chunk: %v", err)
		}
	}

	return nil
}

// READ :: Client side
func downloadFile(client pb.HyDfsServiceClient, hydfsFileName string, localFileName string) error {
	// Create request for reading file
	req := &pb.FileReadRequest{HydfsFileName: hydfsFileName}

	// Call ReadFile RPC method
	stream, err := client.ReadFile(context.Background(), req)
	if err != nil {
		return fmt.Errorf("could not open stream: %v", err)
	}

	// Open local file for writing
	localFile, err := os.Create(localFileName)
	if err != nil {
		return fmt.Errorf("could not create local file: %v", err)
	}
	defer localFile.Close()

	// Receive and write chunks to local file
	for {
		res, err := stream.Recv()
		if err == io.EOF {
			break // End of stream indicates all data has been received
		}
		if err != nil {
			return fmt.Errorf("failed to receive chunk: %v", err)
		}

		_, err = localFile.Write(res.Chunk)
		if err != nil {
			return fmt.Errorf("failed to write chunk to local file: %v", err)
		}
	}

	log.Printf("File downloaded successfully local to %s", localFileName)
	return nil
}

func (hy *HyDfsService) HandleRequestTypeOnReceive(fileReq pb.FileTransferRequest, fileData []byte) error {

	fileWMetadata := models.FileWMetaData{
		FileName:     fileReq.DestFileName,
		ClientPeerId: config.Conf.PeerId,
		TimeStamp:    time.Now(),
	}

	if fileReq.TransferRequestType == pb.TransferRequestType_CREATE_TransferFileType {
		if _, err := os.Stat(fileReq.DestFileName); err == nil {
			// File exists
			log.Println("DUPLICATE create check File already exists: ", fileReq.DestFileName)
			return fmt.Errorf("file already exists: %s. Cannot create Perform append", fileReq.DestFileName)
		}
	}

	if fileReq.TransferRequestType == pb.TransferRequestType_CREATE_TransferFileType ||
		fileReq.TransferRequestType == pb.TransferRequestType_APPEND_TransferFileType || fileReq.TransferRequestType == pb.TransferRequestType_STREAM_Results {
		//
		aolEntry := models.GetAolEntry(fileData, fileWMetadata)
		//fmt.Println("calling aol write for data", aolEntry)
		appendLogs.WriteLog(aolEntry)
		appendLogs.ReplicateLogEntryToNeighbors(aolEntry, hy.Ms)
		return nil
	}
	if fileReq.TransferRequestType == pb.TransferRequestType_REPLICATE_Transfer_TransferFileType {
		// Write to local AOL using WriteLog function
		//File is present delete it
		log.Println("failure replicate transfer request")
		if _, err := os.Stat(fileReq.DestFileName); err == nil {
			log.Println("File exists, deleting it")
			//Delete the destination file
			err := os.Remove(fileReq.DestFileName)
			if err != nil {
				log.Println("Error deleting file: ", err)
				return err
			}
			log.Println("File deleted: ", fileReq.DestFileName)
		}
		fileWMetadata.RequestType = pb.TransferRequestType_REPLICATE_Transfer_TransferFileType.String()
		aolEntry := models.GetAolEntry(fileData, fileWMetadata)
		appendLogs.WriteLog(aolEntry)
		log.Println("Failed replica reached aol")
		return nil
	}
	return nil

}

func (s *Server) ReplicateLogEntry(ctx context.Context, req *pb.ReplicateRequest) (*pb.ReplicateResponse, error) {
	// Extract data from request
	aolEntryBytes := req.GetAolEntry()

	// Write to local AOL using WriteLog function
	var aolEntry models.AolEntry
	json.Unmarshal(aolEntryBytes, &aolEntry)

	log.Printf("\n Received AOL replica for file: %s from %d", aolEntry.FileMetadata.FileName, aolEntry.FileMetadata.ClientPeerId)

	appendLogs.WriteLog(aolEntry)

	log.Println("AOL Replication successfull")

	return &pb.ReplicateResponse{Success: true}, nil
}

// MultiAppend is called when a VM receives a multiappend request from another VM.
func (s *Server) MultiAppend(ctx context.Context, req *pb.FileTransferRequest) (*pb.MultiAppendResponse, error) {
	fmt.Printf("\nReceived multiappend request: Appending %s to %s on target %s", req.SrcFileName, req.DestFileName, req.TargetAddr)

	// Trigger internal file transfer process
	req.SrcNodeId = int32(config.Conf.NodeId)
	req.SrcPeerd = int32(config.Conf.PeerId)

	conn, err := grpc.DialContext(ctx, req.TargetAddr, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(60*1024*1024), // Set to 60MB, adjust as necessary
	))

	if err != nil {
		log.Printf("Failed to connect to target Server: %v", err)

		return &pb.MultiAppendResponse{Response: pb.ResponseType_FAIL}, fmt.Errorf("failed to connect to target Server: %v", err)
	}
	defer conn.Close()

	transferReq := &pb.FileTransferRequest{
		SrcFileName:         req.SrcFileName,
		DestFileName:        req.DestFileName,
		TargetAddr:          req.TargetAddr,
		SrcNodeId:           int32(config.Conf.NodeId),
		SrcPeerd:            int32(config.Conf.PeerId),
		DestNodeId:          req.DestNodeId,
		DestPeerId:          req.DestPeerId,
		TransferRequestType: pb.TransferRequestType_APPEND_TransferFileType,
		FileData:            req.FileData,
	}

	client := pb.NewHyDfsServiceClient(conn)
	fmt.Println("calling upload file from multiappend")
	err = uploadFile(client, *transferReq)
	fmt.Println("upload file done in multiappend")

	if err != nil {
		return &pb.MultiAppendResponse{Response: pb.ResponseType_FAIL}, fmt.Errorf("failed to append file: %v", err)
	}

	return &pb.MultiAppendResponse{Response: pb.ResponseType_SUCCESS}, nil
}

// MergeFile is called when a replica receives a merge request from the primary VM.
func (s *Server) MergeFile(ctx context.Context, req *pb.FileTransferRequest) (*pb.MergeResponse, error) {
	log.Printf("Received merge request: Overriding %s with data from %s", req.DestFileName, req.SrcFileName)
	fmt.Println("Received merge request: Overriding %s with data from %s", req.DestFileName, req.SrcFileName)
	// Ensure directory exists
	if err := os.MkdirAll(utils.HDFS_FILE_PATH, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	appendLogs.MergeAoLOnRead(req.DestFileName)

	// Delete any existing file before writing new data
	if _, err := os.Stat(req.DestFileName); err == nil {
		log.Printf("Deleting existing file: %s", req.DestFileName)
		if err := os.Remove(req.DestFileName); err != nil {
			fmt.Println("overriiding file")
			return nil, fmt.Errorf("failed to delete existing file: %v", err)
		}
	}

	// Open or create destination file

	destFile, err := os.OpenFile(req.DestFileName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open destination file: %v", err)
	}
	defer destFile.Close()

	// Write received data into destination file
	_, err = destFile.Write(req.FileData)
	if err != nil {
		return nil, fmt.Errorf("failed to write data to destination file: %v", err)
	}

	fmt.Println("Successfully merged contents of %s", req.DestFileName)

	log.Printf("Successfully merged contents of %s", req.DestFileName)

	return &pb.MergeResponse{Message: "Merge completed successfully", Response: pb.ResponseType_SUCCESS}, nil
}

// NotifyPrimaryToMerge is called by non-primary VMs to notify the primary VM to trigger a merge.
func (s *Server) NotifyPrimaryToMerge(ctx context.Context, req *pb.MergeRequest) (*pb.MergeResponse, error) {
	fmt.Println("Received notification to trigger merge for file: %s", req.HydfsFileName)
	log.Printf("Received notification to trigger merge for file: %s", req.HydfsFileName)

	// Trigger merge on replicas (this is done by the primary)
	fileHashId := config.GetFileHashId(req.HydfsFileName)
	destPeerIds := utils.FileRouting(fileHashId, s.Hs.Ms.MemberShipList)
	var replicaNodes []int
	for _, destPeerId := range destPeerIds[1:] {
		destNodeId := s.Hs.Ms.MemberShipList[s.Hs.Ms.MemberShipList.GetIndexOfPeer(destPeerId)].NodeId
		replicaNodes = append(replicaNodes, destNodeId)
	}

	fmt.Println("Primary VM triggering merge on replicas: ", replicaNodes)
	err := s.Hs.TriggerMerge(req.HydfsFileName, replicaNodes)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger merge: %v", err)
	}

	return &pb.MergeResponse{Message: "Merge triggered successfully", Response: pb.ResponseType_SUCCESS}, nil
}

func (s *Server) WriteStreamResults(ctx context.Context, req *pb.FileTransferRequest) (*pb.FileSendResponse, error) {
	// Process the received tuples
	var fileName string
	var fileData []byte

	// Append received chunk to file data buffer
	fileData = req.FileData
	fileName = req.DestFileName

	// Ensure the directory exists
	if err := os.MkdirAll(utils.HDFS_FILE_PATH, 0755); err != nil {
		return &pb.FileSendResponse{}, fmt.Errorf("failed to create directory: %v", err)
	}

	// Save reconstructed file data to disk

	fmt.Println("the destination filename", fileName)
	fmt.Println("the file data", string(fileData))
	//file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)'
	err := s.Hs.HandleRequestTypeOnReceive(*req, fileData)

	// err := os.WriteFile(fileName, fileData, 0644)
	if err != nil {
		return &pb.FileSendResponse{}, fmt.Errorf("write operation failed: %v", err)
	}

	// Send confirmation back to client
	return &pb.FileSendResponse{Message: "File uploaded successfully"}, nil

}
