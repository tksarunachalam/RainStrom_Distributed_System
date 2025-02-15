// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.27.3
// source: mp3.proto

package protos

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	HyDfsService_ExecuteFileSystem_FullMethodName    = "/hydfs.HyDfsService/ExecuteFileSystem"
	HyDfsService_SendFile_FullMethodName             = "/hydfs.HyDfsService/SendFile"
	HyDfsService_ReceiveFile_FullMethodName          = "/hydfs.HyDfsService/ReceiveFile"
	HyDfsService_ReplicateLogEntry_FullMethodName    = "/hydfs.HyDfsService/ReplicateLogEntry"
	HyDfsService_ReadFile_FullMethodName             = "/hydfs.HyDfsService/ReadFile"
	HyDfsService_MultiAppend_FullMethodName          = "/hydfs.HyDfsService/MultiAppend"
	HyDfsService_MergeFile_FullMethodName            = "/hydfs.HyDfsService/MergeFile"
	HyDfsService_NotifyPrimaryToMerge_FullMethodName = "/hydfs.HyDfsService/NotifyPrimaryToMerge"
	HyDfsService_WriteStreamResults_FullMethodName   = "/hydfs.HyDfsService/WriteStreamResults"
)

// HyDfsServiceClient is the client API for HyDfsService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type HyDfsServiceClient interface {
	ExecuteFileSystem(ctx context.Context, in *FileRequest, opts ...grpc.CallOption) (*FilResponse, error)
	SendFile(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[FileSendRequest, FileSendResponse], error)
	ReceiveFile(ctx context.Context, in *FileReceiveRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[FileReceiveReponse], error)
	ReplicateLogEntry(ctx context.Context, in *ReplicateRequest, opts ...grpc.CallOption) (*ReplicateResponse, error)
	ReadFile(ctx context.Context, in *FileReadRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[FileReadResponse], error)
	MultiAppend(ctx context.Context, in *FileTransferRequest, opts ...grpc.CallOption) (*MultiAppendResponse, error)
	MergeFile(ctx context.Context, in *FileTransferRequest, opts ...grpc.CallOption) (*MergeResponse, error)
	NotifyPrimaryToMerge(ctx context.Context, in *MergeRequest, opts ...grpc.CallOption) (*MergeResponse, error)
	WriteStreamResults(ctx context.Context, in *FileTransferRequest, opts ...grpc.CallOption) (*FileSendResponse, error)
}

type hyDfsServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewHyDfsServiceClient(cc grpc.ClientConnInterface) HyDfsServiceClient {
	return &hyDfsServiceClient{cc}
}

