# Lab 3 — List of Requirements: Terms of Reference (ToR)

**Project:** Online Tickets (Project 3)
**Version:** 1.3 (22 Oct 2026)
**Inputs:** project brief (`docs/PROJECT_BRIEF.md`), elicitation results (Lab 2,
`lab02-requirements-gathering.md`). The "Source" column traces every requirement to the brief
(**B**), to a Lab 2 question (**Q#**) or, where neither applies, to a documented team decision
(**T**).

---

## 1. Introduction

### 1.1 Purpose
This document specifies what the Online Tickets system shall do and the quality levels it shall
meet. It is the agreement between the customer and the development team and the base for design,
testing and acceptance.

### 1.2 Scope
Online Tickets is a web system for booking and selling tickets to concerts, cinema sessions,
theatre performances and exhibitions. It covers: the event programme, seat selection with
seat-dependent prices, booking with a booking code, payment by bank card (online) or cash (box
office), delivery of tickets, purchase history, loyalty and season tickets, and back-office
management of events, sessions and prices.

Out of scope: the payment provider and the courier service themselves (external systems), access
control at the venue entrance, native mobile apps.

### 1.3 Definitions
See the glossary in `lab01-case-study.md`, section 5. In addition:
- **Paid order** — an order whose payment is confirmed by the payment provider (card) or recorded
  by a cashier (cash) and that has not been refunded. Refunded orders do not count as paid orders.
- **Seat hold** — a 10-minute reservation of the seats of a confirmed pay-by-card-now order while the customer pays at the provider (BR-7); it is not a booking.
- **p95** — the 95th percentile of response times measured during a load test.

### 1.4 Business goals
| ID | Goal | Metric | Source |
|---|---|---|---|
| BG-1 | Move ticket sales online | ≥ 40 % of all tickets sold online within 12 months after launch | Q2 |
| BG-2 | Never sell one seat twice | 0 double-sold seats per year | Q2, Q19 |
| BG-3 | Fast and simple purchase | Median purchase time (event page → confirmation) ≤ 3 minutes | Q2 |
| BG-4 | Keep frequent customers | Season tickets available to loyal customers | B, Q3 |

## 2. Overall description

### 2.1 User classes and external systems
| Actor | Description |
|---|---|
| Guest | Not signed in. Can browse the programme, events, sessions and seat maps. |
| Customer | Registered user. Books and buys tickets; has a purchase history. Sub-types: **regular customer** (0–3 paid orders) and **loyal customer** (more than 3 paid orders). |
| Cashier | Box office employee. Finds bookings by code and accepts cash payments. |
| Event manager | Creates events and sessions, sets prices, sees sales reports. |
| Administrator | Manages venues, halls, seat maps and staff accounts. |
| Payment provider | External system that charges bank cards and confirms payments. |
| Courier service | External system that delivers printed tickets. |

### 2.2 Operating environment
- Web application used in desktop and mobile browsers (latest two major versions of Chrome,
  Firefox, Safari, Edge).
- Server side runs in Docker containers on Linux servers hosted in Russia.

### 2.3 Business rules
| ID | Rule | Source |
|---|---|---|
| BR-1 | Purchasing and booking require a registered customer account. Guests can only browse. | Q1, B ("purchase history") |
| BR-2 | A customer may buy or book **at most 3 tickets for one event in one order**. The limit applies to regular and loyal customers; in a season-ticket order it applies to each event separately. | B, Q12 |
| BR-3 | A customer is **loyal** when they have **more than 3 paid orders**; refunded orders do not count. | B, Q3 |
| BR-4 | A booking can be made only for a session that starts more than 24 hours later. It holds its seats until **24 hours before the session start** (for a booking with sessions of several events, the earliest session); then it expires automatically and the seats return to sale. | B, Q13 |
| BR-5 | Only an order paid by card can be refunded, and only if the earliest session of the order starts more than 24 hours later. Cash-paid orders and later requests are not refunded. | Q9, T |
| BR-6 | Card data is entered only on the payment provider's page; the system never receives or stores card numbers. | Q8 |
| BR-7 | When the customer confirms an order to pay by card now, its seats are held for 10 minutes while the customer pays at the payment provider. If the payment is not confirmed within this time, the order expires and the seats return to sale. Bookings (BR-4) have no such hold. | Q19 |

### 2.4 Assumptions and dependencies
| ID | Assumption / dependency |
|---|---|
| A-1 | There is no real customer; the Lab 2 answers are the team's working assumptions and are treated as confirmed. |
| A-2 | The payment provider offers a hosted payment page, asynchronous payment confirmations (callbacks) and a refund API. |
| A-3 | The courier service accepts delivery requests and reports delivery status over an HTTP API. |
| A-4 | Venues, halls and seat maps for the first release can be loaded from seed data (see FR-37). |
| A-5 | The brief states the 3-ticket limit only for regular customers; Q12 confirms that BR-2 applies to loyal customers as well. |

---

## 3. Functional requirements (prioritised, MoSCoW)

Priority: **M** = Must have, **S** = Should have, **C** = Could have, **W** = Won't have (this release).
Each requirement is verified by an acceptance test derived from its wording.

### 3.1 Programme and event information
| ID | Requirement | Priority | Justification | Source |
|---|---|---|---|---|
| FR-01 | The system shall display the programme of upcoming events, filterable by event type (concert, cinema, theatre, exhibition). | M | Entry point of every purchase; brief requires viewing the programme. | B, Q10 |
| FR-02 | The system shall filter the programme by a date range. | S | Common need, but the type filter and the list already allow a purchase. | Q10 |
| FR-03 | The system shall display an event page with title, type, description, duration and age rating. | M | Brief requires information about each event. | B, Q10 |
| FR-04 | The system shall list the sessions of an event, each with date, start time, venue and hall. | M | Brief requires the schedule. | B, Q10 |
| FR-05 | The system shall display the ticket price of each price zone of a session. | M | Brief: prices are part of the schedule. | B, Q11 |
| FR-06 | For a seated session, the system shall display the hall seat map with row, seat number, price and status (free, held/booked, sold) of each seat. | M | Seat-dependent prices need a seat choice. | B, Q10 |

### 3.2 Accounts
| ID | Requirement | Priority | Justification | Source |
|---|---|---|---|---|
| FR-07 | The system shall register a customer with name, e-mail and password. | M | BR-1: history needs an account. | B, Q1 |
| FR-08 | The system shall authenticate a registered user with e-mail and password. | M | Needed for every purchase and every staff function. | B |
| FR-09 | The system shall show the customer's status (regular or loyal) on their profile page. | S | Customer must know if season tickets are available; loyalty itself works without it (FR-30). | B, Q3 |

### 3.3 Selection and ordering
| ID | Requirement | Priority | Justification | Source |
|---|---|---|---|---|
| FR-10 | The system shall let a signed-in customer select free seats on the seat map of a seated session. | M | Core purchase step. | B, Q10 |
| FR-11 | The system shall hold the seats of a confirmed pay-by-card-now order for 10 minutes while the customer pays at the payment provider (BR-7). | M | Prevents two customers paying for one seat (BG-2). | Q19 |
| FR-12 | For an exhibition session, the system shall let the customer choose a number of general-admission tickets, limited by the remaining session capacity. | M | Exhibitions have no seats. | B |
| FR-13 | The system shall let the customer add add-on services (business lunch, masterclass, presentation) to an exhibition ticket, limited by each add-on's remaining capacity for the session. | S | Brief feature; exhibition tickets can be sold without it. | B, Q14 |
| FR-14 | The system shall reject an order that contains more than 3 tickets for one event (BR-2). | M | Explicit business rule of the brief. | B, Q12 |
| FR-15 | The system shall let the customer choose a payment method: bank card online or cash at the box office. | M | Brief requirement. | B, Q5 |
| FR-16 | The system shall let the customer choose a delivery option: electronic ticket (default) or courier delivery to an address within the city for a fixed delivery fee. | S | Brief step, but e-ticket alone covers day one; courier delivery can wait (Q4). | B, Q4, Q15 |
| FR-17 | Before the transaction, the system shall show an order summary (session, seats, add-ons, price per item, delivery fee, total, payment method, delivery option) and require explicit confirmation. | M | Brief: "confirming the purchase". | B, Q10 |

### 3.4 Booking
| ID | Requirement | Priority | Justification | Source |
|---|---|---|---|---|
| FR-18 | The system shall create a booking for the selected tickets without payment, issue a unique 8-character booking code and display the code to the customer. | M | Brief requirement; day-one feature (Q4). | B, Q4, Q13 |
| FR-19 | The system shall expire an unpaid booking 24 hours before the start of its earliest session and return its seats to sale (BR-4). | M | Brief requirement. | B, Q13 |
| FR-20 | The system shall let the customer pay for an active booking online by card. | M | Booking must lead to a purchase. | B, Q13 |
| FR-21 | The system shall let a cashier find an active booking by its booking code. | M | Cash payment needs the code. | B, Q5 |
| FR-22 | The system shall let a cashier record the cash payment of an active booking, which marks its tickets as paid. | M | Brief: cash at the box office. | B, Q5 |
| FR-23 | The system shall let the customer cancel their active booking, which releases its seats. | S | Frees seats early; expiry (FR-19) covers it otherwise. | Q9, Q13 |

### 3.5 Payment, delivery, tickets
| ID | Requirement | Priority | Justification | Source |
|---|---|---|---|---|
| FR-24 | The system shall take a card payment through the external payment provider's hosted page and mark the order as paid only after the provider confirms the payment. | M | Brief: "processes the ticket purchase transaction"; BR-6. | B, Q8 |
| FR-25 | If a card payment fails or is declined, the system shall keep the order unpaid and let the customer retry the payment until the seat hold (BR-7) or the booking (BR-4) expires. | M | No tickets without money; no seats lost forever. | Q19 |
| FR-26 | After an order with courier delivery is paid, the system shall send a delivery request (order number, address, tickets) to the courier service. | S | Brief requirement, but courier delivery is not needed on day one (Q4); depends on FR-16. | B, Q4, Q15 |
| FR-27 | The system shall store the delivery request status (requested, delivered, failed) received from the courier service. | C | Useful for support; not needed to deliver. | Q15 |
| FR-28 | The system shall issue an electronic ticket with a unique ticket code for each paid ticket and make it available in the customer's account. | M | The customer needs proof of purchase; e-ticket is the default delivery. | Q15 |

### 3.6 History, loyalty, season tickets, refunds
| ID | Requirement | Priority | Justification | Source |
|---|---|---|---|---|
| FR-29 | The system shall show the customer's purchase history: all orders and bookings with event, session, tickets, booking code, total and status (booked, paid, cancelled, expired, refunded). | M | Brief: track purchase history and statuses; day-one feature (Q4). | B, Q1, Q4 |
| FR-30 | The system shall mark a customer as loyal when the number of their paid orders (refunded orders excluded) is greater than 3 (BR-3). | M | Brief requirement; paid orders must be counted from the first release so that season tickets (FR-31) work when they are released. | B, Q3 |
| FR-31 | The system shall let a loyal customer create one season-ticket order that contains tickets for sessions of 2 or more different events. | S | Brief feature for loyal customers; core sales work without it. | B, Q3 |
| FR-32 | The system shall reject a season-ticket order from a customer who is not loyal. | S | Enforces BR-3; needed only together with FR-31. | B, Q3 |
| FR-33 | The system shall refund a paid card order on the customer's request if the earliest session of the order starts more than 24 hours later (BR-5). | C | Business rule from Q9; at launch refunds can be done manually in the provider's dashboard. | Q9 |

### 3.7 Back office
| ID | Requirement | Priority | Justification | Source |
|---|---|---|---|---|
| FR-34 | The system shall let an event manager create and edit events. | M | No programme without events. | B, Q11 |
| FR-35 | The system shall let an event manager create a session of an event in a hall, with a date, start time and a price for each price zone of the hall. | M | No schedule or prices otherwise. | B, Q11 |
| FR-36 | The system shall let an event manager define add-on services with a price and capacity for an exhibition session. | S | Depends on FR-13. | B, Q14 |
| FR-37 | The system shall let an administrator create venues and halls with a seat map (rows, seats, price zones) or a general-admission capacity. | S | Halls can be loaded from seed data at launch (A-4). | Q11 |
| FR-38 | The system shall show an event manager a sales report per session and per event: tickets sold, revenue and occupancy in %. | C | Requested (Q16) but listed as "later" (Q4); not needed to sell tickets. | Q4, Q16 |
| FR-39 | The system shall let a cashier sell tickets for cash to a customer at the box office without a prior booking. | C | Not in the brief; customers can buy online by card up to the session start, and the cashier can take cash for bookings (FR-22). | Q5 |
| FR-44 | The system shall offer booking only for sessions that start more than 24 hours later (BR-4). | M | Brief: the booking code guarantees the purchase until one day before the event; a later booking would have no valid period. | B, Q13 |
| FR-45 | If the payment provider confirms a payment after the seat hold or booking of the order has expired, the system shall not issue tickets and shall request an automatic refund of that payment. | M | Keeps NFR-04 and NFR-05 consistent when a payment arrives late. | Q8, Q19 |
| FR-46 | When the customer confirms an order with the payment method "cash at the box office", the system shall turn the order into a booking (FR-18) instead of starting a card payment. | M | Links the brief's cash payment to the booking code. | B, Q5 |
| FR-47 | The system shall accept courier delivery only for card payment: it shall reject an order that combines courier delivery with the payment method "cash at the box office" and shall not let a cashier record a cash payment (FR-22) for a booking with courier delivery. | S | The courier request is sent only after payment (FR-26), and a customer who pays cash is already at the box office; needed only together with FR-16. | Q15, T |

FR-44…FR-47 are appended at the end of the table to keep the IDs used by other labs stable;
logically they belong to sections 3.3 (FR-47), 3.4 (FR-44, FR-46) and 3.5 (FR-45).

### 3.8 Won't have (this release)
| ID | Requirement | Priority | Justification | Source |
|---|---|---|---|---|
| FR-40 | The system shall recommend events to a customer based on their purchase history. | W | Not requested for day one (Q4); needs sales data first. | Q4 |
| FR-41 | The system shall let a customer sign in with a social-network account. | W | E-mail sign-in is enough; adds a third-party dependency. | T |
| FR-42 | The system shall change ticket prices automatically based on demand. | W | Prices are set by event managers (Q11). | Q11 |
| FR-43 | The system shall be available as native mobile applications. | W | Responsive web covers phones (Q20). | Q20 |

---

## 4. Non-functional requirements (prioritised, ISO/IEC 25010:2011)

| ID | ISO 25010 characteristic | Requirement | Priority | Justification | Source |
|---|---|---|---|---|---|
| NFR-01 | Performance efficiency — time behaviour | The system shall display the seat map of a session in ≤ 1 s (p95), measured from the session click to the rendered map, while 2 000 users are concurrently active. | M | Customer target; on-sale peaks. | Q17 |
| NFR-02 | Performance efficiency — time behaviour | Read API endpoints (programme, event, session, seat map) shall respond in ≤ 300 ms (p95) at 200 requests/s. | S | Server-side budget that supports NFR-01. | Q17 |
| NFR-03 | Performance efficiency — capacity | The system shall sustain 2 000 concurrent users and 50 order confirmations/s for 5 minutes with an HTTP 5xx error rate < 0.1 %. | S | Team estimate of the on-sale peak; the 2 000-user load itself is already enforced by NFR-01 (M). | Q17 |
| NFR-04 | Functional suitability — functional correctness | A seat shall never be part of more than one active hold, booking or paid order. Likewise, sold general-admission tickets and add-on places shall never exceed their capacity. Verified by a test of 100 parallel requests for the same seat (or the last free place): exactly 1 succeeds. | M | BG-2, critical for the customer. | Q19 |
| NFR-05 | Reliability — fault tolerance | An order shall be marked paid only after the payment provider has confirmed its payment; receiving the same provider confirmation more than once shall not change the result (idempotency). Verified by replaying a confirmation 10 times. | M | No tickets without money, no double charging. | Q8, Q19 |
| NFR-06 | Reliability — availability | The system shall be available ≥ 99.5 % of each calendar month, excluding planned maintenance of at most 2 hours per month between 02:00 and 05:00 MSK. | S | Customer accepts a night window (Q18). | Q18 |
| NFR-07 | Reliability — recoverability | A database backup shall be taken every 24 hours; recovery point objective ≤ 24 h, recovery time objective ≤ 4 h. | S | Customer accepts losing at most one day of data (Q18). | Q18 |
| NFR-08 | Security — confidentiality | All client–server traffic shall use TLS 1.2 or higher. | M | Personal data and sign-in. | Q7 |
| NFR-09 | Security — confidentiality | Passwords shall be stored only as bcrypt hashes (cost ≥ 10). | M | Standard practice; protects accounts after a data leak. | Q7 |
| NFR-10 | Security — confidentiality | The system shall not receive, store or log bank card numbers (PCI DSS SAQ-A scope). | M | BR-6. | Q8 |
| NFR-11 | Security — authenticity, integrity | Every API call that changes data shall be rejected (HTTP 401/403) unless it comes from an authenticated user whose role (customer, cashier, event manager, administrator) permits the operation. | M | Role separation for staff functions. | Q5 |
| NFR-12 | Security — accountability | Cash payments, refunds and price changes shall be written to an audit log with user, time and action. | S | Money operations by staff must be traceable. | Q5, Q9 |
| NFR-13 | Usability — operability | A customer shall be able to complete a purchase of a seated ticket in at most 6 screens from the event page to the confirmation. | S | Supports BG-3. | Q2, Q10 |
| NFR-14 | Portability — adaptability | At viewport widths from 360 px to 1920 px, every page of the web UI shall provide all its functions without horizontal scrolling. | M | Phones and desktops are both required. | Q20 |
| NFR-15 | Usability — accessibility | The purchase flow shall meet WCAG 2.1 level AA. | C | Good practice; not requested by the customer. | T |
| NFR-16 | Maintainability — testability | Domain logic (pricing, limits, booking expiry, loyalty) shall have automated unit tests with ≥ 70 % statement coverage. | S | Rules change often; tests prevent regressions. | Q9, Q12 |
| NFR-17 | Maintainability — modularity | Payment and courier integrations shall be accessed only through internal interfaces, so that replacing a provider requires changes in one adapter module only. | S | Providers may change; mocks are used now (AC-6). | Q5, Q21 |
| NFR-18 | Portability — installability | The whole system shall start on a clean Linux host with Docker installed using one command (`docker compose up`) in ≤ 10 minutes. | S | Simple deployment and demo. | Q6 |
| NFR-19 | Compatibility — interoperability | The public API shall be described in OpenAPI 3. | C | Helps the frontend and future partners. | Q5 |
| NFR-20 | Security — confidentiality (regulatory, see note) | Personal data of customers shall be stored only on servers located in Russia (Federal Law 152-FZ). | M | Legal requirement. | Q7 |
| NFR-21 | Performance efficiency — capacity | The system shall support 50 000 concurrent users. | W | Not needed for the expected load (Q17); avoids premature scaling cost. | Q17 |
| NFR-22 | Usability — operability | The system shall provide the UI in more than one language. | W | One regional market at launch; a single UI language (English, the course language) is enough. | T |
| NFR-23 | Security — authenticity | Access tokens shall expire 24 hours after issue; an expired token shall be rejected. | M | Limits the damage of a stolen token. | Q7 |

Note: ISO/IEC 25010:2011 has no separate "compliance" characteristic. NFR-20 is a legal
requirement; it is classified under security because it governs where personal data may be
held, and it also appears as architectural constraint AC-7.

---

## 5. Architectural constraints

| ID | Constraint | Reason |
|---|---|---|
| AC-1 | Backend in **Go**; frontend in **React** (TypeScript). | Team skills and the course decision (T). |
| AC-2 | **Client–server web architecture**: browser SPA ↔ REST/JSON API over HTTPS. | Same system for customers and box office staff (Q5); browsers on all devices (Q20). |
| AC-3 | A relational DBMS with ACID transactions and row-level locking (**PostgreSQL**). | Needed for NFR-04 (no double sale) and NFR-05 (payment consistency). |
| AC-4 | Card payments only through an **external payment provider** with a hosted payment page; the system receives only the payment result. | BR-6, NFR-10 (Q8). |
| AC-5 | The courier service is an **external HTTP API**; the system sends requests and receives status updates. | Q5, Q15. |
| AC-6 | External providers are not available to the course team; they are replaced by **mock adapters** with the same interface. | No contracts or budget for real providers (Q21). |
| AC-7 | Hosting on servers in Russia; deployment in **Docker** containers. | NFR-18, NFR-20 (Q7). |
| AC-8 | Only free and open-source software. | Zero budget (Q21). |
| AC-9 | Team of 3–4 students; first release by **December 2026**. | Course schedule (Q6); team size (Q21). |

## 6. Release plan and acceptance

- **Release 1 (December 2026, AC-9):** all Must requirements; Should requirements as time allows.
- **Later releases:** remaining Should and Could requirements. Won't items are out of scope.

The release is accepted when all **Must** requirements pass their acceptance tests, NFR-04 passes
the parallel-purchase test, and the demo scenario (browse → select seats → book → pay → see
history) runs end to end.

## 7. Traceability summary (Lab 2 → Lab 3)

| Lab 2 question | Requirements |
|---|---|
| Q1 | BR-1, FR-07, FR-29 |
| Q2 | BG-1, BG-2, BG-3, NFR-13 |
| Q3 | BG-4, BR-3, FR-09, FR-30, FR-31, FR-32 |
| Q4 | MoSCoW priorities (Must = day-one list), FR-16, FR-18, FR-26, FR-29, FR-38, FR-40 |
| Q5 | FR-15, FR-21, FR-22, FR-39, FR-46, NFR-11, NFR-12, NFR-17, NFR-19, AC-2, AC-5 |
| Q6 | NFR-18, AC-9 |
| Q7 | NFR-08, NFR-09, NFR-20, NFR-23, AC-7 |
| Q8 | BR-6, FR-24, FR-45, NFR-05, NFR-10, AC-4 |
| Q9 | BR-5, FR-23, FR-33, NFR-12, NFR-16 |
| Q10 | FR-01, FR-02, FR-03, FR-04, FR-06, FR-10, FR-17, NFR-13 |
| Q11 | FR-05, FR-34, FR-35, FR-37, FR-42 |
| Q12 | BR-2, FR-14, NFR-16 |
| Q13 | BR-4, FR-18, FR-19, FR-20, FR-23, FR-44 |
| Q14 | FR-13, FR-36 |
| Q15 | FR-16, FR-26, FR-27, FR-28, FR-47, AC-5 |
| Q16 | FR-38 |
| Q17 | NFR-01, NFR-02, NFR-03, NFR-21 |
| Q18 | NFR-06, NFR-07 |
| Q19 | BG-2, BR-7, FR-11, FR-25, FR-45, NFR-04, NFR-05 |
| Q20 | FR-43, NFR-14, AC-2 |
| Q21 | NFR-17, AC-6, AC-8, AC-9 |
| T (team) | BR-5 (card-only refunds), FR-41, FR-47, NFR-15, NFR-22, AC-1 |
