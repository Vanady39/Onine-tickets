# Lab 4 — Choosing a Life-Cycle Model and a Development Methodology

**Project:** Online Tickets (Project 3)

## 1. Decision

| | Choice |
|---|---|
| Software life-cycle model | **Incremental model** (iterative-incremental delivery) |
| Development methodology | **Scrum** for project management, with selected **XP engineering practices** (automated tests, continuous integration, small releases, collective code ownership, refactoring) |

A life-cycle model fixes *which phases exist and in which order*; a methodology prescribes *how
the team works day to day* to move the system through these phases. We choose them separately
below.

## 2. Project profile (inputs to the decision)

| Aspect | Online Tickets |
|---|---|
| Product / project type | Business web application (B2C e-commerce) with payments. Not life-critical; money and seat consistency are business-critical. |
| Project scale | Small to medium: 42 in-scope functional and 21 non-functional requirements, plus 6 "Won't" items (Lab 3); estimated 15–25 thousand lines of Go and TypeScript. |
| Team size | 3–4 students, one team, no separate testers. |
| Team competence | Mixed: one member has built and deployed web services before; the others know Go and React from courses but have little production experience. With 3–4 members this is 25–33 % "competent and experienced". |
| Business factors | No real bureaucracy; the "customer" is the instructor plus our own assumptions (Lab 2). Weekly labs give a fixed rhythm of reviews. Fixed end date (December 2026). |
| Artifacts necessity | High. Final documentation is 30 % of the course grade and lab deliverables another 25 %; later labs still require sequence/state diagrams, an implementation, testing and maintenance plan, design patterns and user documentation. Artifacts are produced one per week, in step with the labs, not up front. |
| Requirements stability | Medium-low: requirements come from assumed answers and change after each lab and instructor feedback. |
| Technological factors | Known stack (Go, React, PostgreSQL, Docker); external payment and courier systems are mocked behind interfaces (AC-6), so integration risk is low and can be postponed. |

## 3. Choice of the life-cycle model

| Model | Fit for Online Tickets | Verdict |
|---|---|---|
| Build-and-Fix | No planning or documentation. Unsuitable for a non-trivial system with payments and a graded ToR. | Rejected |
| Waterfall | Needs stable requirements up front; ours change after every lab. Working software appears only at the end, so the risk of missing customer needs is high. | Rejected |
| Rapid prototyping | Useful for checking the seat-map UI with the customer, but not a full model on its own; prototype code tends to be reused when it should be rewritten. | Used only as a technique (UI mock-ups) |
| Spiral | Strong risk analysis, but designed for large internal projects; formal risk management for each loop is too heavy for 3–4 students. Our main risks (double sale, payment consistency) are known and handled by design. | Rejected |
| **Incremental** | Delivers a working product early and grows it: browse → seat selection and booking → card payment and loyalty → add-ons, delivery and back office. Each increment is demo-able at a lab. Matches the MoSCoW priorities of Lab 3: all Must functional requirements are in I1–I3. Requires an open, modular architecture — we plan one (layers with interfaces for payment and courier). | **Selected** |

**Planned increments**

| Increment | Content (Lab 3 IDs) | Priority covered |
|---|---|---|
| I1 | Programme, event page, sessions, seat map, accounts; event and session management for the event manager (FR-01…FR-08, FR-34, FR-35). Venues and halls come from seed data until FR-37. | Must (FR-02 Should) |
| I2 | Seat selection and 10-minute hold, general-admission tickets, limit of 3 tickets per event, payment-method choice, order summary, booking (only more than 24 h before the session) with booking code and expiry, cash orders turned into bookings, cash payment at the box office, booking cancellation (FR-10…FR-12, FR-14, FR-15, FR-17…FR-19, FR-21…FR-23, FR-44, FR-46); no double sale (NFR-04) | Must (FR-23 Should) |
| I3 | Card payment via the provider adapter, including paying for a booking online and auto-refund of a payment confirmed after expiry; e-tickets, purchase history, loyalty status (FR-09, FR-20, FR-24, FR-25, FR-28…FR-30, FR-45); payment consistency (NFR-05) | Must (FR-09 Should) |
| I4 | Add-ons, delivery options and the courier adapter, season tickets, refunds, venue/hall and add-on administration, sales report, box-office sale without booking (FR-13, FR-16, FR-26, FR-27, FR-31…FR-33, FR-36…FR-39) | Should / Could |

Together the four increments cover every in-scope functional requirement (FR-01…FR-39, FR-44…FR-46); the "Won't"
items (FR-40…FR-43, NFR-21, NFR-22) are not planned. The other non-functional requirements are
cross-cutting: each enters the Definition of Done from the increment where it first applies (for
example security NFR-08…NFR-11 and NFR-23 from I1).

The main disadvantage of the incremental model — it can degrade into Build-and-Fix without an
open architecture — is addressed by fixing the architecture before I2 (class diagram lab) and by
keeping integrations behind interfaces.

## 4. Choice of the methodology

### 4.1 "Choose Your Weapon Wisely" questionnaire (Rockwood)

Scores are taken from the weight tables in the handbook (1 = least suitable, 3 = most suitable).