func (c *hyDfsServiceClient) ExecuteFileSystem(ctx context.Context, in *FileRequest, opts ...grpc.CallOption) (*FilResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(FilResponse)
	err := c.cc.Invoke(ctx, HyDfsService_ExecuteFileSystem_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *hyDfsServiceClient) SendFile(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[FileSendRequest, FileSendResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &HyDfsService_ServiceDesc.Streams[0], HyDfsService_SendFile_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[FileSendRequest, FileSendResponse]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type HyDfsService_SendFileClient = grpc.ClientStreamingClient[FileSendRequest, FileSendResponse]

func (c *hyDfsServiceClient) ReceiveFile(ctx context.Context, in *FileReceiveRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[FileReceiveReponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &HyDfsService_ServiceDesc.Streams[1], HyDfsService_ReceiveFile_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[FileReceiveRequest, FileReceiveReponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type HyDfsService_ReceiveFileClient = grpc.ServerStreamingClient[FileReceiveReponse]

func (c *hyDfsServiceClient) ReplicateLogEntry(ctx context.Context, in *ReplicateRequest, opts ...grpc.CallOption) (*ReplicateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(ReplicateResponse)
	err := c.cc.Invoke(ctx, HyDfsService_ReplicateLogEntry_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *hyDfsServiceClient) ReadFile(ctx context.Context, in *FileReadRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[FileReadResponse], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &HyDfsService_ServiceDesc.Streams[2], HyDfsService_ReadFile_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[FileReadRequest, FileReadResponse]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type HyDfsService_ReadFileClient = grpc.ServerStreamingClient[FileReadResponse]

func (c *hyDfsServiceClient) MultiAppend(ctx context.Context, in *FileTransferRequest, opts ...grpc.CallOption) (*MultiAppendResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MultiAppendResponse)
	err := c.cc.Invoke(ctx, HyDfsService_MultiAppend_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *hyDfsServiceClient) MergeFile(ctx context.Context, in *FileTransferRequest, opts ...grpc.CallOption) (*MergeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MergeResponse)
	err := c.cc.Invoke(ctx, HyDfsService_MergeFile_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *hyDfsServiceClient) NotifyPrimaryToMerge(ctx context.Context, in *MergeRequest, opts ...grpc.CallOption) (*MergeResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MergeResponse)
	err := c.cc.Invoke(ctx, HyDfsService_NotifyPrimaryToMerge_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *hyDfsServiceClient) WriteStreamResults(ctx context.Context, in *FileTransferRequest, opts ...grpc.CallOption) (*FileSendResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(FileSendResponse)
	err := c.cc.Invoke(ctx, HyDfsService_WriteStreamResults_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// HyDfsServiceServer is the server API for HyDfsService service.
// All implementations must embed UnimplementedHyDfsServiceServer
// for forward compatibility.
type HyDfsServiceServer interface {
	ExecuteFileSystem(context.Context, *FileRequest) (*FilResponse, error)
	SendFile(grpc.ClientStreamingServer[FileSendRequest, FileSendResponse]) error
	ReceiveFile(*FileReceiveRequest, grpc.ServerStreamingServer[FileReceiveReponse]) error
	ReplicateLogEntry(context.Context, *ReplicateRequest) (*ReplicateResponse, error)
	ReadFile(*FileReadRequest, grpc.ServerStreamingServer[FileReadResponse]) error
	MultiAppend(context.Context, *FileTransferRequest) (*MultiAppendResponse, error)
	MergeFile(context.Context, *FileTransferRequest) (*MergeResponse, error)
	NotifyPrimaryToMerge(context.Context, *MergeRequest) (*MergeResponse, error)
	WriteStreamResults(context.Context, *FileTransferRequest) (*FileSendResponse, error)
	mustEmbedUnimplementedHyDfsServiceServer()
}

// UnimplementedHyDfsServiceServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedHyDfsServiceServer struct{}

func (UnimplementedHyDfsServiceServer) ExecuteFileSystem(context.Context, *FileRequest) (*FilResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExecuteFileSystem not implemented")
}
func (UnimplementedHyDfsServiceServer) SendFile(grpc.ClientStreamingServer[FileSendRequest, FileSendResponse]) error {
	return status.Errorf(codes.Unimplemented, "method SendFile not implemented")
}
func (UnimplementedHyDfsServiceServer) ReceiveFile(*FileReceiveRequest, grpc.ServerStreamingServer[FileReceiveReponse]) error {
	return status.Errorf(codes.Unimplemented, "method ReceiveFile not implemented")
}
func (UnimplementedHyDfsServiceServer) ReplicateLogEntry(context.Context, *ReplicateRequest) (*ReplicateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ReplicateLogEntry not implemented")
}
func (UnimplementedHyDfsServiceServer) ReadFile(*FileReadRequest, grpc.ServerStreamingServer[FileReadResponse]) error {
	return status.Errorf(codes.Unimplemented, "method ReadFile not implemented")
}
func (UnimplementedHyDfsServiceServer) MultiAppend(context.Context, *FileTransferRequest) (*MultiAppendResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MultiAppend not implemented")
}
func (UnimplementedHyDfsServiceServer) MergeFile(context.Context, *FileTransferRequest) (*MergeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method MergeFile not implemented")
}
func (UnimplementedHyDfsServiceServer) NotifyPrimaryToMerge(context.Context, *MergeRequest) (*MergeResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method NotifyPrimaryToMerge not implemented")
}
func (UnimplementedHyDfsServiceServer) WriteStreamResults(context.Context, *FileTransferRequest) (*FileSendResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method WriteStreamResults not implemented")
}
func (UnimplementedHyDfsServiceServer) mustEmbedUnimplementedHyDfsServiceServer() {}
func (UnimplementedHyDfsServiceServer) testEmbeddedByValue()                      {}

// UnsafeHyDfsServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to HyDfsServiceServer will
// result in compilation errors.
type UnsafeHyDfsServiceServer interface {
	mustEmbedUnimplementedHyDfsServiceServer()
}

func RegisterHyDfsServiceServer(s grpc.ServiceRegistrar, srv HyDfsServiceServer) {
	// If the following call pancis, it indicates UnimplementedHyDfsServiceServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&HyDfsService_ServiceDesc, srv)
}

func _HyDfsService_ExecuteFileSystem_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FileRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HyDfsServiceServer).ExecuteFileSystem(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HyDfsService_ExecuteFileSystem_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HyDfsServiceServer).ExecuteFileSystem(ctx, req.(*FileRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _HyDfsService_SendFile_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(HyDfsServiceServer).SendFile(&grpc.GenericServerStream[FileSendRequest, FileSendResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type HyDfsService_SendFileServer = grpc.ClientStreamingServer[FileSendRequest, FileSendResponse]

func _HyDfsService_ReceiveFile_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(FileReceiveRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(HyDfsServiceServer).ReceiveFile(m, &grpc.GenericServerStream[FileReceiveRequest, FileReceiveReponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type HyDfsService_ReceiveFileServer = grpc.ServerStreamingServer[FileReceiveReponse]

func _HyDfsService_ReplicateLogEntry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(ReplicateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HyDfsServiceServer).ReplicateLogEntry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HyDfsService_ReplicateLogEntry_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HyDfsServiceServer).ReplicateLogEntry(ctx, req.(*ReplicateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _HyDfsService_ReadFile_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(FileReadRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(HyDfsServiceServer).ReadFile(m, &grpc.GenericServerStream[FileReadRequest, FileReadResponse]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type HyDfsService_ReadFileServer = grpc.ServerStreamingServer[FileReadResponse]

func _HyDfsService_MultiAppend_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FileTransferRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HyDfsServiceServer).MultiAppend(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HyDfsService_MultiAppend_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HyDfsServiceServer).MultiAppend(ctx, req.(*FileTransferRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _HyDfsService_MergeFile_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FileTransferRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HyDfsServiceServer).MergeFile(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HyDfsService_MergeFile_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HyDfsServiceServer).MergeFile(ctx, req.(*FileTransferRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _HyDfsService_NotifyPrimaryToMerge_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MergeRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HyDfsServiceServer).NotifyPrimaryToMerge(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HyDfsService_NotifyPrimaryToMerge_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HyDfsServiceServer).NotifyPrimaryToMerge(ctx, req.(*MergeRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _HyDfsService_WriteStreamResults_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FileTransferRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(HyDfsServiceServer).WriteStreamResults(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: HyDfsService_WriteStreamResults_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(HyDfsServiceServer).WriteStreamResults(ctx, req.(*FileTransferRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// HyDfsService_ServiceDesc is the grpc.ServiceDesc for HyDfsService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var HyDfsService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "hydfs.HyDfsService",
	HandlerType: (*HyDfsServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "ExecuteFileSystem",
			Handler:    _HyDfsService_ExecuteFileSystem_Handler,
		},
		{
			MethodName: "ReplicateLogEntry",
			Handler:    _HyDfsService_ReplicateLogEntry_Handler,
		},
		{
			MethodName: "MultiAppend",
			Handler:    _HyDfsService_MultiAppend_Handler,
		},
		{
			MethodName: "MergeFile",
			Handler:    _HyDfsService_MergeFile_Handler,
		},
		{
			MethodName: "NotifyPrimaryToMerge",
			Handler:    _HyDfsService_NotifyPrimaryToMerge_Handler,
		},
		{
			MethodName: "WriteStreamResults",
			Handler:    _HyDfsService_WriteStreamResults_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "SendFile",
			Handler:       _HyDfsService_SendFile_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "ReceiveFile",
			Handler:       _HyDfsService_ReceiveFile_Handler,
			ServerStreams: true,
		},
		{
			StreamName:    "ReadFile",
			Handler:       _HyDfsService_ReadFile_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "mp3.proto",
}
