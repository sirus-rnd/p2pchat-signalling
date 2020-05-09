package signaling

import (
	"context"

	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/protos"
)

const (
	// SDPOferrCommand emitted when peer send sdp offer command to another peer
	SDPOferrCommand = "chat.sdp.oferr"
	// SDPAnswerCommand emitted when peer send sdp answer command to another peer oferring
	SDPAnswerCommand = "chat.sdp.answer"
)

const (
	SDPOffer    = "offer"
	SDPAnswer   = "answer"
	SDPPranswer = "pranswer"
	SDPRollback = "rollback"
)

// SDPCommand related to session description command emitted by peers
type SDPCommand struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Type        string `json:"type"`
	Description string `json:"description"`
}

// ISignalingService is service related to signaling from peer to peer
type ISignalingService interface {
	GetCommands() chan *SDPCommand
	SetCommands(commands chan *SDPCommand)
	GetRoomEvents() chan *room.RoomEvent
	SetRoomEvents(chan *room.RoomEvent)
	MyProfile(ctx context.Context) (*protos.Profile, error)
	UpdateProfile(ctx context.Context, param protos.UpdateProfileParam) (*protos.Profile, error)
	MyRooms(ctx context.Context) (*protos.Rooms, error)
	MyRoomInfo(ctx context.Context, param protos.GetRoomParam) (*protos.Room, error)
	OfferSDP(ctx context.Context, param protos.SDPParam) error
	AnswerSDP(ctx context.Context, param protos.SDPParam) error
	SubscribeSDPCommand(
		ctx context.Context,
		commands <-chan *SDPCommand,
		protoEvents chan<- *protos.SDP,
	) error
	SubscribeRoomEvent(
		ctx context.Context,
		commands <-chan *room.RoomEvent,
		protoEvents chan<- *protos.RoomEvent,
	) error
}
