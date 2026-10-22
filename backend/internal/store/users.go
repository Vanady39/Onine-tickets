package store

import (
	"context"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

func (s *Store) CreateUser(ctx context.Context, u domain.User) (domain.User, error) {
	err := s.pool.QueryRow(ctx,
		`INSERT INTO users (email, name, role, password_hash) VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		u.Email, u.Name, u.Role, u.PasswordHash).Scan(&u.ID, &u.CreatedAt)
	if isUniqueViolation(err, "users_email_key") {
		return domain.User{}, domain.ErrEmailTaken
	}
	return u, err
}

const userCols = `id, email, name, role, password_hash, created_at`

func scanUser(row interface{ Scan(...any) error }) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.PasswordHash, &u.CreatedAt)
	return u, notFound(err)
}

func (s *Store) UserByEmail(ctx context.Context, email string) (domain.User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE email = $1`, email))
}

func (s *Store) UserByID(ctx context.Context, id int64) (domain.User, error) {
	return scanUser(s.pool.QueryRow(ctx, `SELECT `+userCols+` FROM users WHERE id = $1`, id))
}

// PaidOrderCount counts orders that are paid and not refunded (BR-3).
func (s *Store) PaidOrderCount(ctx context.Context, customerID int64) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE customer_id = $1 AND status = 'paid'`,
		customerID).Scan(&n)
	return n, err
}

func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&n)
	return n, err
}
