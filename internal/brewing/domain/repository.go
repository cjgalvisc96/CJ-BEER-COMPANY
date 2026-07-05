package domain

import "context"

type BatchRepository interface {
	Save(ctx context.Context, batch *Batch) error
	FindByID(ctx context.Context, id BatchID) (*Batch, error)
	FindAll(ctx context.Context) ([]*Batch, error)
}
