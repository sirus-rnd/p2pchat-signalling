package room

import (
	"go.sirus.dev/p2p-comm/signalling/protos"
)

// RoomModelToProto will convert room model to it's proto representation
func RoomModelToProto(model *RoomModel) *protos.Room {
	room := &protos.Room{
		Id:          model.ID,
		Name:        model.Name,
		Photo:       model.Photo,
		Description: model.Description,
	}
	users := []*protos.User{}
	for _, member := range model.Members {
		user := UserModelToProto(member)
		users = append(users, user)
	}
	room.Users = users
	return room
}

// UserModelToProto will convert user model to it's proto representation
func UserModelToProto(model *UserModel) *protos.User {
	return &protos.User{
		Id:    model.ID,
		Name:  model.Name,
		Photo: model.Photo,
	}
}
