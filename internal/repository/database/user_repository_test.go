package database

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
)

func newMockDB(t *testing.T) (*sqlx.DB, sqlmock.Sqlmock) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")
	return sqlxDB, mock
}

func TestUserRepository_Create(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	testUser := &user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		Password:  "hashed_password",
		Email:     "john@example.com",
		Country:   "US",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO users").WithArgs(
			testUser.ID, testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password,
			testUser.Email, testUser.Country, testUser.CreatedAt, testUser.UpdatedAt,
		).WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, testUser)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("email taken", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO users").WithArgs(
			testUser.ID, testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password,
			testUser.Email, testUser.Country, testUser.CreatedAt, testUser.UpdatedAt,
		).WillReturnError(errors.New("unique constraint email"))

		err := repo.Create(ctx, testUser)
		assert.Equal(t, user.ErrEmailTaken, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nickname taken", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO users").WithArgs(
			testUser.ID, testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password,
			testUser.Email, testUser.Country, testUser.CreatedAt, testUser.UpdatedAt,
		).WillReturnError(errors.New("unique constraint nickname"))

		err := repo.Create(ctx, testUser)
		assert.Equal(t, user.ErrNicknameTaken, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("other error", func(t *testing.T) {
		mock.ExpectExec("INSERT INTO users").WithArgs(
			testUser.ID, testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password,
			testUser.Email, testUser.Country, testUser.CreatedAt, testUser.UpdatedAt,
		).WillReturnError(errors.New("database error"))

		err := repo.Create(ctx, testUser)
		assert.Error(t, err)
		assert.NotEqual(t, user.ErrEmailTaken, err)
		assert.NotEqual(t, user.ErrNicknameTaken, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByID(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	testUser := user.User{
		ID:        userID,
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		Password:  "hashed_password",
		Email:     "john@example.com",
		Country:   "US",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "nickname", "password_hash", "email", "country", "created_at", "updated_at"}).
			AddRow(testUser.ID, testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password, testUser.Email, testUser.Country, testUser.CreatedAt, testUser.UpdatedAt)

		mock.ExpectQuery("SELECT \\* FROM users WHERE id = \\$1").WithArgs(userID).WillReturnRows(rows)

		result, err := repo.GetByID(ctx, userID)
		assert.NoError(t, err)
		assert.Equal(t, testUser.ID, result.ID)
		assert.Equal(t, testUser.FirstName, result.FirstName)
		assert.Equal(t, testUser.LastName, result.LastName)
		assert.Equal(t, testUser.Nickname, result.Nickname)
		assert.Equal(t, testUser.Email, result.Email)
		assert.Equal(t, testUser.Country, result.Country)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM users WHERE id = \\$1").WithArgs(userID).WillReturnError(sql.ErrNoRows)

		result, err := repo.GetByID(ctx, userID)
		assert.Error(t, err)
		assert.Equal(t, user.ErrNotFound, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM users WHERE id = \\$1").WithArgs(userID).WillReturnError(errors.New("database error"))

		result, err := repo.GetByID(ctx, userID)
		assert.Error(t, err)
		assert.NotEqual(t, user.ErrNotFound, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByEmail(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()
	email := "john@example.com"

	testUser := user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		Password:  "hashed_password",
		Email:     email,
		Country:   "US",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "nickname", "password_hash", "email", "country", "created_at", "updated_at"}).
			AddRow(testUser.ID, testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password, testUser.Email, testUser.Country, testUser.CreatedAt, testUser.UpdatedAt)

		mock.ExpectQuery("SELECT \\* FROM users WHERE email = \\$1").WithArgs(email).WillReturnRows(rows)

		result, err := repo.GetByEmail(ctx, email)
		assert.NoError(t, err)
		assert.Equal(t, testUser.ID, result.ID)
		assert.Equal(t, testUser.Email, result.Email)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM users WHERE email = \\$1").WithArgs(email).WillReturnError(sql.ErrNoRows)

		result, err := repo.GetByEmail(ctx, email)
		assert.Error(t, err)
		assert.Equal(t, user.ErrNotFound, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_GetByNickname(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()
	nickname := "johndoe"

	testUser := user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  nickname,
		Password:  "hashed_password",
		Email:     "john@example.com",
		Country:   "US",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "nickname", "password_hash", "email", "country", "created_at", "updated_at"}).
			AddRow(testUser.ID, testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password, testUser.Email, testUser.Country, testUser.CreatedAt, testUser.UpdatedAt)

		mock.ExpectQuery("SELECT \\* FROM users WHERE nickname = \\$1").WithArgs(nickname).WillReturnRows(rows)

		result, err := repo.GetByNickname(ctx, nickname)
		assert.NoError(t, err)
		assert.Equal(t, testUser.ID, result.ID)
		assert.Equal(t, testUser.Nickname, result.Nickname)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectQuery("SELECT \\* FROM users WHERE nickname = \\$1").WithArgs(nickname).WillReturnError(sql.ErrNoRows)

		result, err := repo.GetByNickname(ctx, nickname)
		assert.Error(t, err)
		assert.Equal(t, user.ErrNotFound, err)
		assert.Nil(t, result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Update(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	testUser := &user.User{
		ID:        uuid.New(),
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		Password:  "hashed_password",
		Email:     "john@example.com",
		Country:   "US",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("UPDATE users SET").WithArgs(
			testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password,
			testUser.Email, testUser.Country, sqlmock.AnyArg(), testUser.ID,
		).WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Update(ctx, testUser)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("UPDATE users SET").WithArgs(
			testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password,
			testUser.Email, testUser.Country, sqlmock.AnyArg(), testUser.ID,
		).WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, testUser)
		assert.Error(t, err)
		assert.Equal(t, user.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("email taken", func(t *testing.T) {
		mock.ExpectExec("UPDATE users SET").WithArgs(
			testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password,
			testUser.Email, testUser.Country, sqlmock.AnyArg(), testUser.ID,
		).WillReturnError(errors.New("unique constraint email"))

		err := repo.Update(ctx, testUser)
		assert.Equal(t, user.ErrEmailTaken, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("nickname taken", func(t *testing.T) {
		mock.ExpectExec("UPDATE users SET").WithArgs(
			testUser.FirstName, testUser.LastName, testUser.Nickname, testUser.Password,
			testUser.Email, testUser.Country, sqlmock.AnyArg(), testUser.ID,
		).WillReturnError(errors.New("unique constraint nickname"))

		err := repo.Update(ctx, testUser)
		assert.Equal(t, user.ErrNicknameTaken, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_Delete(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM users WHERE id = \\$1").WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.Delete(ctx, userID)
		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("not found", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM users WHERE id = \\$1").WithArgs(userID).WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Delete(ctx, userID)
		assert.Error(t, err)
		assert.Equal(t, user.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		mock.ExpectExec("DELETE FROM users WHERE id = \\$1").WithArgs(userID).WillReturnError(errors.New("database error"))

		err := repo.Delete(ctx, userID)
		assert.Error(t, err)
		assert.NotEqual(t, user.ErrNotFound, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestUserRepository_List(t *testing.T) {
	db, mock := newMockDB(t)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()

	testUsers := []user.User{
		{
			ID:        uuid.New(),
			FirstName: "John",
			LastName:  "Doe",
			Nickname:  "johndoe",
			Password:  "hashed_password",
			Email:     "john@example.com",
			Country:   "US",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
		{
			ID:        uuid.New(),
			FirstName: "Jane",
			LastName:  "Smith",
			Nickname:  "janesmith",
			Password:  "hashed_password",
			Email:     "jane@example.com",
			Country:   "UK",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		},
	}

	t.Run("list all users", func(t *testing.T) {
		params := user.ListParams{
			Limit:     10,
			Offset:    0,
			OrderBy:   "created_at",
			OrderDesc: true,
		}

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(2)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).WillReturnRows(countRows)

		userRows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "nickname", "password_hash", "email", "country", "created_at", "updated_at"})
		for _, u := range testUsers {
			userRows.AddRow(u.ID, u.FirstName, u.LastName, u.Nickname, u.Password, u.Email, u.Country, u.CreatedAt, u.UpdatedAt)
		}

		mock.ExpectQuery("SELECT \\* FROM users ORDER BY created_at DESC LIMIT \\$1 OFFSET \\$2").WithArgs(params.Limit, params.Offset).WillReturnRows(userRows)

		users, count, err := repo.List(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, int64(2), count)
		assert.Len(t, users, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with filters", func(t *testing.T) {
		params := user.ListParams{
			Limit:     10,
			Offset:    0,
			OrderBy:   "created_at",
			OrderDesc: true,
			Filters: []user.Filter{
				{Field: "country", Value: "US"},
			},
		}

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(1)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE country ILIKE \$1`).WithArgs("%US%").WillReturnRows(countRows)

		userRows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "nickname", "password_hash", "email", "country", "created_at", "updated_at"})
		userRows.AddRow(
			testUsers[0].ID, testUsers[0].FirstName, testUsers[0].LastName, testUsers[0].Nickname,
			testUsers[0].Password, testUsers[0].Email, testUsers[0].Country, testUsers[0].CreatedAt, testUsers[0].UpdatedAt,
		)

		mock.ExpectQuery("SELECT \\* FROM users WHERE country ILIKE \\$1 ORDER BY created_at DESC LIMIT \\$2 OFFSET \\$3").WithArgs("%US%", params.Limit, params.Offset).WillReturnRows(userRows)

		users, count, err := repo.List(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
		assert.Len(t, users, 1)
		assert.Equal(t, "US", users[0].Country)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("empty result", func(t *testing.T) {
		params := user.ListParams{
			Limit:     10,
			Offset:    0,
			OrderBy:   "created_at",
			OrderDesc: true,
			Filters: []user.Filter{
				{Field: "country", Value: "FR"},
			},
		}

		countRows := sqlmock.NewRows([]string{"count"}).AddRow(0)
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE country ILIKE \$1`).WithArgs("%FR%").WillReturnRows(countRows)

		userRows := sqlmock.NewRows([]string{"id", "first_name", "last_name", "nickname", "password_hash", "email", "country", "created_at", "updated_at"})

		mock.ExpectQuery("SELECT \\* FROM users WHERE country ILIKE \\$1 ORDER BY created_at DESC LIMIT \\$2 OFFSET \\$3").WithArgs("%FR%", params.Limit, params.Offset).WillReturnRows(userRows)

		users, count, err := repo.List(ctx, params)
		assert.NoError(t, err)
		assert.Equal(t, int64(0), count)
		assert.Len(t, users, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("database error", func(t *testing.T) {
		params := user.ListParams{
			Limit:     10,
			Offset:    0,
			OrderBy:   "created_at",
			OrderDesc: true,
		}

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users`).WillReturnError(errors.New("database error"))

		users, count, err := repo.List(ctx, params)
		assert.Error(t, err)
		assert.Equal(t, int64(0), count)
		assert.Len(t, users, 0)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
