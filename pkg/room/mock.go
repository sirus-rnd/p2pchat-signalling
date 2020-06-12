package room

import "syreclabs.com/go/faker"

// FakeUser return random fake user data
func FakeUser() *UserModel {
	online := false
	if faker.RandomChoice([]string{"y", "n"}) == "y" {
		online = true
	}
	return &UserModel{
		ID:     faker.RandomString(5),
		Name:   faker.Name().Name(),
		Photo:  faker.Avatar().String(),
		Online: online,
	}
}

// FakeRoom return random fake room data
func FakeRoom() *RoomModel {
	return &RoomModel{
		ID:          faker.RandomString(5),
		Name:        faker.Commerce().ProductName(),
		Photo:       faker.Avatar().String(),
		Description: faker.Lorem().Sentence(5),
	}
}
