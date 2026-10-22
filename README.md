# Online Tickets

Course project for **Software Architecture (F26, Innopolis University)** — Project 3: an online
system for booking and selling tickets to concerts, cinema, theatre and exhibitions.

- Lab deliverables: [`docs/labs/`](docs/labs/) (rubric in [`LAB_TASKS.md`](docs/labs/LAB_TASKS.md))
- UML diagrams (PlantUML source + SVG): [`docs/diagrams/`](docs/diagrams/)
- Backend: Go 1.26, PostgreSQL 16 — [`backend/`](backend/)
- Frontend: React + TypeScript + Vite — [`frontend/`](frontend/)

## Quick start (Docker)

```bash
docker compose up --build
```

Open <http://localhost:8080>. The database is seeded with demo venues, events and accounts
(password for all demo accounts: `demo-pass-123`, local demo only):

| Account | Role |
|---|---|
| `customer@tickets.local` | regular customer |
| `loyal@tickets.local` | loyal customer (4 paid orders → can buy season tickets) |
| `cashier@tickets.local` | box office cashier |
| `manager@tickets.local` | event manager |
| `admin@tickets.local` | administrator |

Card payments go to a **mock payment provider** page (no card data is asked; choose "Pay" or
"Decline"). Courier delivery uses a **mock courier service** that logs the request.

Environment variables (see `docker-compose.yml`): `JWT_SECRET`, `PAYMENT_SECRET`,
`POSTGRES_PASSWORD`, `PUBLIC_URL` (public origin, default `http://localhost:8080`), `WEB_PORT`,
`SEED_DEMO`, `SEED_PASSWORD`. Change the secrets for anything other than a local demo.

## Architecture

Client–server web application, modular monolith with a hexagonal (ports and adapters) backend —
justification in [Lab 6](docs/labs/lab06-class-diagram.md).

```
React SPA ──HTTPS/JSON──► nginx ──► Go API ──SQL──► PostgreSQL
                                     ├─ httpapi   (REST adapter, auth, roles)
                                     ├─ app       (use cases + ports: Repository, PaymentGateway, CourierService)
                                     ├─ domain    (entities, business rules — no dependencies)
                                     ├─ store     (PostgreSQL adapter, migrations)
                                     └─ payment / courier (mock adapters)
```

Key rules and where they live:

| Rule | Implementation |
|---|---|
| Max 3 tickets for one event per order (every customer) | `domain.BuildOrder` |
| Loyal = more than 3 paid orders; season ticket (2+ events) only for loyal | `domain.IsLoyal`, `domain.BuildOrder` |
| Booking with code, valid until 24 h before the earliest session | `domain.Order`, `store.ExpireDue`, `app.ExpiryJob` |
| 10-minute seat hold while paying by card | `domain.HoldDuration`, `domain.Order.MarkPaid` |
| A seat is never sold twice | unique partial index `tickets_one_active_seat` + transaction in `store.CreateOrder` |
| Payment confirmed only by signed provider callback; late confirmation refunded | `app.OrderService.ConfirmPayment` |

## Development

Backend (needs PostgreSQL):

```bash
docker run -d --name ot-pg -e POSTGRES_USER=tickets -e POSTGRES_PASSWORD=tickets -e POSTGRES_DB=tickets -p 55432:5432 postgres:16-alpine
```

```bash
cd backend && DATABASE_URL='postgres://tickets:tickets@localhost:55432/tickets?sslmode=disable' ADDR=:18080 PUBLIC_URL=http://localhost:18080 SEED_DEMO=true go run ./cmd/api
```

Frontend (proxies `/api` and `/mock-provider` to `http://localhost:18080`):

```bash
cd frontend && npm install && npm run dev
```

Tests (unit tests always; end-to-end API tests run against PostgreSQL when `TEST_DATABASE_URL` is
set — each run creates and drops its own database):

```bash
cd backend && TEST_DATABASE_URL='postgres://tickets:tickets@localhost:55432/tickets?sslmode=disable' go test ./...
```

The end-to-end suite covers card purchase, decline and hold expiry, booking paid in cash at the box
office, booking expiry, ticket limit, season tickets, courier delivery, exhibition add-ons, refunds,
late payment confirmation, back-office reports, and 100 parallel orders for one seat (exactly one
succeeds — NFR-04).

Diagrams: edit `docs/diagrams/*.puml` and run `scripts/render-diagrams.sh` (Docker), or paste the
`.puml` file into <https://www.plantuml.com/plantuml> or draw.io (Arrange → Insert → Advanced →
PlantUML).

## REST API

| Method and path | Role | Purpose |
|---|---|---|
| `GET /api/events?type=&from=&to=` | public | programme (FR-01, FR-02) |
| `GET /api/events/{id}` | public | event with upcoming sessions |
| `GET /api/sessions/{id}`, `GET /api/sessions/{id}/seats` | public | session details, seat map |
| `POST /api/auth/register`, `POST /api/auth/login`, `GET /api/me` | — / any | accounts, loyalty status |
| `POST /api/orders` | customer | buy (card) or book (code) |
| `GET /api/orders`, `GET /api/orders/{id}` | customer | purchase history |
| `POST /api/orders/{id}/pay` · `/cancel` · `/refund` | customer | payment, cancellation, refund |
| `POST /api/payments/callback` | provider | signed payment result |
| `GET /api/boxoffice/bookings/{code}`, `POST …/{code}/pay-cash` | cashier | box office |
| `GET/POST /api/admin/events`, `PUT /api/admin/events/{id}`, `GET …/{id}/report` | manager | events, sales report |
| `POST /api/admin/sessions`, `GET /api/admin/venues` | manager | schedule sessions and prices |
| `POST /api/admin/venues` | admin | venues, halls, seat maps |

Known limits: the mock provider keeps payments in memory (refunds of payments made before an API
restart fail); courier status is set to `requested`/`failed` only — there is no courier callback
(FR-27, Could).
