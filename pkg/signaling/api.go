package signaling

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/jinzhu/gorm"
	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/protos"
	"go.uber.org/zap"
)

const (
	CredentialTypePassword = iota
	CredentialTypeOAuth
)

const (
	ContextInvalidError = "context invalid"
)

// NewAPI will create new instance of signaling API
func NewAPI(
	db *gorm.DB,
	logger *zap.SugaredLogger,
	ICEServers *[]*ICEServer,
) *API {
	return &API{
		DB:         db,
		Logger:     logger,
		ICEServers: ICEServers,
	}
}

// ICEServer define ICE server configuration for peer to make ICE candidates between them
type ICEServer struct {
	URL            string `json:"url"`
	Username       string `json:"username"`
	CredentialType int    `json:"credential_type"`
	Password       string `json:"password"`
	AccessToken    string `json:"access_token"`
	MacKey         string `json:"mac_key"`
}

// SDPTypeProtoToCommand mapping from  proto to command
var SDPTypeProtoToCommand = map[protos.SDPTypes]string{
	0: SDPOffer,
	1: SDPAnswer,
	2: SDPPranswer,
	3: SDPRollback,
}

// SDPTypeCommandToProto mapping from command to proto
var SDPTypeCommandToProto = map[string]protos.SDPTypes{
	SDPOffer:    protos.SDPTypes(0),
	SDPAnswer:   protos.SDPTypes(1),
	SDPPranswer: protos.SDPTypes(2),
	SDPRollback: protos.SDPTypes(3),
}

// API act as intermediate between peers,
// make signal between them so they can communicate
type API struct {
	DB         *gorm.DB
	Logger     *zap.SugaredLogger
	ICEServers *[]*ICEServer
	Commands   chan *SDPCommand
	Events     chan *room.RoomEvent
}

// GetCommands return SDP command channel
func (a *API) GetCommands() chan *SDPCommand {
	return a.Commands
}

// SetCommands will set SDP commands channel
func (a *API) SetCommands(commands chan *SDPCommand) {
	a.Commands = commands
}

// GetRoomEvents will return channel use to publish room events
func (a *API) GetRoomEvents() chan *room.RoomEvent {
	return a.Events
}

// SetRoomEvents will set channel use to publish room events
func (a *API) SetRoomEvents(events chan *room.RoomEvent) {
	a.Events = events
}

// GetUserContext will return user context for an invocation
func (a *API) GetUserContext(ctx context.Context) (*room.UserModel, error) {
	userID, ok := ctx.Value(room.UserIDKey).(string)
	if !ok {
		return nil, fmt.Errorf(ContextInvalidError)
	}
	user := &room.UserModel{}
	err := a.DB.
		Where(&room.UserModel{ID: userID}).
		First(user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(room.UserNotFoundError)
		}
		return nil, err
	}
	return user, nil
}

// MyProfile will return user profile and configuration
func (a *API) MyProfile(ctx context.Context) (*protos.Profile, error) {
	user, err := a.GetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	return a.GetMyProfile(user), nil
}

// GetMyProfile will add user & ICE server config to profile information
func (a *API) GetMyProfile(user *room.UserModel) *protos.Profile {
	// map ice server configurations
	servers := []*protos.ICEServer{}
	for _, ice := range *a.ICEServers {
		server := &protos.ICEServer{Url: ice.URL}
		server.CredentialType = protos.ICECredentialType(ice.CredentialType)
		server.Username = ice.Username
		server.Password = ice.Password
		server.AccessToken = ice.AccessToken
		server.MacKey = ice.MacKey
		servers = append(servers, server)
	}
	return &protos.Profile{
		Id:      user.ID,
		Name:    user.Name,
		Photo:   user.Photo,
		Servers: servers,
	}
}

// UpdateProfile will update user profile information like photo, name etc.
func (a *API) UpdateProfile(ctx context.Context, param *protos.UpdateProfileParam) (*protos.Profile, error) {
	user, err := a.GetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	// update user info
	user.Name = param.Name
	user.Photo = param.Photo
	// save user info
	err = a.DB.Save(user).Error
	if err != nil {
		return nil, err
	}
	// publish user profile updated events
	rooms := &[]room.RoomModel{}
	err = a.DB.Model(user).Related(rooms, "Rooms").Error
	if err != nil {
		return nil, err
	}
	roomIDs := []string{}
	for _, r := range *rooms {
		roomIDs = append(roomIDs, r.ID)
	}
	a.Events <- &room.RoomEvent{
		Time:  time.Now(),
		Event: room.UserProfileUpdated,
		Payload: &room.UserInstanceEventPayload{
			ID:      user.ID,
			Name:    user.Name,
			Photo:   user.Photo,
			RoomIDs: roomIDs,
		},
	}
	// return profile information
	return a.GetMyProfile(user), nil
}

