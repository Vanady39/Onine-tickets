package domain

import "time"

type EventType string

const (
	Concert    EventType = "concert"
	Cinema     EventType = "cinema"
	Theatre    EventType = "theatre"
	Exhibition EventType = "exhibition"
)

func (t EventType) Valid() bool {
	switch t {
	case Concert, Cinema, Theatre, Exhibition:
		return true
	}
	return false
}

type Event struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Type        EventType `json:"type"`
	Description string    `json:"description"`
	DurationMin int       `json:"durationMin"`
	AgeRating   string    `json:"ageRating"`
	ImageURL    string    `json:"imageUrl"`
	Sessions    []Session `json:"sessions,omitempty"`
	// NextSessionAt is the start of the earliest upcoming session (programme listing only).
	NextSessionAt *time.Time `json:"nextSessionAt,omitempty"`
	MinPrice      *Money     `json:"minPrice,omitempty"`
}

type Venue struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Halls   []Hall `json:"halls,omitempty"`
}

type HallKind string

const (
	Seated           HallKind = "seated"
	GeneralAdmission HallKind = "general_admission"
)

type Hall struct {
	ID        int64       `json:"id"`
	VenueID   int64       `json:"venueId"`
	VenueName string      `json:"venueName"`
	Address   string      `json:"address"`
	Name      string      `json:"name"`
	Kind      HallKind    `json:"kind"`
	Capacity  int         `json:"capacity"`
	Zones     []PriceZone `json:"zones,omitempty"`
}

func (h Hall) IsSeated() bool { return h.Kind == Seated }

type PriceZone struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type Seat struct {
	ID     int64 `json:"id"`
	ZoneID int64 `json:"zoneId"`
	Row    int   `json:"row"`
	Number int   `json:"number"`
}

type ZonePrice struct {
	ZoneID   int64  `json:"zoneId"`
	ZoneName string `json:"zoneName"`
	Price    Money  `json:"price"`
}

type AddOnKind string

const (
	BusinessLunch AddOnKind = "business_lunch"
	Masterclass   AddOnKind = "masterclass"
	Presentation  AddOnKind = "presentation"
)

func (k AddOnKind) Valid() bool {
	switch k {
	case BusinessLunch, Masterclass, Presentation:
		return true
	}
	return false
}

type AddOnService struct {
	ID        int64     `json:"id"`
	Kind      AddOnKind `json:"kind"`
	Name      string    `json:"name"`
	Price     Money     `json:"price"`
	Capacity  int       `json:"capacity"`
	Remaining int       `json:"remaining"`
}

// BookingWindow is how long before the session start bookings stay valid (BR-4).
const BookingWindow = 24 * time.Hour

type Session struct {
	ID         int64          `json:"id"`
	EventID    int64          `json:"eventId"`
	EventTitle string         `json:"eventTitle"`
	EventType  EventType      `json:"eventType"`
	Hall       Hall           `json:"hall"`
	StartsAt   time.Time      `json:"startsAt"`
	BasePrice  Money          `json:"basePrice"` // general-admission price
	Prices     []ZonePrice    `json:"prices"`
	AddOns     []AddOnService `json:"addOns"`
	// Remaining general-admission places (exhibitions only).
	Remaining int `json:"remaining"`
}

// BookingDeadline is the moment an unpaid booking for this session expires.
func (s Session) BookingDeadline() time.Time { return s.StartsAt.Add(-BookingWindow) }

func (s Session) CanBook(now time.Time) bool { return now.Before(s.BookingDeadline()) }

func (s Session) Started(now time.Time) bool { return !now.Before(s.StartsAt) }

func (s Session) PriceForZone(zoneID int64) (Money, bool) {
	for _, p := range s.Prices {
		if p.ZoneID == zoneID {
			return p.Price, true
		}
	}
	return 0, false
}

func (s Session) AddOn(id int64) (AddOnService, bool) {
	for _, a := range s.AddOns {
		if a.ID == id {
			return a, true
		}
	}
	return AddOnService{}, false
}

type SeatStatus string

const (
	SeatFree SeatStatus = "free"
	SeatHeld SeatStatus = "held" // in checkout or booked
	SeatSold SeatStatus = "sold"
)

type SeatState struct {
	Seat
	ZoneName string     `json:"zoneName"`
	Price    Money      `json:"price"`
	Status   SeatStatus `json:"status"`
}
