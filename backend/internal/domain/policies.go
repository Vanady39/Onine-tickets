package domain

import "time"

// MaxTicketsPerEvent limits tickets for one event in one order, for every customer (BR-2).
const MaxTicketsPerEvent = 3

// LoyaltyThreshold: a customer with more than this many paid orders is loyal (BR-3).
const LoyaltyThreshold = 3

func IsLoyal(paidOrders int) bool { return paidOrders > LoyaltyThreshold }

// OrderItem is one requested ticket.
type OrderItem struct {
	SessionID int64   `json:"sessionId"`
	SeatID    int64   `json:"seatId,omitempty"` // required for seated sessions, 0 otherwise
	AddOnIDs  []int64 `json:"addOnIds,omitempty"`
}

// OrderRequest is what the customer confirms at checkout.
type OrderRequest struct {
	Items         []OrderItem   `json:"items"`
	PaymentMethod PaymentMethod `json:"paymentMethod"`
	// Book = reserve now with a booking code and pay later (online or cash).
	// Card without Book = pay now; seats are held for HoldDuration.
	// Cash always means a booking paid at the box office.
	Book            bool           `json:"book"`
	DeliveryOption  DeliveryOption `json:"deliveryOption"`
	DeliveryAddress string         `json:"deliveryAddress,omitempty"`
}

// BuildOrder validates a request against business rules and prices it. Seat and capacity
// availability is checked later by the store inside the order transaction.
// sessions must contain every session referenced by the request; seatZones maps each
// requested seat ID to its hall and price zone (missing entry = unknown seat).
func BuildOrder(customerID int64, paidOrders int, req OrderRequest, sessions map[int64]Session,
	seatZones map[int64]SeatZone, now time.Time) (Order, error) {

	if len(req.Items) == 0 {
		return Order{}, ErrEmptyOrder
	}
	if req.PaymentMethod != Card && req.PaymentMethod != Cash {
		return Order{}, ErrValidation
	}
	if req.DeliveryOption == "" {
		req.DeliveryOption = ETicket
	}
	if req.DeliveryOption != ETicket && req.DeliveryOption != Courier {
		return Order{}, ErrValidation
	}
	booking := req.Book || req.PaymentMethod == Cash

	delivery := Delivery{Option: req.DeliveryOption}
	if req.DeliveryOption == Courier {
		if req.PaymentMethod != Card {
			return Order{}, ErrCourierNeedsCard
		}
		if req.DeliveryAddress == "" {
			return Order{}, ErrAddressRequired
		}
		delivery.Address = req.DeliveryAddress
		delivery.Fee = CourierFee
	}

	perEvent := map[int64]int{}
	seen := map[[2]int64]bool{}
	var tickets []Ticket
	var earliest time.Time
	total := delivery.Fee

	for _, it := range req.Items {
		s, ok := sessions[it.SessionID]
		if !ok {
			return Order{}, ErrNotFound
		}
		if s.Started(now) {
			return Order{}, ErrSessionStarted
		}
		if booking && !s.CanBook(now) {
			return Order{}, ErrBookingClosed
		}
		perEvent[s.EventID]++
		if perEvent[s.EventID] > MaxTicketsPerEvent {
			return Order{}, ErrTicketLimit
		}

		t := Ticket{SessionID: s.ID, EventID: s.EventID, Active: true}
		if s.Hall.IsSeated() {
			if it.SeatID == 0 {
				return Order{}, ErrSeatRequired
			}
			sz, ok := seatZones[it.SeatID]
			if !ok || sz.HallID != s.Hall.ID {
				return Order{}, ErrSeatNotInHall
			}
			key := [2]int64{s.ID, it.SeatID}
			if seen[key] {
				return Order{}, ErrDuplicateSeat
			}
			seen[key] = true
			price, ok := s.PriceForZone(sz.ZoneID)
			if !ok {
				return Order{}, ErrSeatNotInHall
			}
			t.SeatID, t.Price = it.SeatID, price
		} else {
			if it.SeatID != 0 {
				return Order{}, ErrValidation
			}
			t.Price = s.BasePrice
		}

		if len(it.AddOnIDs) > 0 && s.EventType != Exhibition {
			return Order{}, ErrAddOnNotAllowed
		}
		for _, id := range it.AddOnIDs {
			a, ok := s.AddOn(id)
			if !ok {
				return Order{}, ErrUnknownAddOn
			}
			t.AddOns = append(t.AddOns, TicketAddOn{AddOnID: a.ID, Name: a.Name, Price: a.Price})
		}

		total += t.Total()
		tickets = append(tickets, t)
		if earliest.IsZero() || s.StartsAt.Before(earliest) {
			earliest = s.StartsAt
		}
	}

	o := Order{
		CustomerID:      customerID,
		PaymentMethod:   req.PaymentMethod,
		BookingDeadline: earliest.Add(-BookingWindow),
		Total:           total,
		Delivery:        delivery,
		Tickets:         tickets,
		CreatedAt:       now,
	}
	if o.IsSeasonTicket() && !IsLoyal(paidOrders) {
		return Order{}, ErrNotLoyal
	}
	if booking {
		o.Status = Booked
	} else {
		o.Status = PendingPayment
		hold := now.Add(HoldDuration)
		o.HoldExpiresAt = &hold
	}
	return o, nil
}

// SeatZone locates a seat: its hall and price zone.
type SeatZone struct {
	HallID int64
	ZoneID int64
}
