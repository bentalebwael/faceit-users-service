package cache

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bentalebwael/faceit-users-service/internal/config"
	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
)

type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	return ret.(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	args := m.Called(ctx, email)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	return ret.(*user.User), args.Error(1)
}

func (m *MockUserRepository) GetByNickname(ctx context.Context, nickname string) (*user.User, error) {
	args := m.Called(ctx, nickname)
	ret := args.Get(0)
	if ret == nil {
		return nil, args.Error(1)
	}
	return ret.(*user.User), args.Error(1)
}

func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	args := m.Called(ctx, u)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserRepository) List(ctx context.Context, params user.ListParams) ([]user.User, int64, error) {
	args := m.Called(ctx, params)
	retUsers := args.Get(0)
	retCount := args.Get(1)
	var users []user.User
	var count int64
	if retUsers != nil {
		users = retUsers.([]user.User)
	}
	if retCount != nil {
		count = retCount.(int64)
	}
	return users, count, args.Error(2)
}

func setupCacheTest(t *testing.T) (*CacheDecorator, *MockUserRepository, redismock.ClientMock) {
	mockRepo := new(MockUserRepository)
	db, mockRedis := redismock.NewClientMock()
	cfg := &config.RedisConfig{
		Addr: "localhost:6379",
	}

	cache := NewCacheDecorator(mockRepo, db, cfg)
	return cache, mockRepo, mockRedis
}

