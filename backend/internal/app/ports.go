// Package app holds the application services (use cases). They orchestrate domain rules and
// talk to the outside world only through the ports declared here.
package app

import (
	"context"
	"crypto/rand"
	"time"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

// Repository is the persistence port, implemented by the PostgreSQL store.
type Repository interface {
	ListEvents(ctx context.Context, f domain.EventFilter, now time.Time) ([]domain.Event, error)
	GetEvent(ctx context.Context, id int64, now time.Time) (domain.Event, error)
	Session(ctx context.Context, id int64) (domain.Session, error)
	Sessions(ctx context.Context, ids []int64) (map[int64]domain.Session, error)
	SeatZones(ctx context.Context, seatIDs []int64) (map[int64]domain.SeatZone, error)
	SeatMap(ctx context.Context, sessionID int64) ([]domain.SeatState, error)

	CreateUser(ctx context.Context, u domain.User) (domain.User, error)
	UserByEmail(ctx context.Context, email string) (domain.User, error)
	UserByID(ctx context.Context, id int64) (domain.User, error)
	PaidOrderCount(ctx context.Context, customerID int64) (int, error)

	CreateOrder(ctx context.Context, o *domain.Order) error
	GetOrder(ctx context.Context, id int64) (domain.Order, error)
	OrderByBookingCode(ctx context.Context, code string) (domain.Order, error)
	OrderByPaymentRef(ctx context.Context, ref string) (domain.Order, error)
	CustomerOrders(ctx context.Context, customerID int64) ([]domain.Order, error)
	UpdateOrder(ctx context.Context, id int64, fn func(*domain.Order) error) (domain.Order, error)
	ExpireDue(ctx context.Context, now time.Time) (int, error)

	CreateEvent(ctx context.Context, e domain.Event) (domain.Event, error)
	UpdateEvent(ctx context.Context, e domain.Event) (domain.Event, error)
	AllEvents(ctx context.Context) ([]domain.Event, error)
	CreateSession(ctx context.Context, ns domain.NewSession) (int64, error)
	CreateVenue(ctx context.Context, name, address string, halls []domain.HallSpec) (int64, error)
	Venues(ctx context.Context) ([]domain.Venue, error)
	SalesReport(ctx context.Context, eventID int64) (domain.EventReport, error)
	Audit(ctx context.Context, userID int64, action, details string) error
}

// PaymentGateway is the port to the external card payment provider (AC-4). Card data is
// entered only on the provider's page (BR-6).
type PaymentGateway interface {
	// CreatePayment registers a payment and returns its reference and the provider page URL.
	CreatePayment(ctx context.Context, orderID int64, amount domain.Money) (ref, redirectURL string, err error)
	// VerifyCallback checks that a payment result really comes from the provider.
	VerifyCallback(ref, status, signature string) bool
	Refund(ctx context.Context, ref string, amount domain.Money) error
}

// DeliveryRequest is sent to the courier service for a paid order (FR-26).
type DeliveryRequest struct {
	OrderID     int64
	Address     string
	TicketCodes []string
}

// CourierService is the port to the external courier company (AC-5).
type CourierService interface {
	RequestDelivery(ctx context.Context, req DeliveryRequest) (ref string, err error)
}

// Clock returns the current time; tests replace it.
type Clock func() time.Time

const codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // no 0/O, 1/I

// newCode returns a random code from codeAlphabet.
func newCode(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}
