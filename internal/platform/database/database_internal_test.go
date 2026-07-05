package database

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stub driver: connects successfully so the happy path of open() is
// provable without a real database.
type stubDriver struct{}

func (stubDriver) Open(string) (driver.Conn, error) { return stubConn{}, nil }

type stubConn struct{}

func (stubConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (stubConn) Close() error                        { return nil }
func (stubConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }

func TestOpenSucceedsWhenDatabaseAnswers(t *testing.T) {
	sql.Register("database-stub", stubDriver{})

	db, err := open("database-stub", "ignored")

	require.NoError(t, err)
	require.NotNil(t, db)
	_ = db.Close()
}

func TestOpenFailsOnUnknownDriver(t *testing.T) {
	_, err := open("definitely-not-registered", "ignored")

	assert.ErrorContains(t, err, "open database")
}
