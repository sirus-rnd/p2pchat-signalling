package server

import (
	"context"
	"time"

	"github.com/nats-io/nats.go"
	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/protos"
	"go.uber.org/zap"
)

// NewRoomManagementService will create new instance of RoomManagementService
func NewRoomManagementService(
	roomManager room.IRoomManager,
	logger *zap.SugaredLogger,
	nats *nats.EncodedConn,
	eventNamespace string,
	accessSecret string,
) *RoomManagementService {
	return &RoomManagementService{
		RoomManager:    roomManager,
		Logger:         logger,
		Nats:           nats,
		EventNamespace: eventNamespace,
		AccessSecret:   accessSecret,
	}
}

// RoomManagementService will implement room management server
type RoomManagementService struct {
	protos.UnimplementedRoomManagementServiceServer
	RoomManager    room.IRoomManager
	Logger         *zap.SugaredLogger
	Nats           *nats.EncodedConn
	EventNamespace string
	AccessSecret   string
}

// RegisterUser will register new user that can participate in a room
func (s *RoomManagementService) RegisterUser(
	ctx context.Context,
	req *protos.NewUserParam,
) (*protos.User, error) {
	return s.RoomManager.RegisterUser(ctx, req)
}

// GetUser return user information by it's id
func (s *RoomManagementService) GetUser(
	ctx context.Context,
	req *protos.GetUserParam,
) (*protos.User, error) {
	return s.RoomManager.GetUser(ctx, req)
}

// GetUsers return list of user registered on system
func (s *RoomManagementService) GetUsers(
	ctx context.Context,
	req *protos.PaginationParam,
) (*protos.Users, error) {
	return s.RoomManager.GetUsers(ctx, req)
}

// GetUserAccessToken will return access token used by peer as an user identification form
func (s *RoomManagementService) GetUserAccessToken(
	ctx context.Context,
	req *protos.GetUserParam,
) (*protos.UserAccessToken, error) {
	return s.RoomManager.GetUserAccessToken(ctx, req)
}

// UpdateUserProfile will update user profile informations
func (s *RoomManagementService) UpdateUserProfile(
	ctx context.Context,
	req *protos.UpdateUserProfileParam,
) (*protos.User, error) {
	return s.RoomManager.UpdateUserProfile(ctx, req)
}

// RemoveUser will remove a user from system
func (s *RoomManagementService) RemoveUser(
	ctx context.Context,
	req *protos.GetUserParam,
) (*protos.User, error) {
	return s.RoomManager.RemoveUser(ctx, req)
}

// CreateRoom will create a new room for user to participate in
func (s *RoomManagementService) CreateRoom(
	ctx context.Context,
	req *protos.NewRoomParam,
) (*protos.Room, error) {
	return s.RoomManager.Create(ctx, req)
}

// GetRoom will return a room and it's participant by it's id
func (s *RoomManagementService) GetRoom(
	ctx context.Context,
	req *protos.GetRoomParam,
) (*protos.Room, error) {
	return s.RoomManager.GetByID(ctx, req)
}

// GetRooms will return all room registered on system
func (s *RoomManagementService) GetRooms(
	ctx context.Context,
	req *protos.PaginationParam,
) (*protos.Rooms, error) {
	return s.RoomManager.GetAll(ctx, req)
}

// UpdateRoomProfile will update room profile like description and photo
func (s *RoomManagementService) UpdateRoomProfile(
	ctx context.Context,
	req *protos.UpdateRoomProfileParam,
) (*protos.Room, error) {
	return s.RoomManager.UpdateProfile(ctx, req)
}

// AddUserToRoom to a room
func (s *RoomManagementService) AddUserToRoom(
	ctx context.Context,
	req *protos.UserRoomParam,
) (*protos.Room, error) {
	return s.RoomManager.AddUser(ctx, req)
}

// KickUserFromRoom from a room
func (s *RoomManagementService) KickUserFromRoom(
	ctx context.Context,
	req *protos.UserRoomParam,
) (*protos.Room, error) {
	return s.RoomManager.KickUser(ctx, req)
}

// DestroyRoom will destroy a room
func (s *RoomManagementService) DestroyRoom(
	ctx context.Context,
	req *protos.GetRoomParam,
) (*protos.Room, error) {
	return s.RoomManager.Destroy(ctx, req)
}

// Run signaling service
// this should be called before serve service to network
// - wire room event and SDP command to NATS message bus
func (s *RoomManagementService) Run() {
	// wired nats to publish channel
	r1c := make(chan *room.RoomEvent)
	defer close(r1c)
	s.RoomManager.SetEvents(r1c)
	go s.PublishEvent(r1c)

	// keep it running
	for {
		time.Sleep(time.Second)
	}
}

// PublishEvent will publish room events to NATS / STAN
func (s *RoomManagementService) PublishEvent(
	events chan *room.RoomEvent,
) error {
	for event := range events {
		if event == nil {
			continue
		}
		subject := s.EventNamespace + "." + event.Event
		err := s.Nats.Publish(subject, event.Payload)
		if err != nil {
			s.Logger.Error(err)
			continue
		}
	}
	return nil
}
