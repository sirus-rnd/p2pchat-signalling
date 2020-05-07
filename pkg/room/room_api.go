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
	UserAlreadyExist  = "user with same id already exists"
	UserNotFoundError = "user not found"
	RoomAlreadyExist  = "room with same id already exists"
	RoomNotFoundError = "room not found"
)

// NewRoomAPI will create new instance of room API
func NewRoomAPI(
	db *gorm.DB,
	logger *zap.SugaredLogger,
	accessSecret string,
) *RoomAPI {
	return &RoomAPI{
		DB:           db,
		Logger:       logger,
		AccessSecret: accessSecret,
	}
}

// RoomAPI to manage room & participant in it
type RoomAPI struct {
	DB           *gorm.DB
	Logger       *zap.SugaredLogger
	AccessSecret string
	Events       chan *RoomEvent
}

// GetEvents will return channel use to publish events
func (a *RoomAPI) GetEvents() <-chan *RoomEvent {
	return a.Events
}

// SetEvents will set channel use to publish events
func (a *RoomAPI) SetEvents(events chan *RoomEvent) {
	a.Events = events
}

// UserModelToProto will convert user model to it's proto representation
func (a *RoomAPI) UserModelToProto(model *UserModel) *protos.User {
	return &protos.User{
		Id:    model.ID,
		Name:  model.Name,
		Photo: model.Photo,
	}
}

// RegisterUser will register new user that can participate in a room
func (a *RoomAPI) RegisterUser(ctx context.Context, param protos.NewUserParam) (*protos.User, error) {
	// check if user presents
	var user *UserModel
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	if user != nil {
		return nil, fmt.Errorf(UserAlreadyExist)
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
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: UserRegistered,
		Payload: &UserInstanceEventPayload{
			UserID: user.ID,
		},
	}
	return a.UserModelToProto(user), nil
}

// GetUser return user information by it's id
func (a *RoomAPI) GetUser(ctx context.Context, param protos.GetUserParam) (*protos.User, error) {
	user := &UserModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(UserNotFoundError)
		}
		return nil, err
	}
	return a.UserModelToProto(user), nil
}

// GetUsers return list of user registered on system
func (a *RoomAPI) GetUsers(ctx context.Context, param protos.PaginationParam) (*protos.Users, error) {
	datas := []UserModel{}
	count := uint64(0)
	keyword := strings.ToLower(param.Keyword)
	err := a.DB.
		Where("LOWER(name) LIKE ?", "%"+keyword+"%").
		Offset(int(param.Offset)).
		Limit(int(param.Limit)).
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
		user := a.UserModelToProto(&data)
		users = append(users, user)
	}
	return &protos.Users{
		Users: users,
		Count: count,
	}, nil
}

// GetUserAccessToken will return access token used by peer as an user identification form
func (a *RoomAPI) GetUserAccessToken(ctx context.Context, param protos.GetUserParam) (*protos.UserAccessToken, error) {
	// get user information
	user := &UserModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
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
func (a *RoomAPI) UpdateUserProfile(ctx context.Context, param protos.UpdateUserProfileParam) (*protos.User, error) {
	// get user information
	user := &UserModel{}
	err := a.DB.Where(&UserModel{ID: param.Id}).
		First(user).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
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
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: UserProfileUpdated,
		Payload: &UserInstanceEventPayload{
			UserID: user.ID,
		},
	}
	return a.UserModelToProto(user), nil
}

// RemoveUser will remove a user from system
func (a *RoomAPI) RemoveUser(ctx context.Context, param protos.GetRoomParam) (*protos.User, error) {
	// remove user
	var user *UserModel
	err := a.DB.Where(&UserModel{ID: param.Id}).Delete(user).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(UserNotFoundError)
		}
		return nil, err
	}
	// publish user removed events
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: UserRemoved,
		Payload: &UserInstanceEventPayload{
			UserID: user.ID,
		},
	}
	return a.UserModelToProto(user), nil
}

