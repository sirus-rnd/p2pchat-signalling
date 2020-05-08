package signaling_test

import (
	"context"

	"github.com/jinzhu/gorm"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"go.sirus.dev/p2p-comm/signalling/pkg/connector"
	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/pkg/signaling"
	"go.sirus.dev/p2p-comm/signalling/protos"
	"go.uber.org/zap"
)

var _ = Describe("API", func() {
	var (
		db          *gorm.DB
		logger      *zap.SugaredLogger
		ICEServers  *[]*signaling.ICEServer
		SDPCommands chan *signaling.SDPCommand
		roomEvents  chan *room.RoomEvent
		api         signaling.API
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
		SDPCommands = make(chan *signaling.SDPCommand)
		ICEServers = &[]*signaling.ICEServer{}
		api = signaling.API{db, logger, ICEServers, SDPCommands, roomEvents}
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

	Describe("GetCommands", func() {
		It("should return command channel", func() {})
	})

	Describe("SetCommands", func() {
		It("should set command channel", func() {})
	})

	Describe("GetRoomEvents", func() {
		It("should return room events channel", func() {})
	})

	Describe("SetRoomEvents", func() {
		It("should set room events channel", func() {})
	})

	Describe("GetUserContext", func() {
		It("should return current user context", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			res, err := api.GetUserContext(ctx)
			Expect(err).To(BeNil())
			Expect(res.ID).To(Equal(u1.ID))
			Expect(res.Name).To(Equal(u1.Name))
			Expect(res.Photo).To(Equal(u1.Photo))
		})

		When("user not exist", func() {
			It("should return user not exist error", func() {})
		})

		When("context not contain user key", func() {
			It("should return context invalid error", func() {})
		})
	})

	Describe("MyProfile", func() {
		It("should return current user context and ICE server list", func() {})
	})

	Describe("UpdateProfile", func() {
		It("should update user profile information", func() {})
		It("should return updated user profile information", func() {})
		It("should publish update profile event", func() {})
	})

	Describe("MyRooms", func() {
		It("should return list of my rooms", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			res, err := api.MyRooms(ctx)
			Expect(err).To(BeNil())
			Expect(res.Count).To(Equal(uint64(2)))
			Expect(res.Rooms).To(HaveLen(2))
			Expect(res.Rooms[0].Id).To(Or(
				Equal(r1.ID),
				Equal(r2.ID),
			))
			Expect(res.Rooms[0].Name).To(Or(
				Equal(r1.Name),
				Equal(r2.Name),
			))
			Expect(res.Rooms[0].Description).To(Or(
				Equal(r1.Description),
				Equal(r2.Description),
			))
			Expect(res.Rooms[0].Photo).To(Or(
				Equal(r1.Photo),
				Equal(r2.Photo),
			))
			Expect(res.Rooms[0].Users).To(Or(
				ConsistOf(
					room.UserModelToProto(u1),
					room.UserModelToProto(u2),
				),
				ConsistOf(
					room.UserModelToProto(u1),
					room.UserModelToProto(u3),
				),
			))
		})

		When("user dont join any room", func() {
			It("should return 0 room", func() {})
		})
	})

	Describe("MyRoomInfo", func() {
		It("should return room with list of member", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			res, err := api.MyRoomInfo(ctx, protos.GetRoomParam{
				Id: r1.ID,
			})
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(r1.ID))
			Expect(res.Name).To(Equal(r1.Name))
			Expect(res.Photo).To(Equal(r1.Photo))
			Expect(res.Description).To(Equal(r1.Description))
			Expect(res.Users).To(HaveLen(2))
			Expect(res.Users).To(ConsistOf(
				room.UserModelToProto(u1),
				room.UserModelToProto(u2),
			))
		})

		When("user not join this room", func() {
			It("should return room not found error", func() {})
		})
	})

	Describe("OfferSDP", func() {
		It("should publish SDP offer command from user", func() {})
	})

	Describe("AnswerSDP", func() {
		It("should publish SDP answer command from user", func() {})
	})

	Describe("SubscribeSDPCommand", func() {
		When("a peer send SDP offer command", func() {
			It("should send SDP offer event to target user", func() {})
		})
		When("a peer send SDP answer command", func() {
			It("should send SDP answer event to target user", func() {})
		})
		When("a peer not a target user", func() {
			It("should not receive SDP answer event", func() {})
			It("should not receive SDP offer event", func() {})
		})
	})

	Describe("IsUserInMyRooms", func() {
		When("user in my room", func() {
			It("should return true", func() {})
		})
		When("user not in my room", func() {
			It("should return false", func() {})
		})
	})

	Describe("SubscribeRoomEvent", func() {
		When("user joined my room", func() {
			It("should receive user joined room event", func() {})
		})
		When("user joined to other room", func() {
			It("should not receive user joined room event", func() {})
		})
		When("user left my room", func() {
			It("should receive user left room event", func() {})
		})
		When("user left other room", func() {
			It("should not receive user left room event", func() {})
		})
		When("new room created with me in it", func() {
			It("should receive room created event", func() {})
		})
		When("new room created without me in it", func() {
			It("should not receive room created event", func() {})
		})
		When("my room profile updated", func() {
			It("should receive room profile updated event", func() {})
		})
		When("other room profile updated", func() {
			It("should not receive room profile updated event", func() {})
		})
		When("my room destroyed", func() {
			It("should receive room destoryed event", func() {})
		})
		When("other room destroyed", func() {
			It("should not receive room destoryed event", func() {})
		})
		When("user in my room has profile updated", func() {
			It("should receive user profile updated event", func() {})
		})
		When("user in other room has profile updated", func() {
			It("should not receive user profile updated event", func() {})
		})
		When("user in my room has removed", func() {
			It("should receive user removed event", func() {})
		})
		When("user in other room has removed", func() {
			It("should not receive user removed event", func() {})
		})
	})
})