func TestCacheDecorator_Create(t *testing.T) {
	cache, mockRepo, mockRedis := setupCacheTest(t)
	ctx := context.Background()

	testUser := &user.User{
		ID:        uuid.New(),
		FirstName: "Cache",
		LastName:  "Test",
		Nickname:  "cachetest",
		Password:  "hashed",
		Email:     "cache@test.com",
		Country:   "CT",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	userData, err := json.Marshal(testUser)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		mockRepo.On("Create", ctx, testUser).Return(nil).Once()

		mockRedis.ExpectSet(userKey(testUser.ID), userData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(emailKey(testUser.Email), userData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(nickKey(testUser.Nickname), userData, cache.ttl).SetVal("OK")

		err := cache.Create(ctx, testUser)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("repo error", func(t *testing.T) {
		repoErr := errors.New("repo create error")
		mockRepo.On("Create", ctx, testUser).Return(repoErr).Once()

		err := cache.Create(ctx, testUser)
		assert.Equal(t, repoErr, err)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet()) // No expectations set
	})

	t.Run("cache error", func(t *testing.T) {
		cacheErr := errors.New("redis error")
		mockRepo.On("Create", ctx, testUser).Return(nil).Once()

		mockRedis.ExpectSet(userKey(testUser.ID), userData, cache.ttl).SetErr(cacheErr)

		err := cache.Create(ctx, testUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error caching user")
		assert.Contains(t, err.Error(), cacheErr.Error())
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})
}

func TestCacheDecorator_GetByID(t *testing.T) {
	cache, mockRepo, mockRedis := setupCacheTest(t)
	ctx := context.Background()
	userID := uuid.New()

	testUser := &user.User{ID: userID, Email: "get@id.com", Nickname: "getid"}
	userData, err := json.Marshal(testUser)
	require.NoError(t, err)

	t.Run("cache hit", func(t *testing.T) {
		mockRedis.ExpectGet(userKey(userID)).SetVal(string(userData))

		result, err := cache.GetByID(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testUser, result)
		mockRepo.AssertNotCalled(t, "GetByID")
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("cache miss", func(t *testing.T) {
		mockRedis.ExpectGet(userKey(userID)).SetErr(redis.Nil)
		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil).Once()

		mockRedis.ExpectSet(userKey(testUser.ID), userData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(emailKey(testUser.Email), userData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(nickKey(testUser.Nickname), userData, cache.ttl).SetVal("OK")

		result, err := cache.GetByID(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testUser, result)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("cache miss, repo error", func(t *testing.T) {
		repoErr := errors.New("repo get error")
		mockRedis.ExpectGet(userKey(userID)).SetErr(redis.Nil)
		mockRepo.On("GetByID", ctx, userID).Return(nil, repoErr).Once()

		result, err := cache.GetByID(ctx, userID)
		assert.Equal(t, repoErr, err)
		assert.Nil(t, result)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("cache miss, cache write error", func(t *testing.T) {
		cacheErr := errors.New("redis set error")
		mockRedis.ExpectGet(userKey(userID)).SetErr(redis.Nil)
		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil).Once()

		mockRedis.ExpectSet(userKey(testUser.ID), userData, cache.ttl).SetErr(cacheErr)

		result, err := cache.GetByID(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error caching user")
		assert.Nil(t, result) // Error occurred during caching, so return nil
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("cache read error (not redis.Nil)", func(t *testing.T) {
		cacheErr := errors.New("redis get error")
		mockRedis.ExpectGet(userKey(userID)).SetErr(cacheErr)

		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil).Once()
		mockRedis.ExpectSet(userKey(testUser.ID), userData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(emailKey(testUser.Email), userData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(nickKey(testUser.Nickname), userData, cache.ttl).SetVal("OK")

		result, err := cache.GetByID(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testUser, result)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("cache hit, unmarshal error", func(t *testing.T) {
		mockRedis.ExpectGet(userKey(userID)).SetVal("invalid json")

		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil).Once()
		mockRedis.ExpectSet(userKey(testUser.ID), userData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(emailKey(testUser.Email), userData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(nickKey(testUser.Nickname), userData, cache.ttl).SetVal("OK")

		result, err := cache.GetByID(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testUser, result)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})
}

// Similar tests should be written for GetByEmail and GetByNickname
// They follow the same pattern as GetByID, just using different cache keys.

func TestCacheDecorator_Update(t *testing.T) {
	cache, mockRepo, mockRedis := setupCacheTest(t)
	ctx := context.Background()
	userID := uuid.New()

	oldUser := &user.User{ID: userID, Email: "old@update.com", Nickname: "oldupdate"}
	updatedUser := &user.User{ID: userID, Email: "new@update.com", Nickname: "newupdate"}

	updatedUserData, err := json.Marshal(updatedUser)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		// Mock GetByID for invalidation check
		mockRepo.On("GetByID", ctx, userID).Return(oldUser, nil).Once()
		// Mock Update
		mockRepo.On("Update", ctx, updatedUser).Return(nil).Once()

		// Expect invalidation of old keys
		mockRedis.ExpectDel(userKey(oldUser.ID)).SetVal(1)
		mockRedis.ExpectDel(emailKey(oldUser.Email)).SetVal(1)
		mockRedis.ExpectDel(nickKey(oldUser.Nickname)).SetVal(1)

		// Expect caching of new user
		mockRedis.ExpectSet(userKey(updatedUser.ID), updatedUserData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(emailKey(updatedUser.Email), updatedUserData, cache.ttl).SetVal("OK")
		mockRedis.ExpectSet(nickKey(updatedUser.Nickname), updatedUserData, cache.ttl).SetVal("OK")

		// Mock GetByID for getting updated user
		mockRepo.On("GetByID", ctx, updatedUser.ID).Return(updatedUser, nil).Once()

		err := cache.Update(ctx, updatedUser)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("GetByID error", func(t *testing.T) {
		getErr := errors.New("get error before update")
		mockRepo.On("GetByID", ctx, userID).Return(nil, getErr).Once()

		err := cache.Update(ctx, updatedUser)
		assert.Equal(t, getErr, err)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("Update error", func(t *testing.T) {
		updateErr := errors.New("update repo error")
		mockRepo.On("GetByID", ctx, userID).Return(oldUser, nil).Once()
		mockRepo.On("Update", ctx, updatedUser).Return(updateErr).Once()

		err := cache.Update(ctx, updatedUser)
		assert.Equal(t, updateErr, err)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("Invalidation error", func(t *testing.T) {
		cacheErr := errors.New("redis del error")
		mockRepo.On("GetByID", ctx, userID).Return(oldUser, nil).Once()
		mockRepo.On("Update", ctx, updatedUser).Return(nil).Once()

		mockRedis.ExpectDel(userKey(oldUser.ID)).SetErr(cacheErr)

		err := cache.Update(ctx, updatedUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error invalidating cache")
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("Caching error after update", func(t *testing.T) {
		cacheErr := errors.New("redis set error")
		mockRepo.On("GetByID", ctx, userID).Return(oldUser, nil).Once()
		mockRepo.On("Update", ctx, updatedUser).Return(nil).Once()

		mockRedis.ExpectDel(userKey(oldUser.ID)).SetVal(1)
		mockRedis.ExpectDel(emailKey(oldUser.Email)).SetVal(1)
		mockRedis.ExpectDel(nickKey(oldUser.Nickname)).SetVal(1)

		mockRedis.ExpectSet(userKey(updatedUser.ID), updatedUserData, cache.ttl).SetErr(cacheErr)

		mockRepo.On("GetByID", ctx, updatedUser.ID).Return(updatedUser, nil).Once()

		err := cache.Update(ctx, updatedUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error caching updated user")
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})
}

func TestCacheDecorator_Delete(t *testing.T) {
	cache, mockRepo, mockRedis := setupCacheTest(t)
	ctx := context.Background()
	userID := uuid.New()

	testUser := &user.User{ID: userID, Email: "delete@me.com", Nickname: "deleteme"}

	t.Run("success", func(t *testing.T) {
		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil).Once()
		// Mock Delete
		mockRepo.On("Delete", ctx, userID).Return(nil).Once()

		mockRedis.ExpectDel(userKey(testUser.ID)).SetVal(1)
		mockRedis.ExpectDel(emailKey(testUser.Email)).SetVal(1)
		mockRedis.ExpectDel(nickKey(testUser.Nickname)).SetVal(1)

		err := cache.Delete(ctx, userID)
		assert.NoError(t, err)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("GetByID error", func(t *testing.T) {
		getErr := errors.New("get error before delete")
		mockRepo.On("GetByID", ctx, userID).Return(nil, getErr).Once()

		err := cache.Delete(ctx, userID)
		assert.Equal(t, getErr, err)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("Delete error", func(t *testing.T) {
		deleteErr := errors.New("delete repo error")
		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil).Once()
		mockRepo.On("Delete", ctx, userID).Return(deleteErr).Once()

		err := cache.Delete(ctx, userID)
		assert.Equal(t, deleteErr, err)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})

	t.Run("Invalidation error", func(t *testing.T) {
		cacheErr := errors.New("redis del error")
		mockRepo.On("GetByID", ctx, userID).Return(testUser, nil).Once()
		mockRepo.On("Delete", ctx, userID).Return(nil).Once()

		mockRedis.ExpectDel(userKey(testUser.ID)).SetErr(cacheErr)

		err := cache.Delete(ctx, userID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error invalidating cache")
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet())
	})
}

func TestCacheDecorator_List(t *testing.T) {
	cache, mockRepo, mockRedis := setupCacheTest(t)
	ctx := context.Background()

	params := user.ListParams{Limit: 10, Offset: 0}
	expectedUsers := []user.User{{ID: uuid.New()}}
	expectedCount := int64(1)

	t.Run("list bypasses cache", func(t *testing.T) {
		mockRepo.On("List", ctx, params).Return(expectedUsers, expectedCount, nil).Once()

		users, count, err := cache.List(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, expectedUsers, users)
		assert.Equal(t, expectedCount, count)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet()) // No expectations set
	})

	t.Run("list repo error", func(t *testing.T) {
		repoErr := errors.New("list repo error")
		mockRepo.On("List", ctx, params).Return(nil, int64(0), repoErr).Once()

		users, count, err := cache.List(ctx, params)
		assert.Equal(t, repoErr, err)
		assert.Nil(t, users)
		assert.Equal(t, int64(0), count)
		mockRepo.AssertExpectations(t)
		assert.NoError(t, mockRedis.ExpectationsWereMet()) // No expectations set
	})
}
