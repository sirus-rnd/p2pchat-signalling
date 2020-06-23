package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/nats-io/nats.go"
	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/pkg/signaling"
	"go.sirus.dev/p2p-comm/signalling/pkg/utils"
	"go.sirus.dev/p2p-comm/signalling/protos"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// NewSignalingService will create new instance of SignalingService
func NewSignalingService(
	signaling signaling.ISignaling,
	logger *zap.SugaredLogger,
	nats *nats.EncodedConn,
	eventNamespace string,
	accessSecret string,
) *SignalingService {
	return &SignalingService{
		Signaling:      signaling,
		Logger:         logger,
		Nats:           nats,
		EventNamespace: eventNamespace,
		AccessSecret:   accessSecret,
	}
}

// SignalingService will implement signaling service server
type SignalingService struct {
	protos.UnimplementedSignalingServiceServer
	Signaling      signaling.ISignaling
	Logger         *zap.SugaredLogger
	Nats           *nats.EncodedConn
	EventNamespace string
	AccessSecret   string
}

// SetUserContext will set user access context for each grpc calls
// based on access token at metadata
func (s *SignalingService) SetUserContext(ctx context.Context) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "metadata not found")
	}
	tokens := md.Get("token")
	if len(tokens) == 0 {
		return nil, status.Error(codes.PermissionDenied, "token not found on metadata")
	}
	claims, err := utils.ValidateToken(s.AccessSecret, tokens[0])
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, "invalid token")
	}
	userID, ok := claims[room.UserIDKey].(string)
	if !ok {
		return nil, status.Error(codes.PermissionDenied, "user context not found")
	}
	ctx = context.WithValue(ctx, room.UserIDKey, userID)
	return ctx, nil
}

// GetProfile return user profile and configuration
func (s *SignalingService) GetProfile(
	ctx context.Context,
	req *empty.Empty,
) (*protos.Profile, error) {
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.Signaling.MyProfile(ctx)
}

// UpdateProfile will update user profile information like photo, name etc.
func (s *SignalingService) UpdateProfile(
	ctx context.Context,
	req *protos.UpdateProfileParam,
) (*protos.Profile, error) {
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.Signaling.UpdateProfile(ctx, req)
}

// GetMyRooms will return list of room peer participates in
func (s *SignalingService) GetMyRooms(
	ctx context.Context,
	req *empty.Empty,
) (*protos.Rooms, error) {
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.Signaling.MyRooms(ctx)
}

// GetRoom will return detailed information about a room peer participates in
func (s *SignalingService) GetRoom(
	ctx context.Context,
	req *protos.GetRoomParam,
) (*protos.Room, error) {
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.Signaling.MyRoomInfo(ctx, req)
}

// GetUser return user information by it's id
func (s *SignalingService) GetUser(
	ctx context.Context,
	req *protos.GetUserParam,
) (*protos.User, error) {
	return s.Signaling.GetUser(ctx, req)
}

// OfferSessionDescription will send session description offer from a peer to target peers
func (s *SignalingService) OfferSessionDescription(
	ctx context.Context,
	req *protos.SDPParam,
) (*empty.Empty, error) {
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	err = s.Signaling.OfferSDP(ctx, req)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// AnswerSessionDescription will answer SDP offer from a peer
func (s *SignalingService) AnswerSessionDescription(
	ctx context.Context,
	req *protos.SDPParam,
) (*empty.Empty, error) {
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	err = s.Signaling.AnswerSDP(ctx, req)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// SubscribeSDPCommand will subscribe SDP commands from other peers
func (s *SignalingService) SubscribeSDPCommand(
	req *empty.Empty,
	srv protos.SignalingService_SubscribeSDPCommandServer,
) error {
	ctx := srv.Context()
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return err
	}
	commands := make(chan *signaling.SDPCommand)
	sdps := make(chan *protos.SDP)
	var errc error
	sub, err := s.SubscribeNatsSDPCommand(commands, nil)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	go func() {
		err := s.Signaling.SubscribeSDPCommand(ctx, commands, sdps)
		if err != nil {
			errc = err
		}
		close(commands)
		close(sdps)
	}()
	// uggly patch to ACK request
	// send ack to subscriber to trigger on header resolver
	srv.Send(&protos.SDP{})
	// get real SDP response
	for sdp := range sdps {
		err := srv.Send(sdp)
		if err != nil {
			return err
		}
	}
	return errc
}

// SubscribeRoomEvent will subscribe changes in a rooms or channel
func (s *SignalingService) SubscribeRoomEvent(
	req *empty.Empty,
	srv protos.SignalingService_SubscribeRoomEventServer,
) error {
	ctx := srv.Context()
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return err
	}
	events := make(chan *room.RoomEvent)
	protoEvents := make(chan *protos.RoomEvent)
	var errc error
	sub, err := s.SubscribeNatsRoomEvent(events, nil)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	go func() {
		err := s.Signaling.SubscribeRoomEvent(ctx, events, protoEvents)
		if err != nil {
			errc = err
		}
		close(events)
		close(protoEvents)
	}()
	for event := range protoEvents {
		err := srv.Send(event)
		if err != nil {
			return err
		}
	}
	return errc
}

