// Package services holds the Warehouses read-model services —
// BrewUp.Warehouses.ReadModel/Services (IAvailabilityQueryService). Two
// adapters: in-memory (dev/tests) and Postgres (production).
package services

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
)

// AvailabilityService is the in-memory adapter.
type AvailabilityService struct {
	mu             sync.RWMutex
	availabilities map[string]dtos.Availability
}

func NewAvailabilityService() *AvailabilityService {
	return &AvailabilityService{availabilities: make(map[string]dtos.Availability)}
}

// UpsertAvailability writes the projection for a beer.
func (s *AvailabilityService) UpsertAvailability(_ context.Context, availability dtos.Availability) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.availabilities[availability.BeerId] = availability
	return nil
}

func (s *AvailabilityService) GetAvailability(_ context.Context, beerId string) (dtos.Availability, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	availability, ok := s.availabilities[beerId]
	if !ok {
		return dtos.Availability{}, fmt.Errorf("%w: availability %s", muflone.ErrNotFound, beerId)
	}
	return availability, nil
}

func (s *AvailabilityService) GetAvailabilities(_ context.Context) ([]dtos.Availability, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	availabilities := make([]dtos.Availability, 0, len(s.availabilities))
	for _, availability := range s.availabilities {
		availabilities = append(availabilities, availability)
	}
	sort.Slice(availabilities, func(i, j int) bool {
		return availabilities[i].BeerName < availabilities[j].BeerName
	})
	return availabilities, nil
}
