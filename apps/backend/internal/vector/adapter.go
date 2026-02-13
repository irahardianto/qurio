package vector

import (
	"context"

	"github.com/weaviate/weaviate-go-client/v5/weaviate"
	"github.com/weaviate/weaviate/entities/models"
)

type WeaviateClientAdapter struct {
	Client *weaviate.Client
}

func NewWeaviateClientAdapter(client *weaviate.Client) *WeaviateClientAdapter {
	return &WeaviateClientAdapter{Client: client}
}

func (a *WeaviateClientAdapter) ClassExists(ctx context.Context, className string) (bool, error) {
	return a.Client.Schema().ClassExistenceChecker().WithClassName(className).Do(ctx)
}

func (a *WeaviateClientAdapter) CreateClass(ctx context.Context, class *models.Class) error {
	return a.Client.Schema().ClassCreator().WithClass(class).Do(ctx)
}

func (a *WeaviateClientAdapter) GetClass(ctx context.Context, className string) (*models.Class, error) {
	return a.Client.Schema().ClassGetter().WithClassName(className).Do(ctx)
}

func (a *WeaviateClientAdapter) AddProperty(ctx context.Context, className string, property *models.Property) error {
	return a.Client.Schema().PropertyCreator().WithClassName(className).WithProperty(property).Do(ctx)
}
