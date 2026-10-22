package httpapi

import (
	"net/http"
	"time"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

// --- catalogue

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.EventFilter{Type: domain.EventType(q.Get("type"))}
	var err error
	if f.From, err = parseDate(q.Get("from"), 0); err != nil {
		s.fail(w, r, err)
		return
	}
	// "to" is inclusive: sessions up to the end of that day.
	if f.To, err = parseDate(q.Get("to"), 24*time.Hour); err != nil {
		s.fail(w, r, err)
		return
	}
	events, err := s.Catalog.Events(r.Context(), f)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// parseDate parses YYYY-MM-DD in Moscow time and adds shift; empty means no bound.
func parseDate(v string, shift time.Duration) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.ParseInLocation(time.DateOnly, v, moscow)
	if err != nil {
		return nil, domain.ErrValidation
	}
	t = t.Add(shift)
	return &t, nil
}

var moscow = time.FixedZone("MSK", 3*60*60)

func (s *Server) getEvent(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		var e domain.Event
		if e, err = s.Catalog.Event(r.Context(), id); err == nil {
			writeJSON(w, http.StatusOK, e)
			return
		}
	}
	s.fail(w, r, err)
}

func (s *Server) getSession(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		var se domain.Session
		if se, err = s.Catalog.Session(r.Context(), id); err == nil {
			writeJSON(w, http.StatusOK, se)
			return
		}
	}
	s.fail(w, r, err)
}

func (s *Server) getSeats(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		var seats []domain.SeatState
		if seats, err = s.Catalog.SeatMap(r.Context(), id); err == nil {
			writeJSON(w, http.StatusOK, seats)
			return
		}
	}
	s.fail(w, r, err)
}

// --- accounts

type credentials struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string      `json:"token"`
	User  domain.User `json:"user"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := decode(r, &c); err != nil {
		s.fail(w, r, err)
		return
	}
	if _, err := s.Accounts.Register(r.Context(), c.Name, c.Email, c.Password); err != nil {
		s.fail(w, r, err)
		return
	}
	token, u, err := s.Accounts.Login(r.Context(), c.Email, c.Password)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: u})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := decode(r, &c); err != nil {
		s.fail(w, r, err)
		return
	}
	token, u, err := s.Accounts.Login(r.Context(), c.Email, c.Password)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: u})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	p, err := s.Accounts.Profile(r.Context(), userFrom(r).ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// --- orders

func (s *Server) placeOrder(w http.ResponseWriter, r *http.Request) {
	var req domain.OrderRequest
	if err := decode(r, &req); err != nil {
		s.fail(w, r, err)
		return
	}
	o, err := s.Orders.PlaceOrder(r.Context(), userFrom(r).ID, req)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	orders, err := s.Orders.History(r.Context(), userFrom(r).ID)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, orders)
}

func (s *Server) getOrder(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		var o domain.Order
		if o, err = s.Orders.Order(r.Context(), userFrom(r).ID, id); err == nil {
			writeJSON(w, http.StatusOK, o)
			return
		}
	}
	s.fail(w, r, err)
}

func (s *Server) pay(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		var url string
		if url, err = s.Orders.StartCardPayment(r.Context(), userFrom(r).ID, id); err == nil {
			writeJSON(w, http.StatusOK, map[string]string{"redirectUrl": url})
			return
		}
	}
	s.fail(w, r, err)
}

func (s *Server) cancel(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		var o domain.Order
		if o, err = s.Orders.Cancel(r.Context(), userFrom(r).ID, id); err == nil {
			writeJSON(w, http.StatusOK, o)
			return
		}
	}
	s.fail(w, r, err)
}

func (s *Server) refund(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		var o domain.Order
		if o, err = s.Orders.Refund(r.Context(), userFrom(r).ID, id); err == nil {
			writeJSON(w, http.StatusOK, o)
			return
		}
	}
	s.fail(w, r, err)
}

// paymentCallback is the provider webhook; the signature proves its origin (NFR-05).
func (s *Server) paymentCallback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Ref       string `json:"ref"`
		Status    string `json:"status"`
		Signature string `json:"signature"`
	}
	if err := decode(r, &body); err != nil {
		s.fail(w, r, err)
		return
	}
	if err := s.Orders.ConfirmPayment(r.Context(), body.Ref, body.Status, body.Signature); err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

// --- box office

func (s *Server) findBooking(w http.ResponseWriter, r *http.Request) {
	o, err := s.BoxOffice.FindBooking(r.Context(), r.PathValue("code"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) payCash(w http.ResponseWriter, r *http.Request) {
	o, err := s.BoxOffice.AcceptCash(r.Context(), userFrom(r).ID, r.PathValue("code"))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// --- back office

func (s *Server) adminEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.Back.Events(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

type eventInput struct {
	Title       string           `json:"title"`
	Type        domain.EventType `json:"type"`
	Description string           `json:"description"`
	DurationMin int              `json:"durationMin"`
	AgeRating   string           `json:"ageRating"`
	ImageURL    string           `json:"imageUrl"`
}

func (in eventInput) event(id int64) domain.Event {
	return domain.Event{ID: id, Title: in.Title, Type: in.Type, Description: in.Description,
		DurationMin: in.DurationMin, AgeRating: in.AgeRating, ImageURL: in.ImageURL}
}

func (s *Server) createEvent(w http.ResponseWriter, r *http.Request) {
	var in eventInput
	if err := decode(r, &in); err != nil {
		s.fail(w, r, err)
		return
	}
	e, err := s.Back.CreateEvent(r.Context(), in.event(0))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, e)
}

func (s *Server) updateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	var in eventInput
	if err := decode(r, &in); err != nil {
		s.fail(w, r, err)
		return
	}
	e, err := s.Back.UpdateEvent(r.Context(), in.event(id))
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, e)
}

func (s *Server) createSession(w http.ResponseWriter, r *http.Request) {
	var ns domain.NewSession
	if err := decode(r, &ns); err != nil {
		s.fail(w, r, err)
		return
	}
	id, err := s.Back.CreateSession(r.Context(), userFrom(r).ID, ns)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err == nil {
		var rep domain.EventReport
		if rep, err = s.Back.SalesReport(r.Context(), id); err == nil {
			writeJSON(w, http.StatusOK, rep)
			return
		}
	}
	s.fail(w, r, err)
}

func (s *Server) venues(w http.ResponseWriter, r *http.Request) {
	v, err := s.Back.Venues(r.Context())
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}

func (s *Server) createVenue(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name    string            `json:"name"`
		Address string            `json:"address"`
		Halls   []domain.HallSpec `json:"halls"`
	}
	if err := decode(r, &in); err != nil {
		s.fail(w, r, err)
		return
	}
	id, err := s.Back.CreateVenue(r.Context(), in.Name, in.Address, in.Halls)
	if err != nil {
		s.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
