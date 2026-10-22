// Package httpapi is the REST/JSON adapter (AC-2). Handlers decode requests, call the
// application services and map domain errors to HTTP status codes.
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Vanady/online-tickets/backend/internal/app"
	"github.com/Vanady/online-tickets/backend/internal/domain"
)

type Server struct {
	Accounts  *app.AccountService
	Catalog   *app.CatalogService
	Orders    *app.OrderService
	BoxOffice *app.BoxOfficeService
	Back      *app.BackOfficeService
	// Extra serves additional routes (the mock payment provider page).
	Extra http.Handler
	Log   *slog.Logger
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	customer := s.require(domain.RoleCustomer)
	cashier := s.require(domain.RoleCashier, domain.RoleAdmin)
	manager := s.require(domain.RoleManager, domain.RoleAdmin)
	admin := s.require(domain.RoleAdmin)
	anyUser := s.require(domain.RoleCustomer, domain.RoleCashier, domain.RoleManager, domain.RoleAdmin)

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// Catalogue (public).
	mux.HandleFunc("GET /api/events", s.listEvents)
	mux.HandleFunc("GET /api/events/{id}", s.getEvent)
	mux.HandleFunc("GET /api/sessions/{id}", s.getSession)
	mux.HandleFunc("GET /api/sessions/{id}/seats", s.getSeats)

	// Accounts.
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.Handle("GET /api/me", anyUser(s.me))

	// Orders (customer).
	mux.Handle("POST /api/orders", customer(s.placeOrder))
	mux.Handle("GET /api/orders", customer(s.history))
	mux.Handle("GET /api/orders/{id}", customer(s.getOrder))
	mux.Handle("POST /api/orders/{id}/pay", customer(s.pay))
	mux.Handle("POST /api/orders/{id}/cancel", customer(s.cancel))
	mux.Handle("POST /api/orders/{id}/refund", customer(s.refund))

	// Payment provider webhook.
	mux.HandleFunc("POST /api/payments/callback", s.paymentCallback)

	// Box office.
	mux.Handle("GET /api/boxoffice/bookings/{code}", cashier(s.findBooking))
	mux.Handle("POST /api/boxoffice/bookings/{code}/pay-cash", cashier(s.payCash))

	// Back office.
	mux.Handle("GET /api/admin/events", manager(s.adminEvents))
	mux.Handle("POST /api/admin/events", manager(s.createEvent))
	mux.Handle("PUT /api/admin/events/{id}", manager(s.updateEvent))
	mux.Handle("GET /api/admin/events/{id}/report", manager(s.report))
	mux.Handle("POST /api/admin/sessions", manager(s.createSession))
	mux.Handle("GET /api/admin/venues", manager(s.venues))
	mux.Handle("POST /api/admin/venues", admin(s.createVenue))

	if s.Extra != nil {
		mux.Handle("/mock-provider/", s.Extra)
	}
	return s.logRequests(mux)
}

// --- middleware

type ctxKey struct{}

type principal struct {
	ID   int64
	Role domain.Role
}

func userFrom(r *http.Request) principal { return r.Context().Value(ctxKey{}).(principal) }

// require authenticates the bearer token and checks the role (NFR-11).
func (s *Server) require(roles ...domain.Role) func(http.HandlerFunc) http.Handler {
	return func(next http.HandlerFunc) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !ok {
				writeError(w, http.StatusUnauthorized, "sign in required")
				return
			}
			id, role, err := s.Accounts.Authenticate(token)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "session expired, sign in again")
				return
			}
			for _, allowed := range roles {
				if role == allowed {
					ctx := context.WithValue(r.Context(), ctxKey{}, principal{ID: id, Role: role})
					next(w, r.WithContext(ctx))
					return
				}
			}
			writeError(w, http.StatusForbidden, "not allowed for your role")
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (s *Server) logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(sw, r)
		s.Log.Info("http", "method", r.Method, "path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds())
	})
}

// --- helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errors.Join(domain.ErrValidation, errors.New("malformed JSON body"))
	}
	return nil
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, domain.ErrNotFound
	}
	return id, nil
}

// fail maps domain errors to HTTP responses; unknown errors become 500 and are logged.
func (s *Server) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "not found")
	case errors.Is(err, domain.ErrBadCredentials):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrEmailTaken), errors.Is(err, domain.ErrSeatTaken), errors.Is(err, domain.ErrSoldOut):
		writeError(w, http.StatusConflict, err.Error())
	case isBusinessError(err):
		writeError(w, http.StatusUnprocessableEntity, messageOf(err))
	default:
		s.Log.Error("request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

var businessErrors = []error{
	domain.ErrValidation, domain.ErrEmptyOrder, domain.ErrTicketLimit, domain.ErrNotLoyal,
	domain.ErrSessionStarted, domain.ErrBookingClosed, domain.ErrSeatRequired, domain.ErrSeatNotInHall,
	domain.ErrDuplicateSeat, domain.ErrAddOnNotAllowed, domain.ErrUnknownAddOn, domain.ErrCourierNeedsCard,
	domain.ErrAddressRequired, domain.ErrInvalidTransition, domain.ErrHoldExpired, domain.ErrRefundClosed, domain.ErrRefundCardOnly,
	domain.ErrPaymentUnconfirmed,
}

func isBusinessError(err error) bool {
	for _, b := range businessErrors {
		if errors.Is(err, b) {
			return true
		}
	}
	return false
}

// messageOf drops the generic "invalid request" prefix of wrapped validation errors.
func messageOf(err error) string {
	msg := err.Error()
	if m, ok := strings.CutPrefix(msg, domain.ErrValidation.Error()+": "); ok {
		return m
	}
	return strings.ReplaceAll(msg, domain.ErrValidation.Error()+"\n", "")
}
