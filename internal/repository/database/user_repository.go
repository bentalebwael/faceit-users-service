package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/bentalebwael/faceit-users-service/internal/domain/user"
)

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (
			id, first_name, last_name, nickname, password_hash,
			email, country, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)`

	_, err := r.db.ExecContext(ctx, query,
		u.ID, u.FirstName, u.LastName, u.Nickname, u.Password,
		u.Email, u.Country, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			if strings.Contains(err.Error(), "email") {
				return user.ErrEmailTaken
			}
			if strings.Contains(err.Error(), "nickname") {
				return user.ErrNicknameTaken
			}
		}
		return fmt.Errorf("error creating user: %w", err)
	}

	return nil
}

func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	var u user.User
	err := r.db.GetContext(ctx, &u, "SELECT * FROM users WHERE id = $1", id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("error getting user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	err := r.db.GetContext(ctx, &u, "SELECT * FROM users WHERE email = $1", email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("error getting user by email: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByNickname(ctx context.Context, nickname string) (*user.User, error) {
	var u user.User
	err := r.db.GetContext(ctx, &u, "SELECT * FROM users WHERE nickname = $1", nickname)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("error getting user by nickname: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users SET 
			first_name = $1, last_name = $2, nickname = $3,
			password_hash = $4, email = $5, country = $6,
			updated_at = $7
		WHERE id = $8`

	result, err := r.db.ExecContext(ctx, query,
		u.FirstName, u.LastName, u.Nickname, u.Password,
		u.Email, u.Country, time.Now().UTC(), u.ID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "unique constraint") {
			if strings.Contains(err.Error(), "email") {
				return user.ErrEmailTaken
			}
			if strings.Contains(err.Error(), "nickname") {
				return user.ErrNicknameTaken
			}
		}
		return fmt.Errorf("error updating user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}
	if rows == 0 {
		return user.ErrNotFound
	}

	return nil
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("error deleting user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}
	if rows == 0 {
		return user.ErrNotFound
	}

	return nil
}

func (r *UserRepository) List(ctx context.Context, params user.ListParams) ([]user.User, int64, error) {
	var conditions []string
	var filterArgs []interface{}
	argCount := 1

	for _, filter := range params.Filters {
		conditions = append(conditions, fmt.Sprintf("%s ILIKE $%d", filter.Field, argCount))
		filterArgs = append(filterArgs, fmt.Sprintf("%%%v%%", filter.Value))
		argCount++
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	countQuery := "SELECT COUNT(*) FROM users" + whereClause
	var totalCount int64
	err := r.db.GetContext(ctx, &totalCount, countQuery, filterArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting users: %w", err)
	}

	selectQuery := "SELECT * FROM users" + whereClause
	selectQuery += fmt.Sprintf(" ORDER BY %s", params.OrderBy)
	if params.OrderDesc {
		selectQuery += " DESC"
	} else {
		selectQuery += " ASC"
	}

	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argCount, argCount+1)
	args := append(filterArgs, params.Limit, params.Offset)

	users := make([]user.User, 0)
	err = r.db.SelectContext(ctx, &users, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("error listing users: %w", err)
	}

	return users, totalCount, nil
}
