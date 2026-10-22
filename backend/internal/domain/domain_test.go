package domain

import (
	"errors"
	"testing"
	"time"
)

var now = time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)

// Fixtures: a seated concert (event 1) in hall 10, a second concert (event 2) in the same hall,
// and an exhibition (event 3) in a general-admission hall 20.
func fixtures() (map[int64]Session, map[int64]SeatZone) {
	seated := Hall{ID: 10, Kind: Seated, Capacity: 100}
	ga := Hall{ID: 20, Kind: GeneralAdmission, Capacity: 50}
	sessions := map[int64]Session{
		1: {ID: 1, EventID: 1, EventType: Concert, Hall: seated, StartsAt: now.Add(72 * time.Hour),
			Prices: []ZonePrice{{ZoneID: 100, Price: RUB(1000)}, {ZoneID: 101, Price: RUB(3000)}}},
		2: {ID: 2, EventID: 2, EventType: Concert, Hall: seated, StartsAt: now.Add(96 * time.Hour),
			Prices: []ZonePrice{{ZoneID: 100, Price: RUB(800)}, {ZoneID: 101, Price: RUB(2000)}}},
		3: {ID: 3, EventID: 3, EventType: Exhibition, Hall: ga, StartsAt: now.Add(48 * time.Hour),
			BasePrice: RUB(500), AddOns: []AddOnService{{ID: 7, Name: "Lunch", Price: RUB(1200), Capacity: 10}}},
		4: {ID: 4, EventID: 1, EventType: Concert, Hall: seated, StartsAt: now.Add(10 * time.Hour),
			Prices: []ZonePrice{{ZoneID: 100, Price: RUB(1000)}}},
	}
	seats := map[int64]SeatZone{
		1001: {HallID: 10, ZoneID: 100}, 1002: {HallID: 10, ZoneID: 100},
		1003: {HallID: 10, ZoneID: 101}, 1004: {HallID: 10, ZoneID: 100},
		9001: {HallID: 99, ZoneID: 900},
	}
	return sessions, seats
}

func build(t *testing.T, paid int, req OrderRequest) (Order, error) {
	t.Helper()
	sessions, seats := fixtures()
	return BuildOrder(42, paid, req, sessions, seats, now)
}

func TestBuildOrderPricesSeatsAndHoldsForCardPayment(t *testing.T) {
	o, err := build(t, 0, OrderRequest{
		Items:         []OrderItem{{SessionID: 1, SeatID: 1001}, {SessionID: 1, SeatID: 1003}},
		PaymentMethod: Card,
	})
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != PendingPayment || o.HoldExpiresAt == nil || !o.HoldExpiresAt.Equal(now.Add(HoldDuration)) {
		t.Fatalf("want pending payment with 10 min hold, got %s %v", o.Status, o.HoldExpiresAt)
	}
	if o.Total != RUB(4000) {
		t.Fatalf("total = %s, want 4000", o.Total)
	}
	if want := now.Add(48 * time.Hour); !o.BookingDeadline.Equal(want) {
		t.Fatalf("deadline = %v, want %v", o.BookingDeadline, want)
	}
}

func TestTicketLimitAppliesPerEventForEveryCustomer(t *testing.T) {
	four := []OrderItem{{SessionID: 1, SeatID: 1001}, {SessionID: 1, SeatID: 1002},
		{SessionID: 1, SeatID: 1003}, {SessionID: 1, SeatID: 1004}}
	for _, paid := range []int{0, 10} {
		if _, err := build(t, paid, OrderRequest{Items: four, PaymentMethod: Card}); !errors.Is(err, ErrTicketLimit) {
			t.Fatalf("paid=%d: want ErrTicketLimit, got %v", paid, err)
		}
	}
	// Three tickets are fine.
	if _, err := build(t, 0, OrderRequest{Items: four[:3], PaymentMethod: Card}); err != nil {
		t.Fatal(err)
	}
}

