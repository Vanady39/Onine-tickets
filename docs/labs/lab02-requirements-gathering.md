# Lab 2 — Requirements Gathering: Questions to the Customer

**Project:** Online Tickets (Project 3)
**Customer:** an event-ticketing company that sells tickets for its partner venues (concert halls,
cinemas, theatres, exhibition centres).
**Goal of the interview:** collect high-level requirements: business goals, boundaries and
constraints, functionality and quality attributes.

Question type: **O** = open-ended (explore), **C** = closed-ended (confirm one specific thing).

## 1. Questions

### A. Business goals

| # | Type | Question | Why we ask |
|---|---|---|---|
| Q1 | O | How do customers buy tickets today, and what problems does this cause for them and for you? | Understand the current process and the customer's main pains. |
| Q2 | O | What would make you say in one year that the system is a success? Please name numbers you would measure. | Get success metrics that become business goals and quality attributes. |
| Q3 | O | Why do you want to give special rights to loyal customers, and what do you expect them to do differently? | Understand the business value behind the loyalty and season ticket rules. |
| Q4 | O | If we cannot deliver everything in the first release, which features must customers have on day one, and which can wait? | Get the input for MoSCoW prioritisation. |

### B. Boundaries and constraints

| # | Type | Question | Why we ask |
|---|---|---|---|
| Q5 | O | Which systems do you already use (box office software, payment provider, courier company, CRM) and which of them must the new system connect to? | Find the system boundary and external interfaces. |
| Q6 | C | Is there a fixed launch date for the first release? | Time constraint. |
| Q7 | C | Must customers' personal data be stored on servers in Russia (Federal Law 152-FZ)? | Regulatory and hosting constraint. |
| Q8 | C | Must customers enter card details only on an external payment provider's page? | Security/regulatory constraint (PCI DSS scope). |
| Q9 | O | Which business rules about refunds and cancellations must the system follow? | Business rules that limit the purchase flow. |

### C. Functionality

| # | Type | Question | Why we ask |
|---|---|---|---|
| Q10 | O | Walk us through how a customer chooses an event, a session and a seat. What information must they see at each step? | Main purchase flow and event/schedule data. |
| Q11 | O | How are seat prices set for a cinema, theatre or concert hall? Who enters the seat maps and the prices? | Seat-dependent pricing and the event manager role. |
| Q12 | C | Is the limit of three tickets per event counted per order, or per customer for the whole event? | Clarify the ticket limit rule. |
| Q13 | O | Until when does a booking hold the seats, and what happens to it if the customer does not pay in time? | Booking code rules and booking expiry. |
| Q14 | O | Which add-on services can be sold with exhibition tickets, and do they have their own capacity limits? | Exhibition add-ons. |
| Q15 | C | Do you need delivery of printed tickets by courier in addition to electronic tickets? | Delivery options and the courier integration. |
| Q16 | O | Which reports or statistics do your managers need from the system? | Back-office functionality. |

### D. Quality attributes

| # | Type | Question | Why we ask |
|---|---|---|---|
| Q17 | O | How many people do you expect at the moment a popular concert goes on sale, and how fast must the seat map respond to them? | Performance and scalability targets. |
| Q18 | C | Is it acceptable that the site is unavailable for up to 2 hours per month for maintenance? | Availability target and maintenance window. |
| Q19 | C | Must the same seat never be sold to two customers, even under peak load? | Consistency (data integrity) requirement. |
| Q20 | C | Do customers need to use the system from mobile phones? | Usability / portability. |

### E. Budget and resources

| # | Type | Question | Why we ask |
|---|---|---|---|
| Q21 | O | What budget and people are available to build and run the system, and how does this limit paid software or external services? | Budget and resource constraints. |

## 2. Elicitation results (assumed answers)

There is no real customer for the course project, so the team played the customer role. The
answers below are our working assumptions; they are the input for the requirements list in Lab 3.

| # | Answer (assumption) |
|---|---|
| Q1 | Customers queue at box offices or call by phone; they cannot see free seats; many tickets stay unsold; there is no customer history. |
| Q2 | 40 % of all tickets sold online within 12 months; zero double-sold seats; a typical purchase (from the event page to confirmation) takes under 3 minutes. |
| Q3 | To keep customers who come often: a loyal customer (more than 3 paid orders) can buy a season ticket, one order with tickets for sessions of 2 or more different events. |
| Q4 | Day one: event programme, seat selection, card and cash payment, booking with a code, electronic tickets, purchase history. Later: courier delivery, reports, recommendations. |
| Q5 | Card payments through an external payment provider; a partner courier company with an HTTP API; box office cashiers will use the same system through a staff interface. |
| Q6 | Yes — end of the semester (December 2026). |
| Q7 | Yes. |
| Q8 | Yes; the system must never receive or store card numbers. |
| Q9 | A paid ticket can be refunded up to 24 hours before the session; after that no refunds. |
| Q10 | Programme with filters by type and date → event page with description and sessions → hall seat map with free/taken seats and prices → payment method → delivery option → confirmation. |
| Q11 | An administrator enters the seat map of each hall once; seats belong to price zones; the event manager sets a price per zone for each session. |
| Q12 | Up to 3 tickets for one event in one order, for every customer (regular and loyal); loyal customers additionally get season tickets. |
| Q13 | The booking is valid until 24 hours before the session starts. After that it expires and the seats return to sale. The customer can also cancel a booking earlier. |
| Q14 | Business lunch, masterclass, presentation; each add-on has a limited number of places per session. |
| Q15 | Yes: electronic ticket (default) or courier delivery inside the city for an extra fee. |
| Q16 | Tickets sold and revenue per event and per session; hall occupancy. |
| Q17 | Up to 2 000 users at the same time; the seat map must open in under 1 second. |
| Q18 | Yes, if the maintenance is at night (02:00–05:00 MSK). Outside this window the site should be up at least 99.5 % of the month; after a failure, losing at most one day of data is acceptable. |
| Q19 | Yes, this is critical. While a customer pays by card, their seats may be held for them for up to 10 minutes. |
| Q20 | Yes, phones and desktops. |
| Q21 | There is no money budget; the system is built by a team of 3–4 students during the semester. Only free and open-source software can be used. There are no contracts with a real payment provider or courier company, so test (mock) versions of these services are acceptable. |

## 3. Requirements candidates found

| Source | Candidate requirement |
|---|---|
| Q10, Q11 | Browse programme; view event details and sessions; seat map with seat-dependent prices. |
| Q12 | Limit of 3 tickets for one event in one order, for every customer. |
| Q3 | Season ticket (sessions of 2 or more different events in one order) for loyal customers (more than 3 paid orders). |
| Q13 | Booking with code, valid until 24 hours before the session; the customer can cancel it. |
| Q8, Q5 | Card payment on the payment provider's page; cash payment at the box office. |
| Q15 | E-ticket or courier delivery; delivery request sent to the courier service. |
| Q14 | Exhibition add-on services with capacity. |
| Q9 | Refund up to 24 hours before the session. |
| Q1, Q16 | Purchase history with statuses; sales reports for managers. |
| Q17–Q20 | Performance, availability, consistency (10-minute seat hold during card checkout), responsive UI. |
| Q6–Q8 | Deadline, personal data stored in Russia, no card data storage. |
| Q21 | Zero budget: free and open-source software only; mock payment and courier services; team of 3–4 students. |
| Q2, Q4 | Business goals with metrics (online share, zero double sales, purchase time); day-one scope for MoSCoW. |
