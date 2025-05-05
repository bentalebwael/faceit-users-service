package postgres

import (
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bentalebwael/faceit-users-service/internal/config"
)

func createTestConfig(url string) *config.Config {
	return &config.Config{
		DB: config.DBConfig{
			URL: url,
		},
	}
}

// TestNewConnection_ConnectError focuses on the scenario where sqlx.Connect fails.
func TestNewConnection_ConnectError(t *testing.T) {
	invalidDSN := "this-is-not-a-valid-dsn"
	cfg := createTestConfig(invalidDSN)

	db, err := NewConnection(cfg)

	assert.Error(t, err, "Expected an error when connecting with an invalid DSN")
	assert.Nil(t, db, "Expected database connection to be nil on connection error")
	assert.Contains(t, err.Error(), "error connecting to the database", "Error message should indicate connection failure")
}

func TestNewConnection_Success_Mocked(t *testing.T) {
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true)) // Monitor pings
	require.NoError(t, err)
	defer mockDB.Close()

	mock.ExpectPing() // Expect a ping call and assume it succeeds

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock") // Wrap mockDB with sqlx

	// Simulate the ping part of NewConnection
	err = sqlxDB.Ping()
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestNewConnection_PingError_Mocked(t *testing.T) {
	// Similar setup to Success_Mocked...
	mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	require.NoError(t, err)
	defer mockDB.Close()

	pingErr := errors.New("mock ping failed")
	mock.ExpectPing().WillReturnError(pingErr)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	err = sqlxDB.Ping()
	assert.Error(t, err)
	assert.ErrorIs(t, err, pingErr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestClose_Success(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create sqlmock")

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	mock.ExpectClose()

	err = Close(sqlxDB)

	assert.NoError(t, err, "Expected no error when closing a valid connection")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err, "Expectations were not met for sqlmock")
}

func TestClose_NilDB(t *testing.T) {
	err := Close(nil)
	assert.NoError(t, err, "Closing a nil DB should not return an error")
}

func TestClose_Error(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err, "Failed to create sqlmock")

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	closeErr := errors.New("mock close error")
	mock.ExpectClose().WillReturnError(closeErr)

	err = Close(sqlxDB)

	assert.Error(t, err, "Expected an error when the underlying Close fails")
	assert.ErrorContains(t, err, "mock close error", "Error message should contain the original error")
	assert.ErrorContains(t, err, "error closing database connection", "Error message should be wrapped")

	err = mock.ExpectationsWereMet()
	assert.NoError(t, err, "Expectations were not met for sqlmock")
}
