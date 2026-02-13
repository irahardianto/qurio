package vector

import (
	"context"
	"testing"

	"github.com/weaviate/weaviate/entities/models"
)

type MockSchemaClient struct {
	CreatedClass    *models.Class
	ExistingClass   *models.Class
	AddedProperties []*models.Property
}

func (m *MockSchemaClient) ClassExists(ctx context.Context, className string) (bool, error) {
	if m.ExistingClass != nil {
		return true, nil
	}
	return false, nil
}

func (m *MockSchemaClient) CreateClass(ctx context.Context, class *models.Class) error {
	m.CreatedClass = class
	return nil
}

func (m *MockSchemaClient) GetClass(ctx context.Context, className string) (*models.Class, error) {
	return m.ExistingClass, nil
}

func (m *MockSchemaClient) AddProperty(ctx context.Context, className string, property *models.Property) error {
	m.AddedProperties = append(m.AddedProperties, property)
	return nil
}

func TestEnsureSchema_CreatesClass(t *testing.T) {
	client := &MockSchemaClient{}
	if err := EnsureSchema(context.Background(), client); err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	if client.CreatedClass == nil {
		t.Fatal("Class not created")
	}

	expectedProps := map[string]string{
		"sourceId": "string",
		"url":      "string",
		"type":     "string",
		"language": "string",
	}

	for _, prop := range client.CreatedClass.Properties {
		if expectedType, ok := expectedProps[prop.Name]; ok {
			if len(prop.DataType) == 0 || prop.DataType[0] != expectedType {
				t.Errorf("Property %s has wrong DataType: %v (expected %s)", prop.Name, prop.DataType, expectedType)
			}
		}
	}
}

func TestEnsureSchema_AddsMissingProperties(t *testing.T) {
	// Simulate existing class without new properties
	existingClass := &models.Class{
		Class: "DocumentChunk",
		Properties: []*models.Property{
			{Name: "content", DataType: []string{"text"}},
			{Name: "sourceId", DataType: []string{"string"}},
		},
	}

	client := &MockSchemaClient{
		ExistingClass: existingClass,
	}

	if err := EnsureSchema(context.Background(), client); err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	if client.CreatedClass != nil {
		t.Fatal("Should not recreate class if it exists")
	}

	if len(client.AddedProperties) == 0 {
		t.Fatal("Should have added properties")
	}

	addedNames := make(map[string]bool)
	for _, p := range client.AddedProperties {
		addedNames[p.Name] = true
	}

	if !addedNames["type"] {
		t.Error("Missing 'type' property")
	}
	if !addedNames["language"] {
		t.Error("Missing 'language' property")
	}
	if addedNames["content"] {
		t.Error("Should not re-add existing 'content' property")
	}
}

func TestEnsureSchema_AddsNewMetadataProperties(t *testing.T) {
	// Simulate existing class with all old properties but missing new metadata ones
	existingClass := &models.Class{
		Class: "DocumentChunk",
		Properties: []*models.Property{
			{Name: "content", DataType: []string{"text"}},
			{Name: "sourceId", DataType: []string{"string"}},
			{Name: "sourceName", DataType: []string{"text"}},
			{Name: "chunkIndex", DataType: []string{"int"}},
			{Name: "title", DataType: []string{"text"}},
			{Name: "url", DataType: []string{"string"}},
			{Name: "type", DataType: []string{"string"}},
			{Name: "language", DataType: []string{"string"}},
		},
	}

	client := &MockSchemaClient{
		ExistingClass: existingClass,
	}

	if err := EnsureSchema(context.Background(), client); err != nil {
		t.Fatalf("EnsureSchema failed: %v", err)
	}

	addedNames := make(map[string]bool)
	for _, p := range client.AddedProperties {
		addedNames[p.Name] = true
	}

	if !addedNames["author"] {
		t.Error("Missing 'author' property")
	}
	if !addedNames["createdAt"] {
		t.Error("Missing 'createdAt' property")
	}
	if !addedNames["pageCount"] {
		t.Error("Missing 'pageCount' property")
	}
}
