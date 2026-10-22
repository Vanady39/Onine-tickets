# Lab 1 — Case Study Selection and Team

## 1. Team

| Role in project | Name | Contact |
|---|---|---|
| Team lead / architect | _to fill in_ | _@telegram_ |
| Backend developer (Go) | _to fill in_ | |
| Frontend developer (React) | _to fill in_ | |
| QA / documentation | _to fill in_ | |

Group size: 3–4 students (course rule). The team and the case study are final for the rest of the
course.

## 2. Selected case study

**Project 3 — Online Tickets** (one of the 10 case studies offered by the course): an online
system for booking and selling tickets to concerts, cinema, theatre performances and exhibitions.
Full original statement: `docs/PROJECT_BRIEF.md`.

## 3. Why this case study

1. **Clear business domain.** Every team member has bought tickets online, so we can play both the
   customer and the analyst role and check requirements against real experience.
2. **Rich architecture problems in a small scope.** Seat reservation under concurrent demand,
   time-limited bookings, payment transactions and an external courier service give real
   quality-attribute trade-offs (consistency, availability, security) in a software-only system.
3. **Good fit for every lab.** Several actor types (guest, regular and loyal customer, cashier,
   event manager, administrator, payment provider, courier service) make use case, class,
   sequence and state diagrams meaningful.
4. **Fits our skills.** The team works with Go and React; the system is a classic web
   client–server application that we can implement and demo by the end of the course. The payment
   provider and the courier service are mocked, so the demo needs no external contracts.

## 4. Initial understanding of the system

| Item | Summary |
|---|---|
| Problem | Customers want to see what events are on and buy a specific seat without queuing at the box office; organisers want to sell more tickets and know their customers. |
| Event types | Concert, film (cinema), theatre performance, exhibition. |
| Key objects | Event, session (date, time, venue, hall), hall seat map with price zones, ticket, booking with booking code, order, payment, delivery request, season ticket, exhibition add-on service. |
| Customer types | Guest (not registered), regular customer, loyal customer (more than 3 paid orders). |
| Key rules | Every customer (regular or loyal): at most 3 tickets for one event in one order; pays by bank card online or in cash at the box office. Loyal customer: can additionally buy a season ticket covering sessions of 2 or more different events. A booking (with its booking code) holds the seats until 24 hours before the session start; then it expires. Tickets paid by card can be delivered by courier. |
| External systems | Payment provider (bank card payments), courier service (ticket delivery); both are mocked in our implementation. |

## 5. Glossary

| Term | Meaning in this project |
|---|---|
| Event | A show offered for sale: a concert, film, theatre performance or exhibition, with a description. |
| Programme | The list of upcoming events offered for sale. |
| Session | One scheduled occurrence of an event: date, start time, venue and hall, ticket prices. |
| Venue | A building where events take place (concert hall, cinema, theatre, exhibition centre); it has one or more halls. |
| Hall | A room of a venue where a session happens. It has a seat map (rows, seats, price zones) or, for exhibitions, a general-admission capacity. |
| Price zone | A group of seats with the same price for a session (e.g. stalls, balcony, VIP). |
| Ticket | The right to attend one session: one seat in a seated hall, or one general-admission place at an exhibition. |
| Add-on service | Extra paid service attached to an exhibition ticket: business lunch, masterclass, presentation. |
| Seat hold | A 10-minute hold on the seats of an order the customer confirmed to pay by card now; if the payment is not confirmed in time, the seats return to sale. |
| Booking | An unpaid order whose tickets are reserved for the customer until 24 hours before the session start; then it expires and the seats return to sale. Identified by a booking code. |
| Booking code | Short unique code issued for a booking; the customer uses it to pay online or shows it to the cashier to pay in cash at the box office. |
| Order | A set of tickets a customer books or buys in one checkout: tickets for one session, or tickets for sessions of 2 or more different events (season ticket). At most 3 tickets for one event per order. |
| Purchase | A completed (paid) order. Purchases count towards loyalty; refunded orders do not. |
| E-ticket | Electronic ticket issued after payment; the default delivery option. |
| Season ticket | An order, available only to loyal customers, that covers tickets for sessions of 2 or more different events. |
| Regular customer | A registered customer with 0–3 completed purchases (paid orders). |
| Loyal customer | A registered customer with more than 3 completed purchases (paid, not refunded orders). |
| Box office | A physical ticket desk where a cashier accepts cash payments. |
| Payment provider | External system that charges bank cards and confirms payments. |
| Courier service | External system that delivers printed tickets to the customer's address. |