// SendICECandidate will send ICE candidate to a user
func (s *SignalingService) SendICECandidate(
	ctx context.Context,
	req *protos.ICEParam,
) (*empty.Empty, error) {
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	err = s.Signaling.SendICECandidate(ctx, req)
	if err != nil {
		return nil, err
	}
	return &empty.Empty{}, nil
}

// SubscribeICECandidate will subscribe ICE candidate offers to a user
func (s *SignalingService) SubscribeICECandidate(
	req *empty.Empty,
	srv protos.SignalingService_SubscribeICECandidateServer,
) error {
	ctx := srv.Context()
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return err
	}
	offers := make(chan *signaling.ICEOffer)
	protoOffers := make(chan *protos.ICEOffer)
	var errc error
	sub, err := s.SubscribeNatsICEOffer(offers, nil)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()
	go func() {
		err := s.Signaling.SubscribeICECandidate(ctx, offers, protoOffers)
		if err != nil {
			errc = err
		}
		close(offers)
		close(protoOffers)
	}()
	for offer := range protoOffers {
		err := srv.Send(offer)
		if err != nil {
			return err
		}
	}
	return errc
}

// SubscribeOnlineStatus act as pull-on switch mechanism for user online state
// when user call this function user status will change to online
// status will pull back to offline after this function exit
func (s *SignalingService) SubscribeOnlineStatus(
	srv protos.SignalingService_SubscribeOnlineStatusServer,
) error {
	ctx := srv.Context()
	ctx, err := s.SetUserContext(ctx)
	if err != nil {
		return err
	}
	statusChanges := make(chan *signaling.OnlineStatus)
	protoStatusChanges := make(chan *protos.OnlineStatus)
	var errc error
	sub, err := s.SubscribeNatsOnlineStatus(statusChanges, nil)
	if err != nil {
		return err
	}
	defer sub.Unsubscribe()

	// forward heartbeat
	heartbeat := make(chan *protos.Heartbeat)
	go func() error {
		for {
			beat, err := srv.Recv()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			heartbeat <- beat
		}
	}()

	go func() {
		err := s.Signaling.SubscribeOnlineStatus(ctx, heartbeat, statusChanges, protoStatusChanges)
		if err != nil {
			errc = err
		}
		close(statusChanges)
		close(protoStatusChanges)
	}()
	for statusChange := range protoStatusChanges {
		err := srv.Send(statusChange)
		if err != nil {
			return err
		}
	}
	return errc
}

// Run signaling service
// this should be called before serve service to network
// - wire room event and SDP command to NATS message bus
func (s *SignalingService) Run() {
	// wired nats to publish channel
	r1c := make(chan *room.RoomEvent)
	s1c := make(chan *signaling.SDPCommand)
	ioc := make(chan *signaling.ICEOffer)
	uol := make(chan *signaling.OnlineStatus)
	defer close(r1c)
	defer close(s1c)
	defer close(ioc)
	defer close(uol)
	s.Signaling.SetRoomEvents(r1c)
	s.Signaling.SetCommands(s1c)
	s.Signaling.SetICEOffers(ioc)
	s.Signaling.SetOnlineStatus(uol)
	go s.PublishRoomEvent(r1c)
	go s.PublishSDPCommand(s1c)
	go s.PublishICEOffer(ioc)
	go s.PublishOnlineStatus(uol)

	// keep it running
	for {
		time.Sleep(time.Second)
	}
}

// PublishRoomEvent will publish room events to NATS / STAN
func (s *SignalingService) PublishRoomEvent(
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

// SubscribeNatsRoomEvent will subscribe native nats event
// parsed event payload and passed it to room events channel
func (s *SignalingService) SubscribeNatsRoomEvent(
	events chan<- *room.RoomEvent,
	queue *string,
) (*nats.Subscription, error) {
	handler := func(m *nats.Msg) {
		eventPattern := regexp.MustCompile(fmt.Sprintf("^%s.", s.EventNamespace))
		subject := eventPattern.ReplaceAllString(m.Subject, "")
		var payload interface{}

		switch {

		// participant event on room
		case utils.ContainString([]string{
			room.UserLeftRoom,
			room.UserJoinedRoom,
		}, subject):
			payload = &room.RoomParticipantEventPayload{}
			err := json.Unmarshal(m.Data, payload)
			if err != nil {
				s.Logger.Error(err)
				return
			}

		// room instance event
		case utils.ContainString([]string{
			room.RoomCreated,
			room.RoomProfileUpdated,
			room.RoomDestroyed,
		}, subject):
			payload = &room.RoomInstanceEventPayload{}
			err := json.Unmarshal(m.Data, payload)
			if err != nil {
				s.Logger.Error(err)
				return
			}

		// user instance event
		case utils.ContainString([]string{
			room.UserRegistered,
			room.UserProfileUpdated,
			room.UserRemoved,
		}, subject):
			payload = &room.UserInstanceEventPayload{}
			err := json.Unmarshal(m.Data, payload)
			if err != nil {
				s.Logger.Error(err)
				return
			}

		default:
			return
		}

		// send event to channel
		if payload == nil {
			return
		}
		events <- &room.RoomEvent{
			Event:   subject,
			Payload: payload,
			Time:    time.Now(),
		}
	}
	if queue != nil {
		return s.Nats.QueueSubscribe(s.EventNamespace+".chat.room.*", *queue, handler)
	}
	return s.Nats.Subscribe(s.EventNamespace+".chat.room.*", handler)
}

// SDPPayload data structure on NATS message
type SDPPayload struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Description string `json:"description"`
}

