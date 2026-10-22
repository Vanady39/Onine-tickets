package domain

import "errors"

// Business rule violations. The HTTP layer maps them to 4xx responses.
var (
	ErrNotFound           = errors.New("not found")
	ErrValidation         = errors.New("invalid request")
	ErrEmptyOrder         = errors.New("order has no tickets")
	ErrTicketLimit        = errors.New("at most 3 tickets for one event in one order")
	ErrNotLoyal           = errors.New("season tickets are available to loyal customers only")
	ErrSessionStarted     = errors.New("session has already started")
	ErrBookingClosed      = errors.New("booking closes 24 hours before the session")
	ErrSeatRequired       = errors.New("seat must be chosen for a seated session")
	ErrSeatNotInHall      = errors.New("seat does not belong to the session hall")
	ErrDuplicateSeat      = errors.New("the same seat is selected twice")
	ErrSeatTaken          = errors.New("seat is no longer available")
	ErrSoldOut            = errors.New("not enough places left")
	ErrAddOnNotAllowed    = errors.New("add-on services are available for exhibition tickets only")
	ErrUnknownAddOn       = errors.New("add-on does not belong to the session")
	ErrCourierNeedsCard   = errors.New("courier delivery requires payment by card")
	ErrAddressRequired    = errors.New("delivery address is required for courier delivery")
	ErrInvalidTransition  = errors.New("operation is not allowed in the current order status")
	ErrHoldExpired        = errors.New("seat hold has expired, please start again")
	ErrRefundClosed       = errors.New("refunds close 24 hours before the session")
	ErrRefundCardOnly     = errors.New("only card payments can be refunded online")
	ErrEmailTaken         = errors.New("e-mail is already registered")
	ErrBadCredentials     = errors.New("wrong e-mail or password")
	ErrForbidden          = errors.New("forbidden")
	ErrPaymentUnconfirmed = errors.New("payment was not confirmed by the provider")
	// ErrDuplicateCode means a generated booking or ticket code collided; the caller retries.
	ErrDuplicateCode = errors.New("duplicate code")
)
