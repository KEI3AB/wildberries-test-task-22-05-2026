package main

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/rueidis"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"

	trendRepository "github.com/wildberries-test-task-22-05-2026/internal/repository/memory"
	stopListRepository "github.com/wildberries-test-task-22-05-2026/internal/repository/stoplist"

	transport "github.com/wildberries-test-task-22-05-2026/internal/transport/grpc"
	kafkaTransport "github.com/wildberries-test-task-22-05-2026/internal/transport/kafka"

	"github.com/wildberries-test-task-22-05-2026/internal/transport/grpc/pb"
	"github.com/wildberries-test-task-22-05-2026/internal/usecase"
)

func initKafkaConsumer(ctx context.Context, cfg KafkaConfig) (*kgo.Client, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ConsumeTopics(cfg.Topic),
		kgo.ConsumerGroup(cfg.GroupID),

		kgo.FetchMaxWait(cfg.MaxWait),
		kgo.FetchMaxBytes(cfg.FetchMaxBytes),
		kgo.ConnIdleTimeout(cfg.ConnTimeout),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, err
	}

	if err := client.Ping(ctx); err != nil {
		client.Close()
		return nil, err
	}

	slog.Info("connected to kafka",
		slog.Any("brokers", cfg.Brokers),
		slog.String("topic", cfg.Topic),
	)

	return client, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logHandler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(logHandler)
	slog.SetDefault(logger)

	// CONFIG
	cfg, err := NewConfig()
	if err != nil {
		slog.Error("critical config load error", slog.Any("err", err))
		os.Exit(1)
	}

	// REDIS
	opts := rueidis.ClientOption{
		InitAddress:      cfg.Redis.Addresses,
		BlockingPoolSize: cfg.Redis.BlockingPoolSize,
		Dialer: net.Dialer{
			Timeout: cfg.Redis.DialTimeout,
		},
		ConnWriteTimeout: cfg.Redis.ReadTimeout,
	}
	redisClient, err := rueidis.NewClient(opts)
	if err != nil {
		slog.Error("critical redis init error", slog.Any("err", err))
		os.Exit(1)
	}

	defer func() {
		slog.Info("closing redis client")
		redisClient.Close()
	}()

	// KAFKA
	kafkaClient, err := initKafkaConsumer(ctx, cfg.Kafka)
	if err != nil {
		slog.Error("critical kafka init error")
		os.Exit(1)
	}

	defer func() {
		slog.Info("closing kafka client")
		kafkaClient.Close()
	}()

	// Repository
	stopListRepo := stopListRepository.NewRedisRepository(redisClient)
	trendRepo := trendRepository.NewRingBuffer()

	stopListRepo.StartSync(ctx)
	trendRepo.StartCleaner(ctx)

	// Usecase
	stopListUC := usecase.NewStopListManager(stopListRepo)
	trendUC := usecase.NewTrendManager(trendRepo, stopListUC)

	kafkaConsumer := kafkaTransport.NewConsumer(kafkaClient, trendUC)
	go kafkaConsumer.Start(ctx)

	// Handler
	stopListHandler := transport.NewStopListHandler(stopListUC)
	trendHandler := transport.NewTrendHandler(trendUC)

	// gRPC
	kaOptions := grpc.KeepaliveParams(keepalive.ServerParameters{
		MaxConnectionIdle: cfg.GRPC.MaxConnIdle,
		Timeout:           cfg.GRPC.Timeout,
	})

	interceptors := grpc.ChainUnaryInterceptor(
		recovery.UnaryServerInterceptor(),
	)

	grpcServer := grpc.NewServer(
		kaOptions,
		interceptors,
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxRecvMsgSize),
	)

	pb.RegisterStopListServiceServer(grpcServer, stopListHandler)
	pb.RegisterTrendServiceServer(grpcServer, trendHandler)

	listener, err := net.Listen("tcp", ":"+cfg.GRPC.Port)
	if err != nil {
		slog.Error("critical listen tcp port error", slog.Any("err", err))
		os.Exit(1)
	}

	go func() {
		slog.Info("gRPC started", slog.String("port", cfg.GRPC.Port))
		if err := grpcServer.Serve(listener); err != nil {
			slog.Error("gRPC server failed", slog.Any("err", err))
		}
	}()

	<-ctx.Done()
	grpcServer.GracefulStop()
}
