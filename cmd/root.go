package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"go.sirus.dev/p2p-comm/signalling/pkg/connector"
	"go.sirus.dev/p2p-comm/signalling/pkg/room"
	"go.sirus.dev/p2p-comm/signalling/pkg/server"
	"go.sirus.dev/p2p-comm/signalling/pkg/signaling"
)

var rootCmd = &cobra.Command{
	Use:   "signalling",
	Short: "peer-to-peer chat signalling server",
	Run: func(cmd *cobra.Command, args []string) {
		// initiate config
		var err error
		conf, err := LoadConfig()
		if err != nil {
			log.Fatalf("Error loading configurations %v", err)
		}

		// setup logger
		logger, err := connector.CreateLogger(conf.LogLevel)
		if err != nil {
			log.Fatalf("Error setup logger %v", err)
		}
		logger.Info("logger setup finish")

		// connect to postgres
		models := []interface{}{}
		models = append(models, room.Models...)
		db, err := connector.ConnectToPostgres(conf.Postgres, models)
		if err != nil {
			logger.Fatalf("failed to open postgres -> %v", err)
		}
		logger.Info("postgres connected")

		// connect to nats
		logger.Debug("connect to nats")
		nc, err := nats.Connect(conf.NatsURL)
		if err != nil {
			logger.Fatalf("failed to connect to nats", err)
		}
		logger.Debug("encode nats connecting")
		natsConn, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
		if err != nil {
			logger.Fatalf("failed to encode nats connection", err)
		}
		logger.Info("nats connected")

		// instantiacte room manager and signaling API
		roomManagerAPI := room.NewAPI(db, logger, conf.AccessSecret)
		signalingAPI := signaling.NewAPI(db, logger, conf.ICEServers)

		// create services
		roomManagerSvc := server.NewRoomManagementService(
			roomManagerAPI, logger, natsConn,
			conf.EventNamespace, conf.AccessSecret,
		)
		signalingSvc := server.NewSignalingService(signalingAPI, logger, natsConn,
			conf.EventNamespace, conf.AccessSecret,
		)

		// create server
		logger.Infof("start server on port %d", conf.Port)
		svc := server.New(signalingSvc, roomManagerSvc, conf.Port)
		err = svc.Start()
		if err != nil {
			logger.Errorf("failed to start server", err)
		}
	},
}

func init() {
	rootCmd.Version = Version
}

// Execute root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