func TestSeasonTicketRequiresLoyalty(t *testing.T) {
	items := []OrderItem{{SessionID: 1, SeatID: 1001}, {SessionID: 2, SeatID: 1001}}
	if _, err := build(t, 3, OrderRequest{Items: items, PaymentMethod: Card}); !errors.Is(err, ErrNotLoyal) {
		t.Fatalf("3 paid orders: want ErrNotLoyal, got %v", err)
	}
	o, err := build(t, 4, OrderRequest{Items: items, PaymentMethod: Card})
	if err != nil {
		t.Fatal(err)
	}
	if !o.IsSeasonTicket() {
		t.Fatal("want season ticket")
	}
}

func TestBookingRules(t *testing.T) {
	o, err := build(t, 0, OrderRequest{Items: []OrderItem{{SessionID: 1, SeatID: 1001}}, PaymentMethod: Cash})
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != Booked || o.HoldExpiresAt != nil {
		t.Fatalf("cash must create a booking, got %s", o.Status)
	}
	// Session 4 starts in 10 h: booking is closed, buying by card is still allowed.
	late := []OrderItem{{SessionID: 4, SeatID: 1001}}
	if _, err := build(t, 0, OrderRequest{Items: late, PaymentMethod: Card, Book: true}); !errors.Is(err, ErrBookingClosed) {
		t.Fatalf("want ErrBookingClosed, got %v", err)
	}
	if _, err := build(t, 0, OrderRequest{Items: late, PaymentMethod: Card}); err != nil {
		t.Fatal(err)
	}
}

func TestSeatValidation(t *testing.T) {
	cases := map[string]struct {
		items []OrderItem
		want  error
	}{
		"missing seat":   {[]OrderItem{{SessionID: 1}}, ErrSeatRequired},
		"foreign seat":   {[]OrderItem{{SessionID: 1, SeatID: 9001}}, ErrSeatNotInHall},
		"unknown seat":   {[]OrderItem{{SessionID: 1, SeatID: 5}}, ErrSeatNotInHall},
		"duplicate seat": {[]OrderItem{{SessionID: 1, SeatID: 1001}, {SessionID: 1, SeatID: 1001}}, ErrDuplicateSeat},
		"seat at GA":     {[]OrderItem{{SessionID: 3, SeatID: 1001}}, ErrValidation},
		"add-on concert": {[]OrderItem{{SessionID: 1, SeatID: 1001, AddOnIDs: []int64{7}}}, ErrAddOnNotAllowed},
		"bad add-on":     {[]OrderItem{{SessionID: 3, AddOnIDs: []int64{8}}}, ErrUnknownAddOn},
		"no items":       {nil, ErrEmptyOrder},
		"bad session":    {[]OrderItem{{SessionID: 99}}, ErrNotFound},
	}
	for name, c := range cases {
		if _, err := build(t, 0, OrderRequest{Items: c.items, PaymentMethod: Card}); !errors.Is(err, c.want) {
			t.Errorf("%s: want %v, got %v", name, c.want, err)
		}
	}
}

func TestExhibitionWithAddOnAndCourier(t *testing.T) {
	o, err := build(t, 0, OrderRequest{
		Items:           []OrderItem{{SessionID: 3, AddOnIDs: []int64{7}}, {SessionID: 3}},
		PaymentMethod:   Card,
		DeliveryOption:  Courier,
		DeliveryAddress: "Innopolis, Universitetskaya 1",
	})
	if err != nil {
		t.Fatal(err)
	}
	// 500 + 1200 + 500 + courier 300
	if o.Total != RUB(2500) {
		t.Fatalf("total = %s, want 2500", o.Total)
	}
	if _, err := build(t, 0, OrderRequest{Items: []OrderItem{{SessionID: 3}}, PaymentMethod: Cash,
		DeliveryOption: Courier, DeliveryAddress: "x"}); !errors.Is(err, ErrCourierNeedsCard) {
		t.Fatalf("want ErrCourierNeedsCard, got %v", err)
	}
	if _, err := build(t, 0, OrderRequest{Items: []OrderItem{{SessionID: 3}}, PaymentMethod: Card,
		DeliveryOption: Courier}); !errors.Is(err, ErrAddressRequired) {
		t.Fatalf("want ErrAddressRequired, got %v", err)
	}
}

