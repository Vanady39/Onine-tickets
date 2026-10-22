// Package seed fills an empty database with demo venues, events, sessions and accounts.
package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/Vanady/online-tickets/backend/internal/app"
	"github.com/Vanady/online-tickets/backend/internal/domain"
	"github.com/Vanady/online-tickets/backend/internal/store"
)

// Demo accounts. Passwords are demo values for local runs only (see README).
var Accounts = []struct {
	Name, Email string
	Role        domain.Role
}{
	{"Demo Cashier", "cashier@tickets.local", domain.RoleCashier},
	{"Demo Manager", "manager@tickets.local", domain.RoleManager},
	{"Demo Admin", "admin@tickets.local", domain.RoleAdmin},
	{"Anna Loyal", "loyal@tickets.local", domain.RoleCustomer},
	{"Ivan Regular", "customer@tickets.local", domain.RoleCustomer},
}

var msk = time.FixedZone("MSK", 3*60*60)

// Run seeds the database unless it already has users. now anchors session dates.
func Run(ctx context.Context, st *store.Store, accounts *app.AccountService, password string, now time.Time) error {
	n, err := st.CountUsers(ctx)
	if err != nil || n > 0 {
		return err
	}

	users := map[string]domain.User{}
	for _, a := range Accounts {
		var u domain.User
		if a.Role == domain.RoleCustomer {
			u, err = accounts.Register(ctx, a.Name, a.Email, password)
		} else {
			u, err = accounts.CreateStaff(ctx, a.Name, a.Email, password, a.Role)
		}
		if err != nil {
			return fmt.Errorf("seed user %s: %w", a.Email, err)
		}
		users[a.Email] = u
	}

	venues := []struct {
		name, address string
		halls         []domain.HallSpec
	}{
		{"Grand Concert Hall", "Kazan, Pushkina St 1", []domain.HallSpec{{Name: "Main Hall", Kind: domain.Seated, Zones: []domain.ZoneSpec{
			{Name: "VIP", Rows: 2, SeatsPerRow: 12}, {Name: "Parterre", Rows: 6, SeatsPerRow: 16}, {Name: "Balcony", Rows: 4, SeatsPerRow: 18}}}}},
		{"Cinema Star", "Kazan, Baumana St 50", []domain.HallSpec{{Name: "Hall 1", Kind: domain.Seated, Zones: []domain.ZoneSpec{
			{Name: "Standard", Rows: 8, SeatsPerRow: 14}, {Name: "Recliner VIP", Rows: 2, SeatsPerRow: 8}}}}},
		{"Kachalov Drama Theatre", "Kazan, Svobody Sq 1", []domain.HallSpec{{Name: "Main Stage", Kind: domain.Seated, Zones: []domain.ZoneSpec{
			{Name: "Stalls", Rows: 8, SeatsPerRow: 18}, {Name: "Amphitheatre", Rows: 4, SeatsPerRow: 20}}}}},
		{"Kazan Expo Centre", "Kazan, Orenburgsky Trakt 8", []domain.HallSpec{
			{Name: "Pavilion A", Kind: domain.GeneralAdmission, Capacity: 150}}},
	}
	for _, v := range venues {
		if _, err := st.CreateVenue(ctx, v.name, v.address, v.halls); err != nil {
			return fmt.Errorf("seed venue %s: %w", v.name, err)
		}
	}
	allVenues, err := st.Venues(ctx)
	if err != nil {
		return err
	}
	halls := map[string]domain.Hall{}
	for _, v := range allVenues {
		for _, h := range v.Halls {
			halls[v.Name] = h
		}
	}
	prices := func(venue string, rub ...int64) []domain.ZonePrice {
		var out []domain.ZonePrice
		for i, z := range halls[venue].Zones {
			out = append(out, domain.ZonePrice{ZoneID: z.ID, Price: domain.RUB(rub[i])})
		}
		return out
	}
	// at returns 19:00 MSK (or the given hour) in `days` days.
	at := func(days, hour int) time.Time {
		d := now.In(msk).AddDate(0, 0, days)
		return time.Date(d.Year(), d.Month(), d.Day(), hour, 0, 0, 0, msk)
	}

	type sess struct {
		venue  string
		starts time.Time
		zones  []int64 // roubles per zone (seated)
		base   int64   // roubles (general admission)
		addOns []domain.AddOnService
	}
	expoAddOns := []domain.AddOnService{
		{Kind: domain.BusinessLunch, Name: "Business lunch at the Expo café", Price: domain.RUB(1200), Capacity: 40},
		{Kind: domain.Masterclass, Name: "Masterclass: painting with light", Price: domain.RUB(2500), Capacity: 15},
		{Kind: domain.Presentation, Name: "Curator's presentation", Price: domain.RUB(600), Capacity: 60},
	}
	events := []struct {
		e        domain.Event
		sessions []sess
	}{
		{domain.Event{Title: "Rachmaninoff Night", Type: domain.Concert, DurationMin: 120, AgeRating: "6+",
			Description: "The State Symphony Orchestra of Tatarstan performs Rachmaninoff's Piano Concerto No. 2 and Symphonic Dances."},
			[]sess{{venue: "Grand Concert Hall", starts: at(3, 19), zones: []int64{6000, 3500, 1800}},
				{venue: "Grand Concert Hall", starts: at(10, 19), zones: []int64{6000, 3500, 1800}}}},
		{domain.Event{Title: "Indie Rock Evening", Type: domain.Concert, DurationMin: 150, AgeRating: "16+",
			Description: "Three local indie bands on one stage: guitars, synths and a lot of energy."},
			[]sess{{venue: "Grand Concert Hall", starts: at(6, 20), zones: []int64{4500, 2500, 1500}}}},
		{domain.Event{Title: "Interstellar — Anniversary Screening", Type: domain.Cinema, DurationMin: 169, AgeRating: "12+",
			Description: "Christopher Nolan's space epic on the big screen with remastered sound."},
			[]sess{{venue: "Cinema Star", starts: now.Add(12 * time.Hour).Truncate(time.Hour), zones: []int64{450, 900}},
				{venue: "Cinema Star", starts: at(2, 18), zones: []int64{450, 900}},
				{venue: "Cinema Star", starts: at(4, 21), zones: []int64{500, 1000}}}},
		{domain.Event{Title: "The Grand Budapest Hotel", Type: domain.Cinema, DurationMin: 99, AgeRating: "18+",
			Description: "Wes Anderson's comedy about a legendary concierge and his lobby boy."},
			[]sess{{venue: "Cinema Star", starts: at(5, 19), zones: []int64{400, 850}}}},
		{domain.Event{Title: "The Cherry Orchard", Type: domain.Theatre, DurationMin: 180, AgeRating: "12+",
			Description: "Chekhov's last play in a new production of the Kachalov Drama Theatre."},
			[]sess{{venue: "Kachalov Drama Theatre", starts: at(7, 18), zones: []int64{3000, 1500}},
				{venue: "Kachalov Drama Theatre", starts: at(14, 18), zones: []int64{3000, 1500}}}},
		{domain.Event{Title: "Hamlet", Type: domain.Theatre, DurationMin: 200, AgeRating: "16+",
			Description: "Shakespeare's tragedy with an original score performed live."},
			[]sess{{venue: "Kachalov Drama Theatre", starts: at(9, 19), zones: []int64{3500, 1800}}}},
		{domain.Event{Title: "Modern Art of the Volga Region", Type: domain.Exhibition, DurationMin: 0, AgeRating: "0+",
			Description: "Paintings, installations and media art by 40 artists from the Volga region."},
			[]sess{{venue: "Kazan Expo Centre", starts: at(2, 11), base: 700, addOns: expoAddOns},
				{venue: "Kazan Expo Centre", starts: at(8, 11), base: 700, addOns: expoAddOns}}},
		{domain.Event{Title: "Robots & AI Expo", Type: domain.Exhibition, DurationMin: 0, AgeRating: "6+",
			Description: "Hands-on robotics, AI demos and talks by Innopolis engineers."},
			[]sess{{venue: "Kazan Expo Centre", starts: at(12, 10), base: 900, addOns: []domain.AddOnService{
				{Kind: domain.Presentation, Name: "Talk: robots in everyday life", Price: domain.RUB(500), Capacity: 80},
				{Kind: domain.Masterclass, Name: "Build your first robot", Price: domain.RUB(3000), Capacity: 12}}}}},
	}
	var sessionIDs []int64
	for _, item := range events {
		e, err := st.CreateEvent(ctx, item.e)
		if err != nil {
			return fmt.Errorf("seed event %s: %w", item.e.Title, err)
		}
		for _, s := range item.sessions {
			ns := domain.NewSession{EventID: e.ID, HallID: halls[s.venue].ID, StartsAt: s.starts,
				BasePrice: domain.RUB(s.base), AddOns: s.addOns}
			if s.zones != nil {
				ns.Prices = prices(s.venue, s.zones...)
			}
			id, err := st.CreateSession(ctx, ns)
			if err != nil {
				return fmt.Errorf("seed session of %s: %w", e.Title, err)
			}
			sessionIDs = append(sessionIDs, id)
		}
	}

	// Make the loyal demo customer loyal: four paid single-ticket orders (BR-3).
	return seedPaidOrders(ctx, st, users["loyal@tickets.local"].ID, sessionIDs, now)
}

