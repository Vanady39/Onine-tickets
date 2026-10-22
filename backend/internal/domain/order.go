package domain

import (
	"time"
)

type OrderStatus string

const (
	PendingPayment OrderStatus = "pending_payment" // seats held during card checkout
	Booked         OrderStatus = "booked"          // unpaid booking with a booking code
	Paid           OrderStatus = "paid"
	Cancelled      OrderStatus = "cancelled"
	Expired        OrderStatus = "expired"
	Refunded       OrderStatus = "refunded"
)

// Active orders keep their seats.
func (s OrderStatus) Active() bool { return s == PendingPayment || s == Booked || s == Paid }

type PaymentMethod string

const (
	Card PaymentMethod = "card"
	Cash PaymentMethod = "cash"
)

type DeliveryOption string

const (
	ETicket DeliveryOption = "e_ticket"
	Courier DeliveryOption = "courier"
)

// HoldDuration is how long seats stay held while the customer pays by card (BR-7).
const HoldDuration = 10 * time.Minute

// CourierFee is the flat fee for courier delivery inside the city.
var CourierFee = RUB(300)

type Delivery struct {
	Option     DeliveryOption `json:"option"`
	Address    string         `json:"address,omitempty"`
	Fee        Money          `json:"fee"`
	CourierRef string         `json:"courierRef,omitempty"`
	Status     string         `json:"status,omitempty"` // requested, delivered, failed
}

type TicketAddOn struct {
	AddOnID int64  `json:"addOnId"`
	Name    string `json:"name"`
	Price   Money  `json:"price"`
}

type Ticket struct {
	ID        int64         `json:"id"`
	Code      string        `json:"code"`
	SessionID int64         `json:"sessionId"`
	EventID   int64         `json:"eventId"`
	SeatID    int64         `json:"seatId,omitempty"` // 0 for general admission
	Price     Money         `json:"price"`
	Active    bool          `json:"active"`
	AddOns    []TicketAddOn `json:"addOns"`

	// Read-only details for display.
	EventTitle string    `json:"eventTitle"`
	EventType  EventType `json:"eventType"`
	StartsAt   time.Time `json:"startsAt"`
	VenueName  string    `json:"venueName"`
	HallName   string    `json:"hallName"`
	ZoneName   string    `json:"zoneName,omitempty"`
	Row        int       `json:"row,omitempty"`
	Number     int       `json:"number,omitempty"`
}

// Total price of the ticket including add-ons.
func (t Ticket) Total() Money {
	sum := t.Price
	for _, a := range t.AddOns {
		sum += a.Price
	}
	return sum
}

type Order struct {
	ID            int64         `json:"id"`
	CustomerID    int64         `json:"customerId"`
	Status        OrderStatus   `json:"status"`
	PaymentMethod PaymentMethod `json:"paymentMethod"`
	BookingCode   string        `json:"bookingCode,omitempty"`
	// BookingDeadline = earliest session start − 24 h. Bookings expire then;
	// refunds are allowed only before it.
	BookingDeadline time.Time  `json:"bookingDeadline"`
	HoldExpiresAt   *time.Time `json:"holdExpiresAt,omitempty"`
	PaymentRef      string     `json:"paymentRef,omitempty"`
	Total           Money      `json:"total"`
	Delivery        Delivery   `json:"delivery"`
	Tickets         []Ticket   `json:"tickets"`
	CreatedAt       time.Time  `json:"createdAt"`
	PaidAt          *time.Time `json:"paidAt,omitempty"`
}

// IsSeasonTicket reports whether the order covers sessions of two or more events.
func (o Order) IsSeasonTicket() bool { return distinctEvents(o.Tickets) >= 2 }

func distinctEvents(tickets []Ticket) int {
	seen := map[int64]bool{}
	for _, t := range tickets {
		seen[t.EventID] = true
	}
	return len(seen)
}

// MarkPaid records a confirmed payment (card via provider, or cash at the box office).
func (o *Order) MarkPaid(method PaymentMethod, now time.Time) error {
	switch o.Status {
	case PendingPayment:
		if method != Card {
			return ErrInvalidTransition
		}
		if o.HoldExpiresAt != nil && !now.Before(*o.HoldExpiresAt) {
			return ErrHoldExpired
		}
	case Booked:
		if method == Cash && o.Delivery.Option == Courier {
			return ErrCourierNeedsCard
		}
		if !now.Before(o.BookingDeadline) {
			return ErrBookingClosed
		}
	default:
		return ErrInvalidTransition
	}
	o.Status = Paid
	o.PaymentMethod = method
	o.HoldExpiresAt = nil
	o.PaidAt = &now
	return nil
}

// Cancel releases an unpaid order (hold or booking) on the customer's request.
func (o *Order) Cancel() error {
	if o.Status != PendingPayment && o.Status != Booked {
		return ErrInvalidTransition
	}
	o.Status = Cancelled
	o.deactivate()
	return nil
}

// Expire releases an unpaid order whose hold or booking deadline has passed.
func (o *Order) Expire(now time.Time) error {
	switch {
	case o.Status == PendingPayment && o.HoldExpiresAt != nil && !now.Before(*o.HoldExpiresAt):
	case o.Status == Booked && !now.Before(o.BookingDeadline):
	default:
		return ErrInvalidTransition
	}
	o.Status = Expired
	o.deactivate()
	return nil
}

// Refund returns a paid card order if the earliest session is more than 24 h away (BR-5).
func (o *Order) Refund(now time.Time) error {
	if o.Status != Paid {
		return ErrInvalidTransition
	}
	if o.PaymentMethod != Card {
		return ErrRefundCardOnly
	}
	if !now.Before(o.BookingDeadline) {
		return ErrRefundClosed
	}
	o.Status = Refunded
	o.deactivate()
	return nil
}

func (o *Order) deactivate() {
	for i := range o.Tickets {
		o.Tickets[i].Active = false
	}
}
