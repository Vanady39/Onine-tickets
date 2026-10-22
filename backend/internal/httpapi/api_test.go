package httpapi_test

// End-to-end tests: HTTP API → services → PostgreSQL. They need a PostgreSQL server:
//   TEST_DATABASE_URL=postgres://user:pass@localhost:5432/postgres?sslmode=disable go test ./...
// Each run creates and drops its own database. Without the variable the tests are skipped.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Vanady/online-tickets/backend/internal/app"
	"github.com/Vanady/online-tickets/backend/internal/courier"
	"github.com/Vanady/online-tickets/backend/internal/domain"
	"github.com/Vanady/online-tickets/backend/internal/httpapi"
	"github.com/Vanady/online-tickets/backend/internal/payment"
	"github.com/Vanady/online-tickets/backend/internal/seed"
	"github.com/Vanady/online-tickets/backend/internal/store"
)

const password = "test-pass-123"

type env struct {
	t     *testing.T
	srv   *httptest.Server
	st    *store.Store
	clock *clock
}

type clock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *clock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *clock) Add(d time.Duration) { c.mu.Lock(); c.now = c.now.Add(d); c.mu.Unlock() }

func setup(t *testing.T) *env {
	t.Helper()
	adminURL := os.Getenv("TEST_DATABASE_URL")
	if adminURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, adminURL)
	if err != nil {
		t.Fatal(err)
	}
	dbName := fmt.Sprintf("ot_test_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE DATABASE "+dbName); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(adminURL)
	u.Path = "/" + dbName
	st, err := store.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		st.Close()
		_, _ = admin.Exec(ctx, "DROP DATABASE "+dbName+" WITH (FORCE)")
		admin.Close(ctx)
	})
	if err := st.Migrate(ctx); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	clk := &clock{now: time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)}
	accounts := &app.AccountService{Repo: st, JWTSecret: []byte("test"), Now: clk.Now}
	if err := seed.Run(ctx, st, accounts, password, clk.Now()); err != nil {
		t.Fatal(err)
	}
	var srv *httptest.Server
	pay := payment.NewMock([]byte("pay-secret"), "", func(id int64) string { return fmt.Sprintf("/orders/%d", id) }, log)
	orders := &app.OrderService{Repo: st, Payment: pay, Courier: &courier.Mock{Log: log}, Now: clk.Now, Log: log}
	pay.OnResult = orders.ConfirmPayment
	srv = httptest.NewServer((&httpapi.Server{
		Accounts:  accounts,
		Catalog:   &app.CatalogService{Repo: st, Now: clk.Now},
		Orders:    orders,
		BoxOffice: &app.BoxOfficeService{Repo: st, Now: clk.Now},
		Back:      &app.BackOfficeService{Repo: st, Now: clk.Now},
		Extra:     pay.Handler(),
		Log:       log,
	}).Handler())
	pay.PageURL = srv.URL
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, st: st, clock: clk}
}

// call performs a JSON request and decodes the response into out (if not nil).
func (e *env) call(method, path, token string, body, out any) int {
	e.t.Helper()
	var r io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		r = bytes.NewReader(b)
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, r)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(res.Body)
	if out != nil && res.StatusCode < 300 {
		if err := json.Unmarshal(data, out); err != nil {
			e.t.Fatalf("%s %s: decode %q: %v", method, path, data, err)
		}
	}
	return res.StatusCode
}

func (e *env) login(email string) string {
	e.t.Helper()
	var res struct{ Token string }
	if code := e.call("POST", "/api/auth/login", "", map[string]string{"email": email, "password": password}, &res); code != 200 {
		e.t.Fatalf("login %s: %d", email, code)
	}
	return res.Token
}

// session returns the first upcoming session of the first event of the given type that
// starts more than 24 h after the test clock.
func (e *env) session(typ domain.EventType) domain.Session {
	e.t.Helper()
	var events []domain.Event
	e.call("GET", "/api/events?type="+string(typ), "", nil, &events)
	for _, ev := range events {
		var full domain.Event
		e.call("GET", fmt.Sprintf("/api/events/%d", ev.ID), "", nil, &full)
		for _, s := range full.Sessions {
			if s.CanBook(e.clock.Now()) {
				return s
			}
		}
	}
	e.t.Fatalf("no bookable %s session", typ)
	return domain.Session{}
}

