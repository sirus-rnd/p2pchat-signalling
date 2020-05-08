package room_test

import (
	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.sirus.dev/p2p-comm/signalling/pkg/connector"
	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/pkg/signaling"
	"go.uber.org/zap"
)

var _ = Describe("API", func() {
	var (
		db           *gorm.DB
		logger       *zap.SugaredLogger
		accessSecret string
		roomEvents   chan *room.RoomEvent
		api          room.API
	)

	BeforeEach(func() {
		var err error
		db, err = connector.ConnectToMemmory(room.Models)
		if err != nil {
			Fail(err.Error())
		}
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.FatalLevel + 1) // silent
		loggerRaw, err := config.Build()
		if err != nil {
			Fail(err.Error())
		}
		logger = loggerRaw.Sugar()
		roomEvents = make(chan *room.RoomEvent)
		ICEServers = &[]*signaling.ICEServer{}
		api = room.API{db, logger, accessSecret, roomEvents}
	})

	var (
		r1 *room.RoomModel
		r2 *room.RoomModel
		r3 *room.RoomModel
		r4 *room.RoomModel
		u1 *room.UserModel
		u2 *room.UserModel
		u3 *room.UserModel
		u4 *room.UserModel
		u5 *room.UserModel
		u6 *room.UserModel
		u7 *room.UserModel
	)

	JustBeforeEach(func() {
		r1 = room.FakeRoom()
		r2 = room.FakeRoom()
		r3 = room.FakeRoom()
		r4 = room.FakeRoom() // empty room
		u1 = room.FakeUser()
		u2 = room.FakeUser()
		u3 = room.FakeUser()
		u4 = room.FakeUser()
		u5 = room.FakeUser()
		u6 = room.FakeUser()
		u7 = room.FakeUser()
		db.Create(r1)
		db.Create(r2)
		db.Create(r3)
		db.Create(r4)
		db.Create(u1)
		db.Create(u2)
		db.Create(u3)
		db.Create(u4)
		db.Create(u5)
		db.Create(u6)
		db.Create(u7)
		// add user to each room expect r4
		// and u7 not join to any room
		db.Model(r1).Association("Members").
			Append(u1, u2)
		db.Model(r2).Association("Members").
			Append(u1, u3)
		db.Model(r3).Association("Members").
			Append(u2, u3, u4, u5, u6)
	})

	AfterEach(func() {
		if db != nil {
			db.Close()
		}
		if logger != nil {
			logger.Sync()
		}
	})

	Describe("GetEvents", func() {
		It("should return room events channel", func() {})
	})

	Describe("SetEvents", func() {
		It("should set room events channel", func() {})
	})

	Describe("GetUserInstancePayload", func() {
		It("should return user event payload with their room joined into", func() {})
	})

	Describe("GetRoomInstancePayload", func() {
		It("should return room event payload with list of their member", func() {})
	})

	Describe("RegisterUser", func() {
		It("should register new user to system", func() {})
		It("should publish user registered event", func() {})

		When("user already registered", func() {
			It("should return user already exist error", func() {})
		})
	})

	Describe("GetUser", func() {
		It("should return user by it's id", func() {})

		When("user not exist", func() {
			It("should return user not found error", func() {})
		})
	})

	Describe("GetUsers", func() {
		It("should return list of user", func() {})
		It("should filter user by their name", func() {})
		It("should be able to cap user search result", func() {})
	})

	Describe("GetUserAccessToken", func() {
		It("should return user access token", func() {})
		When("user not exist", func() {
			It("should return user not found error", func() {})
		})
	})

	Describe("UpdateUserProfile", func() {
		It("should update user's profile information", func() {})
		It("should publish user profile updated event", func() {})
		When("user not exist", func() {
			It("should return user not found error", func() {})
		})
	})

	Describe("RemoveUser", func() {
		It("should remove user from system", func() {})
		It("should publish user removed event", func() {})
		When("user not exist", func() {
			It("should return user not found error", func() {})
		})
	})

	Describe("Create", func() {
		It("should add new room to system", func() {})
		It("should publish user created event", func() {})
		When("room already created", func() {
			It("should return room already exist error", func() {})
		})
	})

	Describe("GetByID", func() {
		It("should return room by it's id", func() {})
		When("room not exist", func() {
			It("should return room not found error", func() {})
		})
	})

	Describe("GetAll", func() {
		It("should return list of room", func() {})
		It("should filter room by their name", func() {})
		It("should be able to cap room search result", func() {})
	})

	Describe("UpdateProfile", func() {
		It("should update room's profile information", func() {})
		It("should publish room profile updated event", func() {})
		When("room not exist", func() {
			It("should return room not found error", func() {})
		})
	})

	Describe("AddUser", func() {
		It("should add user to room", func() {})
		It("should publish user joined room event", func() {})
		When("user already added to room", func() {
			It("should not add user to room", func() {})
		})
		When("room not exist", func() {
			It("should return room not found error", func() {})
		})
	})

	Describe("KickUser", func() {
		It("should remove user from a room", func() {})
		It("should publish user left room event", func() {})
		When("user already kicked from room", func() {
			It("should return member not found error", func() {})
		})
		When("room not exist", func() {
			It("should return room not found error", func() {})
		})
	})

	Describe("Destroy", func() {
		It("should remove room from system", func() {})
		It("should publish room removed event", func() {})
		When("room not exist", func() {
			It("should return room not found error", func() {})
		})
	})
})
