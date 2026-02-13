package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/nsqio/go-nsq"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/weaviate/weaviate-go-client/v5/weaviate"

	"qurio/apps/backend/internal/config"
)

type IntegrationSuite struct {
	T        *testing.T
	DB       *sql.DB
	Weaviate *weaviate.Client
	NSQ      *nsq.Producer

	// Containers
	pgContainer       *postgres.PostgresContainer
	weaviateContainer testcontainers.Container
	nsqContainer      testcontainers.Container

	SkipMigrations bool
}

func NewIntegrationSuite(t *testing.T) *IntegrationSuite {
	return &IntegrationSuite{T: t}
}

func (s *IntegrationSuite) Setup() {
	ctx := context.Background()

	// 1. Postgres
	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("qurio_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(s.T, err)
	s.pgContainer = pgContainer

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(s.T, err)

	s.DB, err = sql.Open("postgres", connStr)
	require.NoError(s.T, err)

	// Run Migrations
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	migrationPath := fmt.Sprintf("file://%s/../../migrations", basepath)

	if !s.SkipMigrations {
		m, err := migrate.New(migrationPath, connStr)
		require.NoError(s.T, err)
		require.NoError(s.T, m.Up())
	}

	// 2. Weaviate
	req := testcontainers.ContainerRequest{
		Image:        "semitechnologies/weaviate:latest",
		ExposedPorts: []string{"8080/tcp", "50051/tcp"},
		Env: map[string]string{
			"AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED": "true",
			"DEFAULT_VECTORIZER_MODULE":               "none",
			"PERSISTENCE_DATA_PATH":                   "/var/lib/weaviate",
		},
		WaitingFor: wait.ForHTTP("/v1/meta").WithPort("8080/tcp").WithStartupTimeout(60 * time.Second),
	}
	weaviateC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(s.T, err)
	s.weaviateContainer = weaviateC

	host, err := weaviateC.Host(ctx)
	require.NoError(s.T, err)
	port, err := weaviateC.MappedPort(ctx, "8080")
	require.NoError(s.T, err)

	cfg := weaviate.Config{
		Host:   fmt.Sprintf("%s:%s", host, port.Port()),
		Scheme: "http",
	}
	s.Weaviate, err = weaviate.NewClient(cfg)
	require.NoError(s.T, err)

	// 3. NSQ
	nsqReq := testcontainers.ContainerRequest{
		Image:        "nsqio/nsq:v1.3.0",
		ExposedPorts: []string{"4150/tcp", "4151/tcp"},
		Cmd:          []string{"/nsqd", "--broadcast-address=localhost", "--max-msg-size=10485760"}, // 10MB limit
		WaitingFor:   wait.ForLog("TCP: listening on").WithStartupTimeout(60 * time.Second),
	}
	nsqC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: nsqReq,
		Started:          true,
	})
	require.NoError(s.T, err)
	s.nsqContainer = nsqC

	nsqHost, err := nsqC.Host(ctx)
	require.NoError(s.T, err)
	nsqPort, err := nsqC.MappedPort(ctx, "4150")
	require.NoError(s.T, err)

	nsqCfg := nsq.NewConfig()
	s.NSQ, err = nsq.NewProducer(fmt.Sprintf("%s:%s", nsqHost, nsqPort.Port()), nsqCfg)
	require.NoError(s.T, err)
}

func (s *IntegrationSuite) Teardown() {
	ctx := context.Background()
	if s.pgContainer != nil {
		if err := s.pgContainer.Terminate(ctx); err != nil {
			slog.Warn("failed to terminate postgres container", "error", err)
		}
	}
	if s.weaviateContainer != nil {
		if err := s.weaviateContainer.Terminate(ctx); err != nil {
			slog.Warn("failed to terminate weaviate container", "error", err)
		}
	}
	if s.nsqContainer != nil {
		if err := s.nsqContainer.Terminate(ctx); err != nil {
			slog.Warn("failed to terminate nsq container", "error", err)
		}
	}
}

func (s *IntegrationSuite) GetAppConfig() *config.Config {
	ctx := context.Background()

	// Postgres
	host, _ := s.pgContainer.Host(ctx)
	port, _ := s.pgContainer.MappedPort(ctx, "5432")

	// Weaviate
	wHost, _ := s.weaviateContainer.Host(ctx)
	wPort, _ := s.weaviateContainer.MappedPort(ctx, "8080")

	// NSQ
	nHost, _ := s.nsqContainer.Host(ctx)
	nPort, _ := s.nsqContainer.MappedPort(ctx, "4150")
	nHTTPPort, _ := s.nsqContainer.MappedPort(ctx, "4151")

	return &config.Config{
		DBHost:         host,
		DBPort:         port.Int(),
		DBUser:         "test",
		DBPass:         "test",
		DBName:         "qurio_test",
		WeaviateHost:   fmt.Sprintf("%s:%s", wHost, wPort.Port()),
		WeaviateScheme: "http",
		NSQDHost:       fmt.Sprintf("%s:%s", nHost, nPort.Port()),
		NSQDHTTP:       fmt.Sprintf("%s:%s", nHost, nHTTPPort.Port()),
	}
}

func (s *IntegrationSuite) Logger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, nil))
}

func (s *IntegrationSuite) GetNSQAddress() string {
	ctx := context.Background()
	host, _ := s.nsqContainer.Host(ctx)
	port, _ := s.nsqContainer.MappedPort(ctx, "4150")
	return fmt.Sprintf("%s:%s", host, port.Port())
}

func (s *IntegrationSuite) ConsumeOne(topic string) *nsq.Message {
	var msg *nsq.Message
	var wg sync.WaitGroup
	wg.Add(1)

	cfg := nsq.NewConfig()
	consumer, err := nsq.NewConsumer(topic, "test-ch-"+topic, cfg)
	require.NoError(s.T, err)

	consumer.AddHandler(nsq.HandlerFunc(func(m *nsq.Message) error {
		msg = m
		wg.Done()
		return nil
	}))

	err = consumer.ConnectToNSQD(s.GetNSQAddress())
	require.NoError(s.T, err)
	defer consumer.Stop()

	// Wait with timeout
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()

	select {
	case <-c:
		return msg
	case <-time.After(5 * time.Second):
		s.T.Fatalf("timeout waiting for message on topic %s", topic)
		return nil
	}
}
