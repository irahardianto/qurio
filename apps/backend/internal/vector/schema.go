package vector

import (
	"context"

	"github.com/weaviate/weaviate/entities/models"
)

// SchemaClient defines the interface for Weaviate schema operations
type SchemaClient interface {
	ClassExists(ctx context.Context, className string) (bool, error)
	CreateClass(ctx context.Context, class *models.Class) error
	GetClass(ctx context.Context, className string) (*models.Class, error)
	AddProperty(ctx context.Context, className string, property *models.Property) error
}

// EnsureSchema checks if the required classes exist and creates them if not
func EnsureSchema(ctx context.Context, client SchemaClient) error {
	className := "DocumentChunk"
	exists, err := client.ClassExists(ctx, className)
	if err != nil {
		return err
	}

	properties := []*models.Property{
		{
			Name:     "content",
			DataType: []string{"text"},
		},
		{
			Name:     "sourceId",
			DataType: []string{"string"}, // UUID as string (exact match)
		},
		{
			Name:     "chunkIndex",
			DataType: []string{"int"},
		},
		{
			Name:     "title",
			DataType: []string{"text"},
		},
		{
			Name:     "url",
			DataType: []string{"string"}, // URL as string (exact match)
		},
		{
			Name:     "type",
			DataType: []string{"string"},
		},
		{
			Name:     "language",
			DataType: []string{"string"},
		},
	}

	if !exists {
		class := &models.Class{
			Class:       className,
			Description: "A chunk of a document",
			Vectorizer:  "none",
			Properties:  properties,
		}
		return client.CreateClass(ctx, class)
	}

	// Class exists, check for missing properties
	class, err := client.GetClass(ctx, className)
	if err != nil {
		return err
	}

	existingProps := make(map[string]bool)
	for _, p := range class.Properties {
		existingProps[p.Name] = true
	}

	for _, p := range properties {
		if !existingProps[p.Name] {
			if err := client.AddProperty(ctx, className, p); err != nil {
				return err
			}
		}
	}

	return nil
}