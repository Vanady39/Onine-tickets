package app

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

// CatalogService covers the programme, event pages and seat maps (UC01–UC04).
type CatalogService struct {
	Repo Repository
	Now  Clock
}

func (s *CatalogService) Events(ctx context.Context, f domain.EventFilter) ([]domain.Event, error) {
	if f.Type != "" && !f.Type.Valid() {
		return nil, fmt.Errorf("%w: unknown event type", domain.ErrValidation)
	}
	return s.Repo.ListEvents(ctx, f, s.Now())
}

func (s *CatalogService) Event(ctx context.Context, id int64) (domain.Event, error) {
	return s.Repo.GetEvent(ctx, id, s.Now())
}

func (s *CatalogService) Session(ctx context.Context, id int64) (domain.Session, error) {
	return s.Repo.Session(ctx, id)
}

func (s *CatalogService) SeatMap(ctx context.Context, sessionID int64) ([]domain.SeatState, error) {
	if _, err := s.Repo.Session(ctx, sessionID); err != nil {
		return nil, err
	}
	return s.Repo.SeatMap(ctx, sessionID)
}

// BoxOfficeService covers the cashier's work (UC17, UC23).
type BoxOfficeService struct {
	Repo Repository
	Now  Clock
}

// FindBooking looks up an order by its booking code (FR-21).
func (s *BoxOfficeService) FindBooking(ctx context.Context, code string) (domain.Order, error) {
	return s.Repo.OrderByBookingCode(ctx, strings.ToUpper(strings.TrimSpace(code)))
}

// AcceptCash records the cash payment of an active booking (FR-22) and audits it (NFR-12).
func (s *BoxOfficeService) AcceptCash(ctx context.Context, cashierID int64, code string) (domain.Order, error) {
	o, err := s.FindBooking(ctx, code)
	if err != nil {
		return o, err
	}
	o, err = s.Repo.UpdateOrder(ctx, o.ID, func(o *domain.Order) error {
		return o.MarkPaid(domain.Cash, s.Now())
	})
	if err != nil {
		return o, err
	}
	err = s.Repo.Audit(ctx, cashierID, "cash_payment", fmt.Sprintf("order %d, booking %s, %s", o.ID, o.BookingCode, o.Total))
	return o, err
}

// BackOfficeService covers events, sessions, venues and reports (UC24–UC27).
type BackOfficeService struct {
	Repo Repository
	Now  Clock
}

func validateEvent(e domain.Event) error {
	switch {
	case strings.TrimSpace(e.Title) == "":
		return fmt.Errorf("%w: title is required", domain.ErrValidation)
	case !e.Type.Valid():
		return fmt.Errorf("%w: unknown event type", domain.ErrValidation)
	case e.DurationMin < 0:
		return fmt.Errorf("%w: duration must not be negative", domain.ErrValidation)
	}
	return nil
}

func (s *BackOfficeService) Events(ctx context.Context) ([]domain.Event, error) {
	return s.Repo.AllEvents(ctx)
}

func (s *BackOfficeService) CreateEvent(ctx context.Context, e domain.Event) (domain.Event, error) {
	if err := validateEvent(e); err != nil {
		return e, err
	}
	return s.Repo.CreateEvent(ctx, e)
}

func (s *BackOfficeService) UpdateEvent(ctx context.Context, e domain.Event) (domain.Event, error) {
	if err := validateEvent(e); err != nil {
		return e, err
	}
	return s.Repo.UpdateEvent(ctx, e)
}

