package room

import (
	"context"

	"go.sirus.dev/p2p-comm/signalling/protos"
)

const (
	// SDPOferrCommand emitted when peer send sdp offer command to another peer
	SDPOferrCommand = "chat.sdp.oferr"
	// SDPAnswerCommand emitted when peer send sdp answer command to another peer oferring
	SDPAnswerCommand = "chat.sdp.answer"
)

const (
	SDPAnswer   = "answer"
	SDPOffer    = "offer"
	SDPPranswer = "pranswer"
	SDPRollback = "rollback"
)

// SDPCommand related to session description command emitted by peers
type SDPCommand struct {
	RoomID      string `json:"room_id"`
	From        string `json:"from"`
	To          string `json:"to"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// IUserService is service related to user management
type IUserService interface {
	GetCommands() <-chan *SDPCommand
	SetCommands(chan *SDPCommand)
	MyProfile(ctx context.Context) (*protos.Profile, error)
	UpdateProfile(ctx context.Context, param protos.UpdateProfileParam) (*protos.Profile, error)
	MyRooms(ctx context.Context, param protos.PaginationParam) (*protos.Rooms, error)
	MyRoomInfo(ctx context.Context) (*protos.Room, error)
	OfferSDP(ctx context.Context, param protos.SDPParam) error
	AnswerSDP(ctx context.Context, param protos.SDPParam) error
	SubscribeSDPCommand(
		ctx context.Context,
		commands <-chan *SDPCommand,
		protoEvents chan<- *protos.SDP,
	) error
	SubscribeRoomEvent(
		ctx context.Context,
		commands <-chan *RoomEvent,
		protoEvents chan<- *protos.RoomEvent,
	) error
}
