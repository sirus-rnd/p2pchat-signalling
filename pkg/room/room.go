package room

import (
	"context"
	"time"

	"go.sirus.dev/p2p-comm/signalling/protos"
)

const (
	// UserLeftRoom emitted when user left a room
	UserLeftRoom = "chat.room.user-left"
	// UserJoinedRoom emitted when user joined on a room
	UserJoinedRoom = "chat.room.user-joined"
	// RoomCreated emitted when new room created
	RoomCreated = "chat.room.created"
	// RoomProfileUpdated emitted when room profile updated
	RoomProfileUpdated = "chat.room.updated"
	// RoomDestroyed emitted when room has been destroyed
	RoomDestroyed = "chat.room.destroyed"
	// UserRegistered emitted when new user registered
	UserRegistered = "chat.user.registered"
	// UserProfileUpdated emitted when user profile updated
	UserProfileUpdated = "chat.user.profile-updated"
	// UserRemoved emitted when user removed from system
	UserRemoved = "chat.user.removed"
)

// RoomEvent contain data emitted by events channel
type RoomEvent struct {
	Event   string      `json:"event"`
	Payload interface{} `json:"payload"`
	Time    time.Time   `json:"time"`
}

// RoomParticipantEventPayload is payload emittend on room participant related events
// like user left or join a room
type RoomParticipantEventPayload struct {
	UserID         string   `json:"user_id"`
	RoomID         string   `json:"room_id"`
	ParticipantIDs []string `json:"participant_ids"`
}

// RoomInstanceEventPayload is payload emittend on room instance related events
// like new room created, destroyed or profile updated
type RoomInstanceEventPayload struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Photo       string   `json:"photo"`
	Description string   `json:"description"`
	MemberIDs   []string `json:"member_ids"`
}

// UserInstanceEventPayload is payload emittend on user instance related events
// like new user created, destroyed or profile updated
type UserInstanceEventPayload struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Photo   string   `json:"photo"`
	RoomIDs []string `json:"room_ids"`
}

// IRoomService is service related to room & authorization management
type IRoomService interface {
	GetEvents() chan *RoomEvent
	SetEvents(chan *RoomEvent)
	RegisterUser(ctx context.Context, param protos.NewUserParam) (*protos.User, error)
	GetUser(ctx context.Context, param protos.GetUserParam) (*protos.User, error)
	GetUsers(ctx context.Context, param protos.PaginationParam) (*protos.Users, error)
	GetUserAccessToken(ctx context.Context, param protos.GetUserParam) (*protos.UserAccessToken, error)
	UpdateUserProfile(ctx context.Context, param protos.UpdateUserProfileParam) (*protos.User, error)
	RemoveUser(ctx context.Context, param protos.GetRoomParam) (*protos.User, error)
	Create(ctx context.Context, param protos.NewRoomParam) (*protos.Room, error)
	GetByID(ctx context.Context, param protos.GetRoomParam) (*protos.Room, error)
	GetAll(ctx context.Context, param protos.PaginationParam) (*protos.Rooms, error)
	UpdateProfile(ctx context.Context, param protos.UpdateRoomProfileParam) (*protos.Room, error)
	AddUser(ctx context.Context, param protos.UserRoomParam) (*protos.Room, error)
	KickUser(ctx context.Context, param protos.UserRoomParam) (*protos.Room, error)
	Destroy(ctx context.Context, param protos.GetRoomParam) (*protos.Room, error)
}
