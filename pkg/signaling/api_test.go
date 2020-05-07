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
	"syreclabs.com/go/faker"
)

func fakeUser() *room.UserModel {
	return &room.UserModel{
		ID:    faker.RandomString(5),
		Name:  faker.Name().Name(),
		Photo: faker.Avatar().String(),
	}
}

func fakeRoom() *room.RoomModel {
	return &room.RoomModel{
		ID:          faker.RandomString(5),
		Name:        faker.Name().Name(),
		Photo:       faker.Avatar().String(),
		Description: faker.Lorem().Sentence(5),
	}
}

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

	AfterEach(func() {
		if db != nil {
			db.Close()
		}
		if logger != nil {
			logger.Sync()
		}
	})

	Describe("GetUserContext", func() {
		It("should return current user context", func() {
			user := fakeUser()
			db.Create(user)
			ctx := context.WithValue(context.Background(), room.UserIDKey, user.ID)
			res, err := api.GetUserContext(ctx)
			Expect(err).To(BeNil())
			Expect(res.ID).To(Equal(user.ID))
			Expect(res.Name).To(Equal(user.Name))
			Expect(res.Photo).To(Equal(user.Photo))
		})
	})

	Describe("MyRooms", func() {
		It("should return list of my rooms", func() {
			r1 := fakeRoom()
			r2 := fakeRoom()
			r3 := fakeRoom()
			u1 := fakeUser()
			u2 := fakeUser()
			u3 := fakeUser()
			db.Create(r1)
			db.Create(r2)
			db.Create(r3)
			db.Create(u1)
			db.Create(u2)
			db.Create(u3)
			db.Model(r1).Association("Members").
				Append(u1, u2)
			db.Model(r2).Association("Members").
				Append(u1, u3)
			db.Model(r3).Association("Members").
				Append(u2, u3)
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
	})

	Describe("MyRoomInfo", func() {
		It("should return room with list of member", func() {
			r1 := fakeRoom()
			r2 := fakeRoom()
			r3 := fakeRoom()
			u1 := fakeUser()
			u2 := fakeUser()
			u3 := fakeUser()
			db.Create(r1)
			db.Create(r2)
			db.Create(r3)
			db.Create(u1)
			db.Create(u2)
			db.Create(u3)
			db.Model(r1).Association("Members").
				Append(u1, u2)
			db.Model(r2).Association("Members").
				Append(u1, u3)
			db.Model(r3).Association("Members").
				Append(u2, u3)
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
	})
})
