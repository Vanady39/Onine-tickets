package domain

import "time"

// EventFilter narrows the programme (FR-01, FR-02).
type EventFilter struct {
	Type EventType
	From *time.Time
	To   *time.Time
}

// NewSession is a session to schedule: zone prices for seated halls, base price and
// add-ons for general-admission (exhibition) halls.
type NewSession struct {
	EventID   int64          `json:"eventId"`
	HallID    int64          `json:"hallId"`
	StartsAt  time.Time      `json:"startsAt"`
	BasePrice Money          `json:"basePrice"`
	Prices    []ZonePrice    `json:"prices"`
	AddOns    []AddOnService `json:"addOns"`
}

// HallSpec describes a hall to create. A seated hall gets its seats from Zones, row by row:
// each zone takes the next Rows rows with SeatsPerRow seats each.
type HallSpec struct {
	Name     string     `json:"name"`
	Kind     HallKind   `json:"kind"`
	Capacity int        `json:"capacity"` // general admission only
	Zones    []ZoneSpec `json:"zones"`
}

type ZoneSpec struct {
	Name        string `json:"name"`
	Rows        int    `json:"rows"`
	SeatsPerRow int    `json:"seatsPerRow"`
}

type SessionReport struct {
	SessionID   int64     `json:"sessionId"`
	StartsAt    time.Time `json:"startsAt"`
	HallName    string    `json:"hallName"`
	VenueName   string    `json:"venueName"`
	Capacity    int       `json:"capacity"`
	TicketsSold int       `json:"ticketsSold"`
	Revenue     Money     `json:"revenue"`
	Occupancy   float64   `json:"occupancy"` // percent of capacity sold
}

type EventReport struct {
	EventID     int64           `json:"eventId"`
	Title       string          `json:"title"`
	TicketsSold int             `json:"ticketsSold"`
	Revenue     Money           `json:"revenue"`
	Sessions    []SessionReport `json:"sessions"`
}