// CreateSession schedules a session (FR-35, FR-36). A seated hall needs a price for every
// zone; a general-admission hall needs a base price and may offer add-ons.
func (s *BackOfficeService) CreateSession(ctx context.Context, managerID int64, ns domain.NewSession) (int64, error) {
	if !ns.StartsAt.After(s.Now()) {
		return 0, fmt.Errorf("%w: session must start in the future", domain.ErrValidation)
	}
	ev, err := s.Repo.GetEvent(ctx, ns.EventID, s.Now())
	if err != nil {
		return 0, err
	}
	hall, err := s.hall(ctx, ns.HallID)
	if err != nil {
		return 0, err
	}
	if hall.IsSeated() {
		if len(ns.AddOns) > 0 {
			return 0, domain.ErrAddOnNotAllowed
		}
		prices := map[int64]domain.Money{}
		for _, p := range ns.Prices {
			if p.Price <= 0 {
				return 0, fmt.Errorf("%w: prices must be positive", domain.ErrValidation)
			}
			prices[p.ZoneID] = p.Price
		}
		for _, z := range hall.Zones {
			if _, ok := prices[z.ID]; !ok {
				return 0, fmt.Errorf("%w: price for zone %q is missing", domain.ErrValidation, z.Name)
			}
		}
		if len(prices) != len(hall.Zones) || len(ns.Prices) != len(prices) {
			return 0, fmt.Errorf("%w: prices must match the hall zones", domain.ErrValidation)
		}
		ns.BasePrice = 0
	} else {
		if ns.BasePrice <= 0 || len(ns.Prices) > 0 {
			return 0, fmt.Errorf("%w: general-admission sessions need a positive base price and no zone prices", domain.ErrValidation)
		}
		if len(ns.AddOns) > 0 && ev.Type != domain.Exhibition {
			return 0, domain.ErrAddOnNotAllowed
		}
		for _, a := range ns.AddOns {
			if !a.Kind.Valid() || strings.TrimSpace(a.Name) == "" || a.Price < 0 || a.Capacity <= 0 {
				return 0, fmt.Errorf("%w: invalid add-on", domain.ErrValidation)
			}
		}
	}
	id, err := s.Repo.CreateSession(ctx, ns)
	if err != nil {
		return 0, err
	}
	err = s.Repo.Audit(ctx, managerID, "session_prices",
		fmt.Sprintf("session %d of event %d: base %s, zones %v, add-ons %d", id, ns.EventID, ns.BasePrice, ns.Prices, len(ns.AddOns)))
	return id, err
}

func (s *BackOfficeService) hall(ctx context.Context, id int64) (domain.Hall, error) {
	venues, err := s.Repo.Venues(ctx)
	if err != nil {
		return domain.Hall{}, err
	}
	for _, v := range venues {
		for _, h := range v.Halls {
			if h.ID == id {
				return h, nil
			}
		}
	}
	return domain.Hall{}, domain.ErrNotFound
}

func (s *BackOfficeService) Venues(ctx context.Context) ([]domain.Venue, error) {
	return s.Repo.Venues(ctx)
}

// CreateVenue creates a venue with halls and seat maps (FR-37).
func (s *BackOfficeService) CreateVenue(ctx context.Context, name, address string, halls []domain.HallSpec) (int64, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(address) == "" || len(halls) == 0 {
		return 0, fmt.Errorf("%w: name, address and at least one hall are required", domain.ErrValidation)
	}
	for _, h := range halls {
		if strings.TrimSpace(h.Name) == "" {
			return 0, fmt.Errorf("%w: hall name is required", domain.ErrValidation)
		}
		switch h.Kind {
		case domain.Seated:
			if len(h.Zones) == 0 {
				return 0, fmt.Errorf("%w: seated hall needs price zones", domain.ErrValidation)
			}
			for _, z := range h.Zones {
				if strings.TrimSpace(z.Name) == "" || z.Rows <= 0 || z.Rows > 100 || z.SeatsPerRow <= 0 || z.SeatsPerRow > 100 {
					return 0, fmt.Errorf("%w: zone needs a name, 1–100 rows and 1–100 seats per row", domain.ErrValidation)
				}
			}
		case domain.GeneralAdmission:
			if h.Capacity <= 0 {
				return 0, fmt.Errorf("%w: capacity must be positive", domain.ErrValidation)
			}
		default:
			return 0, fmt.Errorf("%w: unknown hall kind", domain.ErrValidation)
		}
	}
	return s.Repo.CreateVenue(ctx, name, address, halls)
}

func (s *BackOfficeService) SalesReport(ctx context.Context, eventID int64) (domain.EventReport, error) {
	return s.Repo.SalesReport(ctx, eventID)
}

// ExpiryJob expires stale card holds and unpaid bookings (UC22, FR-19).
type ExpiryJob struct {
	Repo Repository
	Now  Clock
	Log  *slog.Logger
}

func (j *ExpiryJob) RunOnce(ctx context.Context) (int, error) {
	return j.Repo.ExpireDue(ctx, j.Now())
}

// Run calls RunOnce every interval until ctx is cancelled.
func (j *ExpiryJob) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		if n, err := j.RunOnce(ctx); err != nil {
			j.Log.Error("expiry job failed", "err", err)
		} else if n > 0 {
			j.Log.Info("expired orders", "count", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}
