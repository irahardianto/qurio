package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
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

	m, err := migrate.New(migrationPath, connStr)
	require.NoError(s.T, err)
	require.NoError(s.T, m.Up())

	// 2. Weaviate
	req := testcontainers.ContainerRequest{
		Image:        "semitechnologies/weaviate:latest",
		ExposedPorts: []string{"8080/tcp", "50051/tcp"},
		Env: map[string]string{
			"AUTHENTICATION_ANONYMOUS_ACCESS_ENABLED": "true",
			"DEFAULT_VECTORIZER_MODULE":                 "none",
			"PERSISTENCE_DATA_PATH":                     "/var/lib/weaviate",
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
		Cmd:          []string{"/nsqd", "--broadcast-address=localhost"}, // Simplified for test
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
		s.pgContainer.Terminate(ctx)
	}
	if s.weaviateContainer != nil {
		s.weaviateContainer.Terminate(ctx)
	}
	if s.nsqContainer != nil {
		s.nsqContainer.Terminate(ctx)
	}
}