| # | Question | Our answer | RUP | MSS | TSP | XP | Scrum |
|---|---|---|---|---|---|---|---|
| 1 | Team size (developers per team) | a) less than 10 (3–4) | 1 | 1 | 3 | 3 | 3 |
| 2 | Total developers in the project | a) less than 40 | 2 | 2 | 2 | 2 | 2 |
| 3 | Product size and complexity | a) small–medium program: ~20 KLOC is above "a few thousand" but far below option b (hundreds of thousands), so a is the closest | 1 | 1 | 1 | 3 | 3 |
| 4 | Competent and experienced developers | c) 25–33 % (one of 3–4 members)¹ | 2 | 2 | 2 | 2 | 2 |
| 5 | Level of "hacker sentiment" | b) plan and design enough, but don't aim for perfection before coding | 2 | 2 | 1 | 2 | 2 |
| 6 | Management style | a) macro-management: few guidelines, the team self-organises | 2 | 3 | 1 | 3 | 3 |
| 7 | Organisation-wide processes | c) not important (one-semester student team) | 2 | 2 | 2 | 3 | 3 |
| 8 | Confidence needed to adopt a new process | b) fairly mature process is enough | 2 | 2 | 2 | 2 | 2 |
| 9 | Type of product | b) non-life-critical but money-handling (mission-critical for the business) | 2 | 2 | 2 | 2 | 2 |
| 10 | Requirements stability | a) emergent and changing | 2 | 2 | 2 | 3 | 3 |
| 11 | Requirements traceability and formal docs | b) somewhat important (graded ToR and diagrams) | 2 | 2 | 2 | 1 | 1 |
| | **Total** | | **20** | **21** | **20** | **26** | **26** |

¹ With exactly 3 members the share is 33 %, on the border with option d (33–100 %). Option d gives
RUP 21, MSS 22, TSP 19, XP 27, Scrum 27 — the ranking does not change.

XP and Scrum are clearly ahead (26 vs 20–21). The only question where they score below the heavier
processes is traceability (Q11); we handle it with the Definition of Done below.

XP and Scrum tie, so the tally alone does not decide between them. We take **Scrum** as the
management framework because its events map directly onto the lab schedule (sprint review = lab demo), and XP's own
management practices do not fit a student team (no on-site customer, no shared full-time
schedule for pair programming). Scrum, in turn, says nothing about engineering, so we adopt XP's
**engineering practices**.

### 4.2 Comparison of the candidate methodologies

| Methodology | Advantages for us | Disadvantages for us |
|---|---|---|
| RUP | Strong architecture focus and UML artifacts — close to what the course grades. | Heavyweight: dozens of roles and artifacts for 3–4 people; needs training and tooling; slow reaction to changing requirements. |
| Scrum | Short sprints fit the lab schedule; product backlog maps directly to MoSCoW; sprint review = lab demo; adapts to changing requirements; minimal roles. | Says nothing about engineering practices; little documentation by default. |
| XP | Automated tests, CI and refactoring protect the tricky rules (seat consistency, limits, expiry). | Needs an on-site customer (we do not have one); pair programming is hard with student schedules; weak documentation. |

### 4.3 Why Scrum + XP practices

1. **Product/project type.** A web product with a visible UI benefits from frequent demos and
   feedback; nothing in it requires the formal verification that TSP/RUP provide.
2. **Scale and team size.** 3–4 developers fits Scrum easily (the Scrum Guide recommends 10 or
   fewer people). RUP's dozens of roles would give each student several roles at once.
3. **Team competence.** With 25–33 % experienced members, XP's automated tests and code review
   (instead of strict pair programming) let less experienced members work safely.
4. **Business factors.** No bureaucracy and no contract compliance; the instructor acts as the
   Product Owner's stakeholder at weekly labs.
5. **Artifacts necessity.** Written deliverables carry 55 % of the course grade (lab projects 25 % +
   final documentation 30 %), so documentation tasks are backlog items with the same priority as code, and
   the Definition of Done requires updating the ToR and diagrams. This covers the traceability
   weakness of agile methods without adopting RUP.
6. **Technological factors.** Go and React have fast test runners and simple CI (GitHub Actions),
   which makes XP's continuous integration cheap. Mock adapters for payment and courier let us
   test every increment without external systems. Performance targets (NFR-01, NFR-03) are
   checked by a load test in the CI pipeline from I2 onwards, so the risk is found early, not at
   the end.
7. **Changing requirements.** Assumed customer answers will be corrected; Scrum's backlog
   refinement absorbs these changes every sprint.

## 5. How we apply it

| Element | Our setup |
|---|---|
| Sprint length | 2 weeks (two labs per sprint); one increment per sprint (section 3). |
| Roles | Product Owner — team lead (represents the customer, keeps the backlog); Scrum Master — rotates each sprint; Developers — all members. |
| Events | Sprint planning at the start of a sprint; short async daily stand-up in Telegram; sprint review = demo at the lab; retrospective after each review. |
| Artifacts | Product backlog = Lab 3 requirements with MoSCoW priority (GitHub Issues); sprint backlog = GitHub Project board; increment = tagged release. |
| XP practices | Unit tests for domain rules before or with the code (≥ 70 % coverage, NFR-16); CI on every pull request (build, tests, lint); code review of every pull request; refactoring; small releases. |
| Definition of Done | Code merged with review; tests pass in CI; acceptance criteria of the requirement met; ToR / UML diagrams updated if the change affects them; traceability (requirement ID in the issue and pull request title). |