func TestStartedSessionCannotBeSold(t *testing.T) {
	sessions, seats := fixtures()
	_, err := BuildOrder(1, 0, OrderRequest{Items: []OrderItem{{SessionID: 1, SeatID: 1001}}, PaymentMethod: Card},
		sessions, seats, now.Add(73*time.Hour))
	if !errors.Is(err, ErrSessionStarted) {
		t.Fatalf("want ErrSessionStarted, got %v", err)
	}
}

func TestOrderStateMachine(t *testing.T) {
	deadline := now.Add(24 * time.Hour)
	hold := now.Add(HoldDuration)
	newOrder := func(s OrderStatus) *Order {
		return &Order{Status: s, BookingDeadline: deadline, HoldExpiresAt: &hold,
			Tickets: []Ticket{{Active: true}}}
	}

	// Card checkout paid.
	o := newOrder(PendingPayment)
	if err := o.MarkPaid(Card, now); err != nil || o.Status != Paid || o.HoldExpiresAt != nil {
		t.Fatalf("pay pending: %v %s", err, o.Status)
	}
	// A paid order cannot be paid again (idempotent provider callbacks).
	if err := o.MarkPaid(Card, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("double pay: %v", err)
	}
	// A confirmation after the hold ended is rejected (FR-45: refunded by the caller).
	if err := newOrder(PendingPayment).MarkPaid(Card, hold); !errors.Is(err, ErrHoldExpired) {
		t.Fatalf("late card: %v", err)
	}
	// Cash cannot pay a card checkout hold.
	if err := newOrder(PendingPayment).MarkPaid(Cash, now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("cash on hold: %v", err)
	}
	// Booking paid in cash before the deadline, but not after.
	if err := newOrder(Booked).MarkPaid(Cash, now); err != nil {
		t.Fatal(err)
	}
	if err := newOrder(Booked).MarkPaid(Cash, deadline); !errors.Is(err, ErrBookingClosed) {
		t.Fatalf("late cash: %v", err)
	}

	// A booking with courier delivery cannot be paid in cash (FR-47).
	o = newOrder(Booked)
	o.Delivery.Option = Courier
	if err := o.MarkPaid(Cash, now); !errors.Is(err, ErrCourierNeedsCard) {
		t.Fatalf("cash with courier: %v", err)
	}
	if err := o.MarkPaid(Card, now); err != nil {
		t.Fatal(err)
	}

	// Expiry.
	if err := newOrder(PendingPayment).Expire(now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("hold not yet expired: %v", err)
	}
	o = newOrder(PendingPayment)
	if err := o.Expire(hold); err != nil || o.Status != Expired || o.Tickets[0].Active {
		t.Fatalf("expire hold: %v %s", err, o.Status)
	}
	if err := newOrder(Booked).Expire(now); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("booking not yet expired: %v", err)
	}
	if err := newOrder(Booked).Expire(deadline); err != nil {
		t.Fatal(err)
	}
	if err := newOrder(Paid).Expire(deadline.Add(time.Hour)); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("paid must not expire: %v", err)
	}

	// Cancel.
	o = newOrder(Booked)
	if err := o.Cancel(); err != nil || o.Status != Cancelled || o.Tickets[0].Active {
		t.Fatalf("cancel: %v", err)
	}
	if err := newOrder(Paid).Cancel(); !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("cancel paid: %v", err)
	}

	// Refund.
	o = newOrder(Paid)
	o.PaymentMethod = Card
	if err := o.Refund(now); err != nil || o.Status != Refunded || o.Tickets[0].Active {
		t.Fatalf("refund: %v", err)
	}
	o = newOrder(Paid)
	o.PaymentMethod = Card
	if err := o.Refund(deadline); !errors.Is(err, ErrRefundClosed) {
		t.Fatalf("late refund: %v", err)
	}
	o = newOrder(Paid)
	o.PaymentMethod = Cash
	if err := o.Refund(now); !errors.Is(err, ErrRefundCardOnly) {
		t.Fatalf("cash refund: %v", err)
	}
}

func TestLoyalty(t *testing.T) {
	if IsLoyal(3) || !IsLoyal(4) {
		t.Fatal("loyal means more than 3 paid orders")
	}
}
