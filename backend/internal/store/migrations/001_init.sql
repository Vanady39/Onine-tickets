CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    email         TEXT NOT NULL UNIQUE,
    name          TEXT NOT NULL,
    role          TEXT NOT NULL CHECK (role IN ('customer', 'cashier', 'manager', 'admin')),
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE venues (
    id      BIGSERIAL PRIMARY KEY,
    name    TEXT NOT NULL,
    address TEXT NOT NULL
);

CREATE TABLE halls (
    id       BIGSERIAL PRIMARY KEY,
    venue_id BIGINT NOT NULL REFERENCES venues(id),
    name     TEXT NOT NULL,
    kind     TEXT NOT NULL CHECK (kind IN ('seated', 'general_admission')),
    capacity INT NOT NULL CHECK (capacity > 0)
);

CREATE TABLE price_zones (
    id      BIGSERIAL PRIMARY KEY,
    hall_id BIGINT NOT NULL REFERENCES halls(id),
    name    TEXT NOT NULL
);

CREATE TABLE seats (
    id      BIGSERIAL PRIMARY KEY,
    hall_id BIGINT NOT NULL REFERENCES halls(id),
    zone_id BIGINT NOT NULL REFERENCES price_zones(id),
    row_no  INT NOT NULL,
    seat_no INT NOT NULL,
    UNIQUE (hall_id, row_no, seat_no)
);

CREATE TABLE events (
    id           BIGSERIAL PRIMARY KEY,
    title        TEXT NOT NULL,
    type         TEXT NOT NULL CHECK (type IN ('concert', 'cinema', 'theatre', 'exhibition')),
    description  TEXT NOT NULL DEFAULT '',
    duration_min INT NOT NULL DEFAULT 0,
    age_rating   TEXT NOT NULL DEFAULT '',
    image_url    TEXT NOT NULL DEFAULT ''
);

CREATE TABLE sessions (
    id         BIGSERIAL PRIMARY KEY,
    event_id   BIGINT NOT NULL REFERENCES events(id),
    hall_id    BIGINT NOT NULL REFERENCES halls(id),
    starts_at  TIMESTAMPTZ NOT NULL,
    base_price BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX sessions_event_idx ON sessions (event_id, starts_at);

CREATE TABLE zone_prices (
    session_id BIGINT NOT NULL REFERENCES sessions(id),
    zone_id    BIGINT NOT NULL REFERENCES price_zones(id),
    price      BIGINT NOT NULL CHECK (price >= 0),
    PRIMARY KEY (session_id, zone_id)
);

CREATE TABLE add_ons (
    id         BIGSERIAL PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES sessions(id),
    kind       TEXT NOT NULL CHECK (kind IN ('business_lunch', 'masterclass', 'presentation')),
    name       TEXT NOT NULL,
    price      BIGINT NOT NULL CHECK (price >= 0),
    capacity   INT NOT NULL CHECK (capacity > 0)
);

CREATE TABLE orders (
    id                BIGSERIAL PRIMARY KEY,
    customer_id       BIGINT NOT NULL REFERENCES users(id),
    status            TEXT NOT NULL CHECK (status IN
                        ('pending_payment', 'booked', 'paid', 'cancelled', 'expired', 'refunded')),
    payment_method    TEXT NOT NULL CHECK (payment_method IN ('card', 'cash')),
    booking_code      TEXT UNIQUE,
    booking_deadline  TIMESTAMPTZ NOT NULL,
    hold_expires_at   TIMESTAMPTZ,
    payment_ref       TEXT UNIQUE,
    total             BIGINT NOT NULL,
    delivery_option   TEXT NOT NULL CHECK (delivery_option IN ('e_ticket', 'courier')),
    delivery_address  TEXT NOT NULL DEFAULT '',
    delivery_fee      BIGINT NOT NULL DEFAULT 0,
    courier_ref       TEXT NOT NULL DEFAULT '',
    delivery_status   TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL,
    paid_at           TIMESTAMPTZ
);
CREATE INDEX orders_customer_idx ON orders (customer_id, created_at DESC);
CREATE INDEX orders_expiry_idx ON orders (status, hold_expires_at, booking_deadline);

CREATE TABLE tickets (
    id         BIGSERIAL PRIMARY KEY,
    order_id   BIGINT NOT NULL REFERENCES orders(id),
    session_id BIGINT NOT NULL REFERENCES sessions(id),
    seat_id    BIGINT REFERENCES seats(id),
    price      BIGINT NOT NULL,
    code       TEXT NOT NULL UNIQUE,
    active     BOOLEAN NOT NULL DEFAULT TRUE
);
-- NFR-04: a seat can be in at most one active hold, booking or paid order.
CREATE UNIQUE INDEX tickets_one_active_seat ON tickets (session_id, seat_id) WHERE active AND seat_id IS NOT NULL;
CREATE INDEX tickets_order_idx ON tickets (order_id);

CREATE TABLE ticket_add_ons (
    ticket_id BIGINT NOT NULL REFERENCES tickets(id),
    add_on_id BIGINT NOT NULL REFERENCES add_ons(id),
    price     BIGINT NOT NULL,
    PRIMARY KEY (ticket_id, add_on_id)
);

-- NFR-12: money operations by staff and price changes.
CREATE TABLE audit_log (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT REFERENCES users(id),
    action     TEXT NOT NULL,
    details    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