func (e *env) freeSeats(sessionID int64, n int) []int64 {
	e.t.Helper()
	var seats []domain.SeatState
	e.call("GET", fmt.Sprintf("/api/sessions/%d/seats", sessionID), "", nil, &seats)
	var ids []int64
	for _, s := range seats {
		if s.Status == domain.SeatFree && len(ids) < n {
			ids = append(ids, s.ID)
		}
	}
	if len(ids) < n {
		e.t.Fatalf("not enough free seats")
	}
	return ids
}

func items(sessionID int64, seats ...int64) []domain.OrderItem {
	var out []domain.OrderItem
	for _, s := range seats {
		out = append(out, domain.OrderItem{SessionID: sessionID, SeatID: s})
	}
	return out
}

// payViaProvider follows the mock provider page like a browser would.
func (e *env) payViaProvider(token string, orderID int64, result string) {
	e.t.Helper()
	e.submitProvider(e.startPayment(token, orderID), result)
}

func (e *env) startPayment(token string, orderID int64) string {
	e.t.Helper()
	var res struct{ RedirectURL string }
	if code := e.call("POST", fmt.Sprintf("/api/orders/%d/pay", orderID), token, nil, &res); code != 200 {
		e.t.Fatalf("start payment: %d", code)
	}
	return res.RedirectURL
}

func (e *env) submitProvider(pageURL, result string) {
	e.t.Helper()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.PostForm(pageURL, url.Values{"result": {result}})
	if err != nil {
		e.t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		e.t.Fatalf("provider page: %d", resp.StatusCode)
	}
}

func TestCardPurchaseFlow(t *testing.T) {
	e := setup(t)
	token := e.login("customer@tickets.local")
	s := e.session(domain.Concert)
	seats := e.freeSeats(s.ID, 2)

	var o domain.Order
	code := e.call("POST", "/api/orders", token, domain.OrderRequest{Items: items(s.ID, seats...), PaymentMethod: domain.Card}, &o)
	if code != 201 || o.Status != domain.PendingPayment || len(o.Tickets) != 2 {
		t.Fatalf("place order: %d %+v", code, o)
	}
	// Seats are held for others now.
	var map1 []domain.SeatState
	e.call("GET", fmt.Sprintf("/api/sessions/%d/seats", s.ID), "", nil, &map1)
	for _, st := range map1 {
		if st.ID == seats[0] && st.Status != domain.SeatHeld {
			t.Fatalf("seat status %s, want held", st.Status)
		}
	}

	e.payViaProvider(token, o.ID, "succeeded")
	e.call("GET", fmt.Sprintf("/api/orders/%d", o.ID), token, nil, &o)
	if o.Status != domain.Paid || o.PaidAt == nil {
		t.Fatalf("after payment: %s", o.Status)
	}
	var history []domain.Order
	e.call("GET", "/api/orders", token, nil, &history)
	if len(history) != 1 || history[0].ID != o.ID {
		t.Fatalf("history: %+v", history)
	}

	// Refund more than 24 h before the session.
	if code := e.call("POST", fmt.Sprintf("/api/orders/%d/refund", o.ID), token, nil, &o); code != 200 || o.Status != domain.Refunded {
		t.Fatalf("refund: %d %s", code, o.Status)
	}
}

