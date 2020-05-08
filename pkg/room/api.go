package room

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jinzhu/gorm"
	"go.sirus.dev/p2p-comm/signalling/pkg/utils"
	"go.sirus.dev/p2p-comm/signalling/protos"
	"go.uber.org/zap"
)

const (
	UserIDKey = "user_id"
)

const (
	UserAlreadyExistError = "user with same id already exists"
	UserNotFoundError     = "user not found"
	RoomAlreadyExistError = "room with same id already exists"
	RoomNotFoundError     = "room not found"
	MemberNotFoundError   = "member not found"
)

// NewAPI will create new instance of room API
func NewAPI(
	db *gorm.DB,
	logger *zap.SugaredLogger,
	accessSecret string,
) *API {
	return &API{
		DB:           db,
		Logger:       logger,
		AccessSecret: accessSecret,
	}
}

// API to manage room & participant in it
type API struct {
	DB           *gorm.DB
	Logger       *zap.SugaredLogger
	AccessSecret string
	Events       chan *RoomEvent
}

// GetEvents will return channel use to publish events
func (a *API) GetEvents() chan *RoomEvent {
	return a.Events
}

// SetEvents will set channel use to publish events
func (a *API) SetEvents(events chan *RoomEvent) {
	a.Events = events
}

// GetUserInstancePayload return user instance used on user instance payload
func (a *API) GetUserInstancePayload(user *UserModel) (*UserInstanceEventPayload, error) {
	rooms := &[]RoomModel{}
	err := a.DB.Model(user).Related(rooms, "Rooms").Error
	if err != nil {
		return nil, err
	}
	roomIDs := []string{}
	for _, r := range *rooms {
		roomIDs = append(roomIDs, r.ID)
	}
	return &UserInstanceEventPayload{
		ID:      user.ID,
		Name:    user.Name,
		Photo:   user.Photo,
		RoomIDs: roomIDs,
	}, nil
}

// GetRoomInstancePayload return room instance used on room instance payload
func (a *API) GetRoomInstancePayload(room *RoomModel) (*RoomInstanceEventPayload, error) {
	users := &[]UserModel{}
	err := a.DB.Model(room).Related(users, "Members").Error
	if err != nil {
		return nil, err
	}
	userIDs := []string{}
	for _, u := range *users {
		userIDs = append(userIDs, u.ID)
	}
	return &RoomInstanceEventPayload{
		ID:          room.ID,
		Name:        room.Name,
		Photo:       room.Photo,
		Description: room.Description,
		MemberIDs:   userIDs,
	}, nil
}

// RegisterUser will register new user that can participate in a room
func (a *API) RegisterUser(ctx context.Context, param protos.NewUserParam) (*protos.User, error) {
	// check if user presents
	user := &UserModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	if len(user.ID) > 0 {
		return nil, fmt.Errorf(UserAlreadyExistError)
	}
	// create user
	user = &UserModel{
		ID:    param.Id,
		Name:  param.Name,
		Photo: param.Photo,
	}
	err = a.DB.Create(user).Error
	if err != nil {
		return nil, err
	}
	// publish user registered events
	payload, err := a.GetUserInstancePayload(user)
	if err != nil {
		return nil, err
	}
	a.Events <- &RoomEvent{
		Time:    time.Now(),
		Event:   UserRegistered,
		Payload: payload,
	}
	return UserModelToProto(user), nil
}

// GetUser return user information by it's id
func (a *API) GetUser(ctx context.Context, param protos.GetUserParam) (*protos.User, error) {
	user := &UserModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(UserNotFoundError)
		}
		return nil, err
	}
	return UserModelToProto(user), nil
}

// GetUsers return list of user registered on system
func (a *API) GetUsers(ctx context.Context, param protos.PaginationParam) (*protos.Users, error) {
	datas := []UserModel{}
	count := uint64(0)
	keyword := strings.ToLower(param.Keyword)
	err := a.DB.
		Where("LOWER(name) LIKE ?", "%"+keyword+"%").
		Offset(int(param.Offset)).
		Limit(int(param.Limit)).
		Order("id").
		Find(&datas).
		Error
	if err != nil {
		return nil, err
	}
	err = a.DB.
		Model(&UserModel{}).
		Where("LOWER(name) LIKE ?", "%"+keyword+"%").
		Count(&count).Error
	if err != nil {
		return nil, err
	}
	users := []*protos.User{}
	for _, data := range datas {
		user := UserModelToProto(&data)
		users = append(users, user)
	}
	return &protos.Users{
		Users: users,
		Count: count,
	}, nil
}

