### Task 1: Infrastructure & Configuration

**Files:**
- Modify: `apps/backend/internal/config/topics.go:1-20`
- Modify: `apps/backend/internal/config/config.go:1-50`
- Modify: `apps/backend/internal/app/bootstrap.go:1-100`
- Modify: `apps/backend/main.go:1-100`
- Modify: `apps/backend/internal/app/app.go:1-100`

**Requirements:**
- **Acceptance Criteria**
  1. `config.TopicIngestEmbed` constant is defined as `ingest.embed`.
  2. `ENABLE_API` and `ENABLE_EMBEDDER_WORKER` environment variables are loaded into the config.
  3. `ingest.embed` topic is created during application bootstrap.
  4. Application entry point (`main.go`) respects the toggles to start/stop components.

- **Functional Requirements**
  1. Define new topic constant.
  2. Add configuration fields for toggles and `INGESTION_CONCURRENCY`.
  3. Initialize `ingest.embed` topic in `createTopics`.
  4. Conditional startup logic in `main.go`.

- **Non-Functional Requirements**
  - None for this task.

- **Test Coverage**
  - [Unit] `config_test.go` - verify env vars are loaded.
  - [Integration] `bootstrap_test.go` - verify topic creation (if testable).

**Step 1: Write failing test**
```go
// apps/backend/internal/config/config_test.go
package config_test

import (
	"os"
	"testing"
	"qurio/apps/backend/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig_Toggles(t *testing.T) {
	os.Setenv("ENABLE_API", "false")
	os.Setenv("ENABLE_EMBEDDER_WORKER", "true")
	os.Setenv("INGESTION_CONCURRENCY", "10")
	defer os.Unsetenv("ENABLE_API")
	defer os.Unsetenv("ENABLE_EMBEDDER_WORKER")
	defer os.Unsetenv("INGESTION_CONCURRENCY")

	cfg, err := config.Load()
	assert.NoError(t, err)
	assert.False(t, cfg.EnableAPI)
	assert.True(t, cfg.EnableEmbedderWorker)
	assert.Equal(t, 10, cfg.IngestionConcurrency)
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/config/... -v`
Expected: FAIL (fields do not exist)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/config/topics.go
const (
	TopicIngestWeb    = "ingest.task.web"
	TopicIngestFile   = "ingest.task.file"
	TopicIngestResult = "ingest.result"
	TopicIngestEmbed  = "ingest.embed" // Added
)

// apps/backend/internal/config/config.go
type Config struct {
	// ... existing fields
	EnableAPI            bool `envconfig:"ENABLE_API" default:"true"`
	EnableEmbedderWorker bool `envconfig:"ENABLE_EMBEDDER_WORKER" default:"false"`
	IngestionConcurrency int  `envconfig:"INGESTION_CONCURRENCY" default:"1"`
}

// apps/backend/internal/app/bootstrap.go
// In createTopics function:
// ...
if err := producer.Publish(config.TopicIngestEmbed, []byte("ping")); err != nil {
    return fmt.Errorf("failed to create topic %s: %w", config.TopicIngestEmbed, err)
}

// apps/backend/main.go & app.go
// Use cfg.EnableAPI and cfg.EnableEmbedderWorker to conditionally start servers/consumers.
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/config/... -v`
Expected: PASS


### Task 2: Define Payload Schema

**Files:**
- Create: `apps/backend/internal/worker/events.go`

**Requirements:**
- **Acceptance Criteria**
  1. `IngestEmbedPayload` struct is defined with all necessary fields for reconstruction.

- **Functional Requirements**
  1. Include Source metadata (ID, URL, Name).
  2. Include Chunk data (Content, Index, Type, Language).
  3. Include Context metadata (Title, Author, CreatedAt).
  4. Include Tracing (CorrelationID).

- **Non-Functional Requirements**
  - JSON tags for serialization.

- **Test Coverage**
  - [Unit] JSON Marshaling/Unmarshaling test.

**Step 1: Write failing test**
```go
// apps/backend/internal/worker/events_test.go
package worker_test

import (
	"encoding/json"
	"testing"
	"qurio/apps/backend/internal/worker"
	"github.com/stretchr/testify/assert"
)

