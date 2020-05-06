package room

import (
	"context"

	"go.sirus.dev/p2p-comm/signalling/protos"
)

// IRoomService is service related to room & authorization management
type IRoomService interface {
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
