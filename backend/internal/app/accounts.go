package app

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

// TokenTTL is the access token lifetime (NFR-23).
const TokenTTL = 24 * time.Hour

const bcryptCost = 10 // NFR-09

// AccountService covers registration, sign-in and profiles (UC05–UC07).
type AccountService struct {
	Repo      Repository
	JWTSecret []byte
	Now       Clock
}

type Claims struct {
	Role domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// Register creates a customer account (FR-07).
func (s *AccountService) Register(ctx context.Context, name, email, password string) (domain.User, error) {
	return s.createUser(ctx, name, email, password, domain.RoleCustomer)
}

// CreateStaff creates a cashier, manager or administrator account (used by seeding).
func (s *AccountService) CreateStaff(ctx context.Context, name, email, password string, role domain.Role) (domain.User, error) {
	if role == domain.RoleCustomer || !role.Valid() {
		return domain.User{}, fmt.Errorf("%w: staff role required", domain.ErrValidation)
	}
	return s.createUser(ctx, name, email, password, role)
}

func (s *AccountService) createUser(ctx context.Context, name, email, password string, role domain.Role) (domain.User, error) {
	name, email = strings.TrimSpace(name), normalizeEmail(email)
	if name == "" {
		return domain.User{}, fmt.Errorf("%w: name is required", domain.ErrValidation)
	}
	if _, err := mail.ParseAddress(email); err != nil || !strings.Contains(email, "@") {
		return domain.User{}, fmt.Errorf("%w: invalid e-mail", domain.ErrValidation)
	}
	if len(password) < 8 || len(password) > 72 {
		return domain.User{}, fmt.Errorf("%w: password must be 8–72 characters", domain.ErrValidation)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return domain.User{}, err
	}
	return s.Repo.CreateUser(ctx, domain.User{Name: name, Email: email, Role: role, PasswordHash: string(hash)})
}

// Login checks credentials and returns a signed access token (FR-08).
func (s *AccountService) Login(ctx context.Context, email, password string) (string, domain.User, error) {
	u, err := s.Repo.UserByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, domain.ErrNotFound) {
		return "", u, domain.ErrBadCredentials
	}
	if err != nil {
		return "", u, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return "", u, domain.ErrBadCredentials
	}
	now := s.Now()
	claims := Claims{
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprint(u.ID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(TokenTTL)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.JWTSecret)
	return token, u, err
}

// Authenticate validates a token and returns the user ID and role it carries.
func (s *AccountService) Authenticate(token string) (int64, domain.Role, error) {
	var c Claims
	_, err := jwt.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return s.JWTSecret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(s.Now), jwt.WithExpirationRequired())
	if err != nil {
		return 0, "", domain.ErrForbidden
	}
	var id int64
	if _, err := fmt.Sscan(c.Subject, &id); err != nil || !c.Role.Valid() {
		return 0, "", domain.ErrForbidden
	}
	return id, c.Role, nil
}

// Profile returns the user with loyalty status (FR-09).
func (s *AccountService) Profile(ctx context.Context, userID int64) (domain.Profile, error) {
	u, err := s.Repo.UserByID(ctx, userID)
	if err != nil {
		return domain.Profile{}, err
	}
	p := domain.Profile{User: u}
	if u.Role == domain.RoleCustomer {
		if p.PaidOrders, err = s.Repo.PaidOrderCount(ctx, userID); err != nil {
			return p, err
		}
		p.Loyal = domain.IsLoyal(p.PaidOrders)
	}
	return p, nil
}

func normalizeEmail(email string) string { return strings.ToLower(strings.TrimSpace(email)) }
