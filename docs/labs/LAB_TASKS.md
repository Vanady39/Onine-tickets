# Lab Assignments — Grading Rubric (F26 Software Architecture)

Summary of each lab task as published on Moodle (course id 3704). Reviewers check the lab
documents against this file. Each lab: 1 point main assignment + 1 point extra question.

Case study: **Project 3 — Online Tickets** (see `docs/PROJECT_BRIEF.md`).

## Lab 1 — Introduction: selection of case study + group forming (Week 1)
- Form a group of 3–4 students; choose one of the 10 listed case studies (no others accepted).
- Team and case study are final for the rest of the course.

## Lab 2 — Requirements Gathering: questions to the customer (Week 2)
- Prepare a list of questions that help collect **high-level requirements** for successful implementation.
- **Minimum 10 well-formulated questions.**
- Questions must reveal: (a) main **business goals**, (b) **boundaries and constraints**
  (business rules, time, resources, budget, regulatory, technological), (c) **functionality**,
  (d) **quality attributes** (definition of success with metrics).
- Mix open-ended (explore) and closed-ended (confirm) questions. Closed questions: short, clear,
  specific, one thing per question. Ask how to prioritise when there are many needs.
- Recommendation: keep traceability of team contribution (Git, Google Docs, ...).

## Lab 3 — List of Requirements / Terms of Reference (Week 4)
- Create a list of software requirements and document them as a **ToR** with all major sections
  relevant to the project.
- Must include requirements identified during Lab 2.
- Deliverables: **table of prioritised FRs**, **table of prioritised NFRs**, **list of
  architectural constraints**.
- All requirements prioritised with **MoSCoW** (Must / Should / Could / Won't), with justification.
- Good requirement characteristics: atomic, achievable, cohesive, complete, current, independent,
  modifiable, traceable, unambiguous, verifiable (measurable acceptance), necessary.
  "The system shall ..." wording; NFRs measurable (e.g. "≤ 400 ms p95 at 50 RPS"); ISO 25010
  categories for NFRs.

## Lab 4 — Choosing a model and methodology (Week 5)
- For the project describe the most suitable (1) **software life-cycle model** (Build-and-Fix,
  Waterfall, Rapid prototyping, Incremental, Spiral, ...) and (2) **development methodology**
  (RUP, Scrum, XP, ...).
- Justify the choice with strong arguments covering: product/project type, project scale, team size,
  team competence, business factors (bureaucracy), artifacts necessity, technological factors
  (own aspects encouraged).
- Reference tool: Rockwood, "Choose Your Weapon Wisely" — 11-question tally (RUP/MSS/TSP/XP/Scrum
  scored 1–3 per question; highest total = most suitable).

## Lab 5 — Use Case Diagram (Week 6)
- UML use case diagram for the project:
  - **≥ 3 actors**;
  - **≥ 2 components (subsystems)** of the system shown;
  - **≥ 10 use cases**;
  - all relationship types: **Association, Include, Extend, Generalization**.
- Tools: draw.io / PlantUML / StarUML.
- Workflow table per use case (step, actor action, system response, notes; main + alternative path)
  as shown in the lab example.

## Lab 6 — UML Class Diagram (Week 7, opens 22 Oct 2026 19:00 — prepared from lecture material)
- Lab plan: "Detailed class diagram: choice of architecture (final, with justification) — OOAD".
- Lecture method: noun extraction from use cases → preliminary list of classes (filter out
  attributes and out-of-scope nouns) → class diagram with attributes, methods, relationships,
  cardinalities, inheritance.
- Checklist (from the Lab 6 task as quoted in docs/diagrams/class-diagram-simple.puml; confirm
  on Moodle): ≥ 10 classes, each with name, attributes and methods (parameters, visibility);
  ≥ 4 kinds of relationships; multiplicity on every association; parameter direction
  (`in` / `out` / `inout`).
