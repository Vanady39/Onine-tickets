package store

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

// CreateOrder inserts an order and its tickets in one transaction. It locks every session
// of the order, re-checks general-admission and add-on capacity, and relies on the
// tickets_one_active_seat unique index so that a seat can never be taken twice (NFR-04).
func (s *Store) CreateOrder(ctx context.Context, o *domain.Order) error {
	return s.inTx(ctx, func(tx pgx.Tx) error {
		sessionIDs := map[int64]bool{}
		gaCount := map[int64]int{}
		addOnCount := map[int64]int{}
		for _, t := range o.Tickets {
			sessionIDs[t.SessionID] = true
			if t.SeatID == 0 {
				gaCount[t.SessionID]++
			}
			for _, a := range t.AddOns {
				addOnCount[a.AddOnID]++
			}
		}
		ids := make([]int64, 0, len(sessionIDs))
		for id := range sessionIDs {
			ids = append(ids, id)
		}
		// Serialises concurrent orders for the same sessions; seats are additionally
		// protected by the unique index.
		if _, err := tx.Exec(ctx, `SELECT id FROM sessions WHERE id = ANY($1) ORDER BY id FOR UPDATE`, ids); err != nil {
			return err
		}
		for sid, n := range gaCount {
			var free int
			err := tx.QueryRow(ctx, `
				SELECT h.capacity - (SELECT count(*) FROM tickets t WHERE t.session_id = se.id AND t.active)
				FROM sessions se JOIN halls h ON h.id = se.hall_id WHERE se.id = $1`, sid).Scan(&free)
			if err != nil {
				return err
			}
			if free < n {
				return domain.ErrSoldOut
			}
		}
		for aid, n := range addOnCount {
			var free int
			err := tx.QueryRow(ctx, `
				SELECT a.capacity - (SELECT count(*) FROM ticket_add_ons ta JOIN tickets t ON t.id = ta.ticket_id
				                     WHERE ta.add_on_id = a.id AND t.active)
				FROM add_ons a WHERE a.id = $1`, aid).Scan(&free)
			if err != nil {
				return err
			}
			if free < n {
				return domain.ErrSoldOut
			}
		}

		err := tx.QueryRow(ctx, `
			INSERT INTO orders (customer_id, status, payment_method, booking_code, booking_deadline,
			                    hold_expires_at, total, delivery_option, delivery_address, delivery_fee, created_at, paid_at)
			VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`,
			o.CustomerID, o.Status, o.PaymentMethod, o.BookingCode, o.BookingDeadline, o.HoldExpiresAt,
			o.Total, o.Delivery.Option, o.Delivery.Address, o.Delivery.Fee, o.CreatedAt, o.PaidAt).Scan(&o.ID)
		if isUniqueViolation(err, "orders_booking_code_key") {
			return domain.ErrDuplicateCode
		}
		if err != nil {
			return err
		}
		for i := range o.Tickets {
			t := &o.Tickets[i]
			var seat *int64
			if t.SeatID != 0 {
				seat = &t.SeatID
			}
			err := tx.QueryRow(ctx, `
				INSERT INTO tickets (order_id, session_id, seat_id, price, code, active)
				VALUES ($1, $2, $3, $4, $5, TRUE) RETURNING id`,
				o.ID, t.SessionID, seat, t.Price, t.Code).Scan(&t.ID)
			switch {
			case isUniqueViolation(err, "tickets_one_active_seat"):
				return domain.ErrSeatTaken
			case isUniqueViolation(err, "tickets_code_key"):
				return domain.ErrDuplicateCode
			case err != nil:
				return err
			}
			for _, a := range t.AddOns {
				if _, err := tx.Exec(ctx, `INSERT INTO ticket_add_ons (ticket_id, add_on_id, price) VALUES ($1, $2, $3)`,
					t.ID, a.AddOnID, a.Price); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

const orderCols = `id, customer_id, status, payment_method, COALESCE(booking_code, ''), booking_deadline,
	hold_expires_at, COALESCE(payment_ref, ''), total, delivery_option, delivery_address, delivery_fee,
	courier_ref, delivery_status, created_at, paid_at`

func scanOrder(row interface{ Scan(...any) error }) (domain.Order, error) {
	var o domain.Order
	d := &o.Delivery
	err := row.Scan(&o.ID, &o.CustomerID, &o.Status, &o.PaymentMethod, &o.BookingCode, &o.BookingDeadline,
		&o.HoldExpiresAt, &o.PaymentRef, &o.Total, &d.Option, &d.Address, &d.Fee, &d.CourierRef, &d.Status,
		&o.CreatedAt, &o.PaidAt)
	return o, err
}

func (s *Store) GetOrder(ctx context.Context, id int64) (domain.Order, error) {
	return s.orderWhere(ctx, s.pool, `id = $1`, id)
}

func (s *Store) OrderByBookingCode(ctx context.Context, code string) (domain.Order, error) {
	return s.orderWhere(ctx, s.pool, `booking_code = $1`, code)
}

func (s *Store) OrderByPaymentRef(ctx context.Context, ref string) (domain.Order, error) {
	return s.orderWhere(ctx, s.pool, `payment_ref = $1`, ref)
}

func (s *Store) orderWhere(ctx context.Context, q db, cond string, arg any) (domain.Order, error) {
	o, err := scanOrder(q.QueryRow(ctx, `SELECT `+orderCols+` FROM orders WHERE `+cond, arg))
	if err != nil {
		return o, notFound(err)
	}
	tickets, err := loadTickets(ctx, q, []int64{o.ID})
	o.Tickets = tickets[o.ID]
	return o, err
}

// CustomerOrders returns the purchase history, newest first.
func (s *Store) CustomerOrders(ctx context.Context, customerID int64) ([]domain.Order, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+orderCols+` FROM orders WHERE customer_id = $1 ORDER BY created_at DESC, id DESC`, customerID)
	if err != nil {
		return nil, err
	}
	orders := []domain.Order{}
	var ids []int64
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		orders = append(orders, o)
		ids = append(ids, o.ID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	tickets, err := loadTickets(ctx, s.pool, ids)
	if err != nil {
		return nil, err
	}
	for i := range orders {
		orders[i].Tickets = tickets[orders[i].ID]
	}
	return orders, nil
}

func loadTickets(ctx context.Context, q db, orderIDs []int64) (map[int64][]domain.Ticket, error) {
	out := map[int64][]domain.Ticket{}
	if len(orderIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT t.order_id, t.id, t.code, t.session_id, se.event_id, COALESCE(t.seat_id, 0), t.price, t.active,
		       e.title, e.type, se.starts_at, v.name, h.name,
		       COALESCE(z.name, ''), COALESCE(st.row_no, 0), COALESCE(st.seat_no, 0)
		FROM tickets t
		JOIN sessions se ON se.id = t.session_id
		JOIN events e ON e.id = se.event_id
		JOIN halls h ON h.id = se.hall_id
		JOIN venues v ON v.id = h.venue_id
		LEFT JOIN seats st ON st.id = t.seat_id
		LEFT JOIN price_zones z ON z.id = st.zone_id
		WHERE t.order_id = ANY($1)
		ORDER BY se.starts_at, st.row_no, st.seat_no, t.id`, orderIDs)
	if err != nil {
		return nil, err
	}
	pos := map[int64]int{}       // ticket ID -> index in out[order]
	orderOf := map[int64]int64{} // ticket ID -> order ID
	for rows.Next() {
		var orderID int64
		var t domain.Ticket
		if err := rows.Scan(&orderID, &t.ID, &t.Code, &t.SessionID, &t.EventID, &t.SeatID, &t.Price, &t.Active,
			&t.EventTitle, &t.EventType, &t.StartsAt, &t.VenueName, &t.HallName, &t.ZoneName, &t.Row, &t.Number); err != nil {
			rows.Close()
			return nil, err
		}
		t.AddOns = []domain.TicketAddOn{}
		out[orderID] = append(out[orderID], t)
		pos[t.ID] = len(out[orderID]) - 1
		orderOf[t.ID] = orderID
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = q.Query(ctx, `
		SELECT ta.ticket_id, ta.add_on_id, a.name, ta.price
		FROM ticket_add_ons ta JOIN add_ons a ON a.id = ta.add_on_id
		JOIN tickets t ON t.id = ta.ticket_id
		WHERE t.order_id = ANY($1) ORDER BY a.id`, orderIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ticketID int64
		var a domain.TicketAddOn
		if err := rows.Scan(&ticketID, &a.AddOnID, &a.Name, &a.Price); err != nil {
			return nil, err
		}
		oid, i := orderOf[ticketID], pos[ticketID]
		out[oid][i].AddOns = append(out[oid][i].AddOns, a)
	}
	return out, rows.Err()
}

// UpdateOrder locks the order row, lets fn apply a domain transition, and saves the
// resulting state (status, payment, delivery and ticket activity) in the same transaction.
func (s *Store) UpdateOrder(ctx context.Context, id int64, fn func(*domain.Order) error) (domain.Order, error) {
	var o domain.Order
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		var err error
		o, err = scanOrder(tx.QueryRow(ctx, `SELECT `+orderCols+` FROM orders WHERE id = $1 FOR UPDATE`, id))
		if err != nil {
			return notFound(err)
		}
		tickets, err := loadTickets(ctx, tx, []int64{o.ID})
		if err != nil {
			return err
		}
		o.Tickets = tickets[o.ID]
		if err := fn(&o); err != nil {
			return err
		}
		return saveOrder(ctx, tx, &o)
	})
	return o, err
}

func saveOrder(ctx context.Context, tx pgx.Tx, o *domain.Order) error {
	_, err := tx.Exec(ctx, `
		UPDATE orders SET status = $2, payment_method = $3, hold_expires_at = $4, payment_ref = NULLIF($5, ''),
		       paid_at = $6, courier_ref = $7, delivery_status = $8
		WHERE id = $1`,
		o.ID, o.Status, o.PaymentMethod, o.HoldExpiresAt, o.PaymentRef, o.PaidAt, o.Delivery.CourierRef, o.Delivery.Status)
	if err != nil {
		return err
	}
	if !o.Status.Active() {
		_, err = tx.Exec(ctx, `UPDATE tickets SET active = FALSE WHERE order_id = $1`, o.ID)
	}
	return err
}

// ExpireDue expires card holds older than their hold time and bookings past their deadline,
// releasing their seats (FR-19, BR-7). It returns the number of expired orders.
func (s *Store) ExpireDue(ctx context.Context, now time.Time) (int, error) {
	var n int
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			UPDATE orders SET status = 'expired'
			WHERE (status = 'pending_payment' AND hold_expires_at <= $1)
			   OR (status = 'booked' AND booking_deadline <= $1)
			RETURNING id`, now)
		if err != nil {
			return err
		}
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		n = len(ids)
		if n == 0 {
			return nil
		}
		_, err = tx.Exec(ctx, `UPDATE tickets SET active = FALSE WHERE order_id = ANY($1)`, ids)
		return err
	})
	return n, err
}