// MyRooms will return list of room peer participates in
func (a *API) MyRooms(ctx context.Context) (*protos.Rooms, error) {
	user, err := a.GetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	// get room of this user
	datas := []room.RoomModel{}
	err = a.DB.
		Model(user).
		Related(&datas, "Rooms").
		Error
	if err != nil {
		return nil, err
	}
	count := a.DB.
		Model(user).
		Association("Rooms").
		Count()

	// add member information to the datas
	roomIds := []string{}
	for _, data := range datas {
		roomIds = append(roomIds, data.ID)
	}
	err = a.DB.Preload("Members").
		Find(&datas, "id IN (?)", roomIds).
		Error
	if err != nil {
		return nil, err
	}
	rooms := []*protos.Room{}
	for _, data := range datas {
		r := room.RoomModelToProto(&data)
		rooms = append(rooms, r)
	}
	return &protos.Rooms{
		Rooms: rooms,
		Count: uint64(count),
	}, nil
}

// MyRoomInfo will return detailed information about a room peer participates in
func (a *API) MyRoomInfo(ctx context.Context, param *protos.GetRoomParam) (*protos.Room, error) {
	user, err := a.GetUserContext(ctx)
	if err != nil {
		return nil, err
	}
	// get room of this user
	r := &room.RoomModel{}
	err = a.DB.Preload("Members").
		First(r, "id = ?", param.Id).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(room.RoomNotFoundError)
		}
		return nil, err
	}
	// check if user are member this room
	exist := false
	for _, member := range r.Members {
		if member.ID == user.ID {
			exist = true
			break
		}
	}
	if !exist {
		return nil, fmt.Errorf(room.RoomNotFoundError)
	}
	// return room with members
	return room.RoomModelToProto(r), nil
}

// OfferSDP will send session description offer from a peer to target peers
func (a *API) OfferSDP(ctx context.Context, param *protos.SDPParam) error {
	user, err := a.GetUserContext(ctx)
	if err != nil {
		return err
	}
	a.Commands <- &SDPCommand{
		Type:        SDPOffer,
		From:        user.ID,
		To:          param.UserID,
		Description: param.Description,
	}
	return nil
}

// AnswerSDP will answer SDP offer from a peer
func (a *API) AnswerSDP(ctx context.Context, param *protos.SDPParam) error {
	user, err := a.GetUserContext(ctx)
	if err != nil {
		return err
	}
	a.Commands <- &SDPCommand{
		Type:        SDPAnswer,
		From:        user.ID,
		To:          param.UserID,
		Description: param.Description,
	}
	return nil
}

