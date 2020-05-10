package cmd

import (
	"encoding/json"
	"strings"

	"github.com/fatih/structs"
	nats "github.com/nats-io/nats.go"
	"github.com/spf13/viper"
	"go.sirus.dev/p2p-comm/signalling/pkg/connector"
	"go.sirus.dev/p2p-comm/signalling/pkg/signaling"
)

// Config define service configuration structure
type Config struct {
	LogLevel       string                    `mapstructure:"log_level"`
	Postgres       *connector.PostgresConfig `mapstructure:"postgres"`
	Port           int                       `mapstructure:"port"`
	EventNamespace string                    `mapstructure:"event_namespace"`
	AccessSecret   string                    `mapstructure:"access_secret"`
	NatsURL        string                    `mapstructure:"nats_url"`
	ICEServers     *[]*signaling.ICEServer   `mapstructure:"ice_servers"`
}

// DefaultConfig is default configuration
var DefaultConfig = Config{
	LogLevel:       "info",
	Postgres:       connector.DefaultPostgresConfig,
	Port:           9956,
	EventNamespace: "qh",
	NatsURL:        nats.DefaultOptions.Url,
	AccessSecret:   "access-secret",
	ICEServers: &[]*signaling.ICEServer{
		{URL: "stun.l.google.com:19302"},
		{URL: "stun.fwdnet.net"},
		{URL: "stunserver.org"},
	},
}

// String implement string interface
func (c *Config) String() string {
	val, _ := json.Marshal(c)
	return string(val)
}

var conf = viper.New()
var keys []string

// getEnvKeys will read environment keys
func getEnvKeys(fields []*structs.Field, prefix string) {
	for _, field := range fields {
		key := field.Tag("mapstructure")
		if prefix != "" {
			keys = append(keys, prefix+"."+key)
		} else {
			keys = append(keys, key)
		}
		if field.Kind().String() == "ptr" {
			if len(prefix) > 0 {
				key = prefix + "." + key
			}
			getEnvKeys(structs.Fields(field.Value()), key)
		}
	}
}

// LoadConfig will load configurations
func LoadConfig() (*Config, error) {
	// initiate config
	config := DefaultConfig
	// get all configurations keys
	fields := structs.Fields(config)
	getEnvKeys(fields, "")
	// read from config file
	conf.SetConfigName("config")
	conf.AddConfigPath("/etc/signalling/")
	conf.AddConfigPath("$HOME/.signalling")
	conf.AddConfigPath(".")
	conf.SetConfigType("yaml")
	_ = conf.ReadInConfig()
	// replace configurations using environment
	conf.SetEnvPrefix("signalling")
	conf.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	// reflect default configs to get all keys
	for _, key := range keys {
		_ = conf.BindEnv(key)
		val := conf.Get(key)
		conf.Set(key, val)
	}
	// unmarshal configuration
	err := conf.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
