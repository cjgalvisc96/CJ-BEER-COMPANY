package sales_test

import (
	"log/slog"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cjgalvisc96/cj-beer-company/internal/muflone"
	"github.com/cjgalvisc96/cj-beer-company/internal/platform/config"
	"github.com/cjgalvisc96/cj-beer-company/internal/sales"
)

// TestRegisterWiresDurablePersistence: with DB_URL configured the module
// selects the Postgres event store and read model (wiring only — behavior
// is covered by the adapter tests and the compose smoke test).
func TestRegisterWiresDurablePersistence(t *testing.T) {
	db, _, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	injector := do.New()
	do.ProvideValue(injector, slog.Default())
	do.ProvideValue(injector, config.Config{DBURL: "postgres://configured"})
	do.ProvideValue(injector, db)
	bus := muflone.NewServiceBus(slog.Default())
	t.Cleanup(func() { _ = bus.Close() })

	sales.Register(injector, bus)

	assert.NotNil(t, do.MustInvoke[*sales.Facade](injector))
}
