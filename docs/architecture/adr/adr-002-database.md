# ADR-002: Database

## Status: Accepted

## Context

The application needs a persistent data store for users, referee profiles, matches, role slots, availability, assignments, and audit logs. Data is highly relational: assignments reference role slots, role slots reference matches, audit logs reference multiple users and matches, and eligibility queries join referees with matches across multiple criteria.

The developer is fluent in PostgreSQL and SQL. MongoDB was listed as a passing-familiarity option with an explicit preference for PostgreSQL.

Scale is ~22 users with at most a few hundred matches per season. No sharding or horizontal scaling is required.

Infrastructure is Azure, where Azure Database for PostgreSQL Flexible Server is available at low cost and is covered by the free credit allocation.

## Decision

PostgreSQL 16, hosted on Azure Database for PostgreSQL Flexible Server.

## Consequences

**Positive:**
- The data model is relational and benefits from foreign keys, constraints, and JOIN queries. PostgreSQL handles all of this natively.
- The eligibility engine is expressible as a single SQL query (see `data_model.md`) rather than application-side filtering, which is more correct and more efficient.
- Constraints (`CHECK`, `UNIQUE`, `NOT NULL`, `FOREIGN KEY`) enforce domain invariants at the database level, providing a safety net independent of the application layer.
- Developer fluency means schemas and queries are written quickly and correctly.
- `pgx/v5` provides a first-class Go driver with named parameters, structured scanning, and a connection pool.
- Azure Database for PostgreSQL Flexible Server includes: automated daily backups with point-in-time restore, built-in pgBouncer for connection pooling, automatic minor version updates, and monitoring — all managed, no DBA required.
- SSL connections to Azure PostgreSQL are enforced by default.
- `golang-migrate` with PostgreSQL support handles versioned schema migrations.

**Negative:**
- Requires a persistent managed service (Azure Database for PostgreSQL), unlike a file-based database like SQLite. At this scale, SQLite would technically work, but Azure Flexible Server at the lowest tier (Burstable, 1 vCore, 32 GB) costs approximately $15/month, well within the Azure free credit allocation.
- Schema migrations must be managed explicitly. `golang-migrate` handles this, but the developer must write and test migration files.

## Alternatives Considered

**SQLite**: Would simplify local development and eliminate the managed service cost. However, SQLite does not support the same level of concurrent writes (though 22 users is unlikely to create contention), and Azure deployment would require a persistent volume mount rather than a managed service. Given the free credit availability and the developer's PostgreSQL fluency, the managed service is the right call.

**MongoDB**: The developer has passing familiarity and the PRD explicitly notes a preference for PostgreSQL. The relational data model (especially the eligibility queries spanning referees, profiles, matches, and slots) is a poor fit for a document store without significant application-side JOIN logic.

**CockroachDB / PlanetScale / Neon**: Distributed or serverless SQL options. Unnecessary complexity for 22 users. PostgreSQL on Azure is simpler and free under the credit allocation.
