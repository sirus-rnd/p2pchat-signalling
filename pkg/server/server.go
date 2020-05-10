package server

import (
	"fmt"
	"net"

	"go.sirus.dev/p2p-comm/signalling/protos"
	"google.golang.org/grpc"
)

// New wil create new instance of server
func New(
	signalingSvc *SignalingService,
	roomManagerSvc *RoomManagementService,
	port int,
) *Server {
	return &Server{
		SignalingSvc:   signalingSvc,
		RoomManagerSvc: roomManagerSvc,
		Port:           port,
	}
}

// Server act as transport layer
type Server struct {
	SignalingSvc   *SignalingService
	RoomManagerSvc *RoomManagementService
	GRPCServer     *grpc.Server
	Port           int
}

// Start will start serve signaling & room management service in GRPC server
func (s *Server) Start() error {
	options := []grpc.ServerOption{}

	// create gRPC server
	s.GRPCServer = grpc.NewServer(options...)
	go s.SignalingSvc.Run()
	go s.RoomManagerSvc.Run()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		return err
	}
	protos.RegisterSignalingServiceServer(s.GRPCServer, s.SignalingSvc)
	protos.RegisterRoomManagementServiceServer(s.GRPCServer, s.RoomManagerSvc)
	return s.GRPCServer.Serve(lis)
}

// Stop will stop GRPC server
func (s *Server) Stop() error {
	if s.GRPCServer != nil {
		s.GRPCServer.Stop()
	}
	return nil
}
