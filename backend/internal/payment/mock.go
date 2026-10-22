// Package payment contains a mock card payment provider (AC-6). It imitates a hosted payment
// page: the customer is redirected to it, chooses to succeed or decline, and the provider
// sends a signed result to the shop. No card data is ever requested.
package payment

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"sync"

	"github.com/Vanady/online-tickets/backend/internal/domain"
)

type payment struct {
	orderID  int64
	amount   domain.Money
	status   string // pending, succeeded, declined, refunded
	refunded domain.Money
}

// ResultHandler receives the signed payment result, like a provider webhook.
type ResultHandler func(ctx context.Context, ref, status, signature string) error

type Mock struct {
	secret []byte
	// PageURL is the public base URL of the mock provider page, e.g. http://localhost:8080.
	PageURL string
	// ReturnURL builds the shop page the customer returns to after paying.
	ReturnURL func(orderID int64) string
	OnResult  ResultHandler
	Log       *slog.Logger

	mu       sync.Mutex
	payments map[string]*payment
}

func NewMock(secret []byte, pageURL string, returnURL func(int64) string, log *slog.Logger) *Mock {
	return &Mock{secret: secret, PageURL: pageURL, ReturnURL: returnURL, Log: log, payments: map[string]*payment{}}
}

func (m *Mock) CreatePayment(_ context.Context, orderID int64, amount domain.Money) (string, string, error) {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	ref := "PAY-" + hex.EncodeToString(b)
	m.mu.Lock()
	m.payments[ref] = &payment{orderID: orderID, amount: amount, status: "pending"}
	m.mu.Unlock()
	return ref, m.PageURL + "/mock-provider/pay/" + ref, nil
}

func (m *Mock) sign(ref, status string) string {
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(ref + "|" + status))
	return hex.EncodeToString(mac.Sum(nil))
}

func (m *Mock) VerifyCallback(ref, status, signature string) bool {
	return hmac.Equal([]byte(m.sign(ref, status)), []byte(signature))
}

func (m *Mock) Refund(_ context.Context, ref string, amount domain.Money) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.payments[ref]
	if !ok || p.status != "succeeded" {
		return fmt.Errorf("payment %s cannot be refunded", ref)
	}
	if p.refunded+amount > p.amount {
		return fmt.Errorf("refund exceeds payment %s", ref)
	}
	p.refunded += amount
	if p.refunded == p.amount {
		p.status = "refunded"
	}
	m.Log.Info("mock provider: refund", "ref", ref, "amount", amount.String())
	return nil
}

var page = template.Must(template.New("pay").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>MockPay — payment</title>
<style>
 body{font-family:system-ui,sans-serif;background:#f1f3f8;margin:0;display:grid;place-items:center;min-height:100vh;color:#1d2433}
 .card{background:#fff;border-radius:16px;padding:32px;max-width:380px;width:calc(100% - 32px);box-shadow:0 10px 30px rgba(0,0,0,.08)}
 h1{font-size:20px;margin:0 0 4px} .muted{color:#6b7385;font-size:14px}
 .amount{font-size:34px;font-weight:700;margin:20px 0}
 button{width:100%;padding:14px;border:0;border-radius:10px;font-size:16px;font-weight:600;cursor:pointer;margin-top:10px}
 .ok{background:#1f7a4d;color:#fff} .no{background:#eef0f5;color:#1d2433}
 .note{margin-top:18px;font-size:12px;color:#6b7385}
</style></head><body><div class="card">
<h1>MockPay</h1><div class="muted">Test payment provider · order #{{.OrderID}}</div>
{{if eq .Status "pending"}}
<div class="amount">{{.Amount}}</div>
<form method="post"><button class="ok" name="result" value="succeeded">Pay (simulate success)</button>
<button class="no" name="result" value="declined">Decline (simulate failure)</button></form>
<p class="note">This page stands in for a real card provider. It never asks for card data.</p>
{{else}}<p>This payment is already {{.Status}}.</p><a href="{{.Return}}">Back to the shop</a>{{end}}
</div></body></html>`))

// Handler serves the mock provider page at /mock-provider/pay/{ref}.
func (m *Mock) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /mock-provider/pay/{ref}", func(w http.ResponseWriter, r *http.Request) {
		ref := r.PathValue("ref")
		m.mu.Lock()
		p, ok := m.payments[ref]
		var data map[string]any
		if ok {
			data = map[string]any{"OrderID": p.orderID, "Amount": p.amount.String(), "Status": p.status, "Return": m.ReturnURL(p.orderID)}
		}
		m.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := page.Execute(w, data); err != nil {
			m.Log.Error("render payment page", "err", err)
		}
	})
	mux.HandleFunc("POST /mock-provider/pay/{ref}", func(w http.ResponseWriter, r *http.Request) {
		ref := r.PathValue("ref")
		result := r.FormValue("result")
		if result != "succeeded" && result != "declined" {
			http.Error(w, "bad result", http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		p, ok := m.payments[ref]
		pending := ok && p.status == "pending"
		if pending {
			p.status = result
		}
		m.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		if pending && m.OnResult != nil {
			if err := m.OnResult(r.Context(), ref, result, m.sign(ref, result)); err != nil {
				m.Log.Error("payment result delivery failed", "ref", ref, "err", err)
			}
		}
		http.Redirect(w, r, m.ReturnURL(p.orderID)+"?payment="+result, http.StatusSeeOther)
	})
	return mux
}