// GetUserAccessToken will return access token used by peer as an user identification form
func (a *API) GetUserAccessToken(ctx context.Context, param protos.GetUserParam) (*protos.UserAccessToken, error) {
	// get user information
	user := &UserModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(UserNotFoundError)
		}
		return nil, err
	}
	// create token
	claim := map[string]interface{}{UserIDKey: user.ID}
	token, err := utils.GenerateToken(a.AccessSecret, claim)
	if err != nil {
		return nil, err
	}
	return &protos.UserAccessToken{
		Token: *token,
	}, nil
}

// UpdateUserProfile will update user profile informations
func (a *API) UpdateUserProfile(ctx context.Context, param protos.UpdateUserProfileParam) (*protos.User, error) {
	// get user information
	user := &UserModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(UserNotFoundError)
		}
		return nil, err
	}
	// update profile data
	user.Name = param.Name
	user.Photo = param.Photo
	err = a.DB.Save(user).Error
	if err != nil {
		return nil, err
	}
	// publish user profile updated events
	payload, err := a.GetUserInstancePayload(user)
	if err != nil {
		return nil, err
	}
	a.Events <- &RoomEvent{
		Time:    time.Now(),
		Event:   UserProfileUpdated,
		Payload: payload,
	}
	return UserModelToProto(user), nil
}

// RemoveUser will remove a user from system
func (a *API) RemoveUser(ctx context.Context, param protos.GetUserParam) (*protos.User, error) {
	// remove user
	user := &UserModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).
		Delete(&UserModel{ID: param.Id}).
		Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(UserNotFoundError)
		}
		return nil, err
	}
	// publish user removed events
	payload, err := a.GetUserInstancePayload(user)
	if err != nil {
		return nil, err
	}
	a.Events <- &RoomEvent{
		Time:    time.Now(),
		Event:   UserRemoved,
		Payload: payload,
	}
	return UserModelToProto(user), nil
}

// Create will create a new room for user to participate in
func (a *API) Create(ctx context.Context, param protos.NewRoomParam) (*protos.Room, error) {
	// check if user presents
	room := &RoomModel{}
	err := a.DB.Where(&RoomModel{ID: param.Id}).
		First(room).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	if len(room.ID) > 0 {
		return nil, fmt.Errorf(RoomAlreadyExistError)
	}
	// get all users
	users := []*UserModel{}
	for _, userID := range param.UserIDs {
		users = append(users, &UserModel{ID: userID})
	}
	// save room instance
	room = &RoomModel{
		ID:          param.Id,
		Name:        param.Name,
		Photo:       param.Photo,
		Description: param.Description,
		Members:     users,
	}
	err = a.DB.Create(room).Error
	if err != nil {
		return nil, err
	}
	// get room detail
	err = a.DB.Preload("Members").
		Where(&RoomModel{ID: param.Id}).
		First(room).Error
	if err != nil {
		return nil, err
	}
	// publish new room created events
	payload, err := a.GetRoomInstancePayload(room)
	if err != nil {
		return nil, err
	}
	a.Events <- &RoomEvent{
		Time:    time.Now(),
		Event:   RoomCreated,
		Payload: payload,
	}
	return RoomModelToProto(room), nil
}

// GetByID will return a room and it's participant by it's id
func (a *API) GetByID(ctx context.Context, param protos.GetRoomParam) (*protos.Room, error) {
	// get room detail
	room := &RoomModel{}
	err := a.DB.Preload("Members").
		Where(&RoomModel{ID: param.Id}).
		First(room).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(RoomNotFoundError)
		}
		return nil, err
	}
	return RoomModelToProto(room), nil
}