func seedPaidOrders(ctx context.Context, st *store.Store, customerID int64, sessionIDs []int64, now time.Time) error {
	sessions, err := st.Sessions(ctx, sessionIDs)
	if err != nil {
		return err
	}
	made := 0
	for _, id := range sessionIDs {
		if made == 4 {
			break
		}
		s := sessions[id]
		if !s.Hall.IsSeated() || !s.CanBook(now) {
			continue
		}
		seats, err := st.SeatMap(ctx, id)
		if err != nil || len(seats) == 0 {
			return err
		}
		seat := seats[len(seats)/2]
		paidAt := now.Add(-time.Duration(made+1) * 24 * time.Hour)
		o := domain.Order{
			CustomerID: customerID, Status: domain.Paid, PaymentMethod: domain.Card,
			BookingDeadline: s.BookingDeadline(), Total: seat.Price, CreatedAt: paidAt, PaidAt: &paidAt,
			Delivery: domain.Delivery{Option: domain.ETicket},
			Tickets: []domain.Ticket{{SessionID: id, EventID: s.EventID, SeatID: seat.ID, Price: seat.Price,
				Code: fmt.Sprintf("SEED%06d", id), Active: true}},
		}
		if err := st.CreateOrder(ctx, &o); err != nil {
			return fmt.Errorf("seed paid order: %w", err)
		}
		made++
	}
	return nil
}
