package main

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	AppEnvironment string `env:"APP_ENV" envDefault:"development"`

	GRPC  GRPCConfig  `envPrefix:"GRPC_"`
	Redis RedisConfig `envPrefix:"REDIS_"`
	Kafka KafkaConfig `envPrefix:"KAFKA_"`
}

type GRPCConfig struct {
	Port           string        `env:"PORT" envDefault:"50051"`
	Timeout        time.Duration `env:"TIMEOUT" envDefault:"250ms"`
	MaxConnIdle    time.Duration `env:"MAX_CONN_IDLE" envDefault:"15s"`
	MaxRecvMsgSize int           `env:"MAX_RECV_MSG_SIZE" envDefault:"1048576"` // 1 Мб
}

type RedisConfig struct {
	Addresses        []string      `env:"ADDRESS" envDefault:"127.0.0.1:6379"`
	BlockingPoolSize int           `env:"POOL_SIZE" envDefault:"64"`
	DialTimeout      time.Duration `env:"DIAL_TIMEOUT" envDefault:"1s"`
	ReadTimeout      time.Duration `env:"READ_TIMEOUT" envDefault:"100ms"`
}

type KafkaConfig struct {
	Brokers       []string      `env:"BROKERS" envDefault:"127.0.0.1:9092"`
	Topic         string        `env:"TOPIC" envDefault:"trend-search-events"`
	GroupID       string        `env:"GROUP_ID" envDefault:"trend-service-v1"`
	MaxWait       time.Duration `env:"MAX_WAIT" envDefault:"250ms"`
	BatchSize     int           `env:"BATCH_SIZE" envDefault:"1000"`
	FetchMaxBytes int32         `env:"FETCH_MAX_BYTES" envDefault:"10485760"` // 10 Мб
	ConnTimeout   time.Duration `env:"CONN_TIMEOUT" envDefault:"5s"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("env parsing failed: %w", err)
	}

	return cfg, nil
}
