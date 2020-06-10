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
	signalingPort int,
	roomMngrSvc *RoomManagementService,
	roomMngrPort int,
) *Server {
	return &Server{
		SignalingSvc:  signalingSvc,
		SignalingPort: signalingPort,
		RoomMngrSvc:   roomMngrSvc,
		RoomMngrPort:  roomMngrPort,
	}
}

// Server act as transport layer
type Server struct {
	SignalingSvc    *SignalingService
	SignalingServer *grpc.Server
	SignalingPort   int
	RoomMngrSvc     *RoomManagementService
	RoomMngrServer  *grpc.Server
	RoomMngrPort    int
}

// StartSignaling will start serve signaling service in GRPC server
func (s *Server) StartSignaling() error {
	options := []grpc.ServerOption{}
	s.SignalingServer = grpc.NewServer(options...)
	go s.SignalingSvc.Run()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.SignalingPort))
	if err != nil {
		return err
	}
	protos.RegisterSignalingServiceServer(s.SignalingServer, s.SignalingSvc)

	return s.SignalingServer.Serve(lis)
}

// StartRoomManager will start serve room management service in GRPC server
func (s *Server) StartRoomManager() error {
	options := []grpc.ServerOption{}
	s.RoomMngrServer = grpc.NewServer(options...)
	go s.RoomMngrSvc.Run()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.RoomMngrPort))
	if err != nil {
		return err
	}
	protos.RegisterRoomManagementServiceServer(s.RoomMngrServer, s.RoomMngrSvc)
	return s.RoomMngrServer.Serve(lis)
}

// Stop will stop all gRPC server
func (s *Server) Stop() error {
	if s.SignalingServer != nil {
		s.SignalingServer.Stop()
	}
	if s.RoomMngrServer != nil {
		s.RoomMngrServer.Stop()
	}
	return nil
}
