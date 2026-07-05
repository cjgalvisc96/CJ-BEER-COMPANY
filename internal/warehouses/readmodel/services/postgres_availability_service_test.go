package services_test

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/shared/customtypes"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/dtos"
	"github.com/cjgalvisc96/cj-beer-company/internal/warehouses/readmodel/services"
)

func newPostgresService(t *testing.T) (*services.PostgresAvailabilityService, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return services.NewPostgresAvailabilityService(db), mock
}

func availabilityColumns() *sqlmock.Rows {
	return sqlmock.NewRows([]string{"beer_id", "beer_name", "quantity", "unit_of_measure"})
}

func TestPostgresUpsertAvailability(t *testing.T) {
	service, mock := newPostgresService(t)
	mock.ExpectExec("INSERT INTO availabilities").
		WithArgs("beer-1", "BrewUp IPA", 100, "Lt").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := service.UpsertAvailability(context.Background(), dtos.Availability{
		BeerId: "beer-1", BeerName: "BrewUp IPA", Quantity: customtypes.NewQuantity(100, "Lt"),
	})

	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresUpsertAvailabilityError(t *testing.T) {
	service, mock := newPostgresService(t)
	mock.ExpectExec("INSERT INTO availabilities").WillReturnError(assert.AnError)

	err := service.UpsertAvailability(context.Background(), dtos.Availability{BeerId: "beer-1"})

	assert.ErrorIs(t, err, assert.AnError)
}

func TestPostgresGetAvailability(t *testing.T) {
	service, mock := newPostgresService(t)
	mock.ExpectQuery("SELECT beer_id, beer_name").WithArgs("beer-1").
		WillReturnRows(availabilityColumns().AddRow("beer-1", "BrewUp IPA", 70, "Lt"))

	availability, err := service.GetAvailability(context.Background(), "beer-1")

	require.NoError(t, err)
	assert.Equal(t, 70, availability.Quantity.Value)
}

func TestPostgresGetAvailabilityErrors(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnRows(availabilityColumns())
		_, err := service.GetAvailability(context.Background(), "missing")
		assert.ErrorIs(t, err, muflone.ErrNotFound)
	})
	t.Run("query fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnError(assert.AnError)
		_, err := service.GetAvailability(context.Background(), "beer-1")
		assert.ErrorIs(t, err, assert.AnError)
	})
}

func TestPostgresGetAvailabilities(t *testing.T) {
	service, mock := newPostgresService(t)
	mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnRows(
		availabilityColumns().
			AddRow("beer-1", "Alpha IPA", 20, "Lt").
			AddRow("beer-2", "Zeta Stout", 10, "Lt"))

	availabilities, err := service.GetAvailabilities(context.Background())

	require.NoError(t, err)
	require.Len(t, availabilities, 2)
	assert.Equal(t, "Alpha IPA", availabilities[0].BeerName)
}

func TestPostgresGetAvailabilitiesErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("query fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnError(assert.AnError)
		_, err := service.GetAvailabilities(ctx)
		assert.ErrorIs(t, err, assert.AnError)
	})
	t.Run("scan fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnRows(
			availabilityColumns().AddRow("b", "n", "not-an-int", "Lt"))
		_, err := service.GetAvailabilities(ctx)
		assert.Error(t, err)
	})
	t.Run("iteration fails", func(t *testing.T) {
		service, mock := newPostgresService(t)
		mock.ExpectQuery("SELECT beer_id, beer_name").WillReturnRows(
			availabilityColumns().AddRow("b", "n", 1, "Lt").RowError(0, assert.AnError))
		_, err := service.GetAvailabilities(ctx)
		assert.ErrorIs(t, err, assert.AnError)
	})
}
