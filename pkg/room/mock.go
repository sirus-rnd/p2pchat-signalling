package room

import "syreclabs.com/go/faker"

// FakeUser return random fake user data
func FakeUser() *UserModel {
	return &UserModel{
		ID:    faker.RandomString(5),
		Name:  faker.Name().Name(),
		Photo: faker.Avatar().String(),
	}
}

// FakeRoom return random fake room data
func FakeRoom() *RoomModel {
	return &RoomModel{
		ID:          faker.RandomString(5),
		Name:        faker.Name().Name(),
		Photo:       faker.Avatar().String(),
		Description: faker.Lorem().Sentence(5),
	}
}
