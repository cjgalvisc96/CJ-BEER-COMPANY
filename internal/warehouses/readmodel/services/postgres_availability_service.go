package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
)

// PostgresAvailabilityService is the durable adapter over the
// availabilities projection table versioned in migrations/.
type PostgresAvailabilityService struct {
	db *sql.DB
}

func NewPostgresAvailabilityService(db *sql.DB) *PostgresAvailabilityService {
	return &PostgresAvailabilityService{db: db}
}

func (s *PostgresAvailabilityService) UpsertAvailability(ctx context.Context, availability dtos.Availability) error {
	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO availabilities (beer_id, beer_name, quantity, unit_of_measure)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (beer_id) DO UPDATE SET
		   beer_name = excluded.beer_name,
		   quantity = excluded.quantity,
		   unit_of_measure = excluded.unit_of_measure,
		   projected_at = now()`,
		availability.BeerId, availability.BeerName,
		availability.Quantity.Value, availability.Quantity.UnitOfMeasure,
	); err != nil {
		return fmt.Errorf("project availability %s: %w", availability.BeerId, err)
	}
	return nil
}

func (s *PostgresAvailabilityService) GetAvailability(ctx context.Context, beerId string) (dtos.Availability, error) {
	var availability dtos.Availability
	err := s.db.QueryRowContext(ctx,
		`SELECT beer_id, beer_name, quantity, unit_of_measure
		   FROM availabilities WHERE beer_id = $1`, beerId,
	).Scan(&availability.BeerId, &availability.BeerName,
		&availability.Quantity.Value, &availability.Quantity.UnitOfMeasure)
	if errors.Is(err, sql.ErrNoRows) {
		return dtos.Availability{}, fmt.Errorf("%w: availability %s", muflone.ErrNotFound, beerId)
	}
	if err != nil {
		return dtos.Availability{}, fmt.Errorf("get availability %s: %w", beerId, err)
	}
	return availability, nil
}

func (s *PostgresAvailabilityService) GetAvailabilities(ctx context.Context) ([]dtos.Availability, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT beer_id, beer_name, quantity, unit_of_measure
		   FROM availabilities ORDER BY beer_name`)
	if err != nil {
		return nil, fmt.Errorf("list availabilities: %w", err)
	}
	defer rows.Close()

	availabilities := make([]dtos.Availability, 0)
	for rows.Next() {
		var availability dtos.Availability
		if err := rows.Scan(&availability.BeerId, &availability.BeerName,
			&availability.Quantity.Value, &availability.Quantity.UnitOfMeasure); err != nil {
			return nil, fmt.Errorf("list availabilities: %w", err)
		}
		availabilities = append(availabilities, availability)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list availabilities: %w", err)
	}
	return availabilities, nil
}