// SubscribeSDPCommand will subscribe SDP commands from other peers
func (a *API) SubscribeSDPCommand(
	ctx context.Context,
	commands <-chan *SDPCommand,
	protoEvents chan<- *protos.SDP,
) error {
	user, err := a.GetUserContext(ctx)
	if err != nil {
		return err
	}
	for {
		select {
		case command := <-commands:
			if command == nil {
				continue
			}
			if command.To != user.ID {
				continue
			}
			protoEvents <- &protos.SDP{
				Type:        SDPTypeCommandToProto[command.Type],
				SenderID:    command.From,
				Description: command.Description,
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// IsItMyRooms return true when list of room in one of my rooms
func (a *API) IsItMyRooms(
	me *room.UserModel,
	roomIDs []string,
) (*bool, error) {
	// get my rooms
	myRooms := &[]room.RoomModel{}
	err := a.DB.
		Model(me).
		Related(myRooms, "Rooms").
		Error
	if err != nil {
		return nil, err
	}
	exist := false
	for _, r := range *myRooms {
		if containString(roomIDs, r.ID) {
			exist = true
			break
		}
	}
	return &exist, nil
}

// SubscribeRoomEvent will subscribe changes in a rooms or channel
func (a *API) SubscribeRoomEvent(
	ctx context.Context,
	events <-chan *room.RoomEvent,
	protoEvents chan<- *protos.RoomEvent,
) error {
	user, err := a.GetUserContext(ctx)
	if err != nil {
		return err
	}
	for {
		select {
		case event := <-events:
			if event == nil {
				continue
			}
			// map room event from message bus to proto event
			var roomEvent *protos.RoomEvent
			switch event.Event {
			case room.UserJoinedRoom:
				{
					payload, ok := event.Payload.(room.RoomParticipantEventPayload)
					if !ok {
						continue
					}
					if !containString(payload.ParticipantIDs, user.ID) {
						continue
					}
					roomEvent = &protos.RoomEvent{
						Event: protos.RoomEvents_UserJoinedRoom,
						Payload: &protos.RoomEvent_RoomParticipant{
							RoomParticipant: &protos.RoomParticipantEventPayload{
								ParticipantID: payload.UserID,
								RoomID:        payload.RoomID,
							},
						},
					}
				}
			case room.UserLeftRoom:
				{
					payload, ok := event.Payload.(room.RoomParticipantEventPayload)
					if !ok {
						continue
					}
					if !containString(payload.ParticipantIDs, user.ID) {
						continue
					}
					roomEvent = &protos.RoomEvent{
						Event: protos.RoomEvents_UserLeftRoom,
						Payload: &protos.RoomEvent_RoomParticipant{
							RoomParticipant: &protos.RoomParticipantEventPayload{
								ParticipantID: payload.UserID,
								RoomID:        payload.RoomID,
							},
						},
					}
				}
			case room.RoomCreated:
				{
					payload, ok := event.Payload.(room.RoomInstanceEventPayload)
					if !ok {
						continue
					}
					if !containString(payload.MemberIDs, user.ID) {
						continue
					}
					roomEvent = &protos.RoomEvent{
						Event: protos.RoomEvents_RoomCreated,
						Payload: &protos.RoomEvent_RoomInstance{
							RoomInstance: &protos.RoomInstanceEventPayload{
								Id:          payload.ID,
								Name:        payload.Name,
								Photo:       payload.Photo,
								Description: payload.Description,
							},
						},
					}
				}
			case room.RoomProfileUpdated:
				{
					payload, ok := event.Payload.(room.RoomInstanceEventPayload)
					if !ok {
						continue
					}
					if !containString(payload.MemberIDs, user.ID) {
						continue
					}
					roomEvent = &protos.RoomEvent{
						Event: protos.RoomEvents_RoomProfileUpdated,
						Payload: &protos.RoomEvent_RoomInstance{
							RoomInstance: &protos.RoomInstanceEventPayload{
								Id:          payload.ID,
								Name:        payload.Name,
								Photo:       payload.Photo,
								Description: payload.Description,
							},
						},
					}
				}
			case room.RoomDestroyed:
				{
					payload, ok := event.Payload.(room.RoomInstanceEventPayload)
					if !ok {
						continue
					}
					if !containString(payload.MemberIDs, user.ID) {
						continue
					}
					roomEvent = &protos.RoomEvent{
						Event: protos.RoomEvents_RoomDestroyed,
						Payload: &protos.RoomEvent_RoomInstance{
							RoomInstance: &protos.RoomInstanceEventPayload{
								Id:          payload.ID,
								Name:        payload.Name,
								Photo:       payload.Photo,
								Description: payload.Description,
							},
						},
					}
				}
			case room.UserProfileUpdated:
				{
					payload, ok := event.Payload.(room.UserInstanceEventPayload)
					if !ok {
						continue
					}
					inMyRoom, err := a.IsItMyRooms(user, payload.RoomIDs)
					if err != nil {
						a.Logger.Error(err)
						continue
					}
					if !(*inMyRoom) {
						continue
					}
					roomEvent = &protos.RoomEvent{
						Event: protos.RoomEvents_UserProfileUpdated,
						Payload: &protos.RoomEvent_UserInstance{
							UserInstance: &protos.UserInstanceEventPayload{
								Id:    payload.ID,
								Name:  payload.Name,
								Photo: payload.Photo,
							},
						},
					}
				}
			case room.UserRemoved:
				{
					payload, ok := event.Payload.(room.UserInstanceEventPayload)
					if !ok {
						continue
					}
					inMyRoom, err := a.IsItMyRooms(user, payload.RoomIDs)
					if err != nil {
						a.Logger.Error(err)
						continue
					}
					if !(*inMyRoom) {
						continue
					}
					roomEvent = &protos.RoomEvent{
						Event: protos.RoomEvents_UserRemoved,
						Payload: &protos.RoomEvent_UserInstance{
							UserInstance: &protos.UserInstanceEventPayload{
								Id:    payload.ID,
								Name:  payload.Name,
								Photo: payload.Photo,
							},
						},
					}
				}
			}
			if roomEvent == nil {
				continue
			}
			// set timestamp
			timestamp, err := ptypes.TimestampProto(event.Time)
			if err != nil {
				a.Logger.Error(err)
				continue
			}
			roomEvent.Time = timestamp
			protoEvents <- roomEvent
		case <-ctx.Done():
			return nil
		}
	}
}

func containString(list []string, item string) bool {
	for _, i := range list {
		if i != item {
			continue
		}
		return true
	}
	return false
}
