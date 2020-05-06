package room

import (
	"context"
	"time"

	"go.sirus.dev/p2p-comm/signalling/protos"
)

// SDPEvent from message bus
type SDPEvent struct {
	RoomID      string    `json:"room_id"`
	UserID      string    `json:"user_id"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Time        time.Time `json:"time"`
}

// UserJoinLeaveRoomEvent from message bus
type UserJoinLeaveRoomEvent struct {
	UserID string    `json:"user_id"`
	RoomID string    `json:"room_id"`
	Time   time.Time `json:"time"`
}

// IUserService is service related to user management
type IUserService interface {
	MyProfile(ctx context.Context) (*protos.Profile, error)
	UpdateProfile(ctx context.Context, param protos.UpdateProfileParam) (*protos.Profile, error)
	MyRooms(ctx context.Context, param protos.PaginationParam) (*protos.Rooms, error)
	MyRoomInfo(ctx context.Context) (*protos.Room, error)
	OfferSDP(ctx context.Context, param protos.SDPParam) error
	AnswerSDP(ctx context.Context, param protos.SDPParam) error
	SubscribeSDPEvent(
		ctx context.Context,
		events <-chan *SDPEvent,
		protoEvents chan<- *protos.SDP,
	) error
	SubscribeUserJoinLeaveRoomEvent(
		ctx context.Context,
		events <-chan *UserJoinLeaveRoomEvent,
		protoEvents chan<- *protos.UserJoinLeaveRoomEvent,
	)
}
