// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.22.2
// source: service.proto

package pb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	TegenariaService_SetStatus_FullMethodName = "/pb.TegenariaService/SetStatus"
	TegenariaService_GetStatus_FullMethodName = "/pb.TegenariaService/GetStatus"
)

// TegenariaServiceClient is the client API for TegenariaService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type TegenariaServiceClient interface {
	SetStatus(ctx context.Context, in *StatusContorlRequest, opts ...grpc.CallOption) (*ResponseMessage, error)
	GetStatus(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ResponseMessage, error)
}

type tegenariaServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewTegenariaServiceClient(cc grpc.ClientConnInterface) TegenariaServiceClient {
	return &tegenariaServiceClient{cc}
}

func (c *tegenariaServiceClient) SetStatus(ctx context.Context, in *StatusContorlRequest, opts ...grpc.CallOption) (*ResponseMessage, error) {
	out := new(ResponseMessage)
	err := c.cc.Invoke(ctx, TegenariaService_SetStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *tegenariaServiceClient) GetStatus(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*ResponseMessage, error) {
	out := new(ResponseMessage)
	err := c.cc.Invoke(ctx, TegenariaService_GetStatus_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// TegenariaServiceServer is the server API for TegenariaService service.
// All implementations must embed UnimplementedTegenariaServiceServer
// for forward compatibility
type TegenariaServiceServer interface {
	SetStatus(context.Context, *StatusContorlRequest) (*ResponseMessage, error)
	GetStatus(context.Context, *emptypb.Empty) (*ResponseMessage, error)
	mustEmbedUnimplementedTegenariaServiceServer()
}

// UnimplementedTegenariaServiceServer must be embedded to have forward compatible implementations.
type UnimplementedTegenariaServiceServer struct {
}

func (UnimplementedTegenariaServiceServer) SetStatus(context.Context, *StatusContorlRequest) (*ResponseMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SetStatus not implemented")
}
func (UnimplementedTegenariaServiceServer) GetStatus(context.Context, *emptypb.Empty) (*ResponseMessage, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetStatus not implemented")
}
func (UnimplementedTegenariaServiceServer) mustEmbedUnimplementedTegenariaServiceServer() {}

// UnsafeTegenariaServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to TegenariaServiceServer will
// result in compilation errors.
type UnsafeTegenariaServiceServer interface {
	mustEmbedUnimplementedTegenariaServiceServer()
}

func RegisterTegenariaServiceServer(s grpc.ServiceRegistrar, srv TegenariaServiceServer) {
	s.RegisterService(&TegenariaService_ServiceDesc, srv)
}

func _TegenariaService_SetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(StatusContorlRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TegenariaServiceServer).SetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TegenariaService_SetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TegenariaServiceServer).SetStatus(ctx, req.(*StatusContorlRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _TegenariaService_GetStatus_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(TegenariaServiceServer).GetStatus(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: TegenariaService_GetStatus_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(TegenariaServiceServer).GetStatus(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

// TegenariaService_ServiceDesc is the grpc.ServiceDesc for TegenariaService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var TegenariaService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "pb.TegenariaService",
	HandlerType: (*TegenariaServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SetStatus",
			Handler:    _TegenariaService_SetStatus_Handler,
		},
		{
			MethodName: "GetStatus",
			Handler:    _TegenariaService_GetStatus_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "service.proto",
}
