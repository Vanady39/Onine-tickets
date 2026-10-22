// Package courier contains a mock courier service (AC-6): it accepts delivery requests and
// returns a tracking reference.
package courier

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"

	"github.com/Vanady/online-tickets/backend/internal/app"
)

type Mock struct {
	Log *slog.Logger
}

func (m *Mock) RequestDelivery(_ context.Context, req app.DeliveryRequest) (string, error) {
	if req.Address == "" || len(req.TicketCodes) == 0 {
		return "", errors.New("courier: address and tickets are required")
	}
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	ref := "CR-" + hex.EncodeToString(b)
	m.Log.Info("mock courier: delivery requested", "order", req.OrderID, "ref", ref, "tickets", len(req.TicketCodes))
	return ref, nil
}
