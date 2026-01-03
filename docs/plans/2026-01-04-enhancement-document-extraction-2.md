### Task 1: Update Result Consumer Contract

**Files:**
- Modify: `apps/backend/internal/worker/result_consumer.go`

**Requirements:**
- **Acceptance Criteria**
  1. `ResultConsumer` struct successfully parses the `metadata` JSON object from the `ingest.result` topic.
  2. The `metadata` fields (`author`, `created_at`, `pages`) are extracted and available for embedding.

- **Functional Requirements**
  1. Update `payload` struct in `HandleMessage` to include `Metadata map[string]interface{} 'json:"metadata"' `.
  2. Ensure backward compatibility (if `metadata` is missing, code doesn't panic).

- **Non-Functional Requirements**
  - **Zero Downtime:** Old workers (without metadata) must still work with new backend.

- **Test Coverage**
  - [Unit] `TestHandleMessage_WithMetadata` - Mock NSQ message with `metadata`, verify parser.
  - [Unit] `TestHandleMessage_WithoutMetadata` - Verify existing behavior.

**Step 1: Write failing test**
```go
// apps/backend/internal/worker/result_consumer_test.go

func TestHandleMessage_WithMetadata(t *testing.T) {
    // Setup mock dependencies...
    consumer := NewResultConsumer(...) 
    
    // Create payload with NEW metadata field
    payload := map[string]interface{}{
        "source_id": "123",
        "content": "test content",
        "metadata": map[string]interface{}{
            "author": "John Doe",
            "pages": 10,
        },
    }
    msgBody, _ := json.Marshal(payload)
    msg := &nsq.Message{Body: msgBody}
    
    // Act
    err := consumer.HandleMessage(msg)
    
    // Assert
    // Verify that the Embedder was called with a string containing "Author: John Doe"
    // This will fail because the current implementation ignores 'metadata'
    assert.NoError(t, err)
    mockEmbedder.AssertCalled(t, "Embed", mock.Anything, mock.MatchedBy(func(s string) bool {
        return strings.Contains(s, "Author: John Doe")
    }))
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/...`
Expected: FAIL (Embedder called without Author string)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/worker/result_consumer.go

// Update struct inside HandleMessage
type payload struct {
    // ... existing fields ...
    Metadata map[string]interface{} `json:"metadata"` // Add this
}

// Inside the embedding loop
contextualString := fmt.Sprintf("Title: %s\nSource: %s\nPath: %s\nURL: %s\nType: %s", 
    payload.Title, sourceName, payload.Path, payload.URL, string(c.Type))

// Add metadata to context if present
if author, ok := payload.Metadata["author"].(string); ok {
    contextualString += fmt.Sprintf("\nAuthor: %s", author)
}
if created, ok := payload.Metadata["created_at"].(string); ok {
    contextualString += fmt.Sprintf("\nCreated: %s", created)
}

contextualString += fmt.Sprintf("\n---\n%s", c.Content)
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/...`
Expected: PASS


### Task 2: Vector Schema Migration

**Files:**
- Modify: `apps/backend/internal/vector/schema.go`

**Requirements:**
- **Acceptance Criteria**
  1. Weaviate `DocumentChunk` class has new properties: `author` (text), `createdAt` (date), `pageCount` (int).
  2. `EnsureSchema` function is idempotent (doesn't fail if props exist).

- **Functional Requirements**
  1. Add `author`, `createdAt`, `pageCount` to `properties` list.
  2. Run `EnsureSchema` on startup (already handled by main).

- **Non-Functional Requirements**
  - **Migration Safety:** Must check `ClassExists` and use `AddProperty` for existing classes.

- **Test Coverage**
  - [Unit] `TestEnsureSchema_AddsNewProperties` - Mock SchemaClient, verify `AddProperty` is called for new fields.

**Step 1: Write failing test**
```go
// apps/backend/internal/vector/schema_test.go
func TestEnsureSchema_AddsNewProperties(t *testing.T) {
    mockClient := new(MockSchemaClient)
    // Setup: Class exists but missing 'author'
    mockClient.On("ClassExists", ...).Return(true, nil)
    mockClient.On("GetClass", ...).Return(&models.Class{Properties: oldProps}, nil)
    
    // Expect AddProperty call
    mockClient.On("AddProperty", mock.Anything, "DocumentChunk", mock.MatchedBy(func(p *models.Property) bool {
        return p.Name == "author"
    })).Return(nil)
    
    err := EnsureSchema(context.Background(), mockClient)
    assert.NoError(t, err)
    mockClient.AssertExpectations(t)
}
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/vector/...`
Expected: FAIL (AddProperty not called)

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/vector/schema.go

properties := []*models.Property{
    // ... existing ...
    {
        Name: "author",
        DataType: []string{"text"},
    },
    {
        Name: "createdAt",
        DataType: []string{"date"},
    },
    {
        Name: "pageCount",
        DataType: []string{"int"},
    },
}
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/vector/...`
Expected: PASS


### Task 3: Store Metadata in Weaviate

**Files:**
- Modify: `apps/backend/internal/adapter/weaviate/store.go`
- Modify: `apps/backend/internal/worker/types.go` (Chunk struct)

**Requirements:**
- **Acceptance Criteria**
  1. `StoreChunk` method maps the new Chunk fields to Weaviate properties.

- **Functional Requirements**
  1. Update `worker.Chunk` struct to include `Author`, `CreatedAt`, `PageCount`.
  2. Update `ResultConsumer` to map payload metadata to these Chunk fields.
  3. Update `weaviate.Store` to save them.

- **Non-Functional Requirements**
  None.

- **Test Coverage**
  - [Integration] `TestStoreChunk_PersistsMetadata` - Store chunk, retrieve it, verify metadata.

**Step 1: Write failing test**
```go
// apps/backend/internal/worker/result_consumer_test.go (Integration-like unit test)
// Verify that the consumer populates the Chunk struct correctly before calling Store
```

**Step 2: Verify test fails**
Run: `go test ./apps/backend/internal/worker/...`
Expected: FAIL

**Step 3: Write minimal implementation**
```go
// apps/backend/internal/worker/types.go
type Chunk struct {
    // ...
    Author    string
    CreatedAt string
    PageCount int
}

// apps/backend/internal/worker/result_consumer.go
chunk := Chunk{
    // ...
    Author:    payload.Metadata["author"].(string),
    // ...
}

// apps/backend/internal/adapter/weaviate/store.go
if chunk.Author != "" {
    properties["author"] = chunk.Author
}
// ...
```

**Step 4: Verify test passes**
Run: `go test ./apps/backend/internal/worker/...`
Expected: PASS

```