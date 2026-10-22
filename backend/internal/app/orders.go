package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

const (
	bookingCodeLen = 8
	ticketCodeLen  = 10
	codeAttempts   = 3
)

// OrderService covers buying, booking, paying, cancelling and refunding (UC08–UC21, UC28).
type OrderService struct {
	Repo    Repository
	Payment PaymentGateway
	Courier CourierService
	Now     Clock
	Log     *slog.Logger
}

// PlaceOrder validates and stores an order: a card checkout with a 10-minute hold, or a
// booking with a booking code.
func (s *OrderService) PlaceOrder(ctx context.Context, customerID int64, req domain.OrderRequest) (domain.Order, error) {
	sessionIDs, seatIDs := []int64{}, []int64{}
	for _, it := range req.Items {
		sessionIDs = append(sessionIDs, it.SessionID)
		if it.SeatID != 0 {
			seatIDs = append(seatIDs, it.SeatID)
		}
	}
	sessions, err := s.Repo.Sessions(ctx, sessionIDs)
	if err != nil {
		return domain.Order{}, err
	}
	seats, err := s.Repo.SeatZones(ctx, seatIDs)
	if err != nil {
		return domain.Order{}, err
	}
	paid, err := s.Repo.PaidOrderCount(ctx, customerID)
	if err != nil {
		return domain.Order{}, err
	}
	o, err := domain.BuildOrder(customerID, paid, req, sessions, seats, s.Now())
	if err != nil {
		return domain.Order{}, err
	}

	for attempt := 1; ; attempt++ {
		if o.Status == domain.Booked {
			o.BookingCode = newCode(bookingCodeLen)
		}
		for i := range o.Tickets {
			o.Tickets[i].Code = newCode(ticketCodeLen)
		}
		err = s.Repo.CreateOrder(ctx, &o)
		if !errors.Is(err, domain.ErrDuplicateCode) || attempt == codeAttempts {
			break
		}
	}
	if err != nil {
		return domain.Order{}, err
	}
	return s.Repo.GetOrder(ctx, o.ID)
}

// Order returns one order of the customer.
func (s *OrderService) Order(ctx context.Context, customerID, orderID int64) (domain.Order, error) {
	o, err := s.Repo.GetOrder(ctx, orderID)
	if err != nil {
		return o, err
	}
	if o.CustomerID != customerID {
		return domain.Order{}, domain.ErrNotFound
	}
	return o, nil
}

// History returns all orders of the customer, newest first (FR-29).
func (s *OrderService) History(ctx context.Context, customerID int64) ([]domain.Order, error) {
	return s.Repo.CustomerOrders(ctx, customerID)
}

// StartCardPayment registers a payment at the provider for a card checkout or an active
// booking and returns the provider page to redirect the customer to (FR-20, FR-24).
func (s *OrderService) StartCardPayment(ctx context.Context, customerID, orderID int64) (string, error) {
	var redirect string
	_, err := s.Repo.UpdateOrder(ctx, orderID, func(o *domain.Order) error {
		if o.CustomerID != customerID {
			return domain.ErrNotFound
		}
		now := s.Now()
		switch o.Status {
		case domain.PendingPayment:
			if o.HoldExpiresAt != nil && !now.Before(*o.HoldExpiresAt) {
				return domain.ErrHoldExpired
			}
		case domain.Booked:
			if !now.Before(o.BookingDeadline) {
				return domain.ErrBookingClosed
			}
		default:
			return domain.ErrInvalidTransition
		}
		ref, url, err := s.Payment.CreatePayment(ctx, o.ID, o.Total)
		if err != nil {
			return err
		}
		o.PaymentRef, redirect = ref, url
		return nil
	})
	return redirect, err
}

// ConfirmPayment handles the provider callback. It is idempotent: repeated callbacks do not
// change a paid order (NFR-05). A confirmation that arrives after the order expired or was
// cancelled is refunded automatically (FR-45).
func (s *OrderService) ConfirmPayment(ctx context.Context, ref, status, signature string) error {
	if !s.Payment.VerifyCallback(ref, status, signature) {
		return domain.ErrForbidden
	}
	o, err := s.Repo.OrderByPaymentRef(ctx, ref)
	if err != nil {
		return err
	}
	if status != "succeeded" {
		s.Log.Info("payment declined", "order", o.ID, "ref", ref)
		return nil // order stays unpaid; the hold or booking expires on schedule (FR-25)
	}
	var lateAmount domain.Money
	o, err = s.Repo.UpdateOrder(ctx, o.ID, func(o *domain.Order) error {
		if o.Status == domain.Paid {
			return nil // duplicate callback
		}
		now := s.Now()
		err := o.MarkPaid(domain.Card, now)
		switch {
		case errors.Is(err, domain.ErrHoldExpired), errors.Is(err, domain.ErrBookingClosed):
			lateAmount = o.Total // too late: release the seats now and refund
			return o.Expire(now)
		case errors.Is(err, domain.ErrInvalidTransition):
			lateAmount = o.Total // expired or cancelled meanwhile: refund
			return nil
		}
		return err
	})
	if err != nil {
		return err
	}
	if lateAmount > 0 {
		s.Log.Warn("late payment confirmation, refunding", "order", o.ID, "ref", ref)
		return s.Payment.Refund(ctx, ref, lateAmount)
	}
	s.requestDelivery(ctx, o)
	return nil
}

// requestDelivery sends a courier request for a paid courier order once (FR-26, FR-27).
func (s *OrderService) requestDelivery(ctx context.Context, o domain.Order) {
	if o.Status != domain.Paid || o.Delivery.Option != domain.Courier || o.Delivery.CourierRef != "" {
		return
	}
	req := DeliveryRequest{OrderID: o.ID, Address: o.Delivery.Address}
	for _, t := range o.Tickets {
		req.TicketCodes = append(req.TicketCodes, t.Code)
	}
	ref, err := s.Courier.RequestDelivery(ctx, req)
	status := "requested"
	if err != nil {
		s.Log.Error("courier request failed", "order", o.ID, "err", err)
		status = "failed"
	}
	_, err = s.Repo.UpdateOrder(ctx, o.ID, func(o *domain.Order) error {
		o.Delivery.CourierRef, o.Delivery.Status = ref, status
		return nil
	})
	if err != nil {
		s.Log.Error("saving courier status failed", "order", o.ID, "err", err)
	}
}

// Cancel releases an unpaid order of the customer (FR-23).
func (s *OrderService) Cancel(ctx context.Context, customerID, orderID int64) (domain.Order, error) {
	return s.Repo.UpdateOrder(ctx, orderID, func(o *domain.Order) error {
		if o.CustomerID != customerID {
			return domain.ErrNotFound
		}
		return o.Cancel()
	})
}

// Refund returns a card-paid order more than 24 h before its earliest session (FR-33).
// The provider refund runs inside the order transaction, so a failed refund leaves the
// order paid.
func (s *OrderService) Refund(ctx context.Context, customerID, orderID int64) (domain.Order, error) {
	o, err := s.Repo.UpdateOrder(ctx, orderID, func(o *domain.Order) error {
		if o.CustomerID != customerID {
			return domain.ErrNotFound
		}
		if err := o.Refund(s.Now()); err != nil {
			return err
		}
		return s.Payment.Refund(ctx, o.PaymentRef, o.Total)
	})
	if err != nil {
		return o, err
	}
	if err := s.Repo.Audit(ctx, customerID, "refund", fmt.Sprintf("order %d, %s", o.ID, o.Total)); err != nil {
		s.Log.Error("audit failed", "err", err)
	}
	return o, nil
}