func TestDeclinedPaymentAndHoldExpiry(t *testing.T) {
	e := setup(t)
	token := e.login("customer@tickets.local")
	s := e.session(domain.Theatre)
	seat := e.freeSeats(s.ID, 1)

	var o domain.Order
	e.call("POST", "/api/orders", token, domain.OrderRequest{Items: items(s.ID, seat...), PaymentMethod: domain.Card}, &o)
	e.payViaProvider(token, o.ID, "declined")
	e.call("GET", fmt.Sprintf("/api/orders/%d", o.ID), token, nil, &o)
	if o.Status != domain.PendingPayment {
		t.Fatalf("declined payment changed status to %s", o.Status)
	}
	// After 10 minutes the hold expires and the seat is free again.
	e.clock.Add(domain.HoldDuration)
	job := &app.ExpiryJob{Repo: e.st, Now: e.clock.Now}
	if n, err := job.RunOnce(context.Background()); err != nil || n != 1 {
		t.Fatalf("expiry: %d %v", n, err)
	}
	e.call("GET", fmt.Sprintf("/api/orders/%d", o.ID), token, nil, &o)
	if o.Status != domain.Expired {
		t.Fatalf("status %s, want expired", o.Status)
	}
	var seats []domain.SeatState
	e.call("GET", fmt.Sprintf("/api/sessions/%d/seats", s.ID), "", nil, &seats)
	var free []int64
	for _, st := range seats {
		if st.Status == domain.SeatFree {
			free = append(free, st.ID)
		}
	}
	if !contains(free, seat[0]) {
		t.Fatal("seat was not released")
	}
}

func contains(ids []int64, id int64) bool {
	for _, x := range ids {
		if x == id {
			return true
		}
	}
	return false
}

func TestBookingPaidInCashAtBoxOffice(t *testing.T) {
	e := setup(t)
	token := e.login("customer@tickets.local")
	cashier := e.login("cashier@tickets.local")
	s := e.session(domain.Cinema)
	seats := e.freeSeats(s.ID, 3)

	var o domain.Order
	e.call("POST", "/api/orders", token, domain.OrderRequest{Items: items(s.ID, seats...), PaymentMethod: domain.Cash}, &o)
	if o.Status != domain.Booked || len(o.BookingCode) != 8 {
		t.Fatalf("booking: %+v", o)
	}
	// Customers cannot use the box office API.
	if code := e.call("GET", "/api/boxoffice/bookings/"+o.BookingCode, token, nil, nil); code != 403 {
		t.Fatalf("customer at box office: %d", code)
	}
	var found domain.Order
	if code := e.call("GET", "/api/boxoffice/bookings/"+strings.ToLower(o.BookingCode), cashier, nil, &found); code != 200 || found.ID != o.ID {
		t.Fatalf("find booking: %d", code)
	}
	if code := e.call("POST", "/api/boxoffice/bookings/"+o.BookingCode+"/pay-cash", cashier, nil, &found); code != 200 || found.Status != domain.Paid {
		t.Fatalf("pay cash: %d %s", code, found.Status)
	}
	if code := e.call("POST", "/api/boxoffice/bookings/"+o.BookingCode+"/pay-cash", cashier, nil, nil); code != 422 {
		t.Fatalf("second cash payment: %d", code)
	}
}

func TestBookingExpiresDayBeforeSession(t *testing.T) {
	e := setup(t)
	token := e.login("customer@tickets.local")
	s := e.session(domain.Concert)
	seat := e.freeSeats(s.ID, 1)
	var o domain.Order
	e.call("POST", "/api/orders", token, domain.OrderRequest{Items: items(s.ID, seat...), PaymentMethod: domain.Card, Book: true}, &o)
	if o.Status != domain.Booked {
		t.Fatalf("status %s", o.Status)
	}
	e.clock.Add(s.BookingDeadline().Sub(e.clock.Now()))
	if n, err := (&app.ExpiryJob{Repo: e.st, Now: e.clock.Now}).RunOnce(context.Background()); err != nil || n != 1 {
		t.Fatalf("expiry at %v (deadline %v, order deadline %v): %d %v", e.clock.Now(), s.BookingDeadline(), o.BookingDeadline, n, err)
	}
	token = e.login("customer@tickets.local") // days have passed: the old token expired (NFR-23)
	if code := e.call("GET", fmt.Sprintf("/api/orders/%d", o.ID), token, nil, &o); code != 200 || o.Status != domain.Expired {
		t.Fatalf("status %d %s, want expired", code, o.Status)
	}
	// Booking is now closed for this session; buying by card is still possible.
	seat2 := e.freeSeats(s.ID, 1)
	if code := e.call("POST", "/api/orders", token, domain.OrderRequest{Items: items(s.ID, seat2...), PaymentMethod: domain.Cash}, nil); code != 422 {
		t.Fatalf("late booking: %d", code)
	}
	if code := e.call("POST", "/api/orders", token, domain.OrderRequest{Items: items(s.ID, seat2...), PaymentMethod: domain.Card}, nil); code != 201 {
		t.Fatalf("late card purchase: %d", code)
	}
}