// RoomModelToProto will convert room model to it's proto representation
func (a *RoomAPI) RoomModelToProto(model *RoomModel) *protos.Room {
	room := &protos.Room{
		Id:          model.ID,
		Name:        model.Name,
		Photo:       model.Photo,
		Description: model.Description,
	}
	users := []*protos.User{}
	for _, member := range model.Members {
		user := a.UserModelToProto(member)
		users = append(users, user)
	}
	room.Users = users
	return room
}

// Create will create a new room for user to participate in
func (a *RoomAPI) Create(ctx context.Context, param protos.NewRoomParam) (*protos.Room, error) {
	// check if user presents
	var room *RoomModel
	err := a.DB.Where(&RoomModel{ID: param.Id}).
		First(room).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	if room != nil {
		return nil, fmt.Errorf(RoomAlreadyExist)
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
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: RoomCreated,
		Payload: &RoomInstanceEventPayload{
			RoomID: room.ID,
		},
	}
	return a.RoomModelToProto(room), nil
}

// GetByID will return a room and it's participant by it's id
func (a *RoomAPI) GetByID(ctx context.Context, param protos.GetRoomParam) (*protos.Room, error) {
	// get room detail
	var room *RoomModel
	err := a.DB.Preload("Members").
		Where(&RoomModel{ID: param.Id}).
		First(room).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(RoomNotFoundError)
		}
		return nil, err
	}
	return a.RoomModelToProto(room), nil
}

// GetAll will return all room registered on system
func (a *RoomAPI) GetAll(ctx context.Context, param protos.PaginationParam) (*protos.Rooms, error) {
	datas := []RoomModel{}
	count := uint64(0)
	keyword := strings.ToLower(param.Keyword)
	err := a.DB.
		Preload("Members").
		Where("LOWER(name) LIKE ?", "%"+keyword+"%").
		Offset(int(param.Offset)).
		Limit(int(param.Limit)).
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
	rooms := []*protos.Room{}
	for _, data := range datas {
		room := a.RoomModelToProto(&data)
		rooms = append(rooms, room)
	}
	return &protos.Rooms{
		Rooms: rooms,
		Count: count,
	}, nil
}

// UpdateProfile will update room profile like description and photo
func (a *RoomAPI) UpdateProfile(ctx context.Context, param protos.UpdateRoomProfileParam) (*protos.Room, error) {
	// get room detail
	var room *RoomModel
	err := a.DB.Preload("Members").
		Where(&RoomModel{ID: param.Id}).
		First(room).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
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
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: RoomProfileUpdated,
		Payload: &RoomInstanceEventPayload{
			RoomID: room.ID,
		},
	}
	return a.RoomModelToProto(room), nil
}

// AddUser to a room
func (a *RoomAPI) AddUser(ctx context.Context, param protos.UserRoomParam) (*protos.Room, error) {
	// get room detail
	var room *RoomModel
	err := a.DB.
		Where(&RoomModel{ID: param.RoomID}).
		First(room).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
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
	return a.RoomModelToProto(room), nil
}

// KickUser from a room
func (a *RoomAPI) KickUser(ctx context.Context, param protos.UserRoomParam) (*protos.Room, error) {
	// get room detail
	var room *RoomModel
	err := a.DB.
		Where(&RoomModel{ID: param.RoomID}).
		First(room).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(RoomNotFoundError)
		}
		return nil, err
	}
	// append member to this room
	err = a.DB.Model(&room).
		Association("Members").
		Delete(&UserModel{ID: param.UserID}).Error
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
	// publish user left from room events
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: UserLeftRoom,
		Payload: &RoomParticipantEventPayload{
			UserID: param.UserID,
			RoomID: param.RoomID,
		},
	}
	return a.RoomModelToProto(room), nil
}

// Destroy a room
func (a *RoomAPI) Destroy(ctx context.Context, param protos.GetRoomParam) (*protos.Room, error) {
	// remove room
	var room *RoomModel
	err := a.DB.Where(&UserModel{ID: param.Id}).
		Delete(room).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf(UserNotFoundError)
		}
		return nil, err
	}
	// publish room destroyed events
	a.Events <- &RoomEvent{
		Time:  time.Now(),
		Event: RoomDestroyed,
		Payload: &RoomInstanceEventPayload{
			RoomID: room.ID,
		},
	}
	return a.RoomModelToProto(room), nil
}
