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
		// add some public stun server
		// and private turn server
		ICEServers = &[]*signaling.ICEServer{
			{
				URL: "stun:stun1.l.google.com:19302",
			},
			{
				URL:            "turn:w1.xirsys.com:80?transport=udp",
				Username:       faker.Internet().UserName(),
				CredentialType: signaling.CredentialTypePassword,
				Password:       faker.RandomString(20),
			},
			{
				URL:            "turns:w1.xirsys.com:5349?transport=tcp",
				Username:       faker.Internet().UserName(),
				CredentialType: signaling.CredentialTypeOAuth,
				AccessToken:    faker.RandomString(100),
				MacKey:         faker.RandomString(50),
			},
		}
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

	Describe("GetCommands", func() {
		It("should return command channel", func() {
			e := api.GetCommands()
			Expect(e).To(Equal(SDPCommands))
		})
	})

	Describe("SetCommands", func() {
		It("should set command channel", func() {
			e := make(chan *signaling.SDPCommand)
			api.SetCommands(e)
			Expect(api.Commands).To(Equal(e))
		})
	})

	Describe("GetRoomEvents", func() {
		It("should return room events channel", func() {
			e := api.GetRoomEvents()
			Expect(e).To(Equal(roomEvents))
		})
	})

	Describe("SetRoomEvents", func() {
		It("should set room events channel", func() {
			e := make(chan *room.RoomEvent)
			api.SetRoomEvents(e)
			Expect(api.Events).To(Equal(e))
		})
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
			It("should return user not found error", func() {
				ctx := context.WithValue(context.Background(), room.UserIDKey, "not-exist-user")
				res, err := api.GetUserContext(ctx)
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(room.UserNotFoundError))
			})
		})

		When("context not contain user key", func() {
			It("should return context invalid error", func() {
				ctx := context.Background()
				res, err := api.GetUserContext(ctx)
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(signaling.ContextInvalidError))
			})
		})
	})

	Describe("MyProfile", func() {
		It("should return current user context and ICE server list", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			res, err := api.MyProfile(ctx)
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(u1.ID))
			Expect(res.Name).To(Equal(u1.Name))
			Expect(res.Photo).To(Equal(u1.Photo))
			Expect(res.Servers).To(ConsistOf(
				&protos.ICEServer{
					Url: (*ICEServers)[0].URL,
				},
				&protos.ICEServer{
					Url:            (*ICEServers)[1].URL,
					CredentialType: protos.ICECredentialType_PASSWORD,
					Username:       (*ICEServers)[1].Username,
					Password:       (*ICEServers)[1].Password,
				},
				&protos.ICEServer{
					Url:            (*ICEServers)[2].URL,
					CredentialType: protos.ICECredentialType_OAUTH,
					Username:       (*ICEServers)[2].Username,
					AccessToken:    (*ICEServers)[2].AccessToken,
					MacKey:         (*ICEServers)[2].MacKey,
				},
			))
		})
	})

	Describe("UpdateProfile", func() {
		It("should update user profile information", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			param := &protos.UpdateProfileParam{
				Name:  faker.Name().Name(),
				Photo: faker.Avatar().String(),
			}
			go func() { <-roomEvents }()
			api.UpdateProfile(ctx, param)
			res, err := api.MyProfile(ctx)
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(u1.ID))
			Expect(res.Name).To(Equal(param.Name))
			Expect(res.Photo).To(Equal(param.Photo))
			Expect(res.Servers).To(ConsistOf(
				&protos.ICEServer{
					Url: (*ICEServers)[0].URL,
				},
				&protos.ICEServer{
					Url:            (*ICEServers)[1].URL,
					CredentialType: protos.ICECredentialType_PASSWORD,
					Username:       (*ICEServers)[1].Username,
					Password:       (*ICEServers)[1].Password,
				},
				&protos.ICEServer{
					Url:            (*ICEServers)[2].URL,
					CredentialType: protos.ICECredentialType_OAUTH,
					Username:       (*ICEServers)[2].Username,
					AccessToken:    (*ICEServers)[2].AccessToken,
					MacKey:         (*ICEServers)[2].MacKey,
				},
			))
		})

		It("should return updated user profile information", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			param := &protos.UpdateProfileParam{
				Name:  faker.Name().Name(),
				Photo: faker.Avatar().String(),
			}
			go func() { <-roomEvents }()
			res, err := api.UpdateProfile(ctx, param)
			Expect(err).To(BeNil())
			Expect(res.Id).To(Equal(u1.ID))
			Expect(res.Name).To(Equal(param.Name))
			Expect(res.Photo).To(Equal(param.Photo))
			Expect(res.Servers).To(ConsistOf(
				&protos.ICEServer{
					Url: (*ICEServers)[0].URL,
				},
				&protos.ICEServer{
					Url:            (*ICEServers)[1].URL,
					CredentialType: protos.ICECredentialType_PASSWORD,
					Username:       (*ICEServers)[1].Username,
					Password:       (*ICEServers)[1].Password,
				},
				&protos.ICEServer{
					Url:            (*ICEServers)[2].URL,
					CredentialType: protos.ICECredentialType_OAUTH,
					Username:       (*ICEServers)[2].Username,
					AccessToken:    (*ICEServers)[2].AccessToken,
					MacKey:         (*ICEServers)[2].MacKey,
				},
			))
		})

		It("should publish update profile event", func() {
			param := &protos.UpdateProfileParam{
				Name:  faker.Name().Name(),
				Photo: faker.Avatar().String(),
			}
			go func() {
				ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
				api.UpdateProfile(ctx, param)
			}()
			event := <-roomEvents
			Expect(event.Event).To(Equal(room.UserProfileUpdated))
			payload := event.Payload.(*room.UserInstanceEventPayload)
			Expect(payload.ID).To(Equal(u1.ID))
			Expect(payload.Name).To(Equal(param.Name))
			Expect(payload.Photo).To(Equal(param.Photo))
			Expect(payload.RoomIDs).To(ConsistOf(
				r1.ID, r2.ID,
			))
		})
	})

	Describe("MyRooms", func() {
		It("should return list of my rooms", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			res, err := api.MyRooms(ctx)
			Expect(err).To(BeNil())
			Expect(res.Count).To(Equal(uint64(2)))
			Expect(res.Rooms).To(HaveLen(2))
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
		})

		When("user dont join any room", func() {
			It("should return 0 room", func() {
				ctx := context.WithValue(context.Background(), room.UserIDKey, u7.ID)
				res, err := api.MyRooms(ctx)
				Expect(err).To(BeNil())
				Expect(res.Count).To(Equal(uint64(0)))
				Expect(res.Rooms).To(HaveLen(0))
			})
		})
	})

	Describe("MyRoomInfo", func() {
		It("should return room with list of member", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			res, err := api.MyRoomInfo(ctx, &protos.GetRoomParam{
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
			It("should return room not found error", func() {
				ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
				res, err := api.MyRoomInfo(ctx, &protos.GetRoomParam{
					Id: r3.ID,
				})
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(room.RoomNotFoundError))
			})
		})

		When("room not exist", func() {
			It("should return room not found error", func() {
				ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
				res, err := api.MyRoomInfo(ctx, &protos.GetRoomParam{
					Id: "not-exist-room",
				})
				Expect(res).To(BeNil())
				Expect(err.Error()).To(Equal(room.RoomNotFoundError))
			})
		})
	})

	Describe("OfferSDP", func() {
		It("should publish SDP offer command from user", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
			param := &protos.SDPParam{
				Description: faker.Lorem().Paragraph(3),
				UserID:      u2.ID,
			}
			go func() {
				err := api.OfferSDP(ctx, param)
				Expect(err).To(BeNil())
			}()
			command := <-SDPCommands
			Expect(command.Type).To(Equal(signaling.SDPOffer))
			Expect(command.From).To(Equal(u1.ID))
			Expect(command.To).To(Equal(param.UserID))
			Expect(command.Description).To(Equal(param.Description))
		})
	})

	Describe("AnswerSDP", func() {
		It("should publish SDP answer command from user", func() {
			ctx := context.WithValue(context.Background(), room.UserIDKey, u2.ID)
			param := &protos.SDPParam{
				Description: faker.Lorem().Paragraph(3),
				UserID:      u1.ID,
			}
			go func() {
				err := api.AnswerSDP(ctx, param)
				Expect(err).To(BeNil())
			}()
			command := <-SDPCommands
			Expect(command.Type).To(Equal(signaling.SDPAnswer))
			Expect(command.From).To(Equal(u2.ID))
			Expect(command.To).To(Equal(param.UserID))
			Expect(command.Description).To(Equal(param.Description))
		})
	})

	Describe("SubscribeSDPCommand", func() {
		When("a peer send SDP offer command", func() {
			It("should send SDP offer event to target user", func(done Done) {
				commands := make(chan *signaling.SDPCommand)
				sdps := make(chan *protos.SDP)
				command := &signaling.SDPCommand{
					Type:        signaling.SDPOffer,
					From:        u2.ID,
					To:          u1.ID,
					Description: faker.Lorem().Paragraph(5),
				}
				go func() {
					ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeSDPCommand(ctx, commands, sdps)
				}()
				go func() {
					commands <- command
				}()
				sdp := <-sdps
				Expect(sdp.SenderID).To(Equal(command.From))
				Expect(sdp.Description).To(Equal(command.Description))
				Expect(sdp.Type).To(Equal(protos.SDPTypes_Offer))
				close(done)
			}, 0.3)
		})

		When("a peer send SDP answer command", func() {
			It("should send SDP answer event to target user", func(done Done) {
				commands := make(chan *signaling.SDPCommand)
				sdps := make(chan *protos.SDP)
				command := &signaling.SDPCommand{
					Type:        signaling.SDPAnswer,
					From:        u1.ID,
					To:          u2.ID,
					Description: faker.Lorem().Paragraph(5),
				}
				go func() {
					ctx := context.WithValue(context.Background(), room.UserIDKey, u2.ID)
					api.SubscribeSDPCommand(ctx, commands, sdps)
				}()
				go func() {
					commands <- command
				}()
				sdp := <-sdps
				Expect(sdp.SenderID).To(Equal(command.From))
				Expect(sdp.Description).To(Equal(command.Description))
				Expect(sdp.Type).To(Equal(protos.SDPTypes_Answer))
				close(done)
			}, 0.3)
		})

		When("a peer not a target user", func() {
			It("should not receive SDP answer event", func(done Done) {
				commands := make(chan *signaling.SDPCommand)
				sdps := make(chan *protos.SDP)
				command := &signaling.SDPCommand{
					Type:        signaling.SDPAnswer,
					From:        u1.ID,
					To:          u3.ID,
					Description: faker.Lorem().Paragraph(5),
				}
				go func() {
					ctx := context.WithValue(context.Background(), room.UserIDKey, u2.ID)
					api.SubscribeSDPCommand(ctx, commands, sdps)
				}()
				go func() {
					// send empty command
					commands <- nil
					// send first command to u3
					commands <- command
				}()
				Consistently(sdps).ShouldNot(Receive())
				close(done)
			}, 0.3)

			It("should not receive SDP offer event", func(done Done) {
				commands := make(chan *signaling.SDPCommand)
				sdps := make(chan *protos.SDP)
				command := &signaling.SDPCommand{
					Type:        signaling.SDPOffer,
					From:        u2.ID,
					To:          u3.ID,
					Description: faker.Lorem().Paragraph(5),
				}
				go func() {
					ctx := context.WithValue(context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeSDPCommand(ctx, commands, sdps)
				}()
				go func() {
					// send empty command
					commands <- nil
					// send first command to u3
					commands <- command
				}()
				Consistently(sdps).ShouldNot(Receive())
				close(done)
			}, 0.3)
		})
	})

	Describe("IsItMyRooms", func() {
		When("one of my rooms is on room id list", func() {
			It("should return true", func() {
				exist, err := api.IsItMyRooms(u1, []string{
					r4.ID, r1.ID, r3.ID,
				})
				Expect(err).To(BeNil())
				Expect(*exist).To(BeTrue())
			})
		})

		When("any of my rooms are not on room id list", func() {
			It("should return false", func() {
				exist, err := api.IsItMyRooms(u1, []string{
					r3.ID, r4.ID,
				})
				Expect(err).To(BeNil())
				Expect(*exist).To(BeFalse())
			})
		})
	})

	Describe("SubscribeRoomEvent", func() {
		When("user joined my room", func() {
			It("should receive user joined room event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				eventPayload := room.RoomParticipantEventPayload{
					RoomID: r1.ID,
					UserID: u4.ID,
					ParticipantIDs: []string{
						u1.ID, u2.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.UserJoinedRoom,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				e := <-myRoomEvents
				Expect(e.Event).To(Equal(protos.RoomEvents_UserJoinedRoom))
				payload := e.Payload.(*protos.RoomEvent_RoomParticipant)
				Expect(payload.RoomParticipant.ParticipantID).
					To(Equal(eventPayload.UserID))
				Expect(payload.RoomParticipant.RoomID).
					To(Equal(eventPayload.RoomID))
				close(done)
			}, 0.3)
		})

		When("user joined to other room", func() {
			It("should not receive user joined room event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				eventPayload := room.RoomParticipantEventPayload{
					RoomID: r3.ID,
					UserID: u7.ID,
					ParticipantIDs: []string{
						u2.ID, u3.ID, u4.ID, u5.ID, u6.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.UserJoinedRoom,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				Consistently(myRoomEvents).ShouldNot(Receive())
				close(done)
			}, 0.3)
		})

		When("user left my room", func() {
			It("should receive user left room event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				eventPayload := room.RoomParticipantEventPayload{
					RoomID: r1.ID,
					UserID: u2.ID,
					ParticipantIDs: []string{
						u1.ID, u2.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.UserLeftRoom,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				e := <-myRoomEvents
				Expect(e.Event).To(Equal(protos.RoomEvents_UserLeftRoom))
				payload := e.Payload.(*protos.RoomEvent_RoomParticipant)
				Expect(payload.RoomParticipant.ParticipantID).
					To(Equal(eventPayload.UserID))
				Expect(payload.RoomParticipant.RoomID).
					To(Equal(eventPayload.RoomID))
				close(done)
			}, 0.3)
		})

		When("user left other room", func() {
			It("should not receive user left room event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				eventPayload := room.RoomParticipantEventPayload{
					RoomID: r3.ID,
					UserID: u3.ID,
					ParticipantIDs: []string{
						u2.ID, u3.ID, u4.ID, u5.ID, u6.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.UserLeftRoom,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				Consistently(myRoomEvents).ShouldNot(Receive())
				close(done)
			}, 0.3)
		})

		When("new room created with me in it", func() {
			It("should receive room created event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				r5 := room.FakeRoom()
				eventPayload := room.RoomInstanceEventPayload{
					ID:          r5.ID,
					Description: r5.Description,
					Name:        r5.Name,
					Photo:       r5.Photo,
					MemberIDs: []string{
						u1.ID, u3.ID, u5.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.RoomCreated,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				e := <-myRoomEvents
				Expect(e.Event).To(Equal(protos.RoomEvents_RoomCreated))
				payload := e.Payload.(*protos.RoomEvent_RoomInstance)
				Expect(payload.RoomInstance.Id).
					To(Equal(eventPayload.ID))
				Expect(payload.RoomInstance.Name).
					To(Equal(eventPayload.Name))
				Expect(payload.RoomInstance.Description).
					To(Equal(eventPayload.Description))
				Expect(payload.RoomInstance.Photo).
					To(Equal(eventPayload.Photo))
				close(done)
			}, 0.3)
		})

		When("new room created without me in it", func() {
			It("should not receive room created event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				r5 := room.FakeRoom()
				eventPayload := room.RoomInstanceEventPayload{
					ID:          r5.ID,
					Description: r5.Description,
					Name:        r5.Name,
					Photo:       r5.Photo,
					MemberIDs: []string{
						u2.ID, u3.ID, u5.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.RoomCreated,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				Consistently(myRoomEvents).ShouldNot(Receive())
				close(done)
			}, 0.3)
		})

		When("my room profile updated", func() {
			It("should receive room profile updated event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				r5 := room.FakeRoom()
				eventPayload := room.RoomInstanceEventPayload{
					ID:          r5.ID,
					Description: r5.Description,
					Name:        r5.Name,
					Photo:       r5.Photo,
					MemberIDs: []string{
						u1.ID, u3.ID, u5.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.RoomProfileUpdated,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				e := <-myRoomEvents
				Expect(e.Event).To(Equal(protos.RoomEvents_RoomProfileUpdated))
				payload := e.Payload.(*protos.RoomEvent_RoomInstance)
				Expect(payload.RoomInstance.Id).
					To(Equal(eventPayload.ID))
				Expect(payload.RoomInstance.Name).
					To(Equal(eventPayload.Name))
				Expect(payload.RoomInstance.Description).
					To(Equal(eventPayload.Description))
				Expect(payload.RoomInstance.Photo).
					To(Equal(eventPayload.Photo))
				close(done)
			}, 0.3)
		})

		When("other room profile updated", func() {
			It("should not receive room profile updated event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				r5 := room.FakeRoom()
				eventPayload := room.RoomInstanceEventPayload{
					ID:          r5.ID,
					Description: r5.Description,
					Name:        r5.Name,
					Photo:       r5.Photo,
					MemberIDs: []string{
						u2.ID, u3.ID, u5.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.RoomProfileUpdated,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				Consistently(myRoomEvents).ShouldNot(Receive())
				close(done)
			}, 0.3)
		})

		When("my room destroyed", func() {
			It("should receive room destoryed event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				r5 := room.FakeRoom()
				eventPayload := room.RoomInstanceEventPayload{
					ID:          r5.ID,
					Description: r5.Description,
					Name:        r5.Name,
					Photo:       r5.Photo,
					MemberIDs: []string{
						u1.ID, u3.ID, u5.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.RoomDestroyed,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				e := <-myRoomEvents
				Expect(e.Event).To(Equal(protos.RoomEvents_RoomDestroyed))
				payload := e.Payload.(*protos.RoomEvent_RoomInstance)
				Expect(payload.RoomInstance.Id).
					To(Equal(eventPayload.ID))
				Expect(payload.RoomInstance.Name).
					To(Equal(eventPayload.Name))
				Expect(payload.RoomInstance.Description).
					To(Equal(eventPayload.Description))
				Expect(payload.RoomInstance.Photo).
					To(Equal(eventPayload.Photo))
				close(done)
			}, 0.3)
		})

		When("other room destroyed", func() {
			It("should not receive room destoryed event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				r5 := room.FakeRoom()
				eventPayload := room.RoomInstanceEventPayload{
					ID:          r5.ID,
					Description: r5.Description,
					Name:        r5.Name,
					Photo:       r5.Photo,
					MemberIDs: []string{
						u2.ID, u3.ID, u5.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.RoomDestroyed,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				Consistently(myRoomEvents).ShouldNot(Receive())
				close(done)
			}, 0.3)
		})

		When("user in my room has profile updated", func() {
			It("should receive user profile updated event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				eventPayload := room.UserInstanceEventPayload{
					ID:    u2.ID,
					Name:  u2.Name,
					Photo: u2.Photo,
					RoomIDs: []string{
						r1.ID, r3.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.UserProfileUpdated,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				e := <-myRoomEvents
				Expect(e.Event).To(Equal(protos.RoomEvents_UserProfileUpdated))
				payload := e.Payload.(*protos.RoomEvent_UserInstance)
				Expect(payload.UserInstance.Id).
					To(Equal(eventPayload.ID))
				Expect(payload.UserInstance.Name).
					To(Equal(eventPayload.Name))
				Expect(payload.UserInstance.Photo).
					To(Equal(eventPayload.Photo))
				close(done)
			}, 0.3)
		})

		When("user in other room has profile updated", func() {
			It("should not receive user profile updated event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				eventPayload := room.UserInstanceEventPayload{
					ID:    u2.ID,
					Name:  u2.Name,
					Photo: u2.Photo,
					RoomIDs: []string{
						r3.ID, r4.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.UserProfileUpdated,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				Consistently(myRoomEvents).ShouldNot(Receive())
				close(done)
			}, 0.3)
		})

		When("user in my room has removed", func() {
			It("should receive user removed event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				eventPayload := room.UserInstanceEventPayload{
					ID:    u2.ID,
					Name:  u2.Name,
					Photo: u2.Photo,
					RoomIDs: []string{
						r1.ID, r3.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.UserRemoved,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				e := <-myRoomEvents
				Expect(e.Event).To(Equal(protos.RoomEvents_UserRemoved))
				payload := e.Payload.(*protos.RoomEvent_UserInstance)
				Expect(payload.UserInstance.Id).
					To(Equal(eventPayload.ID))
				Expect(payload.UserInstance.Name).
					To(Equal(eventPayload.Name))
				Expect(payload.UserInstance.Photo).
					To(Equal(eventPayload.Photo))
				close(done)
			}, 0.3)
		})

		When("user in other room has removed", func() {
			It("should not receive user removed event", func(done Done) {
				events := make(chan *room.RoomEvent)
				myRoomEvents := make(chan *protos.RoomEvent)
				eventPayload := room.UserInstanceEventPayload{
					ID:    u2.ID,
					Name:  u2.Name,
					Photo: u2.Photo,
					RoomIDs: []string{
						r3.ID, r4.ID,
					},
				}
				event := &room.RoomEvent{
					Event:   room.UserRemoved,
					Payload: eventPayload,
				}
				go func() {
					ctx := context.WithValue(
						context.Background(), room.UserIDKey, u1.ID)
					api.SubscribeRoomEvent(ctx, events, myRoomEvents)
				}()
				go func() {
					events <- event
				}()
				Consistently(myRoomEvents).ShouldNot(Receive())
				close(done)
			}, 0.3)
		})
	})
})
