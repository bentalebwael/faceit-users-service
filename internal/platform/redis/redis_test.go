package redis

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Interaction_PingSuccess(t *testing.T) {
	dialTimeout := 5 * time.Second
	db, mock := redismock.NewClientMock()
	defer db.Close()

	mock.ExpectPing().SetVal("PONG")

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	err := db.Ping(ctx).Err()

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet(), "Ping expectation should be met")
}

func TestNewClient_Interaction_PingFails(t *testing.T) {
	dialTimeout := 5 * time.Second
	db, mock := redismock.NewClientMock()
	defer db.Close()

	expectedErr := errors.New("connection refused")
	mock.ExpectPing().SetErr(expectedErr)

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	err := db.Ping(ctx).Err()

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, mock.ExpectationsWereMet(), "Ping expectation should be met")
}

func TestClose_Success(t *testing.T) {
	mockClient, _ := redismock.NewClientMock()

	err := Close(mockClient)

	assert.NoError(t, err, "Close should not return an error on success path")

}

func TestClose_NilClient(t *testing.T) {
	err := Close(nil)

	assert.NoError(t, err, "Close should handle nil client gracefully without error")
}