func TestBusinessRulesOverHTTP(t *testing.T) {
	e := setup(t)
	regular := e.login("customer@tickets.local")
	loyal := e.login("loyal@tickets.local")
	concert := e.session(domain.Concert)
	theatre := e.session(domain.Theatre)

	// Ticket limit.
	four := e.freeSeats(concert.ID, 4)
	if code := e.call("POST", "/api/orders", regular, domain.OrderRequest{Items: items(concert.ID, four...), PaymentMethod: domain.Card}, nil); code != 422 {
		t.Fatalf("4 tickets: %d", code)
	}
	// Season ticket: loyal only.
	season := append(items(concert.ID, four[0]), items(theatre.ID, e.freeSeats(theatre.ID, 1)...)...)
	if code := e.call("POST", "/api/orders", regular, domain.OrderRequest{Items: season, PaymentMethod: domain.Card}, nil); code != 422 {
		t.Fatalf("regular season ticket: %d", code)
	}
	var p domain.Profile
	e.call("GET", "/api/me", loyal, nil, &p)
	if !p.Loyal || p.PaidOrders != 4 {
		t.Fatalf("loyal profile: %+v", p)
	}
	var o domain.Order
	if code := e.call("POST", "/api/orders", loyal, domain.OrderRequest{Items: season, PaymentMethod: domain.Card}, &o); code != 201 || !o.IsSeasonTicket() {
		t.Fatalf("loyal season ticket: %d", code)
	}
	// Seat already held by the season ticket.
	if code := e.call("POST", "/api/orders", regular, domain.OrderRequest{Items: items(concert.ID, four[0]), PaymentMethod: domain.Card}, nil); code != 409 {
		t.Fatalf("taken seat: %d", code)
	}
	// Courier delivery needs card payment and an address.
	if code := e.call("POST", "/api/orders", regular, domain.OrderRequest{Items: items(concert.ID, four[1]), PaymentMethod: domain.Cash,
		DeliveryOption: domain.Courier, DeliveryAddress: "Innopolis"}, nil); code != 422 {
		t.Fatalf("cash courier: %d", code)
	}
	// Unauthenticated order.
	if code := e.call("POST", "/api/orders", "", domain.OrderRequest{Items: items(concert.ID, four[1]), PaymentMethod: domain.Card}, nil); code != 401 {
		t.Fatalf("anonymous order: %d", code)
	}
}

func TestExhibitionAddOnsAndCourier(t *testing.T) {
	e := setup(t)
	token := e.login("customer@tickets.local")
	s := e.session(domain.Exhibition)
	if len(s.AddOns) == 0 {
		t.Fatal("exhibition has no add-ons")
	}
	var o domain.Order
	code := e.call("POST", "/api/orders", token, domain.OrderRequest{
		Items:         []domain.OrderItem{{SessionID: s.ID, AddOnIDs: []int64{s.AddOns[0].ID}}, {SessionID: s.ID}},
		PaymentMethod: domain.Card, DeliveryOption: domain.Courier, DeliveryAddress: "Innopolis, Universitetskaya 1",
	}, &o)
	want := 2*s.BasePrice + s.AddOns[0].Price + domain.CourierFee
	if code != 201 || o.Total != want {
		t.Fatalf("exhibition order: %d total %s want %s", code, o.Total, want)
	}
	e.payViaProvider(token, o.ID, "succeeded")
	e.call("GET", fmt.Sprintf("/api/orders/%d", o.ID), token, nil, &o)
	if o.Status != domain.Paid || o.Delivery.Status != "requested" || o.Delivery.CourierRef == "" {
		t.Fatalf("courier not requested: %+v", o.Delivery)
	}
	var after domain.Session
	e.call("GET", fmt.Sprintf("/api/sessions/%d", s.ID), "", nil, &after)
	if after.Remaining != s.Remaining-2 || after.AddOns[0].Remaining != s.AddOns[0].Remaining-1 {
		t.Fatalf("capacity not updated: %d, %d", after.Remaining, after.AddOns[0].Remaining)
	}
}