func TestIngestEmbedPayload_Serialization(t *testing.T) {
	payload := worker.IngestEmbedPayload{
		SourceID: "src-1",
		Content: "test content",
	}
	bytes, err := json.Marshal(payload)
	assert.NoError(t, err)
	assert.Contains(t, string(bytes), "source_id")
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: FAIL (undefined)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/worker/events.go
package worker

import "encoding/json"

type IngestEmbedPayload struct {
	SourceID      string            `json:"source_id"`
	SourceURL     string            `json:"source_url"`
	SourceName    string            `json:"source_name"`
	Title         string            `json:"title"`
	Path          string            `json:"path"`
	
	// Chunk Data
	Content       string            `json:"content"`
	ChunkIndex    int               `json:"chunk_index"`
	ChunkType     string            `json:"chunk_type"`
	Language      string            `json:"language"`
	
	// Context Metadata
	Author        string            `json:"author,omitempty"`
	CreatedAt     string            `json:"created_at,omitempty"`
	PageCount     int               `json:"page_count,omitempty"`
	
	CorrelationID string            `json:"correlation_id"`
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: PASS


### Task 3: Refactor ResultConsumer (The Coordinator)

**Files:**
- Modify: `apps/backend/internal/worker/result_consumer.go:100-200`

**Requirements:**
- **Acceptance Criteria**
  1. `ResultConsumer` no longer calls `h.embedder.Embed` or `h.store.StoreChunk`.
  2. `ResultConsumer` splits content using `text.ChunkMarkdown`.
  3. `ResultConsumer` publishes `IngestEmbedPayload` to `ingest.embed`.
  4. `DeleteChunksByURL` is retained.

- **Functional Requirements**
  1. Iterate chunks.
  2. Construct payload.
  3. Publish to NSQ.

- **Non-Functional Requirements**
  - Maintain logging context.

- **Test Coverage**
  - [Unit] `result_consumer_test.go` - mock publisher and verify `Publish` is called N times.

**Step 1: Write failing test**
```go
// apps/backend/internal/worker/result_consumer_test.go
// ... (Add test case for embedding delegation)
func TestResultConsumer_DelegatesEmbedding(t *testing.T) {
    // Setup mocks: Publisher expectation
    mockPublisher.On("Publish", config.TopicIngestEmbed, mock.Anything).Return(nil).Times(2) // Expect 2 chunks
    
    // ... triggering HandleMessage with content that produces 2 chunks
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: FAIL (calls embedder instead of publisher)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/worker/result_consumer.go
// ... inside HandleMessage ...

// 2. Chunk and Publish (Refactored)
if payload.Content != "" {
    chunks := text.ChunkMarkdown(payload.Content, 512, 50)
    for i, c := range chunks {
        embedPayload := IngestEmbedPayload{
            SourceID: payload.SourceID,
            SourceURL: payload.URL,
            // ... map fields
            Content: c.Content,
            ChunkIndex: i,
        }
        bytes, _ := json.Marshal(embedPayload)
        if err := h.publisher.Publish(config.TopicIngestEmbed, bytes); err != nil {
            return err // Durable: Fail if publish fails
        }
    }
}
// Remove Embed/Store calls
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: PASS


### Task 4: Implement Embedding Worker

**Files:**
- Create: `apps/backend/internal/worker/embedder_consumer.go`

**Requirements:**
- **Acceptance Criteria**
  1. `EmbedderConsumer` implements `nsq.Handler`.
  2. Reconstructs composite embedding string.
  3. Calls `Embedder` and `VectorStore`.
  4. Handles errors with backoff (via NSQ default or manual).
  5. Implements "Poison Pill" protection (discards invalid JSON).

- **Functional Requirements**
  1. Unmarshal `IngestEmbedPayload`.
  2. Recreate `contextualString`.
  3. `Embed()` with timeout.
  4. `StoreChunk()` with timeout.

- **Non-Functional Requirements**
  - 60s Timeout context.
  - Concurrency support (structure-wise).

- **Test Coverage**
  - [Unit] `embedder_consumer_test.go` - verify string reconstruction, store calls, and poison pill behavior.

**Step 1: Write failing test**
```go
// apps/backend/internal/worker/embedder_consumer_test.go
package worker_test
// ... Test HandleMessage calls Embed and Store with correct data
// ... Test HandleMessage returns nil (no error) for invalid JSON to prevent retry
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/worker/embedder_consumer.go
type EmbedderConsumer struct {
    embedder Embedder
    store    VectorStore
}

func (h *EmbedderConsumer) HandleMessage(m *nsq.Message) error {
    if len(m.Body) == 0 { return nil } // Acknowledge and drop
    var payload IngestEmbedPayload
    if err := json.Unmarshal(m.Body, &payload); err != nil {
        slog.Error("poison pill: invalid json", "error", err)
        return nil // Acknowledge and drop (Poison Pill)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()
    ctx = middleware.WithCorrelationID(ctx, payload.CorrelationID)

    // Reconstruct contextualString...
    // Call h.embedder.Embed(ctx, ...)
    // Call h.store.StoreChunk(ctx, ...)
    
    return nil // Success
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: PASS


### Task 5: Wire Up Worker & Concurrency

**Files:**
- Modify: `apps/backend/internal/app/app.go`

**Requirements:**
- **Acceptance Criteria**
  1. `EmbedderConsumer` is initialized if `ENABLE_EMBEDDER_WORKER` is true.
  2. `AddConcurrentHandlers` is used with `INGESTION_CONCURRENCY`.

- **Functional Requirements**
  1. Register consumer to `ingest.embed`.
  2. Start consumer.

- **Non-Functional Requirements**
  - Graceful shutdown support.

- **Test Coverage**
  - Manual verification via logs or integration test (Task 7).

**Step 1: Write failing test**
(Hard to unit test App wiring, verify via integration test in Task 7)

**Step 2: Verify test fails**
N/A

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/app/app.go
if a.cfg.EnableEmbedderWorker {
    ec := worker.NewEmbedderConsumer(a.embedder, a.vectorStore)
    consumer, _ := nsq.NewConsumer(config.TopicIngestEmbed, "backend-embedder", nsqConfig)
    consumer.AddConcurrentHandlers(ec, a.cfg.IngestionConcurrency)
    // ... connect and manage stop
}
```

**Step 4: Verify test passes**
Run: `go run ./apps/backend/cmd/server/main.go` (check logs)


### Task 6: Docker Compose Updates

**Files:**
- Modify: `docker-compose.yml:90-130`

**Requirements:**
- **Acceptance Criteria**
  1. `backend` service is duplicated/scaled or split into `backend-api` and `backend-worker`.
  2. `backend-worker` is configured with `ENABLE_API=false` and `ENABLE_EMBEDDER_WORKER=true`.
  3. `backend-api` is configured with `ENABLE_API=true` and `ENABLE_EMBEDDER_WORKER=false`.
  4. `INGESTION_CONCURRENCY` is set.

- **Functional Requirements**
  1. Define `backend-worker` service in `docker-compose.yml`.
  2. Ensure it connects to NSQ, Weaviate, etc.

- **Non-Functional Requirements**
  - None.

- **Test Coverage**
  - Manual verification: `docker compose up` starts services correctly.

**Step 1: Write failing test**
(N/A - Config change)

**Step 2: Verify test fails**
(N/A)

**Step 3: Write minimal implementation**
```yaml
# docker-compose.yml
  backend-worker:
    build: ./apps/backend
    environment:
      - ENABLE_API=false
      - ENABLE_EMBEDDER_WORKER=true
      - INGESTION_CONCURRENCY=${INGESTION_CONCURRENCY:-10}
      # ... other envs matching backend
    depends_on:
      - nsqd
      - weaviate
      - postgres
```

**Step 4: Verify test passes**
Run: `docker compose config` (Verify valid syntax)


### Task 7: Integration Testing (Testcontainers)

**Files:**
- Create: `apps/backend/internal/worker/embedder_integration_test.go`

**Requirements:**
- **Acceptance Criteria**
  1. Test spins up NSQ and consumers.
  2. Test publishes `IngestEmbedPayload` to `ingest.embed`.
  3. Test verifies `EmbedderConsumer` processes message and calls `Embed`/`Store` (using mocks injected into the consumer).
  4. Test verifies concurrency (optional, but good).

- **Functional Requirements**
  1. Use `testutils.NewIntegrationSuite`.
  2. Instantiate `EmbedderConsumer` with mocked dependencies.
  3. Connect to real NSQ (Testcontainer).

- **Non-Functional Requirements**
  - Robust against timing flakes.

- **Test Coverage**
  - This IS the integration test.

**Step 1: Write failing test**
```go
// apps/backend/internal/worker/embedder_integration_test.go
func TestEmbedderPipeline(t *testing.T) {
    s := testutils.NewIntegrationSuite(t) // Starts NSQ
    s.Setup()
    defer s.Teardown()
    
    // Setup Mock Embedder/Store
    mockEmbedder := &MockEmbedder{}
    mockStore := &MockStore{}
    
    // Start Consumer
    consumer := worker.NewEmbedderConsumer(mockEmbedder, mockStore)
    // ... connect to s.NSQDHost ...
    
    // Publish
    s.Publish(config.TopicIngestEmbed, payload)
    
    // Assert
    // Wait for mockStore to be called
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: FAIL (logic not implemented)

**Step 3: Write minimal implementation**
(Implement the test logic defined above)

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/... -v`
Expected: PASS