// PublishSDPCommand will publish SDP command to NATS
func (s *SignalingService) PublishSDPCommand(
	commands chan *signaling.SDPCommand,
) error {
	for command := range commands {
		if command == nil {
			continue
		}
		subject := s.EventNamespace + ".chat.sdp." + command.Type
		err := s.Nats.Publish(subject, &SDPPayload{
			From:        command.From,
			To:          command.To,
			Description: command.Description,
		})
		if err != nil {
			s.Logger.Error(err)
			continue
		}
	}
	return nil
}

// SubscribeNatsSDPCommand will subscribe native nats message
// parsed the payload and passed it to SDP commands channel
func (s *SignalingService) SubscribeNatsSDPCommand(
	commands chan<- *signaling.SDPCommand,
	queue *string,
) (*nats.Subscription, error) {
	handler := func(m *nats.Msg) {
		eventPattern := regexp.MustCompile(fmt.Sprintf("^%s.chat.sdp.", s.EventNamespace))
		SDPType := eventPattern.ReplaceAllString(m.Subject, "")
		payload := &SDPPayload{}
		err := json.Unmarshal(m.Data, payload)
		if err != nil {
			s.Logger.Error(err)
			return
		}
		commands <- &signaling.SDPCommand{
			Type:        SDPType,
			Description: payload.Description,
			From:        payload.From,
			To:          payload.To,
		}
	}
	if queue != nil {
		return s.Nats.QueueSubscribe(s.EventNamespace+".chat.sdp.*", *queue, handler)
	}
	return s.Nats.Subscribe(s.EventNamespace+".chat.sdp.*", handler)
}

// PublishICEOffer will publish ICE candidate offer to NATS
func (s *SignalingService) PublishICEOffer(
	offers chan *signaling.ICEOffer,
) error {
	for offer := range offers {
		if offer == nil {
			continue
		}
		subject := s.EventNamespace + "." + signaling.ICECandidateOffer
		err := s.Nats.Publish(subject, offer)
		if err != nil {
			s.Logger.Error(err)
			continue
		}
	}
	return nil
}

// SubscribeNatsICEOffer will subscribe native nats message
// parsed the payload and passed it to ICE Offer channel
func (s *SignalingService) SubscribeNatsICEOffer(
	offers chan<- *signaling.ICEOffer,
	queue *string,
) (*nats.Subscription, error) {
	handler := func(m *nats.Msg) {
		offer := &signaling.ICEOffer{}
		err := json.Unmarshal(m.Data, offer)
		if err != nil {
			s.Logger.Error(err)
			return
		}
		offers <- offer
	}
	if queue != nil {
		return s.Nats.QueueSubscribe(s.EventNamespace+"."+signaling.ICECandidateOffer, *queue, handler)
	}
	return s.Nats.Subscribe(s.EventNamespace+"."+signaling.ICECandidateOffer, handler)
}

// PublishOnlineStatus will publish user online status changes
func (s *SignalingService) PublishOnlineStatus(
	statusChanges chan *signaling.OnlineStatus,
) error {
	for status := range statusChanges {
		if status == nil {
			continue
		}
		subject := s.EventNamespace + "." + signaling.OnlineStatusChangeEvent
		err := s.Nats.Publish(subject, status)
		if err != nil {
			s.Logger.Error(err)
			continue
		}
	}
	return nil
}

// SubscribeNatsOnlineStatus will subscribe native nats message
// parsed the payload and passed it to online status channel
func (s *SignalingService) SubscribeNatsOnlineStatus(
	statusChanges chan *signaling.OnlineStatus,
	queue *string,
) (*nats.Subscription, error) {
	handler := func(m *nats.Msg) {
		statusChange := &signaling.OnlineStatus{}
		err := json.Unmarshal(m.Data, statusChange)
		if err != nil {
			s.Logger.Error(err)
			return
		}
		statusChanges <- statusChange
	}
	if queue != nil {
		return s.Nats.QueueSubscribe(s.EventNamespace+"."+signaling.OnlineStatusChangeEvent, *queue, handler)
	}
	return s.Nats.Subscribe(s.EventNamespace+"."+signaling.OnlineStatusChangeEvent, handler)
}
