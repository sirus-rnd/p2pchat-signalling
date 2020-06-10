package signaling

import (
	"context"

	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/protos"
)

const (
	// SDPOferrCommand emitted when peer send sdp offer command to another peer
	SDPOferrCommand = "chat.sdp.offer"
	// SDPAnswerCommand emitted when peer send sdp answer command to another peer oferring
	SDPAnswerCommand = "chat.sdp.answer"
	// ICECandidateOffer emitted during candidate negotiations between peers
	ICECandidateOffer = "chat.ice-candidate"
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

// ICEOffer contain ICE candidate offer from user to another user
type ICEOffer struct {
	From      string        `json:"from"`
	To        string        `json:"to"`
	Candidate *ICECandidate `json:"candidate"`
}

// ICECandidate interface—part of the WebRTC API—represents a candidate Internet Connectivity Establishment (ICE) configuration which may be used to establish an RTCPeerConnection
type ICECandidate struct {
	Candidate        string `json:"candidate"`
	Component        string `json:"component"`
	Foundation       string `json:"foundation"`
	Port             int32  `json:"port"`
	Priority         int32  `json:"priority"`
	Protocol         string `json:"protocol"`
	RelatedAddress   string `json:"relatedAddress"`
	RelatedPort      int32  `json:"relatedPort"`
	SDPMLineIndex    int32  `json:"sdpMLineIndex"`
	SDPMid           string `json:"sdpMid"`
	TCPType          string `json:"tcpType"`
	Type             string `json:"type"`
	UsernameFragment string `json:"usernameFragment"`
}

// ISignaling act as intermediary to give signal from peer to other peer
// so they be able communicate to each other within same channel / rooms
type ISignaling interface {
	GetCommands() chan *SDPCommand
	SetCommands(commands chan *SDPCommand)
	GetRoomEvents() chan *room.RoomEvent
	SetRoomEvents(chan *room.RoomEvent)
	GetICEOffers() chan *ICEOffer
	SetICEOffers(offers chan *ICEOffer)
	MyProfile(ctx context.Context) (*protos.Profile, error)
	UpdateProfile(ctx context.Context, param *protos.UpdateProfileParam) (*protos.Profile, error)
	MyRooms(ctx context.Context) (*protos.Rooms, error)
	MyRoomInfo(ctx context.Context, param *protos.GetRoomParam) (*protos.Room, error)
	OfferSDP(ctx context.Context, param *protos.SDPParam) error
	AnswerSDP(ctx context.Context, param *protos.SDPParam) error
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
	SendICECandidate(ctx context.Context, param *protos.ICEParam) error
	SubscribeICECandidate(
		ctx context.Context,
		offers <-chan *ICEOffer,
		protoOffers chan<- *protos.ICEOffer,
	) error
}
