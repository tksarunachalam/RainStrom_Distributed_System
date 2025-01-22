package hydfs

import (
	"context"
	pb "cs425/HyDfs/protos"
	"cs425/HyDfs/utils"
	"fmt"
	"log"
)

type Server struct {
	pb.UnimplementedHyDfsServiceServer
	Hs *HyDfsService
}

func (s *Server) ExecuteFileSystem(ctx context.Context, req *pb.FileRequest) (*pb.FilResponse, error) {
	// Implement your logic here
	log.Println("the request", req)
	fmt.Println("Going to create request", req.TypeReq)

	switch req.TypeReq {
	case pb.RequestType_CREATE:
		//create request
		fmt.Println("Going to create request")
		utils.CreateNewFile(req.HydfsFileName, req.GetFData())
	}
	return &pb.FilResponse{Ack: "Received"}, nil

}

// func (hy *HyDfsService) RunNode(addr string) {
// 	lis, err := net.Listen("tcp", addr)
// 	if err != nil {
// 		log.Fatalf("failed to listen: %v", err)
// 	}
// 	s := grpc.NewServer()
// 	pb.RegisterHyDfsServiceServer(s, &Server{Hs: hy})

// 	if err := s.Serve(lis); err != nil {
// 		log.Fatalf("failed to serve: %v", err)
// 	}
// 	InitHydfsService(ms)
// 	JoinNode := <-hy.JoinNodeSignalChan
// 	log.Println("From Hydfs: new node join", JoinNode)

// }
