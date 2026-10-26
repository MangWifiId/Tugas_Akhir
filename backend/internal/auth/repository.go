package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(
	ctx context.Context,
	name string,
	email string,
	passwordHash string,
) (User, error) {
	var user User

	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, role, is_active, created_at, updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		name,
		email,
		passwordHash,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return User{}, fmt.Errorf("gagal membuat pengguna: %w", err)
	}

	return user, nil
}

func (r *Repository) FindUserByEmail(
	ctx context.Context,
	email string,
) (User, error) {
	var user User

	query := `
		SELECT id, name, email, password_hash, role,
		       is_active, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, pgx.ErrNoRows
		}

		return User{}, fmt.Errorf("gagal mencari pengguna: %w", err)
	}

	return user, nil
}
