package store

import (
	"context"
	"time"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

// ListEvents returns events that have at least one session in [From, To) (From defaults to
// now), with the earliest such session and the lowest price.
func (s *Store) ListEvents(ctx context.Context, f domain.EventFilter, now time.Time) ([]domain.Event, error) {
	from := now
	if f.From != nil && f.From.After(now) {
		from = *f.From
	}
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, e.title, e.type, e.description, e.duration_min, e.age_rating, e.image_url,
		       min(se.starts_at),
		       min(LEAST(COALESCE((SELECT min(zp.price) FROM zone_prices zp WHERE zp.session_id = se.id), se.base_price),
		                 CASE WHEN h.kind = 'general_admission' THEN se.base_price END))
		FROM events e
		JOIN sessions se ON se.event_id = e.id
		JOIN halls h ON h.id = se.hall_id
		WHERE se.starts_at >= $1
		  AND ($2::timestamptz IS NULL OR se.starts_at < $2)
		  AND ($3 = '' OR e.type = $3)
		GROUP BY e.id
		ORDER BY min(se.starts_at), e.id`,
		from, f.To, string(f.Type))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []domain.Event{}
	for rows.Next() {
		var e domain.Event
		var next time.Time
		var minPrice domain.Money
		if err := rows.Scan(&e.ID, &e.Title, &e.Type, &e.Description, &e.DurationMin, &e.AgeRating,
			&e.ImageURL, &next, &minPrice); err != nil {
			return nil, err
		}
		e.NextSessionAt, e.MinPrice = &next, &minPrice
		events = append(events, e)
	}
	return events, rows.Err()
}

// GetEvent returns an event with its upcoming sessions (prices included, add-ons excluded).
func (s *Store) GetEvent(ctx context.Context, id int64, now time.Time) (domain.Event, error) {
	var e domain.Event
	err := s.pool.QueryRow(ctx,
		`SELECT id, title, type, description, duration_min, age_rating, image_url FROM events WHERE id = $1`, id).
		Scan(&e.ID, &e.Title, &e.Type, &e.Description, &e.DurationMin, &e.AgeRating, &e.ImageURL)
	if err != nil {
		return e, notFound(err)
	}
	rows, err := s.pool.Query(ctx, `SELECT id FROM sessions WHERE event_id = $1 AND starts_at >= $2 ORDER BY starts_at`, id, now)
	if err != nil {
		return e, err
	}
	var ids []int64
	for rows.Next() {
		var sid int64
		if err := rows.Scan(&sid); err != nil {
			rows.Close()
			return e, err
		}
		ids = append(ids, sid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return e, err
	}
	sessions, err := s.Sessions(ctx, ids)
	if err != nil {
		return e, err
	}
	e.Sessions = make([]domain.Session, 0, len(ids))
	for _, sid := range ids {
		e.Sessions = append(e.Sessions, sessions[sid])
	}
	return e, nil
}

func (s *Store) Session(ctx context.Context, id int64) (domain.Session, error) {
	m, err := s.Sessions(ctx, []int64{id})
	if err != nil {
		return domain.Session{}, err
	}
	se, ok := m[id]
	if !ok {
		return se, domain.ErrNotFound
	}
	return se, nil
}

// Sessions loads sessions with hall, zone prices, add-ons and remaining capacity.
func (s *Store) Sessions(ctx context.Context, ids []int64) (map[int64]domain.Session, error) {
	return loadSessions(ctx, s.pool, ids)
}

func loadSessions(ctx context.Context, q db, ids []int64) (map[int64]domain.Session, error) {
	out := map[int64]domain.Session{}
	if len(ids) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT se.id, se.event_id, e.title, e.type, se.starts_at, se.base_price,
		       h.id, h.venue_id, v.name, v.address, h.name, h.kind, h.capacity,
		       h.capacity - (SELECT count(*) FROM tickets t WHERE t.session_id = se.id AND t.active)
		FROM sessions se
		JOIN events e ON e.id = se.event_id
		JOIN halls h ON h.id = se.hall_id
		JOIN venues v ON v.id = h.venue_id
		WHERE se.id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var se domain.Session
		h := &se.Hall
		if err := rows.Scan(&se.ID, &se.EventID, &se.EventTitle, &se.EventType, &se.StartsAt, &se.BasePrice,
			&h.ID, &h.VenueID, &h.VenueName, &h.Address, &h.Name, &h.Kind, &h.Capacity, &se.Remaining); err != nil {
			rows.Close()
			return nil, err
		}
		se.Prices, se.AddOns = []domain.ZonePrice{}, []domain.AddOnService{}
		out[se.ID] = se
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = q.Query(ctx, `
		SELECT zp.session_id, zp.zone_id, z.name, zp.price
		FROM zone_prices zp JOIN price_zones z ON z.id = zp.zone_id
		WHERE zp.session_id = ANY($1) ORDER BY zp.price DESC`, ids)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var sid int64
		var p domain.ZonePrice
		if err := rows.Scan(&sid, &p.ZoneID, &p.ZoneName, &p.Price); err != nil {
			rows.Close()
			return nil, err
		}
		se := out[sid]
		se.Prices = append(se.Prices, p)
		out[sid] = se
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rows, err = q.Query(ctx, `
		SELECT a.session_id, a.id, a.kind, a.name, a.price, a.capacity,
		       a.capacity - (SELECT count(*) FROM ticket_add_ons ta JOIN tickets t ON t.id = ta.ticket_id
		                     WHERE ta.add_on_id = a.id AND t.active)
		FROM add_ons a WHERE a.session_id = ANY($1) ORDER BY a.id`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sid int64
		var a domain.AddOnService
		if err := rows.Scan(&sid, &a.ID, &a.Kind, &a.Name, &a.Price, &a.Capacity, &a.Remaining); err != nil {
			return nil, err
		}
		se := out[sid]
		se.AddOns = append(se.AddOns, a)
		out[sid] = se
	}
	return out, rows.Err()
}

// SeatZones returns hall and zone for each of the given seats that exists.
func (s *Store) SeatZones(ctx context.Context, seatIDs []int64) (map[int64]domain.SeatZone, error) {
	out := map[int64]domain.SeatZone{}
	if len(seatIDs) == 0 {
		return out, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT id, hall_id, zone_id FROM seats WHERE id = ANY($1)`, seatIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var z domain.SeatZone
		if err := rows.Scan(&id, &z.HallID, &z.ZoneID); err != nil {
			return nil, err
		}
		out[id] = z
	}
	return out, rows.Err()
}

// SeatMap returns every seat of the session hall with price and current status.
func (s *Store) SeatMap(ctx context.Context, sessionID int64) ([]domain.SeatState, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT st.id, st.zone_id, st.row_no, st.seat_no, z.name, COALESCE(zp.price, 0),
		       CASE WHEN o.status = 'paid' THEN 'sold'
		            WHEN o.status IS NOT NULL THEN 'held'
		            ELSE 'free' END
		FROM sessions se
		JOIN seats st ON st.hall_id = se.hall_id
		JOIN price_zones z ON z.id = st.zone_id
		LEFT JOIN zone_prices zp ON zp.session_id = se.id AND zp.zone_id = st.zone_id
		LEFT JOIN tickets t ON t.session_id = se.id AND t.seat_id = st.id AND t.active
		LEFT JOIN orders o ON o.id = t.order_id
		WHERE se.id = $1
		ORDER BY st.row_no, st.seat_no`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	seats := []domain.SeatState{}
	for rows.Next() {
		var st domain.SeatState
		if err := rows.Scan(&st.ID, &st.ZoneID, &st.Row, &st.Number, &st.ZoneName, &st.Price, &st.Status); err != nil {
			return nil, err
		}
		seats = append(seats, st)
	}
	return seats, rows.Err()
}