// GetAll will return all room registered on system
func (a *API) GetAll(ctx context.Context, param protos.PaginationParam) (*protos.Rooms, error) {
	datas := []RoomModel{}
	count := uint64(0)
	keyword := strings.ToLower(param.Keyword)
	err := a.DB.
		Preload("Members").
		Where("LOWER(name) LIKE ?", "%"+keyword+"%").
		Offset(int(param.Offset)).
		Limit(int(param.Limit)).
		Order("id").
		Find(&datas).
		Error
	if err != nil {
		return nil, err
	}
	err = a.DB.
		Model(&RoomModel{}).
		Where("LOWER(name) LIKE ?", "%"+keyword+"%").
		Count(&count).Error
	if err != nil {
		return nil, err
	}
	rooms := []*protos.Room{}
	for _, data := range datas {
		room := RoomModelToProto(&data)
		rooms = append(rooms, room)
	}
	return &protos.Rooms{
		Rooms: rooms,
		Count: count,
	}, nil
}

// UpdateProfile will update room profile like description and photo
func (a *API) UpdateProfile(ctx context.Context, param protos.UpdateRoomProfileParam) (*protos.Room, error) {
	// get room detail
	room := &RoomModel{}
	err := a.DB.Preload("Members").
		Where(&RoomModel{ID: param.Id}).
		First(room).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(RoomNotFoundError)
		}
		return nil, err
	}
	// update room profile
	room.Name = param.Name
	room.Photo = param.Photo
	room.Description = param.Description
	err = a.DB.Save(room).Error
	if err != nil {
		return nil, err
	}
	// publish room profile updated events
	payload, err := a.GetRoomInstancePayload(room)
	if err != nil {
		return nil, err
	}
	a.Events <- &RoomEvent{
		Time:    time.Now(),
		Event:   RoomProfileUpdated,
		Payload: payload,
	}
	return RoomModelToProto(room), nil
}

// AddUser to a room
func (a *API) AddUser(ctx context.Context, param protos.UserRoomParam) (*protos.Room, error) {
	// get room detail
	room := &RoomModel{}
	err := a.DB.
		Where(&RoomModel{ID: param.RoomID}).
		First(room).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(RoomNotFoundError)
		}
		return nil, err
	}
	// append member to this room
	err = a.DB.Model(&room).
		Association("Members").
		Append(&UserModel{ID: param.UserID}).Error
	if err != nil {
		return nil, err
	}
	// get updated room data
	err = a.DB.Preload("Members").
		Where(&RoomModel{ID: param.RoomID}).
		First(room).Error
	if err != nil {
		return nil, err
	}
	// publish user join room events
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: UserJoinedRoom,
		Payload: &RoomParticipantEventPayload{
			UserID: param.UserID,
			RoomID: param.RoomID,
		},
	}
	return RoomModelToProto(room), nil
}

// KickUser from a room
func (a *API) KickUser(ctx context.Context, param protos.UserRoomParam) (*protos.Room, error) {
	// get room detail
	room := &RoomModel{}
	err := a.DB.
		Where(&RoomModel{ID: param.RoomID}).
		First(room).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(RoomNotFoundError)
		}
		return nil, err
	}
	// remove member from this room
	err = a.DB.Model(&room).
		Association("Members").
		Delete(&UserModel{ID: param.UserID}).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(MemberNotFoundError)
		}
		return nil, err
	}
	// get updated room data
	err = a.DB.Preload("Members").
		Where(&RoomModel{ID: param.RoomID}).
		First(room).Error
	if err != nil {
		return nil, err
	}
	// publish user left from room events
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: UserLeftRoom,
		Payload: &RoomParticipantEventPayload{
			UserID: param.UserID,
			RoomID: param.RoomID,
		},
	}
	return RoomModelToProto(room), nil
}

// Destroy a room
func (a *API) Destroy(ctx context.Context, param protos.GetRoomParam) (*protos.Room, error) {
	// remove room
	room := &RoomModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(room).
		Delete(&UserModel{ID: param.Id}).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(UserNotFoundError)
		}
		return nil, err
	}
	// publish room destroyed events
	payload, err := a.GetRoomInstancePayload(room)
	if err != nil {
		return nil, err
	}
	a.Events <- &RoomEvent{
		Time:    time.Now(),
		Event:   RoomDestroyed,
		Payload: payload,
	}
	return RoomModelToProto(room), nil
}
