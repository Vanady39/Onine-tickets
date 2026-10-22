package store

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

func (s *Store) CreateEvent(ctx context.Context, e domain.Event) (domain.Event, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO events (title, type, description, duration_min, age_rating, image_url)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		e.Title, e.Type, e.Description, e.DurationMin, e.AgeRating, e.ImageURL).Scan(&e.ID)
	return e, err
}

func (s *Store) UpdateEvent(ctx context.Context, e domain.Event) (domain.Event, error) {
	tag, err := s.pool.Exec(ctx, `
		UPDATE events SET title = $2, type = $3, description = $4, duration_min = $5, age_rating = $6, image_url = $7
		WHERE id = $1`,
		e.ID, e.Title, e.Type, e.Description, e.DurationMin, e.AgeRating, e.ImageURL)
	if err == nil && tag.RowsAffected() == 0 {
		err = domain.ErrNotFound
	}
	return e, err
}

// AllEvents lists every event for the back office, including events without sessions.
func (s *Store) AllEvents(ctx context.Context) ([]domain.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, title, type, description, duration_min, age_rating, image_url FROM events ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []domain.Event{}
	for rows.Next() {
		var e domain.Event
		if err := rows.Scan(&e.ID, &e.Title, &e.Type, &e.Description, &e.DurationMin, &e.AgeRating, &e.ImageURL); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *Store) CreateSession(ctx context.Context, ns domain.NewSession) (int64, error) {
	var id int64
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx, `INSERT INTO sessions (event_id, hall_id, starts_at, base_price) VALUES ($1, $2, $3, $4) RETURNING id`,
			ns.EventID, ns.HallID, ns.StartsAt, ns.BasePrice).Scan(&id)
		if err != nil {
			return err
		}
		for _, p := range ns.Prices {
			if _, err := tx.Exec(ctx, `INSERT INTO zone_prices (session_id, zone_id, price) VALUES ($1, $2, $3)`,
				id, p.ZoneID, p.Price); err != nil {
				return err
			}
		}
		for _, a := range ns.AddOns {
			if _, err := tx.Exec(ctx, `INSERT INTO add_ons (session_id, kind, name, price, capacity) VALUES ($1, $2, $3, $4, $5)`,
				id, a.Kind, a.Name, a.Price, a.Capacity); err != nil {
				return err
			}
		}
		return nil
	})
	return id, err
}

func (s *Store) CreateVenue(ctx context.Context, name, address string, halls []domain.HallSpec) (int64, error) {
	var venueID int64
	err := s.inTx(ctx, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx, `INSERT INTO venues (name, address) VALUES ($1, $2) RETURNING id`, name, address).Scan(&venueID); err != nil {
			return err
		}
		for _, h := range halls {
			capacity := h.Capacity
			if h.Kind == domain.Seated {
				capacity = 0
				for _, z := range h.Zones {
					capacity += z.Rows * z.SeatsPerRow
				}
			}
			var hallID int64
			if err := tx.QueryRow(ctx, `INSERT INTO halls (venue_id, name, kind, capacity) VALUES ($1, $2, $3, $4) RETURNING id`,
				venueID, h.Name, h.Kind, capacity).Scan(&hallID); err != nil {
				return err
			}
			if h.Kind != domain.Seated {
				continue
			}
			row := 1
			for _, z := range h.Zones {
				var zoneID int64
				if err := tx.QueryRow(ctx, `INSERT INTO price_zones (hall_id, name) VALUES ($1, $2) RETURNING id`, hallID, z.Name).Scan(&zoneID); err != nil {
					return err
				}
				for r := 0; r < z.Rows; r++ {
					if _, err := tx.Exec(ctx, `
						INSERT INTO seats (hall_id, zone_id, row_no, seat_no)
						SELECT $1, $2, $3, n FROM generate_series(1, $4::int) n`,
						hallID, zoneID, row, z.SeatsPerRow); err != nil {
						return err
					}
					row++
				}
			}
		}
		return nil
	})
	return venueID, err
}

// Venues lists venues with their halls and price zones.
func (s *Store) Venues(ctx context.Context) ([]domain.Venue, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT v.id, v.name, v.address, h.id, h.name, h.kind, h.capacity
		FROM venues v JOIN halls h ON h.venue_id = v.id ORDER BY v.id, h.id`)
	if err != nil {
		return nil, err
	}
	venues := []domain.Venue{}
	hallIdx := map[int64][2]int{}
	for rows.Next() {
		var v domain.Venue
		var h domain.Hall
		if err := rows.Scan(&v.ID, &v.Name, &v.Address, &h.ID, &h.Name, &h.Kind, &h.Capacity); err != nil {
			rows.Close()
			return nil, err
		}
		if len(venues) == 0 || venues[len(venues)-1].ID != v.ID {
			venues = append(venues, v)
		}
		cur := &venues[len(venues)-1]
		h.VenueID, h.VenueName, h.Address, h.Zones = v.ID, v.Name, v.Address, []domain.PriceZone{}
		cur.Halls = append(cur.Halls, h)
		hallIdx[h.ID] = [2]int{len(venues) - 1, len(cur.Halls) - 1}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows, err = s.pool.Query(ctx, `SELECT hall_id, id, name FROM price_zones ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var hallID int64
		var z domain.PriceZone
		if err := rows.Scan(&hallID, &z.ID, &z.Name); err != nil {
			return nil, err
		}
		if i, ok := hallIdx[hallID]; ok {
			h := &venues[i[0]].Halls[i[1]]
			h.Zones = append(h.Zones, z)
		}
	}
	return venues, rows.Err()
}

// SalesReport aggregates paid tickets (with add-ons) of an event per session (FR-38).
func (s *Store) SalesReport(ctx context.Context, eventID int64) (domain.EventReport, error) {
	r := domain.EventReport{EventID: eventID, Sessions: []domain.SessionReport{}}
	if err := s.pool.QueryRow(ctx, `SELECT title FROM events WHERE id = $1`, eventID).Scan(&r.Title); err != nil {
		return r, notFound(err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT se.id, se.starts_at, h.name, v.name, h.capacity,
		       count(t.id),
		       COALESCE(sum(t.price + COALESCE((SELECT sum(ta.price) FROM ticket_add_ons ta WHERE ta.ticket_id = t.id), 0)), 0)
		FROM sessions se
		JOIN halls h ON h.id = se.hall_id
		JOIN venues v ON v.id = h.venue_id
		LEFT JOIN (tickets t JOIN orders o ON o.id = t.order_id AND o.status = 'paid')
		       ON t.session_id = se.id AND t.active
		WHERE se.event_id = $1
		GROUP BY se.id, h.name, v.name, h.capacity
		ORDER BY se.starts_at`, eventID)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var sr domain.SessionReport
		if err := rows.Scan(&sr.SessionID, &sr.StartsAt, &sr.HallName, &sr.VenueName, &sr.Capacity, &sr.TicketsSold, &sr.Revenue); err != nil {
			return r, err
		}
		if sr.Capacity > 0 {
			sr.Occupancy = float64(sr.TicketsSold) * 100 / float64(sr.Capacity)
		}
		r.TicketsSold += sr.TicketsSold
		r.Revenue += sr.Revenue
		r.Sessions = append(r.Sessions, sr)
	}
	return r, rows.Err()
}