// NFR-04: 100 parallel orders for the same seat — exactly one succeeds.
func TestNoDoubleSaleUnderConcurrency(t *testing.T) {
	e := setup(t)
	s := e.session(domain.Concert)
	seat := e.freeSeats(s.ID, 1)
	tokens := []string{e.login("customer@tickets.local"), e.login("loyal@tickets.local")}

	var wg sync.WaitGroup
	results := make(chan int, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results <- e.call("POST", "/api/orders", tokens[i%2], domain.OrderRequest{Items: items(s.ID, seat...), PaymentMethod: domain.Card}, nil)
		}(i)
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for c := range results {
		counts[c]++
	}
	if counts[201] != 1 || counts[409] != 99 {
		t.Fatalf("want 1×201 and 99×409, got %v", counts)
	}
}

func TestLatePaymentConfirmationIsRefunded(t *testing.T) {
	e := setup(t)
	token := e.login("customer@tickets.local")
	s := e.session(domain.Concert)
	var o domain.Order
	e.call("POST", "/api/orders", token, domain.OrderRequest{Items: items(s.ID, e.freeSeats(s.ID, 1)...), PaymentMethod: domain.Card}, &o)
	page := e.startPayment(token, o.ID)
	e.clock.Add(domain.HoldDuration + time.Second)
	e.submitProvider(page, "succeeded") // the customer paid after the hold ended
	e.call("GET", fmt.Sprintf("/api/orders/%d", o.ID), token, nil, &o)
	if o.Status != domain.Expired || o.PaidAt != nil {
		t.Fatalf("late confirmation: status %s", o.Status)
	}
}

func TestBackOfficeCreatesSessionAndReport(t *testing.T) {
	e := setup(t)
	manager := e.login("manager@tickets.local")
	customer := e.login("customer@tickets.local")
	if code := e.call("POST", "/api/admin/events", customer, map[string]any{"title": "x", "type": "concert"}, nil); code != 403 {
		t.Fatalf("customer creates event: %d", code)
	}
	var ev domain.Event
	if code := e.call("POST", "/api/admin/events", manager, map[string]any{"title": "Jazz Night", "type": "concert", "durationMin": 90}, &ev); code != 201 {
		t.Fatalf("create event: %d", code)
	}
	var venues []domain.Venue
	e.call("GET", "/api/admin/venues", manager, nil, &venues)
	hall := venues[0].Halls[0]
	ns := domain.NewSession{EventID: ev.ID, HallID: hall.ID, StartsAt: e.clock.Now().Add(72 * time.Hour)}
	if code := e.call("POST", "/api/admin/sessions", manager, ns, nil); code != 422 {
		t.Fatalf("session without prices: %d", code)
	}
	for _, z := range hall.Zones {
		ns.Prices = append(ns.Prices, domain.ZonePrice{ZoneID: z.ID, Price: domain.RUB(2000)})
	}
	var created struct{ ID int64 }
	if code := e.call("POST", "/api/admin/sessions", manager, ns, &created); code != 201 {
		t.Fatalf("create session: %d", code)
	}
	// Sell one ticket and check the report.
	var o domain.Order
	e.call("POST", "/api/orders", customer, domain.OrderRequest{Items: items(created.ID, e.freeSeats(created.ID, 1)...), PaymentMethod: domain.Card}, &o)
	e.payViaProvider(customer, o.ID, "succeeded")
	var rep domain.EventReport
	if code := e.call("GET", fmt.Sprintf("/api/admin/events/%d/report", ev.ID), manager, nil, &rep); code != 200 {
		t.Fatalf("report: %d", code)
	}
	if rep.TicketsSold != 1 || rep.Revenue != domain.RUB(2000) || rep.Sessions[0].Occupancy <= 0 {
		t.Fatalf("report: %+v", rep)
	}
}
