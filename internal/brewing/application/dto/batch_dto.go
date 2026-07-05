// Package dto defines the data shapes crossing the brewing application
// boundary.
package dto

import (
	"time"

	"github.com/cjgalvisc96/cj-beer-company/internal/brewing/domain"
)

type BatchOutput struct {
	ID          string     `json:"id"`
	BeerID      string     `json:"beer_id"`
	Units       int        `json:"units"`
	Status      string     `json:"status"`
	StartedAt   time.Time  `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func BatchOutputFromEntity(batch *domain.Batch) BatchOutput {
	return BatchOutput{
		ID:          batch.ID().String(),
		BeerID:      batch.BeerID().String(),
		Units:       batch.Units(),
		Status:      string(batch.Status()),
		StartedAt:   batch.StartedAt(),
		CompletedAt: batch.CompletedAt(),
	}
}
