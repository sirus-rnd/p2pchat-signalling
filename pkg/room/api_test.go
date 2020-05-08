package room_test

import (
	"context"

	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.sirus.dev/p2p-comm/signalling/pkg/connector"
	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/pkg/utils"
	"go.sirus.dev/p2p-comm/signalling/protos"
	"go.uber.org/zap"
	"syreclabs.com/go/faker"
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
		accessSecret = faker.RandomString(20)
		roomEvents = make(chan *room.RoomEvent)
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
		r1.ID = "r1"
		r2 = room.FakeRoom()
		r2.ID = "r2"
		r3 = room.FakeRoom()
		r3.ID = "r3"
		r4 = room.FakeRoom() // empty room
		r4.ID = "r4"
		u1 = room.FakeUser()
		u1.ID = "u1"
		u2 = room.FakeUser()
		u2.ID = "u2"
		u3 = room.FakeUser()
		u3.ID = "u3"
		u4 = room.FakeUser()
		u4.ID = "u4"
		u5 = room.FakeUser()
		u5.ID = "u5"
		u6 = room.FakeUser()
		u6.ID = "u6"
		u7 = room.FakeUser()
		u7.ID = "u7"
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
		It("should return room events channel", func() {
			e := api.GetEvents()
			Expect(e).To(Equal(roomEvents))
		})
	})

	Describe("SetEvents", func() {
		It("should set room events channel", func() {
			e := make(chan *room.RoomEvent)
			api.SetEvents(e)
			Expect(api.Events).To(Equal(e))
		})
	})

	Describe("GetUserInstancePayload", func() {
		It("should return user event payload with their room joined into", func() {
			payload, err := api.GetUserInstancePayload(u1)
			Expect(err).To(BeNil())
			Expect(payload.ID).To(Equal(u1.ID))
			Expect(payload.Name).To(Equal(u1.Name))
			Expect(payload.Photo).To(Equal(u1.Photo))
			Expect(payload.RoomIDs).To(ConsistOf(
				r1.ID, r2.ID,
			))
		})
	})

	Describe("GetRoomInstancePayload", func() {
		It("should return room event payload with list of their member", func() {
			payload, err := api.GetRoomInstancePayload(r1)
			Expect(err).To(BeNil())
			Expect(payload.ID).To(Equal(r1.ID))
			Expect(payload.Name).To(Equal(r1.Name))
			Expect(payload.Photo).To(Equal(r1.Photo))
			Expect(payload.Description).To(Equal(r1.Description))
			Expect(payload.MemberIDs).To(ConsistOf(
				u1.ID, u2.ID,
			))
		})
	})

	Describe("RegisterUser", func() {
		It("should register new user to system", func() {
			param := protos.NewUserParam{
				Id:    faker.RandomString(5),
				Name:  faker.Name().Name(),
				Photo: faker.Avatar().String(),
			}
			ctx := context.Background()
			go func() { <-roomEvents }()
			res, err := api.RegisterUser(ctx, param)
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(param.Id))
			Expect(res.Name).To(Equal(param.Name))
			Expect(res.Photo).To(Equal(param.Photo))
			res, err = api.GetUser(ctx, protos.GetUserParam{
				Id: param.Id,
			})
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(param.Id))
			Expect(res.Name).To(Equal(param.Name))
			Expect(res.Photo).To(Equal(param.Photo))
		})

		It("should publish user registered event", func(done Done) {
			param := protos.NewUserParam{
				Id:    faker.RandomString(5),
				Name:  faker.Name().Name(),
				Photo: faker.Avatar().String(),
			}
			go func() {
				ctx := context.Background()
				api.RegisterUser(ctx, param)
			}()
			event := <-roomEvents
			Expect(event.Event).To(Equal(room.UserRegistered))
			payload := event.Payload.(*room.UserInstanceEventPayload)
			Expect(payload.ID).To(Equal(param.Id))
			Expect(payload.Name).To(Equal(param.Name))
			Expect(payload.Photo).To(Equal(param.Photo))
			Expect(payload.RoomIDs).To(HaveLen(0))
			close(done)
		}, 0.3)

		When("user already registered", func() {
			It("should return user already exist error", func() {
				param := protos.NewUserParam{
					Id:    u1.ID,
					Name:  faker.Name().Name(),
					Photo: faker.Avatar().String(),
				}
				ctx := context.Background()
				res, err := api.RegisterUser(ctx, param)
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(room.UserAlreadyExistError))
			})
		})
	})

	Describe("GetUser", func() {
		It("should return user by it's id", func() {
			ctx := context.Background()
			res, err := api.GetUser(ctx, protos.GetUserParam{
				Id: u1.ID,
			})
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(u1.ID))
			Expect(res.Name).To(Equal(u1.Name))
			Expect(res.Photo).To(Equal(u1.Photo))
		})

		When("user not exist", func() {
			It("should return user not found error", func() {
				ctx := context.Background()
				res, err := api.GetUser(ctx, protos.GetUserParam{
					Id: "non-exist-id",
				})
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(room.UserNotFoundError))
			})
		})
	})

	Describe("GetUsers", func() {
		It("should return list of user", func() {
			ctx := context.Background()
			res, err := api.GetUsers(ctx, protos.PaginationParam{
				Limit: 10, Offset: 0, Keyword: "",
			})
			Expect(err).To(BeNil())
			Expect(res.Count).To(Equal(uint64(7)))
			Expect(res.Users).To(HaveLen(7))
			Expect(res.Users).To(ConsistOf(
				room.UserModelToProto(u1),
				room.UserModelToProto(u2),
				room.UserModelToProto(u3),
				room.UserModelToProto(u4),
				room.UserModelToProto(u5),
				room.UserModelToProto(u6),
				room.UserModelToProto(u7),
			))
		})

		It("should filter user by their name", func() {
			ctx := context.Background()
			u1.Name = "Cameron Boyce"
			u2.Name = "Jasmine Chan"
			u3.Name = "Will Smith"
			u4.Name = "Kristen Stewart"
			u5.Name = "Peyton List"
			u6.Name = "Amanda Bynes"
			u7.Name = "Brandon Soo Hoo"
			db.Save(u1)
			db.Save(u2)
			db.Save(u3)
			db.Save(u4)
			db.Save(u5)
			db.Save(u6)
			db.Save(u7)
			res, err := api.GetUsers(ctx, protos.PaginationParam{
				Limit: 10, Offset: 0, Keyword: "an",
			})
			Expect(err).To(BeNil())
			Expect(res.Count).To(Equal(uint64(3)))
			Expect(res.Users).To(HaveLen(3))
			Expect(res.Users).To(ConsistOf(
				room.UserModelToProto(u2),
				room.UserModelToProto(u6),
				room.UserModelToProto(u7),
			))
		})

		It("should be able to cap user search result", func() {
			ctx := context.Background()
			u1.Name = "Cameron Boyce"
			u2.Name = "Jasmine Chan"
			u3.Name = "Will Smith"
			u4.Name = "Kristen Stewart"
			u5.Name = "Peyton List"
			u6.Name = "Amanda Bynes"
			u7.Name = "Brandon Soo Hoo"
			db.Save(u1)
			db.Save(u2)
			db.Save(u3)
			db.Save(u4)
			db.Save(u5)
			db.Save(u6)
			db.Save(u7)
			res, err := api.GetUsers(ctx, protos.PaginationParam{
				Limit: 3, Offset: 2, Keyword: "a",
			})
			Expect(err).To(BeNil())
			Expect(res.Count).To(Equal(uint64(5)))
			Expect(res.Users).To(HaveLen(3))
			Expect(res.Users).To(ConsistOf(
				room.UserModelToProto(u4),
				room.UserModelToProto(u6),
				room.UserModelToProto(u7),
			))
		})
	})

	Describe("GetUserAccessToken", func() {
		It("should return user access token", func() {
			ctx := context.Background()
			res, err := api.GetUserAccessToken(ctx, protos.GetUserParam{
				Id: u1.ID,
			})
			Expect(err).To(BeNil())
			claim, err := utils.ValidateToken(accessSecret, res.Token)
			Expect(err).To(BeNil())
			Expect(claim).To(HaveKeyWithValue(room.UserIDKey, u1.ID))
		})

		When("user not exist", func() {
			It("should return user not found error", func() {
				ctx := context.Background()
				res, err := api.GetUserAccessToken(ctx, protos.GetUserParam{
					Id: "non-exist-id",
				})
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(room.UserNotFoundError))
			})
		})
	})

	Describe("UpdateUserProfile", func() {
		It("should update user's profile information", func() {
			ctx := context.Background()
			param := protos.UpdateUserProfileParam{
				Id:    u1.ID,
				Name:  faker.Name().Name(),
				Photo: faker.Avatar().String(),
			}
			go func() { <-roomEvents }()
			res, err := api.UpdateUserProfile(ctx, param)
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(param.Id))
			Expect(res.Name).To(Equal(param.Name))
			Expect(res.Photo).To(Equal(param.Photo))
		})

		It("should publish user profile updated event", func(done Done) {
			ctx := context.Background()
			param := protos.UpdateUserProfileParam{
				Id:    u1.ID,
				Name:  faker.Name().Name(),
				Photo: faker.Avatar().String(),
			}
			go func() {
				api.UpdateUserProfile(ctx, param)
			}()
			event := <-roomEvents
			Expect(event.Event).To(Equal(room.UserProfileUpdated))
			payload := event.Payload.(*room.UserInstanceEventPayload)
			Expect(payload.ID).To(Equal(param.Id))
			Expect(payload.Name).To(Equal(param.Name))
			Expect(payload.Photo).To(Equal(param.Photo))
			Expect(payload.RoomIDs).To(ConsistOf(
				r1.ID, r2.ID,
			))
			close(done)
		}, 0.3)

		When("user not exist", func() {
			It("should return user not found error", func() {
				ctx := context.Background()
				res, err := api.UpdateUserProfile(ctx, protos.UpdateUserProfileParam{
					Id:    "non-exist-id",
					Name:  faker.Name().Name(),
					Photo: faker.Avatar().String(),
				})
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(room.UserNotFoundError))
			})
		})
	})

	Describe("RemoveUser", func() {
		It("should remove user from system", func() {
			ctx := context.Background()
			param := protos.GetUserParam{
				Id: u1.ID,
			}
			go func() { <-roomEvents }()
			res, err := api.RemoveUser(ctx, param)
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(u1.ID))
			Expect(res.Name).To(Equal(u1.Name))
			Expect(res.Photo).To(Equal(u1.Photo))
		})

		It("should publish user removed event", func(done Done) {
			ctx := context.Background()
			param := protos.GetUserParam{
				Id: u1.ID,
			}
			go func() {
				api.RemoveUser(ctx, param)
			}()
			event := <-roomEvents
			Expect(event.Event).To(Equal(room.UserRemoved))
			payload := event.Payload.(*room.UserInstanceEventPayload)
			Expect(payload.ID).To(Equal(u1.ID))
			Expect(payload.Name).To(Equal(u1.Name))
			Expect(payload.Photo).To(Equal(u1.Photo))
			Expect(payload.RoomIDs).To(ConsistOf(
				r1.ID, r2.ID,
			))
			close(done)
		}, 0.3)

		When("user not exist", func() {
			It("should return user not found error", func() {
				ctx := context.Background()
				param := protos.GetUserParam{
					Id: "non-exist-id",
				}
				res, err := api.RemoveUser(ctx, param)
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(room.UserNotFoundError))
			})
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
		It("should return list of room ordered by it's id", func() {
			ctx := context.Background()
			res, err := api.GetAll(ctx, protos.PaginationParam{
				Limit: 10, Offset: 0, Keyword: "",
			})
			Expect(err).To(BeNil())
			Expect(res.Count).To(Equal(uint64(4)))
			Expect(res.Rooms).To(HaveLen(4))
			Expect(res.Rooms[0].Id).To(Equal(r1.ID))
			Expect(res.Rooms[0].Name).To(Equal(r1.Name))
			Expect(res.Rooms[0].Photo).To(Equal(r1.Photo))
			Expect(res.Rooms[0].Description).To(Equal(r1.Description))
			Expect(res.Rooms[0].Users).To(
				ConsistOf(
					room.UserModelToProto(u1),
					room.UserModelToProto(u2),
				),
			)
			Expect(res.Rooms[1].Id).To(Equal(r2.ID))
			Expect(res.Rooms[1].Name).To(Equal(r2.Name))
			Expect(res.Rooms[1].Photo).To(Equal(r2.Photo))
			Expect(res.Rooms[1].Description).To(Equal(r2.Description))
			Expect(res.Rooms[1].Users).To(
				ConsistOf(
					room.UserModelToProto(u1),
					room.UserModelToProto(u3),
				),
			)
			Expect(res.Rooms[2].Id).To(Equal(r3.ID))
			Expect(res.Rooms[2].Name).To(Equal(r3.Name))
			Expect(res.Rooms[2].Photo).To(Equal(r3.Photo))
			Expect(res.Rooms[2].Description).To(Equal(r3.Description))
			Expect(res.Rooms[2].Users).To(
				ConsistOf(
					room.UserModelToProto(u2),
					room.UserModelToProto(u3),
					room.UserModelToProto(u4),
					room.UserModelToProto(u5),
					room.UserModelToProto(u6),
				),
			)
			Expect(res.Rooms[3].Id).To(Equal(r4.ID))
			Expect(res.Rooms[3].Name).To(Equal(r4.Name))
			Expect(res.Rooms[3].Photo).To(Equal(r4.Photo))
			Expect(res.Rooms[3].Description).To(Equal(r4.Description))
			Expect(res.Rooms[3].Users).To(HaveLen(0))
		})

		It("should filter room by their name", func() {
			ctx := context.Background()
			r1.Name = "ruang guru"
			r2.Name = "baseball club"
			r3.Name = "engineering"
			r4.Name = "korean club"
			db.Save(r1)
			db.Save(r2)
			db.Save(r3)
			db.Save(r4)
			res, err := api.GetAll(ctx, protos.PaginationParam{
				Limit: 10, Offset: 0, Keyword: "club",
			})
			Expect(err).To(BeNil())
			Expect(res.Count).To(Equal(uint64(2)))
			Expect(res.Rooms).To(HaveLen(2))
			Expect(res.Rooms[0].Name).To(Equal(r2.Name))
			Expect(res.Rooms[1].Name).To(Equal(r4.Name))
		})

		It("should be able to cap room search result", func() {
			ctx := context.Background()
			r1.Name = "teacher class"
			r2.Name = "baseball club"
			r3.Name = "engineering classic"
			r4.Name = "korean club"
			db.Save(r1)
			db.Save(r2)
			db.Save(r3)
			db.Save(r4)
			res, err := api.GetAll(ctx, protos.PaginationParam{
				Limit: 2, Offset: 1, Keyword: "a",
			})
			Expect(err).To(BeNil())
			Expect(res.Count).To(Equal(uint64(4)))
			Expect(res.Rooms).To(HaveLen(2))
			Expect(res.Rooms[0].Name).To(Equal(r2.Name))
			Expect(res.Rooms[1].Name).To(Equal(r3.Name))
		})
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
